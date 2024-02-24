package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/julienschmidt/httprouter"
)

const (
	StatusEndPnt = "/status" // status endpoint for LIVENESS probing
	HeathEndPnt  = "/health" // health endpoint for READINESS probing

	EthBalanceEndPnt = "/eth/balance/:address" // eth_getBalance proxy endpoint
	EthTx            = "/eth/tx/:id"           // eth_getTransaction proxy endpoint
	EthTxReceipt     = "/eth/receipt/:id"      // eth_getTransactionReceipt proxy endpoint

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
		if err := respondWithJSON(w, http.StatusOK, &StatusResponse{Message: "OK", Version: FullVersion, Service: ServiceName}); err != nil {
			s.logger.Error(err)
		}
	})

}

// HealthResponse contains health probe response fields.
type HealthResponse struct {
	Version  string   `json:"version,omitempty"`
	Service  string   `json:"service,omitempty"`
	Failures []string `json:"failures"`
}

// Health pings the layer one clients. It ensures that the connected geth
// execution clients are ready to accept incoming proxied requests.
func (s *Service) Health() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &HealthResponse{
			Service: ServiceName,
			Version: FullVersion,
		}
		var failures = []string{}
		var httpCode = http.StatusOK

		// check clients
		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		if _, err := s.ethClient.BlockNumber(ctx); err != nil {
			failureArray := strings.Split(err.Error(), "|")
			trimmed := failureArray[0 : len(failureArray)-1]
			failures = append(failures, trimmed...)
		}

		health.Failures = failures

		if len(health.Failures) > 0 {
			httpCode = http.StatusServiceUnavailable
		}

		if err := respondWithJSON(w, httpCode, health); err != nil {
			s.logger.Error(err)
		}
	})
}

// BalanceResp contains balance value formatted as a string.
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
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &BalanceResp{Balance: b.String()}); err != nil {
			s.logger.Error(err)
		}

	})

}

// TxResponse contains ethereum transaction data and a pending flag.
type TxResponse struct {
	Tx        *types.Transaction `json:"tx"`
	IsPending bool               `json:"is_pending"`
}

// Tx handles the eth_getTransaction proxy endpoint.
func (s *Service) Tx() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		txid := p.ByName("id")

		txHash := common.HexToHash(txid)

		if len(txHash.Bytes()) != 32 {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid hash"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		tx, pending, err := s.ethClient.TransactionByHash(ctx, txHash)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &TxResponse{Tx: tx, IsPending: pending}); err != nil {
			s.logger.Error(err)
		}

	})
}

// TxResponse contains ethereum transaction data and a pending flag.
type TxReceiptResponse *types.Receipt

// Tx handles the eth_getTransaction proxy endpoint.
func (s *Service) TxReceipt() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		txid := p.ByName("id")

		txHash := common.HexToHash(txid)

		if len(txHash.Bytes()) != 32 {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid hash"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		tx, err := s.ethClient.TransactionReceipt(ctx, txHash)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if tx == nil {
			respondWithError(w, http.StatusNotFound, fmt.Errorf("not found"))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &tx); err != nil {
			s.logger.Error(err)
		}

	})
}
