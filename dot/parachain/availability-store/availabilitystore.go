// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"context"

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
	return nil
}

func (av *AvailabilityStoreSubsystem) processMessages() {
	for msg := range av.OverseerToSubSystem {
		logger.Debugf("received message %v", msg)
		switch msg := msg.(type) {
		case QueryAvailableData:
			av.handleQueryAvailableData(msg)
		case QueryDataAvailability:
			av.handleQueryDataAvailability(msg)
		case QueryChunk:
			av.handleQueryChunk(msg)
		case QueryChunkSize:
			av.handleQueryChunkSize(msg)
		case QueryAllChunks:
			av.handleQueryAllChunks(msg)
		case QueryChunkAvailability:
			av.handleQueryChunkAvailability(msg)
		case StoreChunk:
			av.handleStoreChunk(msg)
		case StoreAvailableData:
			av.handleStoreAvailableData(msg)
		}
	}
}

func (av *AvailabilityStoreSubsystem) handleQueryAvailableData(msg QueryAvailableData) {
	// TODO: handle query available data
}

func (av *AvailabilityStoreSubsystem) handleQueryDataAvailability(msg QueryDataAvailability) {
	// TODO: handle query data availability
}

func (av *AvailabilityStoreSubsystem) handleQueryChunk(msg QueryChunk) {
	// TODO: handle query chunk
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkSize(msg QueryChunkSize) {
	// TODO: handle query chunk size
}

func (av *AvailabilityStoreSubsystem) handleQueryAllChunks(msg QueryAllChunks) {
	// TODO: handle query all chunks
}

func (av *AvailabilityStoreSubsystem) handleQueryChunkAvailability(msg QueryChunkAvailability) {
	// TODO: handle query chunk availability
}

func (av *AvailabilityStoreSubsystem) handleStoreChunk(msg StoreChunk) {
	// TODO: handle store chunk
}

func (av *AvailabilityStoreSubsystem) handleStoreAvailableData(msg StoreAvailableData) {
	// TODO: handle store available data
}
