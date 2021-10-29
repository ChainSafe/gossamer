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

package transaction

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Validity struct see: https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/sr-primitives/src/transaction_validity.rs#L178
type Validity struct {
	Priority  uint64
	Requires  [][]byte
	Provides  [][]byte
	Longevity uint64
	Propagate bool
}

// NewValidity returns Validity
func NewValidity(priority uint64, requires, provides [][]byte, longevity uint64, propagate bool) *Validity {
	return &Validity{
		Priority:  priority,
		Requires:  requires,
		Provides:  provides,
		Longevity: longevity,
		Propagate: propagate,
	}
}

// ValidTransaction struct
type ValidTransaction struct {
	Extrinsic types.Extrinsic
	Validity  *Validity
}

// NewValidTransaction returns ValidTransaction
func NewValidTransaction(extrinsic types.Extrinsic, validity *Validity) *ValidTransaction {
	return &ValidTransaction{
		Extrinsic: extrinsic,
		Validity:  validity,
	}
}

// StatusNotification represents information about a transaction status update.
type StatusNotification struct {
	Ext                types.Extrinsic
	Status             string
	PeersBroadcastedTo []string
	BlockHash          *common.Hash
}

//nolint
// Status represents possible transaction statuses.
//
// The status events can be grouped based on their kinds as:
// 1. Entering/Moving within the pool:
// 		- `Future`
// 		- `Ready`
// 2. Inside `Ready` queue:
// 		- `Broadcast`
// 3. Leaving the pool:
// 		- `InBlock`
// 		- `Invalid`
// 		- `Usurped`
// 		- `Dropped`
// 	4. Re-entering the pool:
// 		- `Retracted`
// 	5. Block finalized:
// 		- `Finalized`
// 		- `FinalityTimeout`
type Status int64

const (
	// Future status occurs when transaction is part of the future queue.
	Future Status = iota
	// Ready status occurs when transaction is part of the ready queue.
	Ready
	// Broadcast status occurs when transaction has been broadcast to the given peers.
	Broadcast
	// InBlock status occurs when transaction has been included in block with given
	// hash.
	InBlock
	// Retracted status occurs when the block this transaction was included in has
	// been retracted.
	Retracted
	// FinalityTimeout status occurs when the maximum number of finality watchers
	// has been reached,
	// old watchers are being removed.
	FinalityTimeout
	//nolint
	// Finalized status occurs when transaction has been finalized by a finality-gadget,
	// e.g GRANDPA
	Finalized
	// Usurped status occurs when transaction has been replaced in the pool, by another
	// transaction that provides the same tags. (e.g. same (sender, nonce)).
	Usurped
	// Dropped status occurs when transaction has been dropped from the pool because
	// of the limit.
	Dropped
	// Invalid status occurs when transaction is no longer valid in the current state.
	Invalid
)

// String returns string representation of current status.
func (s Status) String() string {
	switch s {
	case Future:
		return "future"
	case Ready:
		return "ready"
	case Broadcast:
		return "broadcast"
	case InBlock:
		return "inBlock"
	case Retracted:
		return "retracted"
	case FinalityTimeout:
		return "finalityTimeout"
	//nolint
	case Finalized:
		return "finalized"
	case Usurped:
		return "usurped"
	case Dropped:
		return "dropped"
	case Invalid:
		return "invalid"
	}
	return "unknown"
}
