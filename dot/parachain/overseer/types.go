// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

type context struct {
	Sender   Sender
	Receiver chan any
	wg       *sync.WaitGroup
	stopCh   chan struct{}
}

type Sender interface {
	SendMessage(msg any) error
}

// ActivatedLeaf is a parachain head which we care to work on.
type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

// ActiveLeavesUpdate changes in the set of active leaves:  the parachain heads which we care to work on.
//
//	note: activated field indicates deltas, not complete sets.
type ActiveLeavesUpdate struct {
	Activated *ActivatedLeaf
}

type Subsystem interface {
	// Run runs the subsystem.
	Run(ctx *context) error
}
