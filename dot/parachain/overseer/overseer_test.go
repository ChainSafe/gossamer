// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	availability_store "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/chainapi"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
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
	t                 *testing.T
	broadcastMessages []any
	broadcastIndex    int
	processes         []func(msg any)
}

func newTestHarness(t *testing.T) *testHarness {
	overseer := NewTestOverseer()
	return &testHarness{
		overseer:       overseer,
		broadcastIndex: 0,
		t:              t,
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
			h.processes[processIndex](msg)
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
	ctrl := gomock.NewController(t)
	harness := newTestHarness(t)

	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := setupTestDB(t)

	availabilityStore, err := availability_store.CreateAndRegister(harness.overseer.GetSubsystemToOverseerChannel(),
		inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = harness.overseer.RegisterSubsystem(availabilityStore)

	activeLeavesUpdate := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   common.Hash{},
			Number: uint32(1),
		},
		Deactivated: []common.Hash{{}},
	}

	harness.broadcastMessages = append(harness.broadcastMessages, activeLeavesUpdate)
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(util.ChainAPIMessage[util.BlockHeader])
		msg2.ResponseChannel <- types.Header{
			Number: 3,
		}
	})
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(util.ChainAPIMessage[util.Ancestors])
		msg2.ResponseChannel <- util.AncestorsResponse{
			Ancestors: []common.Hash{{0x01}, {0x02}},
		}
	})
	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(parachain.RuntimeAPIMessage)

		// return error from runtime call, and check that the subsystem continues to run
		inst := NewMockRuntimeInstance(ctrl)
		inst.EXPECT().ParachainHostCandidateEvents().Return(nil, errors.New("error"))

		msg2.Resp <- inst
	})

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	time.Sleep(11000 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreChunkWorks(t *testing.T) {
	//let store = test_store();
	//
	//test_harness(TestState::default(), store.clone(), |mut virtual_overseer| async move {
	//let candidate_hash = CandidateHash(Hash::repeat_byte(33));
	//let validator_index = ValidatorIndex(5);
	//let n_validators = 10;
	//
	//let chunk = ErasureChunk {
	//chunk: vec![1, 2, 3],
	//index: validator_index,
	//proof: Proof::try_from(vec![vec![3, 4, 5]]).unwrap(),
	//};
	//
	//// Ensure an entry already exists. In reality this would come from watching
	//// chain events.
	//with_tx(&store, |tx| {
	//super::write_meta(
	//tx,
	//&TEST_CONFIG,
	//&candidate_hash,
	//&CandidateMeta {
	//data_available: false,
	//chunks_stored: bitvec::bitvec![u8, BitOrderLsb0; 0; n_validators],
	//state: State::Unavailable(BETimestamp(0)),
	//},
	//);
	//});
	//
	//let (tx, rx) = oneshot::channel();
	//
	//let chunk_msg =
	//AvailabilityStoreMessage::StoreChunk { candidate_hash, chunk: chunk.clone(), tx };
	//
	//overseer_send(&mut virtual_overseer, chunk_msg).await;
	//assert_eq!(rx.await.unwrap(), Ok(()));
	//
	//let (tx, rx) = oneshot::channel();
	//let query_chunk = AvailabilityStoreMessage::QueryChunk(candidate_hash, validator_index, tx);
	//
	//overseer_send(&mut virtual_overseer, query_chunk).await;
	//
	//assert_eq!(rx.await.unwrap().unwrap(), chunk);
	//virtual_overseer
	//});

	//ctrl := gomock.NewController(t)
	harness := newTestHarness(t)
	//TODO(ed): setup db with seed data (so entry exists)
	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := setupTestDB(t)

	availabilityStore, err := availability_store.CreateAndRegister(harness.overseer.GetSubsystemToOverseerChannel(),
		inmemoryDB)
	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = harness.overseer.RegisterSubsystem(availabilityStore)

	msgSenderChan := make(chan any)

	chunkMsg := availability_store.StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Chunk: availability_store.ErasureChunk{
			Chunk: []byte{1, 2, 3},
			Index: 5,
			Proof: []byte{3, 4, 5},
		},
		Sender: msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	harness.processes = append(harness.processes, func(msg any) {

		fmt.Printf("msg: %v\n", msg)
	})
	msgSenderQueryChan := make(chan availability_store.ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 5,
		Sender:         msgSenderQueryChan,
	})
	harness.processes = append(harness.processes, func(msg any) {
		fmt.Printf("msg: %v\n", msg)
	})

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()
	time.Sleep(1000 * time.Millisecond)

	msgSenderChanResult := <-chunkMsg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgSenderChanResult)
	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	fmt.Printf("msgSenderChanResult: %v\n", msgQueryChan)

	time.Sleep(11000 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)

}
