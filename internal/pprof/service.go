// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"context"
	"errors"
	"runtime"

	"github.com/ChainSafe/gossamer/internal/httpserver"
)

// Service is a pprof http server service compatible with the
// dot/service.go interface.
type Service struct {
	settings Settings
	server   httpserver.Runner
	cancel   context.CancelFunc
	done     chan error
}

// NewService creates a pprof server service compatible with the
// dot/service.go interface.
func NewService(settings Settings, logger httpserver.Logger) *Service {
	settings.setDefaults()

	return &Service{
		settings: settings,
		server:   NewServer(settings.ListeningAddress, logger),
		done:     make(chan error),
	}
}

var ErrServerDoneBeforeReady = errors.New("server terminated before being ready")

// Start starts the pprof server service.
func (s *Service) Start() (err error) {
	runtime.SetBlockProfileRate(s.settings.BlockProfileRate)
	runtime.SetMutexProfileFraction(s.settings.MutexProfileRate)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	ready := make(chan struct{})

	go s.server.Run(ctx, ready, s.done)

	select {
	case <-ready:
		return nil
	case err := <-s.done:
		close(s.done)
		if err != nil {
			return err
		}
		return ErrServerDoneBeforeReady
	}
}

// Stop stops the pprof server service.
func (s *Service) Stop() (err error) {
	s.cancel()
	return <-s.done
}
