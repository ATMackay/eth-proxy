package integrationtests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ATMackay/eth-proxy/service"
)

// Test E2E flows against an in-memory Ethereum server.
func Test_E2EStack(t *testing.T) {

	stack := makeEthProxyService(t)

	apiTests := []struct {
		name             string
		endpoint         func() string
		methodType       string
		expectedResponse any
		expectedCode     int
	}{
		{
			"status",
			func() string { return service.StatusEndPnt },
			http.MethodGet,
			&service.StatusResponse{Message: "OK", Version: service.FullVersion, Service: service.ServiceName},
			http.StatusOK,
		},
		{
			"health",
			func() string { return service.HeathEndPnt },
			http.MethodGet,
			&service.HealthResponse{Version: service.FullVersion, Service: "eth-proxy", Failures: []string{}},
			http.StatusOK,
		},
		{
			"eth-balance",
			func() string {
				genesisAddr := stack.node.backend.bankAccount.From
				return fmt.Sprintf("/eth/balance/%v", genesisAddr.Hex())
			},
			http.MethodGet,
			&service.BalanceResp{Balance: oneEther.String()},
			http.StatusOK,
		},
	}

	time.Sleep(10 * time.Millisecond)
	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := executeRequest(tt.methodType, fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), tt.endpoint()))
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

func Test_ConcurrentRequests(t *testing.T) {

	tests := []struct {
		name       string
		url        func() string
		iterations int
	}{
		{
			"mock-stack",
			func() string {
				stack := makeEthProxyService(t)
				genesisAddr := stack.node.backend.bankAccount.From
				endpnt := fmt.Sprintf("/eth/balance/%v", genesisAddr.Hex())
				time.Sleep(10 * time.Millisecond)
				return fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), endpnt)
			},
			100,
		},
		/*
			{
				// Uncomment and execute 'make run' in a separate terminal and then this test to see how well the service handles concurrent requests
			    //
				// Example output
				//
				//  ~/go/src/github.com/ATMackay/eth-proxy/integrationtests$ go test -v -run Test_ConcurrentRequests
				//  === RUN   Test_ConcurrentRequests
				//  === RUN   Test_ConcurrentRequests/real-stack
				//  stack_test.go:133: real-stack: completed 200 requests in 919.66899ms seconds (217.6278563656148 req/s)
				//
				//
				"real-stack",
				func() string {
					return "http://localhost:8080/eth/balance/0xfe3b557e8fb62b89f4916b721be55ceb828dbd73"
				},
				200,
			},
		*/
	}

	for _, tt := range tests {
		N := tt.iterations
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			wg := sync.WaitGroup{}
			for i := 0; i < N; i++ {
				index := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					response, err := executeRequest(http.MethodGet, tt.url())
					if err != nil {
						t.Errorf("%d: %v", index, err)
						return
					}
					if response.StatusCode != http.StatusOK {
						t.Errorf("%d: unexpected error code: %v", index, response.StatusCode)
					}
				}()
			}
			wg.Wait()
			elapsed := time.Since(start)
			t.Logf("%v: completed %d requests in %v seconds (%v req/s)\n", tt.name, N, elapsed, float64(N*1000)/float64(elapsed.Milliseconds()))
		})
	}
}
