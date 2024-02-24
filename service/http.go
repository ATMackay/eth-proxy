package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// endPoint represents an api element.
type endPoint struct {
	path       string
	handler    httprouter.Handle
	methodType string
}

type api struct {
	endpoints []endPoint
}

func makeAPI(endpoints []endPoint) *api {
	r := &api{}
	for _, e := range endpoints {
		r.addEndpoint(e)
	}
	return r
}

func makeServiceAPIs(s *Service) *api {
	return makeAPI([]endPoint{
		endPoint{
			path:       StatusEndPnt,
			handler:    s.Status(),
			methodType: http.MethodGet,
		},
		endPoint{
			path:       HeathEndPnt,
			handler:    s.Health(),
			methodType: http.MethodGet,
		},
		endPoint{
			path:       EthBalanceEndPnt,
			handler:    s.Balance(),
			methodType: http.MethodGet,
		},
		endPoint{
			path:       EthTx,
			handler:    s.Tx(),
			methodType: http.MethodGet,
		},
		endPoint{
			path:       EthTxReceipt,
			handler:    s.TxReceipt(),
			methodType: http.MethodGet,
		},
	},
	)
}

func (a *api) addEndpoint(e endPoint) {
	a.endpoints = append(a.endpoints, e)
}

// routes configures a new httprouter.Router, wrapping each handle (other than the metrics handle)
// with a logger.
func (a *api) routes(l *logrus.Entry) *httprouter.Router {

	router := httprouter.New()

	for _, e := range a.endpoints {
		router.Handle(e.methodType, e.path, logHTTPRequest(l, e.handler))
	}

	// Add metrics server - do not use logging middleware
	router.Handler(http.MethodGet, metricsEndPnt, promhttp.Handler())

	return router
}

type hTTPService struct {
	server *http.Server
	logger *logrus.Entry
}

// NewHTTPService returns a HTTP server with httprouter Router
// handling requests.
func NewHTTPService(port int, api *api, l *logrus.Entry) *hTTPService {

	handler := api.routes(l)

	return &hTTPService{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: l,
	}
}

func (h *hTTPService) Addr() string {
	return h.server.Addr
}

// Start spawns the server which will listen on the TCP address srv.Addr
// for incoming requests.
func (h *hTTPService) Start() {
	go func() {
		if err := h.server.ListenAndServe(); err != nil {
			h.logger.WithFields(logrus.Fields{"error": err}).Warn("serverTerminated")
		}
	}()
}

func (h *hTTPService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}

// HTTP logging middleware

// logHTTPRequest provides logging middleware. It surfaces low level request/response data from the http server.
func logHTTPRequest(entry *logrus.Entry, h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {

		statusRecorder := &responseRecorder{ResponseWriter: w}

		start := time.Now()
		h(statusRecorder, req, p)
		elapsed := time.Since(start)
		if entry == nil {
			return
		}

		httpCode := statusRecorder.statusCode
		entry = entry.WithFields(logrus.Fields{
			"http_method":          req.Method,
			"http_code":            httpCode,
			"elapsed_microseconds": elapsed.Microseconds(),
			"url":                  req.URL.Path,
			"response":             string(statusRecorder.response),
		})
		// only log full request/response data if running in debug mode or if
		// the server returned an error response code.
		if httpCode > 399 {
			entry.Warn("httpErr")
		} else {
			entry.Debug("servedHttpRequest")
		}
	})
}

// responseRecorder is a wrapper for http.ResponseWriter used
// byt logging middleware.
type responseRecorder struct {
	http.ResponseWriter

	statusCode int
	response   []byte
}

func (w *responseRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseRecorder) Write(b []byte) (int, error) {
	w.response = b
	return w.ResponseWriter.Write(b)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	return err
}

func respondWithError(w http.ResponseWriter, code int, msg any) {
	var message string
	switch m := msg.(type) {
	case error:
		message = m.Error()
	case string:
		message = m
	}
	_ = respondWithJSON(w, code, map[string]string{"error": message})
}
