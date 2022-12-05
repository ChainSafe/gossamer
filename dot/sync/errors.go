// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
)

var (
	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = errors.New("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = errors.New("could not verify block")

	// ErrInvalidBlockRequest is returned when an invalid block request is received
	ErrInvalidBlockRequest     = errors.New("invalid block request")
	errInvalidRequestDirection = errors.New("invalid request direction")
	errRequestStartTooHigh     = errors.New("request start number is higher than our best block")

	// chainSync errors
	errEmptyBlockData               = errors.New("empty block data")
	errNilBlockData                 = errors.New("block data is nil")
	errNilResponse                  = errors.New("block response is nil")
	errNilHeaderInResponse          = errors.New("expected header, received none")
	errNilBodyInResponse            = errors.New("expected body, received none")
	errNoPeers                      = errors.New("no peers to sync with")
	errResponseIsNotChain           = errors.New("block response does not form a chain")
	errPeerOnInvalidFork            = errors.New("peer is on an invalid fork")
	errInvalidDirection             = errors.New("direction of request does not match specified start and target")
	errUnknownParent                = errors.New("parent of first block in block response is unknown")
	errUnknownBlockForJustification = errors.New("received justification for unknown block")
	errFailedToGetParent            = errors.New("failed to get parent header")
	errStartAndEndMismatch          = errors.New("request start and end hash are not on the same chain")
	errFailedToGetDescendant        = errors.New("failed to find descendant block")
)
