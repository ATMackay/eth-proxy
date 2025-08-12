package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	yaml "gopkg.in/yaml.v3"
)

const (
	dummyAddr = "0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
	dummyTxid = "0x326c7dbb58eaf646af01f7b6f4fb1e0fb1afe1329ac670ce5945e8fd940ec4d7"
)

var (
	dummyTx = types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1)})
)

// Make sure to write some good tests

var _ SimpleEthClient = (*fakeEthClient)(nil)

type fakeEthClient struct{}

func newFakeEthClient(_ string) (SimpleEthClient, error) {
	return &fakeEthClient{}, nil
}

func newFakeEthClientErr(_ string) (SimpleEthClient, error) {
	return &fakeEthClient{}, errors.New("error")
}

func (f *fakeEthClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (f *fakeEthClient) BlockNumber(context.Context) (uint64, error) {
	return 0, nil
}

func (f *fakeEthClient) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return dummyTx, false, nil
}

func (f *fakeEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (tx *types.Receipt, err error) {
	return &types.Receipt{}, nil
}

func (f *fakeEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}

type fakeEthClientWithErr struct {
	err error
}

func newFakeEthClientWithErr(errMsg string) (SimpleEthClient, error) {
	var embeddedErr error
	if errMsg != "" {
		embeddedErr = errors.New(errMsg)
	}
	return &fakeEthClientWithErr{err: embeddedErr}, nil
}

func (f *fakeEthClientWithErr) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return big.NewInt(0), f.err
}

func (f *fakeEthClientWithErr) BlockNumber(context.Context) (uint64, error) {
	return 0, f.err
}

func (f *fakeEthClientWithErr) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return &types.Transaction{}, false, f.err
}

func (f *fakeEthClientWithErr) TransactionReceipt(ctx context.Context, txHash common.Hash) (tx *types.Receipt, err error) {
	return &types.Receipt{}, f.err
}

func (f *fakeEthClientWithErr) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return f.err
}

type fakeEthClientWithBlock struct {
	fakeEthClient
}

func newFakeEthClientWithBlock(_ string) (SimpleEthClient, error) {
	return &fakeEthClientWithBlock{}, nil
}

func (f *fakeEthClientWithBlock) BlockNumber(context.Context) (uint64, error) {
	return rand.Uint64(), nil
}

func makeTestService(t *testing.T, urls string, constructor func(url string) (SimpleEthClient, error)) *Service {

	l, err := NewLogger("error", "plain")
	if err != nil {
		t.Fatal(err)
	}

	// Use urls to embed err msg (used by )
	cl, err := NewMultiNodeClient(urls, constructor)
	if err != nil {
		t.Fatal(err)
	}

	return New(8080, l, cl)
}

func Test_Logger(t *testing.T) {

	tests := []struct {
		name      string
		loglevel  string
		logformat string
		expectErr bool
	}{
		{
			"normal-info-plain",
			"info",
			"plain",
			false,
		},
		{
			"normal-info-json",
			"info",
			"json",
			false,
		},
		{
			"normal-debug-plain",
			"info",
			"plain",
			false,
		},
		{
			"error-loglevel",
			"invalid",
			"plain",
			true,
		},
		{
			"error-logformat",
			"info",
			"invalid",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewLogger(tt.loglevel, tt.logformat); (err != nil) != tt.expectErr {
				t.Errorf("unexpected error '%v'", err)
			}
		})

	}
}

func Test_StartStop(t *testing.T) {

	srv := makeTestService(t, "-", newFakeEthClient)

	srv.Start()

	srv.Stop(os.Kill)

}

func Test_SantizeConfig(t *testing.T) {

	tests := []struct {
		name           string
		initialConfig  func() Config
		expectedConfig func() Config
	}{
		{
			"empty",
			func() Config {
				return emptyConfig
			},
			func() Config {
				return defaultConfig
			},
		},
		{
			"empty-with-port",
			func() Config {
				cfg := emptyConfig
				cfg.Port = 1
				return cfg
			},
			func() Config {
				cfg := defaultConfig
				cfg.Port = 1
				return cfg
			},
		},
		{
			"empty-with-log-level",
			func() Config {
				cfg := emptyConfig
				cfg.LogLevel = "debug"
				return cfg
			},
			func() Config {
				cfg := defaultConfig
				cfg.LogLevel = "debug"
				return cfg
			},
		},
		{
			"empty-with-log-format",
			func() Config {
				cfg := emptyConfig
				cfg.LogFormat = "json"
				return cfg
			},
			func() Config {
				cfg := defaultConfig
				cfg.LogFormat = "json"
				return cfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.initialConfig()
			c.Sanitize()
			b, _ := yaml.Marshal(c)
			e, _ := yaml.Marshal(tt.expectedConfig())
			if !bytes.Equal(b, e) {
				t.Errorf("returned config not equal to default")
			}
		})
	}
}

