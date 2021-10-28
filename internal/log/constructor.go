package log

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/log/common"
	"github.com/ChainSafe/gossamer/internal/log/production"
)

var _ common.Logger = (*production.Logger)(nil)

var ErrLoggerTypeUnknown = errors.New("logger type is unknown")

// Type is the logger type. It can be log.Production.
type Type uint8

const (
	// Production is the logger type for production logger.
	Production Type = iota
)

// New creates a new logger using the type and options given.
func New(loggerType Type, options ...common.Option) (
	logger common.Logger, err error) {
	switch loggerType {
	case Production:
		return production.New(options...), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrLoggerTypeUnknown, loggerType)
	}
}

// NewFromGlobal creates a new logger from a global logger in a
// thread safe way, using the type and options given.
func NewFromGlobal(loggerType Type, options ...common.Option) (
	logger common.Logger, err error) {
	switch loggerType {
	case Production:
		return production.NewFromGlobal(options...), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrLoggerTypeUnknown, loggerType)
	}
}
