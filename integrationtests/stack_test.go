package integrationtests

import (
	"bytes"
	"context"
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
func Test_E2EStackRead(t *testing.T) {

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
			&service.HealthResponse{Version: service.FullVersion, Service: service.ServiceName, Failures: []string{}},
			http.StatusOK,
		},
		{
			"eth-balance",
			func() string {
				genesisAddr := stack.eth.backend.bankAccount.From
				return fmt.Sprintf("/eth/balance/%v", genesisAddr.Hex())
			},
			http.MethodGet,
			&service.BalanceResp{Balance: oneEther.String()},
			http.StatusOK,
		},
		{
			"metrics",
			func() string { return "/metrics" },
			http.MethodGet,
			nil,
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

			if tt.expectedResponse != nil {

				expectedJSON, _ := json.Marshal(tt.expectedResponse)

				if g, w := b, expectedJSON; !bytes.Equal(g, w) {
					t.Errorf("%v unexpected response, want %s, got %s", tt.name, w, g)
				}
			}

		})

	}

}

func Test_E2EStackTxWrite(t *testing.T) {

	stack := makeEthProxyService(t)

	time.Sleep(10 * time.Millisecond)

	// Check system health
	response, err := executeRequest(http.MethodGet, fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), service.HeathEndPnt))
	if err != nil {
		t.Fatalf("healthcheck err: %v", err)
	}

	if g, w := response.StatusCode, http.StatusOK; g != w {
		t.Fatalf("unexpected response code, want %v got %v", w, g)
	}
	response.Body.Close()

	// create transaction using backend client

	tx, err := stack.eth.backend.newTx()
	if err != nil {
		t.Fatalf("%v", err)
	}

	toAddr := tx.To()
	txHash := tx.Hash()
	amount := tx.Value()

	txBin, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("could not marshal json: %v", err)
	}

	// Send transaction via proxy
	response, err = executeRequest(http.MethodPut, fmt.Sprintf("http://0.0.0.0%v/eth/tx/new/0x%x", stack.service.Server().Addr(), txBin))
	if err != nil {
		t.Fatalf("tx send err: %v", err)
	}
	b, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if g, w := response.StatusCode, http.StatusOK; g != w {
		t.Fatalf("unexpected response code, want %v got %v (body=%s)", w, g, b)
	}
	response.Body.Close()

	txData := &service.TxResponse{}
	if err := json.Unmarshal(b, txData); err != nil {
		t.Fatalf("could not unmarshal response json: %v", err)
	}
	// check matching hashes
	if g, w := txHash.Hex(), txData.Txid; g != w {
		t.Fatalf("unexpected txid, want %s, got %s", w, g)
	}

	// Move the chain forward
	n, err := stack.eth.backend.Client().BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	blkHash := stack.eth.backend.Commit()
	t.Logf("new block: %v", blkHash.Hex())
	m, err := stack.eth.backend.Client().BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m != n+1 {
		t.Fatalf("block not created, was %d, now %d", n, m)
	}

	// query tx by transactionID

	response, err = executeRequest(http.MethodGet, fmt.Sprintf("http://0.0.0.0%v/eth/tx/hash/%v", stack.service.Server().Addr(), txHash.Hex()))
	if err != nil {
		t.Fatalf("tx send err: %v", err)
	}
	b, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if g, w := response.StatusCode, http.StatusOK; g != w {
		t.Fatalf("unexpected response code, want %v got %v (body=%s)", w, g, b)
	}
	response.Body.Close()

	txData = &service.TxResponse{}
	if err := json.Unmarshal(b, txData); err != nil {
		t.Fatalf("could not unmarshal response json: %v", err)
	}

	// verify transaction fields
	txReturned := txData.Tx

	if g, w := txReturned.To().Hex(), toAddr.Hex(); g != w {
		t.Fatalf("unexpected tx send addr, want %s, got %s", w, g)
	}

	if g, w := txReturned.Value().String(), amount.String(); g != w {
		t.Fatalf("unexpected tx value, want %s, got %s", w, g)
	}

	// query destination address balance
	response, err = executeRequest(http.MethodGet, fmt.Sprintf("http://0.0.0.0%v/eth/balance/%v", stack.service.Server().Addr(), toAddr.Hex()))
	if err != nil {
		t.Fatalf("tx send err: %v", err)
	}
	if g, w := response.StatusCode, http.StatusOK; g != w {
		t.Fatalf("unexpected response code, want %v got %v", w, g)
	}
	b, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	balData := &service.BalanceResp{}
	if err := json.Unmarshal(b, balData); err != nil {
		t.Fatalf("could not unmarshal response json: %v", err)
	}

	if g, w := balData.Balance, amount.String(); g != w {
		t.Fatalf("unexpected tx value, want %s, got %s", w, g)
	}
}

func Test_ConcurrentRequests(t *testing.T) {

	stack := makeEthProxyService(t)
	genesisAddr := stack.eth.backend.bankAccount.From
	if err := stack.eth.addTx(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name       string
		url        func() string
		iterations int
	}{
		{
			"mock-stack-read-balance",
			func() string {
				return fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), fmt.Sprintf("/eth/balance/%v", genesisAddr.Hex()))
			},
			100,
		},
		{
			"mock-stack-read-txid",
			func() string {
				tx := stack.eth.txs[1]
				return fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), fmt.Sprintf("/eth/tx/hash/%v", tx.Hash().Hex()))
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
		/*
			{
				// Uncomment and execute 'make run' in a separate terminal and then this test to see how well the service handles concurrent requests
				//
				// Example output
				//
				//  ~/go/src/github.com/ATMackay/eth-proxy/integrationtests$ go test -v -run Test_ConcurrentRequests
				//  === RUN   Test_ConcurrentRequests
				//  === RUN   Test_ConcurrentRequests/real-stack
				//  stack_test.go:133: real-stack: completed 200 requests in 1.039025218s seconds (192.49278152069297 req/s)
				//
				//
				"real-stack",
				func() string {
					return "http://localhost:8080/eth/tx/0x326c7dbb58eaf646af01f7b6f4fb1e0fb1afe1329ac670ce5945e8fd940ec4d7"
				},
				200,
			},
		*/
	}

	for _, tt := range tests {
		N := tt.iterations
		url := tt.url()
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			wg := sync.WaitGroup{}
			for i := 0; i < N; i++ {
				index := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					response, err := executeRequest(http.MethodGet, url)
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