func Test_MultiNodeClient(t *testing.T) {

	tests := []struct {
		name              string
		urls              func() string
		constructor       func(string) (SimpleEthClient, error)
		expectErr         bool
		expectedNodeCount int
	}{
		{
			"single",
			func() string { return "testurl" },
			newFakeEthClient,
			false,
			1,
		},
		{
			"double",
			func() string { return "testurl1,testurl2" },
			newFakeEthClient,
			false,
			2,
		},
		{
			"many",
			func() string {
				var urls string
				for i := 0; i < 100; i++ {
					urls += fmt.Sprintf("url%d,", i)
				}
				return strings.TrimRight(urls, ",")
			},
			newFakeEthClient,
			false,
			100,
		},
		{
			"error",
			func() string { return "testurl1,testurl2" },
			newFakeEthClientErr,
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl, err := NewMultiNodeClient(tt.urls(), tt.constructor)
			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected error: %v", err)

			}
			if err != nil {
				return
			}
			if g, w := len(cl.nodes), tt.expectedNodeCount; g != w {
				t.Errorf("unexpected node count, got %v, want %v", g, w)
			}
			n := len(cl.nodes) - 1
			if len(cl.nodes) > 1 {
				cl.increaseNodePriority(n, fmt.Sprintf("%d", n))
			}
		})
	}

}

