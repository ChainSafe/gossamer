// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"

	"github.com/ChainSafe/gossamer/internal/database"
)

func Register(overseerChan chan<- any, db database.Database, pruning *PruningConfig) (*AvailabilityStoreSubsystem,
	error) {
	availabilityStore := NewAvailabilityStore(db)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	pruningConfig := pruning
	if pruningConfig == nil {
		pruningConfig = &defaultPruningConfig
	}
	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		ctx:                 ctx,
		cancel:              cancel,
		pruningConfig:       *pruningConfig,
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}
