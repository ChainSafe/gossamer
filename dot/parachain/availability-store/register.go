// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import "github.com/ChainSafe/gossamer/dot/state"

func Register(overseerChan chan<- any, st *state.Service) (*AvailabilityStoreSubsystem, error) {
	availabilityStore, err := NewAvailabilityStore(st.DB())
	if err != nil {
		return nil, err
	}

	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}
