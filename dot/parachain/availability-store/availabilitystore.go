// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"context"

	"github.com/ChainSafe/gossamer/internal/database"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-store"))

type AvailabilityStoreSubsystem struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
	availabilityStore   AvailabilityStore
	//pruningConfig PruningConfig
	//clock         Clock
	//metrics       Metrics
}

type AvailabilityStore struct {
	db database.Database
}

type Config struct {
	basepath string
}

func NewAvailabilityStore(config Config) (*AvailabilityStore, error) {

	db, err := database.LoadDatabase(config.basepath,
		false) // TODO: determine if this is the best configuration for this db
	if err != nil {
		return nil, err
	}

	return &AvailabilityStore{
		db: db,
	}, nil
}

func (as *AvailabilityStore) LoadAvailableData(candidate common.Hash) (AvailableData, error) {
	resultBytes, err := as.db.Get(candidate[:]) // TODO: check if this is the correct way to get the value from the
	// database
	if err != nil {
		return AvailableData{}, err
	}
	result := AvailableData{}
	err = scale.Unmarshal(resultBytes, &result)
	return result, err
}

func (av *AvailabilityStoreSubsystem) Run(ctx context.Context, OverseerToSubsystem chan any,
	SubsystemToOverseer chan any) error {
	av.processMessages()
	return nil
}

func (*AvailabilityStoreSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.AvailabilityStore
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
	result, err := av.availabilityStore.LoadAvailableData(msg.CandidateHash)
	if err != nil {
		logger.Errorf("failed to load available data: %w", err)
	}
	msg.Sender <- result
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
