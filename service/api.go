package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/julienschmidt/httprouter"
)

const (
	StatusEndPnt = "/status" // status endpoint for LIVENESS probing
	HeathEndPnt  = "/health" // health endpoint for READINESS probing

	EthBalanceEndPnt = "/eth/balance/:address" // eth_getBalance proxy endpoint
	EthTx            = "/eth/tx/hash/:id"      // eth_getTransaction proxy endpoint
	EthTxReceipt     = "/eth/tx/receipt/:id"   // eth_getTransactionReceipt proxy endpoint
	EthSendTx        = "/eth/tx/new/:txdata"   // eth_sendRawTransaction proxy endpoint

	metricsEndPnt = "/metrics" // Prometheus metrics endpoint

	timeout = 5 * time.Second
)

// StatusResponse contains status response fields.
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

// Status implements the status request endpoint. Always returns OK.
func Status() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := respondWithJSON(w, http.StatusOK, &StatusResponse{Message: "OK", Version: FullVersion, Service: ServiceName}); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
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
func Health(ethClient SimpleEthClient) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &HealthResponse{
			Service: ServiceName,
			Version: FullVersion,
		}
		var failures = []string{}
		var httpCode = http.StatusOK

		// check clients
		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		if _, err := ethClient.BlockNumber(ctx); err != nil {
			failureArray := strings.Split(err.Error(), "|")
			trimmed := failureArray[0 : len(failureArray)-1]
			failures = append(failures, trimmed...)
		}

		health.Failures = failures

		if len(health.Failures) > 0 {
			httpCode = http.StatusServiceUnavailable
		}

		if err := respondWithJSON(w, httpCode, health); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
		}
	})
}

// BalanceResp contains balance value formatted as a string.
type BalanceResp struct {
	Balance string `json:"balance"`
}

// Balance handles the getBalance proxy endpoint.
func Balance(ethClient SimpleEthClient) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		address := p.ByName("address")

		if !common.IsHexAddress(address) {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid address format"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		b, err := ethClient.BalanceAt(ctx, common.HexToAddress(address), nil)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &BalanceResp{Balance: b.String()}); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
		}

	})

}

// TxResponse contains ethereum transaction data and a pending flag.
type TxResponse struct {
	Tx        *types.Transaction `json:"tx,omitempty"`
	Txid      string             `json:"txid,omitempty"`
	IsPending bool               `json:"is_pending,omitempty"`
}

// Tx returns a handler for the eth_getTransaction proxy endpoint.
func Tx(ethClient SimpleEthClient) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		txid := p.ByName("id")

		txHash := common.HexToHash(txid)

		if len(txHash.Bytes()) != 32 {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid hash"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		tx, pending, err := ethClient.TransactionByHash(ctx, txHash)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &TxResponse{Tx: tx, Txid: txHash.Hex(), IsPending: pending}); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
		}

	})
}

// TxResponse contains ethereum transaction data and a pending flag.
type TxReceiptResponse *types.Receipt

// TxReceipt returns a handler for the eth_getTransactionReceipt proxy endpoint.
func TxReceipt(ethClient SimpleEthClient) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		txid := p.ByName("id")

		txHash := common.HexToHash(txid)

		if len(txHash.Bytes()) != 32 {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid hash"))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		tx, err := ethClient.TransactionReceipt(ctx, txHash)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if tx == nil {
			respondWithError(w, http.StatusNotFound, fmt.Errorf("not found"))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &tx); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
		}

	})
}

// SendTx returns a handler for the eth_sendRawTransaction proxy endpoint.
func SendTx(ethClient SimpleEthClient) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		txHex := p.ByName("txdata")

		txBytes, err := hexutil.Decode(txHex)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid tx data: %v", err))
			return
		}

		tx := &types.Transaction{}

		if err := tx.UnmarshalBinary(txBytes); err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not unmarshal tx JSON: %v", err))
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()

		if err := ethClient.SendTransaction(ctx, tx); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("eth client error: %v", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &TxResponse{Txid: tx.Hash().Hex()}); err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("respond error: %v", err))
		}

	})
}
