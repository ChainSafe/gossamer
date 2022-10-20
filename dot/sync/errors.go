// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import "fmt"

var (
	// ErrNilBlockData is returned when trying to process a BlockResponseMessage with nil BlockData
	ErrNilBlockData = fmt.Errorf("got nil BlockData")

	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = fmt.Errorf("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = fmt.Errorf("could not verify block")

	// ErrInvalidBlockRequest is returned when an invalid block request is received
	ErrInvalidBlockRequest        = fmt.Errorf("invalid block request")
	errInvalidRequestDirection    = fmt.Errorf("invalid request direction")
	errRequestStartTooHigh        = fmt.Errorf("request start number is higher than our best block")
	errFailedToGetEndHashAncestor = fmt.Errorf("failed to get ancestor of end block")

	// chainSync errors
	errEmptyBlockData               = fmt.Errorf("empty block data")
	errNilBlockData                 = fmt.Errorf("block data is nil")
	errNilResponse                  = fmt.Errorf("block response is nil")
	errNilHeaderInResponse          = fmt.Errorf("expected header, received none")
	errNilBodyInResponse            = fmt.Errorf("expected body, received none")
	errNoPeers                      = fmt.Errorf("no peers to sync with")
	errResponseIsNotChain           = fmt.Errorf("block response does not form a chain")
	errPeerOnInvalidFork            = fmt.Errorf("peer is on an invalid fork")
	errInvalidDirection             = fmt.Errorf("direction of request does not match specified start and target")
	errUnknownParent                = fmt.Errorf("parent of first block in block response is unknown")
	errUnknownBlockForJustification = fmt.Errorf("received justification for unknown block")
	errFailedToGetParent            = fmt.Errorf("failed to get parent header")
	errStartAndEndMismatch          = fmt.Errorf("request start and end hash are not on the same chain")
	errFailedToGetDescendant        = fmt.Errorf("failed to find descendant block")
)
