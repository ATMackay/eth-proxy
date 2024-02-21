package integrationtests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"net/http/httptest"
	"net/rpc"
	"os"
	"testing"

	"github.com/ATMackay/eth-proxy/service"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

//
// This package uses go-ethereum's simulated package to replicate the behavior of a geth execution client
// which in turn allows us to test the endpoints of our eth-proxy service end-to-end.
//

const simulatedChainID = 1337

var (
	oneEther  = big.NewInt(params.Ether)
	dummyAddr = "0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
)

func makeEthProxyService(t *testing.T) *svcStack {

	bk := newEthBackend(t, common.HexToAddress(dummyAddr))

	// create proxy service
	cfg := &service.Config{
		Port:      8080,
		LogLevel:  "info",
		LogFormat: "plain",
		URLs:      bk.Server.URL,
	}

	l, err := service.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		t.Fatal(err)
	}

	multiClient, err := service.NewMultiNodeClient(cfg.URLs, service.NewEthClient)
	if err != nil {
		panic(err)
	}

	svc := service.New(cfg.Port, l, multiClient)

	svc.Start()

	t.Cleanup(func() { svc.Stop(os.Kill) })

	return &svcStack{
		node:    &ethereumStack{bk},
		service: svc,
	}

}

type blockchainBackend struct {
	*simulated.Backend
	BankAccount *EOA
	ChainID     int
	Server      *httptest.Server
}

func newEthBackend(t *testing.T, accounts ...common.Address) *blockchainBackend {

	t.Helper()

	// create new chain with pre-filled genesis accounts
	genesis := make(map[common.Address]core.GenesisAccount)
	for _, account := range accounts {
		genesis[account] = core.GenesisAccount{Balance: oneEther}
	}
	bankAccount := createEOA(t)
	genesis[bankAccount.From] = core.GenesisAccount{Balance: oneEther}

	log.SetDefault(log.NewLogger(log.DiscardHandler()))

	backend := &blockchainBackend{
		Backend: simulated.NewBackend(genesis),
	}
	t.Cleanup(func() { backend.Close() })

	server := httptest.NewServer(newTestServer(backend))

	t.Cleanup(func() { server.Close() })

	return &blockchainBackend{
		BankAccount: bankAccount,
		ChainID:     simulatedChainID,
		Server:      server,
	}
}

type EOA struct {
	*bind.TransactOpts
	PrivateKey *ecdsa.PrivateKey
}

func createEOA(t *testing.T) *EOA {
	t.Helper()
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(priv, big.NewInt(simulatedChainID))
	if err != nil {
		t.Fatal(err)
	}
	return &EOA{
		PrivateKey:   priv,
		TransactOpts: opts,
	}
}

func newTestServer(backend *blockchainBackend) *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName("eth", &backendAPI{backend}); err != nil {
		panic(err)
	}
	return server
}

type backendAPI struct {
	b *blockchainBackend
}

type ethereumStack struct {
	backend *blockchainBackend
}

type svcStack struct {
	node    *ethereumStack
	service *service.Service
}

func (b *blockchainBackend) fund(t *testing.T, to common.Address, amount *big.Int) string {

	t.Helper()

	signedTx := b.makeTx(t, b.BankAccount, &to, amount, nil)
	if err := b.Client().SendTransaction(context.Background(), signedTx); err != nil {
		t.Fatal(err)
	}
	b.Commit()
	return signedTx.Hash().Hex()
}

func (b *blockchainBackend) makeTx(t *testing.T, sender *EOA, to *common.Address, value *big.Int, data []byte) *types.Transaction {

	t.Helper()

	signedTx, err := types.SignTx(b.makeUnsignedTx(t, sender.From, to, value, data), types.LatestSignerForChainID(big.NewInt(int64(b.ChainID))), sender.PrivateKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}
	return signedTx
}

func (b *blockchainBackend) makeUnsignedTx(t *testing.T, from common.Address, to *common.Address, value *big.Int, data []byte) *types.Transaction {
	t.Helper()

	nonce, err := b.Client().PendingNonceAt(context.Background(), from)
	if err != nil {
		t.Fatal(err)
	}
	gasTip, err := b.Client().SuggestGasTipCap(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	gasPrice, err := b.Client().SuggestGasPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	gasPrice = new(big.Int).Add(gasPrice, big.NewInt(1))

	return types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasFeeCap: new(big.Int).Add(gasPrice, gasTip),
		GasTipCap: gasTip,
		Gas:       100000,
		To:        to,
		Value:     value,
		Data:      data,
	})
}
