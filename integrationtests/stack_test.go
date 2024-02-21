package integrationtests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ATMackay/eth-proxy/service"
)

// Test entire stack
func Test_E2EStack(t *testing.T) {

	stack := makeEthProxyService(t)

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
			service.StatusEndPnt,
			http.MethodGet,
			&service.StatusResponse{Message: "OK", Version: service.Version, Service: service.ServiceName},
			http.StatusOK,
		},
		{
			"health",
			service.HeathEndPnt,
			http.MethodGet,
			&service.HealthResponse{Version: "0.1.0", Service: "eth-proxy", Failures: []string{}},
			http.StatusOK,
		},
		{
			"eth-balance",
			fmt.Sprintf("/eth/balance/%v", dummyAddr),
			http.MethodGet,
			&service.BalanceResp{Balance: oneEther.String()},
			http.StatusOK,
		},
	}

	time.Sleep(10 * time.Millisecond)
	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), tt.methodType, fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), tt.endpoint), nil)
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
