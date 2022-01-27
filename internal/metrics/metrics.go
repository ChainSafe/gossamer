// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ChainSafe/gossamer/internal/httpserver"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultInterval = 10 * time.Second

var logger log.LeveledLogger = log.NewFromGlobal(log.AddContext("pkg", "metrics"))

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
	cancel context.CancelFunc
	server *httpserver.Server
	done   chan error
}

// NewServer is a constructor for metrics server
func NewServer(address string) (s *Server) {
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.Handler())
	return &Server{
		server: httpserver.New("metrics", address, m, logger),
	}
}

// Start will start a dedicated metrics server at the given address.
func (s *Server) Start(address string) (err error) {
	logger.Infof("Starting metrics server at http://%s/metrics", address)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	ready := make(chan struct{})
	s.done = make(chan error)

	go s.server.Run(ctx, ready, s.done)

	select {
	case <-ready:
		return nil
	case err := <-s.done:
		close(s.done)
		if err != nil {
			return err
		}
		return fmt.Errorf("metrics server exited unexpectedly")
	}
}

// Stop will stop the metrics server
func (s *Server) Stop() (err error) {
	s.cancel()
	select {
	case err := <-s.done:
		close(s.done)
		if err != nil {
			return err
		}
		return fmt.Errorf("metrics server exited unexpectedly")
	case <-time.NewTimer(30 * time.Second).C:
		return fmt.Errorf("metrics server exit timeout")
	}
}
