// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import "github.com/ChainSafe/gossamer/lib/common"

type Context struct {
	Sender   Sender
	Receiver chan any
}

type Sender interface {
	SendMessage(msg any) error
}

type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

type ActiveLeavesUpdate struct {
	Activated *ActivatedLeaf
}

type Subsystem interface {
	// Run runs the subsystem.
	Run(ctx *Context) error

	// ProcessActiveLeavesUpdate processes the active leaves update.
	//ProcessActiveLeavesUpdate(update ActiveLeavesUpdate) error
}
