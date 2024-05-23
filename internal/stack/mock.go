package stack

import (
	"os"
	"testing"

	"github.com/ATMackay/eth-proxy/proxy"
	"github.com/ethereum/go-ethereum/core/types"
)

func MockEthProxyService(t testing.TB, logLevel string) *SvcStack {

	bk, err := NewEthBackend()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { bk.Close() })

	// create proxy service
	cfg := &proxy.Config{
		Port:      8080,
		LogLevel:  logLevel, // change to 'info' or 'debug' to see the proxy service logs
		LogFormat: "plain",
	}

	l, err := proxy.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		t.Fatal(err)
	}

	ethClient := bk.Client()

	svc := proxy.New(cfg.Port, l, ethClient)

	svc.Start()

	t.Cleanup(func() { svc.Stop(os.Kill) })

	ethStack := EthereumStack{Backend: bk, Txs: make(map[uint64]*types.Transaction)}

	return &SvcStack{
		Eth:     &ethStack,
		Service: svc,
	}

}
