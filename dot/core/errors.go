// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"
	"fmt"
)

var (

	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = errors.New("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = errors.New("could not verify block")

	ErrNilRuntime = errors.New("cannot have nil runtime")

	ErrNilBlockHandlerParameter = errors.New("unable to handle block due to nil parameter")

	// ErrEmptyRuntimeCode is returned when the storage :code is empty
	ErrEmptyRuntimeCode = errors.New("new :code is empty")
)

// ErrNilChannel is returned if a channel is nil
func ErrNilChannel(s string) error {
	return fmt.Errorf("cannot have nil channel %s", s)
}

// ErrMessageCast is returned if unable to cast a network.Message to a type
func ErrMessageCast(s string) error {
	return fmt.Errorf("could not cast network.Message to %s", s)
}

// ErrUnsupportedMsgType is returned if we receive an unknown message type
func ErrUnsupportedMsgType(d byte) error {
	return fmt.Errorf("received unsupported message type %d", d)
}
