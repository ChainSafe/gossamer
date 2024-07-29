// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
)

var (
	errBlockStatePaused       = errors.New("blockstate service has been paused")
	errMaxNumberOfSameRequest = errors.New("max number of same request reached")

	// ErrInvalidBlockRequest is returned when an invalid block request is received
	ErrInvalidBlockRequest     = errors.New("invalid block request")
	errInvalidRequestDirection = errors.New("invalid request direction")
	errRequestStartTooHigh     = errors.New("request start number is higher than our best block")

	// chainSync errors

	errNoPeers           = errors.New("no peers to sync with")
	errPeerOnInvalidFork = errors.New("peer is on an invalid fork")

	errStartAndEndMismatch   = errors.New("request start and end hash are not on the same chain")
	errFailedToGetDescendant = errors.New("failed to find descendant block")
	errAlreadyInDisjointSet  = errors.New("already in disjoint set")
)
