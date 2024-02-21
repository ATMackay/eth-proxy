package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
)

const (
	StatusEndPnt = "/status" // status endpoint for LIVENESS PROBING
	HeathEndPnt  = "/health" // health endpoint for READINESS probing

	EthBalanceEndPnt = "/eth/balance/:address" // getBalance proxy endpoint (syntax compatible with httprouter)

	metricsEndPnt = "/metrics" // Prometheus metrics endpoint
)

// StatusResponse contains status response fields.
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

// Status implements the status request endpoint. Always returns OK.
func (s *Service) Status() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := respondWithJSON(w, http.StatusOK, &StatusResponse{Message: "OK", Version: Version, Service: ServiceName}); err != nil {
			s.logger.Error(err)
		}
	})

}

// HealthResponse contains status response fields.
type HealthResponse struct {
	Version  string   `json:"version,omitempty"`
	Service  string   `json:"service,omitempty"`
	Failures []string `json:"failures"`
}

// Health pings the layer one clients.
//
// TODO
func (s *Service) Health() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &HealthResponse{
			Service: ServiceName,
			Version: Version,
		}
		var failures = []string{}
		var httpCode = http.StatusOK

		// check clients
		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		if _, err := s.ethClient.SyncProgress(ctx); err != nil {
			failures = append(failures, strings.Split(err.Error(), "|")...)
		}

		health.Failures = failures

		if err := respondWithJSON(w, httpCode, health); err != nil {
			s.logger.Error(err)
		}
	})
}

// BalanceResp contains.
type BalanceResp struct {
	Balance string `json:"balance"`
}

// Balance handles the getBalance proxy endpoint.
func (s *Service) Balance() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		address := p.ByName("address")

		if !common.IsHexAddress(address) {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid address format"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		b, err := s.ethClient.BalanceAt(ctx, common.HexToAddress(address), nil)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &BalanceResp{Balance: b.String()}); err != nil {
			s.logger.Error(err)
		}

	})

}
