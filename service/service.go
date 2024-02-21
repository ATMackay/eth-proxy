package service

import (
	"os"

	"github.com/sirupsen/logrus"
)

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

func (s *Service) Start() {
	s.logger.WithFields(logrus.Fields{
		"compilationDate": date,
		"gitCommit":       gitCommitHash,
	}).Infof("starting %v service", ServiceName)
	s.server.Start()
}

func (s *Service) Stop(sig os.Signal) {
	s.logger.WithFields(logrus.Fields{"signal": sig}).Infof("stopping %v service", ServiceName)

	if err := s.server.Stop(); err != nil {
		s.logger.WithFields(logrus.Fields{"error": err}).Error("error stopping server")
	}
}

func (s *Service) Server() *hTTPService {
	return s.server
}
