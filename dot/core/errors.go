// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import "fmt"

var (

	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = fmt.Errorf("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = fmt.Errorf("could not verify block")

	ErrNilRuntime = fmt.Errorf("cannot have nil runtime")

	ErrNilBlockHandlerParameter = fmt.Errorf("unable to handle block due to nil parameter")

	// ErrEmptyRuntimeCode is returned when the storage :code is empty
	ErrEmptyRuntimeCode = fmt.Errorf("new :code is empty")
)
