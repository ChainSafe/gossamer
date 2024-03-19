// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"

	"github.com/ChainSafe/gossamer/internal/database"
)

func CreateAndRegister(overseerChan chan<- any, db database.Database) (*AvailabilityStoreSubsystem, error) {
	availabilityStore := NewAvailabilityStore(db)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		ctx:                 ctx,
		cancel:              cancel,
		pruningConfig:       defaultPruningConfig,
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}

func CreateAndRegisterPruning(overseerChan chan<- any, db database.Database,
	pruning PruningConfig) (*AvailabilityStoreSubsystem,
	error) {
	availabilityStore := NewAvailabilityStore(db)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		ctx:                 ctx,
		cancel:              cancel,
		pruningConfig:       pruning,
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}
