package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
	},
	)
}

func (a *api) addEndpoint(e endPoint) {
	a.endpoints = append(a.endpoints, e)
}

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
		if entry == nil {
			return
		}
		start := time.Now()
		body, err := readBody(req)
		if err != nil {
			entry.WithError(err)
		}
		statusRecorder := &responseRecorder{ResponseWriter: w}
		h(statusRecorder, req, p)
		elapsed := time.Since(start)
		httpCode := statusRecorder.statusCode
		entry = entry.WithFields(logrus.Fields{
			"http_method":          req.Method,
			"http_code":            httpCode,
			"elapsed_microseconds": elapsed.Microseconds(),
		})
		// only log full request/reposne data if running in debug mode
		if entry.Logger.Level >= logrus.DebugLevel {
			entry = entry.WithField("body", body)
			entry = entry.WithField("response", string(statusRecorder.response))
		}
		if httpCode > 399 {
			entry.Warn(req.URL.Path)
		} else {
			entry.Print(req.URL.Path)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter

	statusCode int
	response   []byte
}

func readBody(r *http.Request) (map[string]interface{}, error) {
	body := make(map[string]interface{})
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	defer func() {
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(b))
		r.ContentLength = int64(bytes.NewBuffer(b).Len())
	}()
	return body, nil
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
