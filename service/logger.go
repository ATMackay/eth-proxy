package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Format string

const (
	timeFormat = time.RFC3339Nano

	JSON  Format = "json"
	Plain Format = "plain"
)

// NewLogger initializes a new (logrus) Logger instance
// Supported log formats are: plain, json
func NewLogger(logLevel, logFormat string) (*logrus.Entry, error) {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	format := Format(logFormat)
	if err := checkFormat(format, ServiceName); err != nil {
		return nil, err
	}

	l := newFormattedLogger(lvl, format)

	logger := logrus.NewEntry(l)
	logger.Level = l.Level
	logger = logger.WithFields(logrus.Fields{
		"serviceName": ServiceName,
		"version":     FullVersion,
	})
	return logger, nil
}

// newFormattedLogger returns a formatted Logger object (log format is JSON by default)
func newFormattedLogger(logLevel logrus.Level, logFormat Format) *logrus.Logger {

	l := logrus.New()
	var formatter logrus.Formatter
	// Select log Format
	switch logFormat {
	case Plain:
		formatter = &logrus.TextFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg: "message",
			},
			TimestampFormat: timeFormat,
		}
	default:
		formatter = &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg: "message",
			},
			TimestampFormat: timeFormat,
		}
	}

	l.SetFormatter(formatter)

	l.Level = logLevel

	if logLevel == logrus.DebugLevel {
		l.Warn(fmt.Sprintf("%s RUNNING IN DEBUG MODE. DO NOT RUN IN PRODUCTION ENVIRONMENT", strings.ToUpper(ServiceName)))
	}
	return l
}

// Validation

func checkFormat(w Format, service string) error {
	switch w {
	case JSON, Plain:
		return nil
	default:
		return fmt.Errorf("invalid %s log format input '%v'", service, w)
	}
}
