package pprof

import (
	"context"
	"errors"

	"github.com/ChainSafe/gossamer/internal/httpserver"
)

// Service is a pprof http server service compatible with the
// dot/service.go interface.
type Service struct {
	server httpserver.Runner
	cancel context.CancelFunc
	done   chan error
}

// NewService creates a pprof server service compatible with the
// dot/service.go interface.
func NewService(address string, logger httpserver.Logger) *Service {
	return &Service{
		server: NewServer(address, logger),
		done:   make(chan error),
	}
}

var ErrServerDoneBeforeReady = errors.New("server terminated before being ready")

// Start starts the pprof server service.
func (s *Service) Start() (err error) {
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
