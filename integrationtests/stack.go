package integrationtests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"net/http"
	"os"
	"testing"

	"github.com/ATMackay/eth-proxy/service"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

//
// go-ethereum's simulated package allows us to replicate the behavior of a user-configured geth execution client in memory
// which in turn allows us to test the endpoints of our eth-proxy service end-to-end.
//

const simulatedChainID = 1337

var (
	oneEther  = big.NewInt(params.Ether)
	dummyAddr = "0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
)

func executeRequest(methodType, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func makeEthProxyService(t testing.TB) *svcStack {

	bk := newEthBackend(t, common.HexToAddress(dummyAddr))

	t.Cleanup(func() { bk.Close() })

	// create proxy service
	cfg := &service.Config{
		Port:      8080,
		LogLevel:  "error", // change to 'info' or 'debug' to see the proxy service logs
		LogFormat: "plain",
	}

	l, err := service.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		t.Fatal(err)
	}

	ethClient := bk.Client()

	svc := service.New(cfg.Port, l, ethClient)

	svc.Start()

	t.Cleanup(func() { svc.Stop(os.Kill) })

	return &svcStack{
		node:    &ethereumStack{bk},
		service: svc,
	}

}

type blockchainBackend struct {
	*simulated.Backend
	bankAccount *eoa
}

func newEthBackend(t testing.TB, accounts ...common.Address) *blockchainBackend {

	t.Helper()

	// create new chain & backend with a pre-filled genesis account
	bankAccount := createEOA(t)

	log.SetDefault(log.NewLogger(log.DiscardHandler()))

	backend := &blockchainBackend{
		Backend: simTestBackend(bankAccount.From),
	}
	backend.bankAccount = bankAccount
	t.Cleanup(func() { backend.Close() })

	return backend
}

func simTestBackend(testAddr common.Address) *simulated.Backend {
	return simulated.NewBackend(
		core.GenesisAlloc{
			testAddr: {Balance: oneEther},
		},
	)
}

type eoa struct {
	*bind.TransactOpts
	PrivateKey *ecdsa.PrivateKey
}

func createEOA(t testing.TB) *eoa {
	t.Helper()
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(priv, big.NewInt(simulatedChainID))
	if err != nil {
		t.Fatal(err)
	}
	return &eoa{
		PrivateKey:   priv,
		TransactOpts: opts,
	}
}

type ethereumStack struct {
	backend *blockchainBackend
}

type svcStack struct {
	node    *ethereumStack
	service *service.Service
}

// TODO - create more scenarios in the stack to test
//
//func (b *blockchainBackend) newTx() (*types.Transaction, error) {
//
//	client := b.Client()
//	key := b.bankAccount.PrivateKey
//
//	// create a signed transaction to send
//	head, err := client.HeaderByNumber(context.Background(), nil) // Should be child's, good enough
//	if err != nil {
//		return nil, err
//	}
//	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(params.GWei))
//	addr := crypto.PubkeyToAddress(key.PublicKey)
//	chainid, err := client.ChainID(context.Background())
//	if err != nil {
//		return nil, err
//	}
//	nonce, err := client.PendingNonceAt(context.Background(), addr)
//	if err != nil {
//		return nil, err
//	}
//	tx := types.NewTx(&types.DynamicFeeTx{
//		ChainID:   chainid,
//		Nonce:     nonce,
//		GasTipCap: big.NewInt(params.GWei),
//		GasFeeCap: gasPrice,
//		Gas:       21000,
//		To:        common.HexToAddress(dummyAddr),
//	})
//	return types.SignTx(tx, types.LatestSignerForChainID(chainid), key)
//}
