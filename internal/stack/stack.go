package stack

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ATMackay/eth-proxy/service"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

//
// go-ethereum's simulated package allows us to replicate the behavior of a user-configured geth execution client in memory
// which in turn allows us to test the endpoints of our eth-proxy service end-to-end.
//

const SimulatedChainID = 1337

var (
	OneEther  = big.NewInt(params.Ether)
	DummyAddr = "0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
)

type SvcStack struct {
	Eth     *EthereumStack
	Service *service.Service
}

type BlockchainBackend struct {
	*simulated.Backend
	BankAccount *EOA
}

func NewEthBackend() (*BlockchainBackend, error) {

	// create new chain & backend with a pre-filled genesis account
	bankAccount, err := createEOA()
	if err != nil {
		return nil, err
	}

	log.SetDefault(log.NewLogger(log.DiscardHandler()))

	backend := &BlockchainBackend{
		Backend: simTestBackend(bankAccount.From),
	}
	backend.BankAccount = bankAccount

	return backend, nil
}

func simTestBackend(testAddr common.Address) *simulated.Backend {
	return simulated.NewBackend(
		types.GenesisAlloc{
			testAddr: {Balance: OneEther},
		},
	)
}

type EOA struct {
	*bind.TransactOpts
	PrivateKey *ecdsa.PrivateKey
}

func createEOA() (*EOA, error) {

	priv, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	opts, err := bind.NewKeyedTransactorWithChainID(priv, big.NewInt(SimulatedChainID))
	if err != nil {
		return nil, err
	}
	return &EOA{
		PrivateKey:   priv,
		TransactOpts: opts,
	}, nil
}

type EthereumStack struct {
	Backend *BlockchainBackend
	Txs     map[uint64]*types.Transaction // in-memory map of confirmed txs
}

func (e *EthereumStack) AddTx() error {
	tx, err := e.Backend.NewTx()
	if err != nil {
		return err
	}

	// send tx
	if err := e.Backend.Client().SendTransaction(context.Background(), tx); err != nil {
		return err
	}

	_ = e.Backend.Commit() // commit transaction, move the chain forward

	blkNum, err := e.Backend.Client().BlockNumber(context.Background())
	if err != nil {
		return err
	}
	e.Txs[blkNum] = tx
	return nil
}

func (b *BlockchainBackend) NewTx() (*types.Transaction, error) {

	client := b.Client()

	key := b.BankAccount.PrivateKey

	// create a signed transaction to send
	head, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(params.GWei))

	fromAddr := crypto.PubkeyToAddress(key.PublicKey)
	chainid, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return nil, err
	}

	bal, err := client.PendingBalanceAt(context.Background(), fromAddr)
	if err != nil {
		return nil, err
	}
	if bal.Cmp(new(big.Int).Mul(gasPrice, big.NewInt(21000))) < 0 {
		return nil, fmt.Errorf("insufficient balance %v below (gasPrice) %v x (gasUnit) %v", bal, gasPrice, 21000)
	}

	// send half of the tx balance
	sendAmount := bal.Div(bal, big.NewInt(2))

	toAddr := common.HexToAddress(DummyAddr)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainid,
		Nonce:     nonce,
		GasTipCap: big.NewInt(params.GWei),
		GasFeeCap: gasPrice,
		Gas:       21000,
		To:        &toAddr,
		Value:     sendAmount,
	})
	return types.SignTx(tx, types.LatestSignerForChainID(chainid), key)
}
