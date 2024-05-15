package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ATMackay/eth-proxy/internal/stack"
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
}
