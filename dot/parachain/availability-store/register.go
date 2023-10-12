// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

func Register(overseerChan chan<- any) (*AvailabilityStoreSubsystem, error) {
	availabilityStore, err := NewAvailabilityStore(Config{basepath: "availability_store"})
	if err != nil {
		return nil, err
	}

	availabilityStoreSubsystem := AvailabilityStoreSubsystem{
		SubSystemToOverseer: overseerChan,
		availabilityStore:   *availabilityStore,
	}

	return &availabilityStoreSubsystem, nil
}
