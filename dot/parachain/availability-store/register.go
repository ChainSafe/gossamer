// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import "github.com/ChainSafe/gossamer/dot/state"

func Register(overseerChan chan<- any, st *state.Service) (*AvailabilityStoreSubsystem, error) {
	availabilityStore := NewAvailabilityStore(st.DB())
	availabilityStoreSubsystem, err := NewAvailabilityStoreSubsystem(st.DB())
	if err != nil {
		return nil, err
	}
	availabilityStoreSubsystem.SubSystemToOverseer = overseerChan

	return availabilityStoreSubsystem, nil
}
