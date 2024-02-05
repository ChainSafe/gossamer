// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	types "github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type TestSubsystem struct {
	name string
}

func (s *TestSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
	counter := 0
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("%s ctx error: %v\n", s.name, err)
			}
			fmt.Printf("%s overseer stopping\n", s.name)
			return
		case overseerSignal := <-OverseerToSubSystem:
			fmt.Printf("%s received from overseer %v\n", s.name, overseerSignal)
		default:
			// simulate work, and sending messages to overseer
			r := rand.Intn(1000)
			time.Sleep(time.Duration(r) * time.Millisecond)
			SubSystemToOverseer <- fmt.Sprintf("hello from %v, i: %d", s.name, counter)
			counter++
		}
	}
}

func (s *TestSubsystem) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	fmt.Printf("%s ProcessActiveLeavesUpdateSignal\n", s.name)
	return nil
}

func (s *TestSubsystem) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	fmt.Printf("%s ProcessActiveLeavesUpdateSignal\n", s.name)
	return nil
}

func (s *TestSubsystem) String() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Stop() {}

func TestHandleBlockEvents(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	var finalizedCounter atomic.Int32
	var importedCounter atomic.Int32

	var wg sync.WaitGroup
	wg.Add(4) // number of subsystems * 2

	// mocked subsystems
	go func() {
		for {
			select {
			case msg := <-overseerToSubSystem1:
				go incrementCounters(t, msg, &finalizedCounter, &importedCounter)
				wg.Done()
			case msg := <-overseerToSubSystem2:
				go incrementCounters(t, msg, &finalizedCounter, &importedCounter)
				wg.Done()
			}

		}
	}()

	err := overseer.Start()
	require.NoError(t, err)
	finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{}

	wg.Wait()

	// let subsystems run for a bit
	time.Sleep(4000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	require.Equal(t, int32(2), finalizedCounter.Load())
	require.Equal(t, int32(2), importedCounter.Load())
}

func incrementCounters(t *testing.T, msg any, finalizedCounter *atomic.Int32, importedCounter *atomic.Int32) {
	t.Helper()

	if msg == nil {
		return
	}

	switch msg.(type) {
	case parachaintypes.BlockFinalizedSignal:
		finalizedCounter.Add(1)
	case parachaintypes.ActiveLeavesUpdateSignal:
		importedCounter.Add(1)
	}
}

func TestHandleBlockEvents(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	var finalizedCounter atomic.Int32
	var importedCounter atomic.Int32

	go func() {
		for {
			select {
			case msg := <-overseerToSubSystem1:
				if msg == nil {
					continue
				}

				_, ok := msg.(BlockFinalizedSignal)
				if ok {
					finalizedCounter.Add(1)
				}

				_, ok = msg.(ActiveLeavesUpdateSignal)
				if ok {
					importedCounter.Add(1)
				}
			case msg := <-overseerToSubSystem2:
				if msg == nil {
					continue
				}

				_, ok := msg.(BlockFinalizedSignal)
				if ok {
					finalizedCounter.Add(1)
				}

				_, ok = msg.(ActiveLeavesUpdateSignal)
				if ok {
					importedCounter.Add(1)
				}
			}

		}
	}()

	err := overseer.Start()
	require.NoError(t, err)
	finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{}

	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	require.Equal(t, int32(2), finalizedCounter.Load())
	require.Equal(t, int32(2), importedCounter.Load())
}

func TestSignalAvailabilityStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)
	blockState.EXPECT().GetRuntime(gomock.Any()).Return(wazero.NewTestInstance(t, runtime.WESTEND_RUNTIME_v0942), nil)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := setupTestDB(t)

	availabilityStore, err := availability_store.Register(overseer.GetSubsystemToOverseerChannel(), inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = overseer.RegisterSubsystem(availabilityStore)

	chainApi, err := chainapi.Register(overseer.GetSubsystemToOverseerChannel())
	require.NoError(t, err)
	chainApi.OverseerToSubSystem = overseer.RegisterSubsystem(chainApi)

	err = overseer.Start()
	require.NoError(t, err)

	finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{}

	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)
}

func setupTestDB(t *testing.T) database.Database {
	inmemoryDB := state.NewInMemoryDB(t)
	return inmemoryDB
}

func TestRuntimeApiErrorDoesNotStopTheSubsystem(t *testing.T) {
	ctrl := gomock.NewController(t)

	overseer := NewMockOverseerSystem(ctrl)
	subToOverseer := make(chan any)

	// TODO: add error to availability store to test this
	overseer.EXPECT().GetSubsystemToOverseerChannel().Return(subToOverseer).AnyTimes()
	overseer.EXPECT().RegisterSubsystem(gomock.Any()).Return(subToOverseer).AnyTimes()
	overseer.EXPECT().Start().Return(nil)
	overseer.EXPECT().Stop().Return(nil)

	require.NotNil(t, overseer)

	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := setupTestDB(t)

	availabilityStore, err := availability_store.Register(overseer.GetSubsystemToOverseerChannel(), inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = overseer.RegisterSubsystem(availabilityStore)

	chainApi, err := chainapi.Register(overseer.GetSubsystemToOverseerChannel())
	require.NoError(t, err)
	chainApi.OverseerToSubSystem = overseer.RegisterSubsystem(chainApi)

	err = overseer.Start()
	require.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)
}

// fn runtime_api_error_does_not_stop_the_subsystem()

// fn store_chunk_works()

// fn store_chunk_does_nothing_if_no_entry_already()

// fn query_chunk_checks_meta()

// fn store_available_data_erasure_mismatch()

// fn store_block_works()

// fn store_pov_and_query_chunk_works()

// fn query_all_chunks_works()

// fn stored_but_not_included_data_is_pruned()

// fn stored_data_kept_until_finalized()

// fn we_dont_miss_anything_if_import_notifications_are_missed()

// fn forkfullness_works()

// fn query_chunk_size_works()
