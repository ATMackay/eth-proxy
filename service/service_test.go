package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	yaml "gopkg.in/yaml.v3"
)

const (
	dummyAddr = "0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
)

// Make sure to write some good tests

var _ SimpleEthClient = (*fakeEthClient)(nil)

type fakeEthClient struct{}

func newFakeEthClient(_ string) (SimpleEthClient, error) {
	return &fakeEthClient{}, nil
}

func newFakeEthClientWithErr(_ string) (SimpleEthClient, error) {
	return &fakeEthClient{}, errors.New("error")
}

func (f *fakeEthClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (f *fakeEthClient) BlockNumber(context.Context) (uint64, error) {
	return 0, nil
}

func makeTestService(t *testing.T) *Service {

	l, err := NewLogger("error", "plain")
	if err != nil {
		t.Fatal(err)
	}

	cl, err := NewMultiNodeClient("-,-", newFakeEthClient)
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

	srv := makeTestService(t)

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
				for i := range 100 {
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
			newFakeEthClientWithErr,
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
		})
	}

}

func Test_API(t *testing.T) {

	s := makeTestService(t)

	s.Start()
	defer s.Stop(os.Kill)

	apiTests := []struct {
		name             string
		endpoint         string
		methodType       string
		expectedResponse any
		expectedCode     int
	}{
		//
		// READ REQUESTS
		//
		{
			"status",
			StatusEndPnt,
			http.MethodGet,
			&StatusResponse{Message: "OK", Version: FullVersion, Service: ServiceName},
			http.StatusOK,
		},
		{
			"health",
			HeathEndPnt,
			http.MethodGet,
			&HealthResponse{Version: FullVersion, Service: ServiceName, Failures: []string{}},
			http.StatusOK,
		},
		{
			"eth-balance",
			fmt.Sprintf("/eth/balance/%v", dummyAddr),
			http.MethodGet,
			&BalanceResp{Balance: "0"},
			http.StatusOK,
		},
		{
			"eth-balance-malformed",
			fmt.Sprintf("/eth/balance/%v", "0xnotanaddress"),
			http.MethodGet,
			map[string]string{"error": "invalid address format"},
			http.StatusBadRequest,
		},
	}

	time.Sleep(10 * time.Millisecond)
	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), tt.methodType, fmt.Sprintf("http://0.0.0.0%v%v", s.Server().Addr(), tt.endpoint), nil)
			if err != nil {
				t.Fatalf("%v: %v", tt.name, err)
			}
			req.Header.Set("Content-Type", "application/json")

			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("%v: %v", tt.name, err)
			}
			defer response.Body.Close()

			// Read the response body
			b, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if g, w := response.StatusCode, tt.expectedCode; g != w {
				t.Errorf("%v unexpected response code, want %v got %v", tt.name, w, g)
			}

			expectedJSON, _ := json.Marshal(tt.expectedResponse)

			if g, w := b, expectedJSON; !bytes.Equal(g, w) {
				t.Errorf("%v unexpected response, want %s, got %s", tt.name, w, g)
			}
		})

	}
}
