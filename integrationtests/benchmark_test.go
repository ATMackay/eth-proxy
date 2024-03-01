package integrationtests

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkConcurrentRequests(b *testing.B) {

	stack := makeEthProxyService(b)
	genesisAddr := stack.eth.backend.bankAccount.From
	endpnt := fmt.Sprintf("/eth/balance/%v", genesisAddr.Hex())
	time.Sleep(10 * time.Millisecond)
	url := fmt.Sprintf("http://0.0.0.0%v%v", stack.service.Server().Addr(), endpnt)

	var wg sync.WaitGroup

	numRequests := 10

	var set []time.Duration

	var counter = new(atomic.Int64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		start := time.Now()
		for j := 0; j < numRequests; j++ {
			index := j
			wg.Add(1)
			go func() {
				defer wg.Done()
				response, err := executeRequest(http.MethodGet, url)
				if err != nil {
					b.Errorf("%d: %v", index, err)
					return
				}
				if response.StatusCode != http.StatusOK {
					b.Errorf("%d: unexpected error code: %v", index, response.StatusCode)
				}
				counter.Add(1)
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)
		set = append(set, elapsed)
	}

	// calculate mean
	var sum time.Duration
	for _, d := range set {
		sum += d
	}
	mean := sum / time.Duration(len(set))

	b.Logf("executed %d requests in %v - mean duration: %v per %v requests (%v req/s)\n",
		counter.Load(),
		sum,
		mean,
		numRequests,
		float64(numRequests*1000000)/float64(mean.Microseconds()),
	)
}
