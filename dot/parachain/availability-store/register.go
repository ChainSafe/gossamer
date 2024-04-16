// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"github.com/ChainSafe/gossamer/internal/database"
)

func Register(overseerChan chan<- any, db database.Database, pruning *PruningConfig) (*AvailabilityStoreSubsystem,
	error) {

	availabilityStoreSubsystem := NewAvailabilityStoreSubsystem(db)

	if pruning != nil {
		availabilityStoreSubsystem.pruningConfig = *pruning
	}
	availabilityStoreSubsystem.SubSystemToOverseer = overseerChan
	return availabilityStoreSubsystem, nil
}