func Test_API(t *testing.T) {

	apiTests := []struct {
		name               string
		urls               string
		serviceConstructor func(urls string) *Service
		endpoint           func() string
		methodType         string
		expectedResponse   any
		expectedCode       int
	}{
		//
		// READ REQUESTS
		//
		{
			"status",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return StatusEndPnt },
			http.MethodGet,
			&StatusResponse{Message: "OK", Version: Version, Service: ServiceName},
			http.StatusOK,
		},
		{
			"health",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return HeathEndPnt },
			http.MethodGet,
			&HealthResponse{Version: Version, Service: ServiceName, Failures: []string{}},
			http.StatusOK,
		},
		{
			"eth-balance",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return fmt.Sprintf("%v%v", EthV0BalancePrfx, dummyAddr) },
			http.MethodGet,
			&BalanceResponse{Balance: "0"},
			http.StatusOK,
		},
		{
			"eth-tx",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return fmt.Sprintf("%v%v", EthV0TxPrfx, dummyTxid) },
			http.MethodGet,
			&TxResponse{Tx: dummyTx, Txid: dummyTxid, IsPending: false},
			http.StatusOK,
		},
		{
			"eth-tx-receipt",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return fmt.Sprintf("%v%v", EthV0TxReceiptPrfx, dummyTxid) },
			http.MethodGet,
			&types.Receipt{},
			http.StatusOK,
		},
		{
			"eth-tx-send",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string {
				b, _ := dummyTx.MarshalBinary()
				return fmt.Sprintf("%v0x%x", EthV0SendTxPrfx, b)
			},
			http.MethodPost,
			&TxResponse{Txid: dummyTx.Hash().Hex()},
			http.StatusOK,
		},
		//
		// CLIENT ERRORS
		//
		{
			"eth-balance-malformed",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string { return fmt.Sprintf("%v0xnotanaddress", EthV0BalancePrfx) },
			http.MethodGet,
			map[string]string{"error": "invalid address format"},
			http.StatusBadRequest,
		},
		{
			"eth-tx-send-malformed",
			"-",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClient) },
			func() string {
				return EthV0SendTxPrfx + "0xnotATx"
			},
			http.MethodPost,
			map[string]string{"error": "invalid tx data: invalid hex string"},
			http.StatusBadRequest,
		},
		//
		// SERVER ERRORS
		//
		{
			"health-node-err",
			"testErr",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithErr) },
			func() string { return HeathEndPnt },
			http.MethodGet,
			&HealthResponse{Version: Version, Service: ServiceName, Failures: []string{"node 0 err: testErr"}},
			http.StatusServiceUnavailable,
		},
		{
			"eth-balance-err",
			"testErr",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithErr) },
			func() string { return fmt.Sprintf("%v%v", EthV0BalancePrfx, dummyAddr) },
			http.MethodGet,
			map[string]string{"error": "eth client error: testErr"},
			http.StatusInternalServerError,
		},
		{
			"eth-tx-err",
			"testErr",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithErr) },
			func() string { return fmt.Sprintf("%v%v", EthV0TxPrfx, dummyTxid) },
			http.MethodGet,
			map[string]string{"error": "eth client error: testErr"},
			http.StatusInternalServerError,
		},
		{
			"eth-receipt-err",
			"testErr",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithErr) },
			func() string { return fmt.Sprintf("%v%v", EthV0TxReceiptPrfx, dummyTxid) },
			http.MethodGet,
			map[string]string{"error": "eth client error: testErr"},
			http.StatusInternalServerError,
		},
		{
			"eth-tx-send-err",
			"testErr",
			func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithErr) },
			func() string {
				b, _ := dummyTx.MarshalBinary()
				return fmt.Sprintf("%v0x%x", EthV0SendTxPrfx, b)
			},
			http.MethodPost,
			map[string]string{"error": "eth client error: testErr"},
			http.StatusInternalServerError,
		},
	}

	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {

			s := tt.serviceConstructor(tt.urls)
			s.Start()
			defer s.Stop(os.Kill)

			time.Sleep(10 * time.Millisecond)

			b, code, err := executeRequest(tt.methodType, fmt.Sprintf("http://0.0.0.0%v%v", s.Server().Addr(), tt.endpoint()))
			if err != nil {
				t.Fatalf("%v: %v", tt.name, err)
			}
			if g, w := code, tt.expectedCode; g != w {
				t.Errorf("%v unexpected response code, want %v got %v", tt.name, w, g)
			}

			expectedJSON, _ := json.Marshal(tt.expectedResponse)

			if g, w := b, expectedJSON; !bytes.Equal(g, w) {
				t.Errorf("%v unexpected response, want %s, got %s", tt.name, w, g)
			}
		})

	}
}

func Test_HealthCheckErr(t *testing.T) {
	{
		apiTests := []struct {
			name               string
			urls               string
			serviceConstructor func(urls string) *Service
			endpoint           string
			methodType         string
			expectedCode       int
			expectedFailures   int
		}{
			{
				"health-nodes-unhealthy",
				"url,url,url",
				func(urls string) *Service { return makeTestService(t, urls, newFakeEthClientWithBlock) },
				HeathEndPnt,
				http.MethodGet,
				http.StatusServiceUnavailable,
				2,
			},
		}

		for _, tt := range apiTests {
			t.Run(tt.name, func(t *testing.T) {

				s := tt.serviceConstructor(tt.urls)
				s.Start()
				defer s.Stop(os.Kill)

				time.Sleep(10 * time.Millisecond)

				b, code, err := executeRequest(tt.methodType, fmt.Sprintf("http://0.0.0.0%v%v", s.Server().Addr(), tt.endpoint))
				if err != nil {
					t.Fatalf("%v: %v", tt.name, err)
				}
				if g, w := code, tt.expectedCode; g != w {
					t.Errorf("%v unexpected response code, want %v got %v", tt.name, w, g)
				}
				h := &HealthResponse{}
				if err := json.Unmarshal(b, h); err != nil {
					t.Fatal(err)
				}

				if g, w := len(h.Failures), tt.expectedFailures; g != w {
					t.Errorf("failures list %v of unexpected length %v, wanted %v", h.Failures, g, w)
				}

			})

		}
	}
}

func executeRequest(methodType, url string) (respBytes []byte, code int, err error) {
	req, err := http.NewRequestWithContext(context.Background(), methodType, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()
	// Read the response body
	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, err
	}
	return b, response.StatusCode, nil
}
