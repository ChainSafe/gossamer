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

	availability_store "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/chainapi"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
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
	fmt.Printf("%s run\n", s.name)
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

func (s *TestSubsystem) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) {
	fmt.Printf("%s ProcessActiveLeavesUpdateSignal\n", s.name)
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

	availabilityStore, err := availability_store.CreateAndRegister(overseer.GetSubsystemToOverseerChannel(), inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = overseer.RegisterSubsystem(availabilityStore)

	chainApi, err := chainapi.Register(overseer.GetSubsystemToOverseerChannel())
	require.NoError(t, err)
	chainApi.OverseerToSubSystem = overseer.RegisterSubsystem(chainApi)

	err = overseer.Start()
	require.NoError(t, err)

	//finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{
		Header: types.Header{
			ParentHash:     common.Hash{},
			Number:         2,
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         scale.VaryingDataTypeSlice{},
		},
		Body: nil,
	}

	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)
}

func setupTestDB(t *testing.T) database.Database {
	inmemoryDB := state.NewInMemoryDB(t)
	return inmemoryDB
}

// TODO: consider removing this test since and replacing with the test harness tests since there is more control over
// the subsystems and the overseer
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

	availabilityStore, err := availability_store.CreateAndRegister(overseer.GetSubsystemToOverseerChannel(), inmemoryDB)
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

type testOverseer struct {
	ctx context.Context

	subsystems           map[Subsystem]chan any
	SubsystemsToOverseer chan any
	wg                   sync.WaitGroup
}

func NewTestOverseer() *testOverseer {
	ctx := context.Background()
	return &testOverseer{
		ctx:                  ctx,
		subsystems:           make(map[Subsystem]chan any),
		SubsystemsToOverseer: make(chan any),
	}
}

func (to *testOverseer) RegisterSubsystem(subsystem Subsystem) chan any {
	overseerToSubSystem := make(chan any)
	to.subsystems[subsystem] = overseerToSubSystem

	return overseerToSubSystem
}

func (to *testOverseer) Start() error {
	// start subsystems
	for subsystem, overseerToSubSystem := range to.subsystems {
		to.wg.Add(1)
		go func(sub Subsystem, overseerToSubSystem chan any) {
			sub.Run(to.ctx, overseerToSubSystem, to.SubsystemsToOverseer)
			logger.Infof("subsystem %v stopped", sub)
			to.wg.Done()
		}(subsystem, overseerToSubSystem)
	}

	return nil
}

func (to *testOverseer) Stop() error {
	return nil
}

func (to *testOverseer) GetSubsystemToOverseerChannel() chan any {
	return to.SubsystemsToOverseer
}

func (to *testOverseer) broadcast(msg any) {
	for _, overseerToSubSystem := range to.subsystems {
		overseerToSubSystem <- msg
	}
}

type testHarness struct {
	overseer          *testOverseer
	broadcastMessages []any
	broadcastIndex    int
	expectedMessages  []any
}

func newTestHarness() *testHarness {
	overseer := NewTestOverseer()
	return &testHarness{
		overseer:       overseer,
		broadcastIndex: 0,
	}
}

func (h *testHarness) triggerBroadcast() {
	h.overseer.broadcast(h.broadcastMessages[h.broadcastIndex])
	h.broadcastIndex++
}

func (h *testHarness) processMessages() {
	processIndex := 0
	for {
		select {
		case msg := <-h.overseer.SubsystemsToOverseer:
			fmt.Printf("harness received from subsystem %v\n", msg)
			fmt.Printf("comparing messages: %v %v\n", msg, h.expectedMessages[processIndex])
			processIndex++
		case <-h.overseer.ctx.Done():
			if err := h.overseer.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			h.overseer.wg.Done()
			return
		}
	}
}

func TestRuntimeApiErrorDoesNotStopTheSubsystemTestHarness(t *testing.T) {
	harness := newTestHarness()

	// TODO: add error to availability store to test this

	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := setupTestDB(t)

	availabilityStore, err := availability_store.CreateAndRegister(harness.overseer.GetSubsystemToOverseerChannel(), inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = harness.overseer.RegisterSubsystem(availabilityStore)

	activeLeavesUpdate := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   common.Hash{},
			Number: uint32(1),
		},
		Deactivated: []common.Hash{common.Hash{}},
	}

	harness.broadcastMessages = append(harness.broadcastMessages, activeLeavesUpdate)
	harness.expectedMessages = append(harness.expectedMessages, activeLeavesUpdate)

	err = harness.overseer.Start()
	require.NoError(t, err)
	go harness.processMessages()

	harness.triggerBroadcast()

	time.Sleep(1000 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}
