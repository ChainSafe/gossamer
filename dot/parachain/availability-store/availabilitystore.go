// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-store"))

type AvailabilityStoreSubsystem struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	//db            interface{} // Define the actual database type
	//config        Config      // Define the actual config type
	//pruningConfig PruningConfig
	//clock         Clock
	//metrics       Metrics
}

func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) error {
	av.processMessages()
}

func (av *AvailabilityStoreSubsystem) processMessages() {
	for msg := range av.OverseerToSubSystem {
		switch msg.(type) {
		case QueryAvailableData:
			av.handleQueryAvailableData(msg.(QueryAvailableData))
		}
	}
	fmt.Printf("AvailabilityStore: Got message %v\n", msg)
}

func (av *AvailabilityStoreSubsystem) handleQueryAvailableData(msg QueryAvailableData) {
	// TODO: handle query available data
}
