package service

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Service is the main application struct containing a SimpleEthClient
// the http server and logger. It can be called to start and stop.
type Service struct {
	ethClient SimpleEthClient
	server    *hTTPService
	logger    *logrus.Entry
}

// New constructs a Service with ethclient, logger and http server.
func New(port int, l *logrus.Entry, client SimpleEthClient) *Service {
	srv := &Service{
		ethClient: client,
		logger:    l,
	}
	httpSrv := NewHTTPService(port, makeServiceAPIs(srv), l)
	srv.server = httpSrv
	return srv
}

// Start creates the HTTP server.
func (s *Service) Start() {
	s.logger.WithFields(logrus.Fields{
		"compilationDate": date,
		"gitCommit":       gitCommitHash,
	}).Infof("starting %v service", ServiceName)
	s.server.Start()

	s.logger.Infof("listening on port %v", s.server.Addr())
}

// Start gracefully shutts down the HTTP server.
func (s *Service) Stop(sig os.Signal) {
	s.logger.WithFields(logrus.Fields{"signal": sig}).Infof("stopping %v service", ServiceName)

	if err := s.server.Stop(); err != nil {
		s.logger.WithFields(logrus.Fields{"error": err}).Error("error stopping server")
	}
}

// Server exposes the http server externally.
func (s *Service) Server() *hTTPService {
	return s.server
}
