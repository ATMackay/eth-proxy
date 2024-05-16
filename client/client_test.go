package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ATMackay/eth-proxy/internal/stack"
	"github.com/ethereum/go-ethereum/common"
)

func TestClient(t *testing.T) {

	// Use mock stack to execute tests - TODO

	s := stack.MockEthProxyService(t, "error")

	genesisAddr := s.Eth.Backend.BankAccount.From

	time.Sleep(10 * time.Millisecond)
	baseUrl := fmt.Sprintf("http://0.0.0.0%v", s.Service.Server().Addr())

	cl := New(baseUrl)

	ctx := context.Background()

	t.Run("status", func(t *testing.T) {

		stat, err := cl.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(*stat)
	})

	t.Run("health", func(t *testing.T) {

		health, err := cl.Health(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(*health)
	})

	t.Run("balance", func(t *testing.T) {

		bal, err := cl.Balance(ctx, genesisAddr)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(*bal)
	})

	t.Run("tx-by-hash-err", func(t *testing.T) {

		_, err := cl.TransactionByHash(ctx, common.Hash{0})
		if err == nil {
			t.Fatal(err)
		}
		t.Log(err)
	})

	t.Run("tx-by-receipt-err", func(t *testing.T) {

		_, err := cl.TransactionReceipt(ctx, common.Hash{0})
		if err == nil {
			t.Fatal(err)
		}
		t.Log(err)
	})

	// create transaction using backend client

	tx, err := s.Eth.Backend.NewTx()
	if err != nil {
		t.Fatalf("%v", err)
	}

	toAddr := tx.To()
	txHash := tx.Hash()
	amount := tx.Value()

	t.Run("send-tx", func(t *testing.T) {
		t.Logf("sending %v ETH to %v", amount, toAddr.Hex())
		txResp, err := cl.SendTransaction(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("transaction ID: %v", txResp.Txid)
		if g, w := txResp.Txid, txHash.Hex(); g != w {
			t.Fatalf("unexpted txid, got %v want %v", g, w)
		}
	})

	blkHash := s.Eth.Backend.Commit()
	t.Logf("new block: %v", blkHash.Hex())

	t.Run("tx-by-hash", func(t *testing.T) {

		txResp, err := cl.TransactionByHash(ctx, txHash)
		if err != nil {
			t.Fatal(err)
		}
		if g, w := txResp.Txid, txHash.Hex(); g != w {
			t.Fatalf("unexpected txid, got %v want %v", g, w)
		}
	})

	t.Run("tx-by-receipt", func(t *testing.T) {

		rec, err := cl.TransactionReceipt(ctx, txHash)
		if err != nil {
			t.Fatal(err)
		}
		if rec.BlockHash.Cmp(blkHash) != 0 {
			t.Fatalf("unexpted blockHash, got %v want %v", rec.BlockHash.Hex(), blkHash.Hex())
		}
	})
}
