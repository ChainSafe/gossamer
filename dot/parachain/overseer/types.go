// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ActivatedLeaf is a parachain head which we care to work on.
type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

// ActiveLeavesUpdate changes in the set of active leaves:  the parachain heads which we care to work on.
//
// note: activated field indicates deltas, not complete sets.
type ActiveLeavesUpdate struct {
	Activated ActivatedLeaf
}

// Subsystem is an interface for subsystems to be registered with the overseer.
type Subsystem interface {
	// Run runs the subsystem.
	Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error
	Name() parachaintypes.SubSystemName
}
