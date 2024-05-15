package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/ATMackay/eth-proxy/service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Client struct {
	baseURL string
	c       *http.Client
	mu      sync.Mutex
	headers http.Header
}

// New returns a new eth-proxy http client.
func New(url string) *Client {
	return &Client{
		baseURL: url,
		c:       new(http.Client),
		mu:      sync.Mutex{},
		headers: makeDefaultHeaders(),
	}
}

func makeDefaultHeaders() http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return h
}

func (client *Client) Status(ctx context.Context) (*service.StatusResponse, error) {
	var status service.StatusResponse
	if err := client.executeRequest(ctx, &status, http.MethodGet, service.StatusEndPnt, nil); err != nil {
		return nil, err
	}
	return &status, nil
}

func (client *Client) Health(ctx context.Context) (*service.HealthResponse, error) {
	var health service.HealthResponse
	if err := client.executeRequest(ctx, &health, http.MethodGet, service.HeathEndPnt, nil); err != nil {
		return nil, err
	}
	return &health, nil
}

func (client *Client) Balance(ctx context.Context, address common.Address) (*service.BalanceResponse, error) {
	var balance service.BalanceResponse
	if err := client.executeRequest(ctx, &balance, http.MethodGet, fmt.Sprintf("/eth/balance/%v", address.Hex()), nil); err != nil {
		return nil, err
	}
	return &balance, nil
}

func (client *Client) TransactionByHash(ctx context.Context, hash common.Hash) (*service.TxResponse, error) {
	var txResponse service.TxResponse
	if err := client.executeRequest(ctx, &txResponse, http.MethodGet, fmt.Sprintf("/eth/tx/hash/%v", hash.Hex()), nil); err != nil {
		return nil, err
	}
	return &txResponse, nil
}

func (client *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	var receipt types.Receipt
	if err := client.executeRequest(ctx, &receipt, http.MethodGet, fmt.Sprintf("/eth/tx/receipt/%v", hash.Hex()), nil); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (client *Client) SendTransaction(ctx context.Context, tx *types.Transaction) (*service.TxResponse, error) {
	b, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	var txResponse service.TxResponse
	if err := client.executeRequest(ctx, &txResponse, http.MethodPost, fmt.Sprintf("/eth/tx/new/%x", b), nil); err != nil {
		return nil, err
	}
	return &txResponse, nil
}

func (client *Client) executeRequest(ctx context.Context, result any, method, path string, body any) (err error) {

	op := &requestOp{
		path:   path,
		method: method,
		msg:    body,
		resp:   make(chan *jsonResult, 1),
	}
	if err := client.sendHTTP(ctx, op, result); err != nil {
		return err
	}

	jsonRes, err := op.wait(ctx)
	if err != nil {
		return err
	}
	if jsonRes.errMsg != nil {
		return fmt.Errorf("%v", jsonRes.errMsg.Error)
	}

	return nil
}

func (client *Client) sendHTTP(ctx context.Context, op *requestOp, result any) error {

	respBody, status, err := client.doRequest(ctx, op.method, op.path, op.msg)
	if err != nil {
		return err
	}

	defer respBody.Close()

	// await response
	var res = &jsonResult{
		result: result,
	}

	// process resp or error
	if status > 399 {
		errMsg := service.JSONError{}
		if err := json.NewDecoder(respBody).Decode(&errMsg); err != nil {
			return err
		}
		res.errMsg = &errMsg
	} else {
		if err := json.NewDecoder(respBody).Decode(&result); err != nil {
			return err
		}
	}

	op.resp <- res

	return nil
}

func (client *Client) doRequest(ctx context.Context, method, path string, msg any) (io.ReadCloser, int, error) {
	// Serialize JSON-encoded method
	var body []byte
	var err error
	if msg != nil {
		body, err = json.Marshal(msg)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	// set headers
	client.mu.Lock()
	req.Header = client.headers.Clone()
	client.mu.Unlock()
	setHeaders(req.Header, headersFromContext(ctx))

	// do request
	resp, err := client.c.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return resp.Body, resp.StatusCode, nil
}

type jsonResult struct {
	result any
	errMsg *service.JSONError
}

type requestOp struct {
	path   string
	method string
	msg    any
	resp   chan *jsonResult
}

func (op *requestOp) wait(ctx context.Context) (*jsonResult, error) {
	select {
	case <-ctx.Done():
		// Send the timeout error
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, nil
	}
}

type mdHeaderKey struct{}

// headersFromContext is used to extract http.Header from context.
func headersFromContext(ctx context.Context) http.Header {
	source, _ := ctx.Value(mdHeaderKey{}).(http.Header)
	return source
}

// setHeaders sets all headers from src in dst.
func setHeaders(dst http.Header, src http.Header) http.Header {
	for key, values := range src {
		dst[http.CanonicalHeaderKey(key)] = values
	}
	return dst
}
