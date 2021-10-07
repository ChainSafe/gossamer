// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"errors"
	"fmt"
)

var (
	errNilBlockState         = errors.New("cannot have nil BlockState")
	errNilStorageState       = errors.New("cannot have nil StorageState")
	errNilVerifier           = errors.New("cannot have nil Verifier")
	errNilBlockImportHandler = errors.New("cannot have nil BlockImportHandler")
	errNilNetwork            = errors.New("cannot have nil Network")
	errNilFinalityGadget     = errors.New("cannot have nil FinalityGadget")
	errNilTransactionState   = errors.New("cannot have nil TransactionState")

	// ErrNilBlockData is returned when trying to process a BlockResponseMessage with nil BlockData
	ErrNilBlockData = errors.New("got nil BlockData")

	// ErrServiceStopped is returned when the service has been stopped
	ErrServiceStopped = errors.New("service has been stopped")

	// ErrInvalidBlock is returned when a block cannot be verified
	ErrInvalidBlock = errors.New("could not verify block")

	// ErrInvalidBlockRequest is returned when an invalid block request is received
	ErrInvalidBlockRequest = errors.New("invalid block request")

	// chainSync errors
	errEmptyBlockData               = errors.New("empty block data")
	errNilBlockData                 = errors.New("block data is nil")
	errNilResponse                  = errors.New("block response is nil")
	errNilHeaderInResponse          = errors.New("expected header, received none")
	errNilBodyInResponse            = errors.New("expected body, received none")
	errNoPeers                      = errors.New("no peers to sync with")
	errResponseIsNotChain           = errors.New("block response does not form a chain")
	errPeerOnInvalidFork            = errors.New("peer is on an invalid fork")
	errWorkerMissingStartNumber     = errors.New("worker has nil start block number")
	errWorkerMissingTargetNumber    = errors.New("worker has nil target block number")
	errInvalidDirection             = errors.New("direction of request does not match specified start and target")
	errUnknownParent                = errors.New("parent of first block in block response is unknown")
	errUnknownBlockForJustification = errors.New("received justification for unknown block")
	errFailedToGetParent            = errors.New("failed to get parent header")
)

// ErrNilChannel is returned if a channel is nil
func ErrNilChannel(s string) error {
	return fmt.Errorf("cannot have nil channel %s", s)
}
