// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// Subsystem is an interface for subsystems to be registered with the overseer.
type Subsystem interface {
	// Run runs the subsystem.
	Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any)
	Name() parachaintypes.SubSystemName
	ProcessActiveLeavesUpdateSignal()
	ProcessBlockFinalizedSignal()
	Stop()
}
