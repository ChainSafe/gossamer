// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

func Register(overseerChan chan<- any) (*AvailabilityStoreSubsystem, error) {
	availabilityStore := AvailabilityStoreSubsystem{
		SubSystemToOverseer: overseerChan,
	}

	return &availabilityStore, nil
}
