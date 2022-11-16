// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ChainSafe/gossamer/internal/httpserver"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultInterval = 10 * time.Second
)

var (
	logger                     Logger = log.NewFromGlobal(log.AddContext("pkg", "metrics"))
	ErrServerEndedUnexpectedly        = fmt.Errorf("metrics server exited unexpectedly")
	ErrServerStopTimeout              = fmt.Errorf("metrics server exit timeout")
)

// IntervalConfig for interval collection
type IntervalConfig struct {
	Publish  bool
	Interval time.Duration
}

// NewIntervalConfig is constructor for IntervalConfig, and uses default metrics interval
func NewIntervalConfig(publish bool) IntervalConfig {
	return IntervalConfig{
		Publish:  publish,
		Interval: defaultInterval,
	}
}

// Server is a metrics http server
type Server struct {
	server *httpserver.Server
}

// NewServer is a constructor for metrics server
func NewServer(address string) (s *Server) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &Server{
		server: httpserver.New(
			httpserver.Address(address),
			httpserver.Handler(mux),
			httpserver.Logger("metrics", logger),
		),
	}
}

// Start starts the metrics server.
// TODO return a runtimeError channel once services can read runtime
// errors.
func (s *Server) Start() (err error) {
	_, err = s.server.Start()
	return err
}

// Stop stops the metrics server.
func (s *Server) Stop() (err error) {
	return s.server.Stop()
}
