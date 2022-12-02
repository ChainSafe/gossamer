// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"
)

var (
	ErrNilRuntime = errors.New("cannot have nil runtime")

	ErrNilBlockHandlerParameter = errors.New("unable to handle block due to nil parameter")

	// ErrEmptyRuntimeCode is returned when the storage :code is empty
	ErrEmptyRuntimeCode = errors.New("new :code is empty")

	errInvalidTransactionQueueVersion = errors.New("invalid transaction queue version")
)
