// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"context"

	"github.com/ChainSafe/gossamer/dot/state"
)

func Register(overseerChan chan<- any, st *state.Service) (*AvailabilityStoreSubsystem, error) {
	availabilityStore := NewAvailabilityStore(st.DB())

	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		pruningConfig:       defaultPruningConfig,
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}
