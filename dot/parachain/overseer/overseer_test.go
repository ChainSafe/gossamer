// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"encoding/binary"
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
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/trie"
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

// TODO(ed): fix this test, not sure why it's returning wrong instance of runtime
func TestSignalAvailabilityStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)
	inst := NewMockRuntimeInstance(ctrl)
	blockState.EXPECT().GetRuntime(gomock.Any()).Return(inst, nil)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	stateService := state.NewService(state.Config{})
	stateService.UseMemDB()

	inmemoryDB := state.NewInMemoryDB(t)

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

	inmemoryDB := availability_store.SetupTestDB(t)

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

func newTestHarness(t *testing.T, seedDB bool) *testHarness {
	overseer := NewTestOverseer()
	harness := &testHarness{
		overseer:       overseer,
		broadcastIndex: 0,
		t:              t,
	}
	var inmemoryDB database.Database
	if seedDB {
		inmemoryDB = availability_store.SetupTestDB(t)
	} else {
		inmemoryDB = state.NewInMemoryDB(t)
	}

	testPruningConfig := availability_store.PruningConfig{
		KeepUnavailableFor: time.Second * 2,
		KeepFinalizedFor:   time.Second * 5,
		PruningInterval:    time.Second * 1,
	}

	availabilityStore, err := availability_store.CreateAndRegisterPruning(harness.overseer.GetSubsystemToOverseerChannel(),
		inmemoryDB, testPruningConfig)

	require.NoError(t, err)

	availabilityStore.OverseerToSubSystem = harness.overseer.RegisterSubsystem(availabilityStore)

	return harness
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
	harness := newTestHarness(t, false)

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

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	time.Sleep(1000 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreChunkWorks(t *testing.T) {
	harness := newTestHarness(t, true)

	msgSenderChan := make(chan any)

	chunkMsg := availability_store.StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Chunk:         availability_store.TestChunk1,
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	msgSenderQueryChan := make(chan availability_store.ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 0,
		Sender:         msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()
	time.Sleep(1000 * time.Millisecond)

	msgSenderChanResult := <-chunkMsg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	fmt.Printf("msgSenderChanResult: %v\n", msgQueryChan)
	require.Equal(t, availability_store.TestChunk1, msgQueryChan)
	time.Sleep(1000 * time.Millisecond)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreChunkDoesNothingIfNoEntryAlready(t *testing.T) {
	harness := newTestHarness(t, false)

	msgSenderChan := make(chan any)

	chunkMsg := availability_store.StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Chunk:         availability_store.TestChunk1,
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	msgSenderQueryChan := make(chan availability_store.ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 0,
		Sender:         msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-chunkMsg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgSenderChanResult)
	require.Equal(t, nil, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	fmt.Printf("msgSenderChanResult: %v\n", msgQueryChan)
	// TODO(ed): confirm this is correct
	require.Equal(t, availability_store.ErasureChunk{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestQueryChunkChecksMetadata(t *testing.T) {
	harness := newTestHarness(t, true)

	msgSenderChan := make(chan bool)

	queryChunkMsg := availability_store.QueryChunkAvailability{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 0,
		Sender:         msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, queryChunkMsg)
	msgSender2Chan := make(chan bool)

	queryChunk2Msg := availability_store.QueryChunkAvailability{
		CandidateHash:  parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		ValidatorIndex: 2,
		Sender:         msgSender2Chan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryChunk2Msg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-queryChunkMsg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgSenderChanResult)
	require.Equal(t, true, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-queryChunk2Msg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgQueryChan)
	require.Equal(t, false, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStorePOVandQueryChunkWorks(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := availability_store.AvailableData{
		PoV: pov,
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(nValidators, availableDataEnc)
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := availability_store.StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       uint32(nValidators),
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	for i := uint(0); i < nValidators; i++ {
		msgSenderQueryChan := make(chan availability_store.ErasureChunk)
		harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryChunk{
			CandidateHash:  candidateHash,
			ValidatorIndex: uint32(i),
			Sender:         msgSenderQueryChan,
		})
		harness.triggerBroadcast()
		msgQueryChan := <-msgSenderQueryChan
		require.Equal(t, chunksExpected[i], msgQueryChan.Chunk)
	}
}

func TestQueryAllChunksWorks(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	candidateHash2 := parachaintypes.CandidateHash{Value: common.Hash{0x02}}
	candidateHash3 := parachaintypes.CandidateHash{Value: common.Hash{0x03}}

	msgChunkSenderChan := make(chan any)

	chunkMsg := availability_store.StoreChunk{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		Chunk:         availability_store.TestChunk1,
		Sender:        msgChunkSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)

	msgSenderQueryChan := make(chan []availability_store.ErasureChunk)
	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryAllChunks{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	})

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryAllChunks{
		CandidateHash: candidateHash2,
		Sender:        msgSenderQueryChan,
	})

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryAllChunks{
		CandidateHash: candidateHash3,
		Sender:        msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	//result from store chunk
	harness.triggerBroadcast()
	msgQueryChan := <-msgChunkSenderChan
	require.Equal(t, nil, msgQueryChan)

	// result from query all chunks for candidatehash
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []availability_store.ErasureChunk{availability_store.TestChunk1, availability_store.TestChunk2},
		msgQueryChan)

	// result from query all chunks for candidatehash2
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []availability_store.ErasureChunk{availability_store.TestChunk1}, msgQueryChan)

	// result from query all chunks for candidatehash3
	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, []availability_store.ErasureChunk{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestQueryChunkSizeWorks(t *testing.T) {
	harness := newTestHarness(t, true)

	msgSenderChan := make(chan uint32)

	queryChunkMsg := availability_store.QueryChunkSize{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, queryChunkMsg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-queryChunkMsg.Sender
	require.Equal(t, uint32(6), msgSenderChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreBlockWorks(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := availability_store.AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(nValidators, availableDataEnc)
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := availability_store.StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       uint32(nValidators),
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	msgSenderQueryChan := make(chan availability_store.AvailableData)
	queryData := availability_store.QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	msgSenderErasureChan := make(chan availability_store.ErasureChunk)
	queryChunk := availability_store.QueryChunk{
		CandidateHash:  candidateHash,
		ValidatorIndex: 5,
		Sender:         msgSenderErasureChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryChunk)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	fmt.Printf("msgSenderChanResult: %v\n", msgSenderChanResult)
	require.Equal(t, nil, msgSenderChanResult)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)

	harness.triggerBroadcast()
	msgSenderErasureChanResult := <-queryChunk.Sender
	expectedChunk := availability_store.ErasureChunk{
		Chunk: chunksExpected[5],
		Index: 5,
		Proof: []byte{},
	}
	require.Equal(t, expectedChunk, msgSenderErasureChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoreAvailableDataErasureMismatch(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := availability_store.AvailableData{
		PoV: pov,
	}

	msgSenderChan := make(chan error)

	blockMsg := availability_store.StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       uint32(nValidators),
		AvailableData:       availableData,
		ExpectedErasureRoot: common.Hash{},
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, availability_store.ErrInvalidErasureRoot, msgSenderChanResult)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestStoredButNotIncludedDataIsPruned(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{0x01}}
	nValidators := uint(10)

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := availability_store.AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	availableDataEnc, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunksExpected, err := erasure.ObtainChunks(nValidators, availableDataEnc)
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()

	for i, chunk := range chunksExpected {
		result := make([]byte, 4)
		binary.BigEndian.PutUint32(result, uint32(i))
		err := tr.Put(result, common.MustBlake2bHash(chunk).ToBytes())
		require.NoError(t, err)
	}
	branchHash, err := trie.V1.Hash(tr)
	require.NoError(t, err)

	msgSenderChan := make(chan error)

	blockMsg := availability_store.StoreAvailableData{
		CandidateHash:       candidateHash,
		NumValidators:       uint32(nValidators),
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	// check that the data is still there
	msgSenderQueryChan := make(chan availability_store.AvailableData)
	queryData := availability_store.QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)

	time.Sleep(10000 * time.Millisecond)

	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	fmt.Printf("msgQueryChan: %v\n", msgQueryChan)

	// trigger pruning
	//harness.triggerBroadcast()
	//
	//// check that the data is pruned
	//harness.triggerBroadcast()
	//msgQueryChan = <-msgSenderQueryChan
	//require.Equal(t, availability_store.AvailableData{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

// TODO(ed): use this as example for activeleavesupdate signal, and check pruning time is set correctly (trace this)
func TestHarnessCandidateEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	harness := newTestHarness(t, false)

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

		inst := NewMockRuntimeInstance(ctrl)
		candidateEvents, err := parachaintypes.NewCandidateEvents()
		require.NoError(t, err)
		result := "0x9001e80300002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f3282a8119a659fb3c94c3c951e5215daa35c741d13719d0be7368ca8e089537b5b3704113bf3c0d7fafa20033cad97002e9b9cad03fb998cb35cd71a9873d87a6af72afd5fa059b18e2e13eb3ac6158212b57ba060772886bf962b12034127908e54749e8014a7525ca4eb8815d1565185b31aecb38121e4347b54ccc8e49d13e2a732f9e332e0b425c741a07206c71d801011c2c0a278c87fefbf8064e0631875d1ed14e57816dde31202e999e6e24ad0f8ae2fc830bc1d325e73c3f3dcd186be8e59074c5553d8b65ed0a7034b310abd88b894263dd5869a8cf87e5ef4f0a79985e134020e8a1e2e211afcd5ac9ec6a2fd21dfb5e1c39b3b670f4415e904061adfacb103f25340b938dbfd331ba478351f9173a81134bff8a8fa1c0fe6be68e9025ea1f9f9f24e7fa5437238b01826b451001da46638f62b334300923068d1691f42d0e400b65ae90c283a08ee958a10a0e8c33eebbcb01d137f1612e39973a10489164135f64606c111f8d5d21ec2f8ae932676beda36b07e3577c7d8103c7cf2cf878ce90806617572612083635c0800000000056175726101016f5b6937ad29ad8721ba848694820840b25d91a631d2cf19f8c63ffcf1fcef260d25513fbaeb64c090d371bdba09663f2cd3bb4a8f8f7838b6620bde5c6e5905000000002000000001e90300002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f84b5c6e57a116cba4148aaad37fbfc3edc54e7728b5192a40391d9c8ef33bc405ecd07dedf12350fca80ea9faa4288186ef3dd0b40badbbc49f858a38cfde88c2bf80adf8f3282e527e242daa821b07b2972d9630b1eea6d2ffbaeb90446a8bc62a19241993b622e6321aa73a113c354958dd7266e7e2a28acde078ae276bcf7cad7aa37dace9d97c31549cc22292b650a04cfbcdc44e20c9225d499c8241735168858f1c5382ae1cbb9ece1165e63f01471338afe7714b8928ef3b2a15e6f86fb4ae217da22aa2880ae8d874d21f8852d76ce7d32554830cb82467a66658784dcc6cda49b51efbe0fd184b974074d1258ca9f15b9fccfedc35b7dee6e1ab9e9bc06de0d02e4b3871341b268751813d0fcfbdb0bbb2ec4fa9798959ab01a48dbe9029ccf0aff2230779960fcd86efb10ab9d36ee09ed20b3404b8da50f63a2f2aaeb6e43470084a5205b111c66b80cabbd010043085f5ba73c3a4e0f19105e32274bf19fab97145aca984aa0eaaa60c4528cdfb9aecfc2dba582729ad01ebf8d72ad39f1d70e0806617572612083635c080000000005617572610101ecd027c637c53ee91d7a0e98cea0e7e72f26f69415985ee9e28509aac917565522d428904e56757639946c30eb6228f11feae3bed1e1e60458922e892f89ba8c010000002100000001d00700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f605b88b35bbbc7463cd981713be480f1e7ea1d12b6c21eacf0c120126d24347f57dd8f8b64cdf03c5a171480ed3b0718ed0d2ba49ed0bc088b33c45e9f40c929db5ceb08943ef0f0cea09c1606e0822d9bf9f0a7297ac08335d13c87f28490ad2532a4846d8b951e37ed71582b2562027373adc871ec734cf1b77fd6de383cd5a2b09b240d86f318e67f5fe77e4b3714d5267699515c6696fbf7eb8d2150826f89faadb98f6b52ca914fb474ae9d10ee57c48669ebcf6568bc6c0a8833466f8174c4eaaceb528233d6b6ca4823171880f8853b0f02e6f829444f6eac27fbfaf477bb8d6c11c2ed5ee8a4e0933aa661d2e44e03dc8823cb180cb069463128a2c766b405ebd52f05687eb3288165c4808ab492750a0c0ad69f3d29134eb883cf5de9028ad5bd67ae9e27b97131608d072644251f45c38a49ed0ac3dcdfe3e6c6c9178e8620d600c955c50fe95eecec5125170c2ee7031c13b46c6c64d26278c805e74f5d98a89770c37ca072e8a11ec0adb6e178a689d285b066c93bf63444a1be78977081c6fa0806617572612083635c0800000000056175726101017a9dd3c3f5b3c34e7804b2a4e92c08a4aba153098f51229d81be212fd0345e2be37a41cbe3265e448b8afc03475e33b2b9cb762a9df3e04bcdae19af3aef0d8c020000002200000001d60700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fa23d6b412989d9b453ba20b524d14a4932218d5d20cb57529b5d8481fd9d1717c79dc1dd9fbc475efab87f98c901499443b467cb30fef5e85ffb9429b028fb69b7632d7583557ea812404498c4c544c574b43b952c0f642f208a47c837871974d324446e0b5362729c6f287ae6781da900bf67a02a0675327c8b0c4eb716d79c12dcfed6687a9e1cbf5a58b4d4c1428cddfc93b6031fe5a45441dc6a363f43676fa41104feb13bb292af09d5929eccb40b00b1ba9856a4a7727e3a82f9689c8d58eed3e4d8a61504356c2f5f230bd53ac9ad08605c6366a65ee919decea46ef17520623e2407d6c2a89b4c660b073fbbaf0418452b0959d1018293e923cc0c70ef48a7f0993dd843c38142939ccc1d8937f7600ec33f8e203529f3c0bb741bde0d1e56f9f2bbf7b0f5b44f5a8fd26c1641938379cb13bde0d422c3e5db5675964cd736a2d500e8c1d62f574c4a67bc94e0d38ce45fa94b7fd1af9dfd8c1902f34802a4f849af258f105cc555928f36e117e2efd519ee691784304be5a7c7fdfe5ba3128576150c06617572612083635c08000000000466726f6e091b01ccd9ece066c3e5c23701aef8a8d166534049e0866bc96618feb63b27516af2bfd4c90a72b83065ae02bc073d72ec49901539e3dc74b84025913b2599c392f6daf5109eae1cef4251a2652df775975fcbd3267c254655f0623cff8d48c24d3a0f935472c4b01eeb46871659e4085d8afbb3091dd6a9559d56a889ecdd239ce4cafa23ba44589f0ec8aab1726096d29ee0c9e2676cb1b26cf2da888205cbd2dfaf1d2e87536aa7ca2330a777b9a43a8523bcfd9d9f041d661c688a8c2791fa5927ffd02ac2ea36c1f3f8a3250b94398171098e4de128bcc6333d68e8de6176f3209f66cef788ef9cf9bcdfd1560c5081311c00fa0d4a4d952bfe6c879f3752f7a1015e5ecbfcf729e0713df598b79f1b7981acd0595d8bdd4cf33f3a3b4d44bfa16b0fe8275f58a908984a0cea1fa464abe66705577c82cb660ba42391739003a6ec0dfccf329c2b6d81fe7ea47987e94b00fab2444a1c34eb3e1be1a21a31b6bf0a2c01828ca4247cdff7b0769ecbe0d186ca9af743e0ab05a823ae65bfe7fe871f7d1e5e2fe06d88139bfdc57f9169ae6f869c155bf1dab7d97ddd8ed0b31e2d7b6a1097a9e0e1ebe09539865f3abdfb20d83f39cd9a9ddbe70f2264cc5cc34cdaa45942ce51e3af015302a9a25df2ddc220657322ad6b03e843c24779d4ca45cc4ee30a6830e3c82855458becde33a4481a1e0bb4a548c244b7ecb8be06c6cd182f5e7d3e39010b0cbaff53d43f3bb471e994e01e8e79fba4dee058c59c0eb2364ef45fc1371f62f14dce74c67cafe44cba43cce596b6a37b0044ca9613180029a14a77908f74613db7650b31a2176cfd2cf7ef35e680b45d72bbf45c56a4b8a6327926f708a899548f224faa0cc95e0b58066d7a526f92bf53524d25e5e25c39a9f7523576f0af291e0ae37c7cd5282b14737a46a01b52d5b0a46b7c4674fb28551d2db05d6b6bf9ca560b49911d8087a6898b672e75d9447edf4700e7935cc0899cece13496a9f53e3533ab2e2f32814e36d2162d1316497ed6112c2b8fdb36bad16f697098f1445ccc3ff6efd641cde1b90e8012eecf0c3c5a005eed41d5a312d6ddc76c78b2ddc7219bcc62dc3c2e0dd84ee7c161a2181bd844830d8a533fe8a73ccb7af3a290f1cfd0758182572b5a4b9da22f93854ae35d0b6e79f7b289253ae05ae173ade67ee65f312577e0a1623b4c14982f33db8e10ed43874de707843f49e7d186a0b3ce48f5efa449647f230eb490ce8ab38fb460ebc8b6a26b5331a0dc5224b1914fa1c9a0e98b2092769b3e46066aabe127f3866f55d56faa702b5ce9faf3fbe22ab352469f5f86719624375497b2e9142e12b3f18dbfa0d6545c7637eb25f6f0243e14a7de2a79f5a725e484bd484cf4e8af7b2ad7e781bf6d5cc72b0f14abc95a407cba0bd01c4d47e192d9eced14bc33d509313d37ac40d2abd213d3961547fec1df1ce3450f1271575684cfaaa6867d7ac9b9fa9e939267e11f1bcece14cad2c9956fb316b8e4063db5389738faa82e1cac2685c1bb14c35f2846b1fd44076df9851ff3f6043fc3824e44ec0fd70f684d2140525e37f1ca00190672cf0d3f496d917e4e63eb23cf89aec96c62e352bae3010a165807e51ee2d8e90178f9285da18ed954fde7c7614fe88015f06cd290455c05667c58cd8210dbfb528bfeb70878776e9ace349aafe5ff9340b4598bd70097f8faa9e37a9e3caa2a628e67d83117107e99257785a45c144d29e970c424a0eda79be84fb94287e56f9fb3e746da3847a56403e9618a5420e93832989fb58116d712ef4b1d2d23659682053b38d6410322a2f07444ada2aee3ef85d631e01cfaf5ed28bec9dcb2bae747f6c63e9e4a12a8814da4ec2ac713565574227d197851422b823db3f5eb669fe9ea324f0b939835b73eab6eaf8393221ce5540462b71770bbaa8b77682198e9830f75537221146d306ad07bf3f9de76cdd485a669607d9c5c8c517cbb87730b6e90a783b0f1bad7aac92ae0a0972a8424f6683c42d78400442e9271108953f1d40e0af58e8da33c1d9fc1aa77b572f0672a3020ef1465b1687381cd70e5b1ece01a7c08be69f55b1e0683c1f742dae07de6657e6e88f97b6d3314dc30e461ae492f8e3a180ba1eeecc64a856be520f8198a1c2a4e0bc6d295a2fc13d4bb81ada3553923ece3cb67047b7347ffcd13c6bae49885be17de3d478d3aa8dd2f964ec3ede11d84348f7a40015e588ef9c9a446a3699de1cbe1afa916d10356f412c32e085c3ec0115fd488d3785f9552f893bbe0b22c3773df967b7e36713177151c7c21b9b9b1b367953ab7199fb03947c31a57a94329c6edb9b6ca20d762423fefd38b1ef2ba3f179998a5a5bce656d8b5f14dff06715909d5b7a6343003bf33c0377003d680eeafe54a348cb7815e33c842eb658ffdf3151507e496343305617572610101744acd32f811aae5d80e20534e44195d03abd773156a54dc42204e7c33370702db76b09ff7448955b79a53f0ae958312e2bdaef5a6662350349ce955b8561286050000002500000001d70700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f7efb9393ca81fe0825670473a2d400ac00ad22c1f401a14bc154c5f92ad2443192f6141b3e3c8ff6efa5a85a4433a49aa88b225d64a40a6c500854222e4957243012e86d4a295ca3422a771ca745241b2ff615dc7a71758ca806b1f48525836b78e14daf47e1789603860a916feb342edb58a2b7acb095c4e93a52b987442ad39c854c794e49072f464fd7e0a418d9c7eaa790c89320f830a4919c109dd9795bd4c96ae86882d18ae5a8edbebc1be36c269313f50ee8e1ea30ad62770fe9f88ed5f6cf4e7fe30b0808f01d7b1e9571367979d1cbb8359088cf98ae22e8abab91ccfad85a4a6ce77f402b64b0f0524a831daf156f8af91f9d5d62ce4c687947c3fef1228b81636a938a7a242ad5f78aed0a218d582faa5176c28748621c0e24b9e902757eab60aabf67c35b6f64b7dfe695fa70c9213e15d0cf39dbc547924a5b2ef9f69d630014ff54af3384cb0391d00b06d7309bf0869519cba60f2bc41be171ed2f372f3e1b71a4f5d6958d3b1fd4e43f3173218aa9092f05bac9e7f4cdd4ec490f4d177a0806617572612083635c080000000005617572610101fab1ead14c7c2cff2e1f1ba2f2c0fad3d5a5d5b6c3cfde4e88dde08d978dfa1da06bcb5daedc8e960b11f9012473e7c237b5864609cacc9b4fe9832785864b83060000002600000001d80700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fe4bd11531d6789a4478dbf7a36a3c7d23b58c5a1debdc455e1fbaf1b5b2ab412ec6a464b90ff6ada8a89e3f8d180e74e293a8b2f01c8499ddb1e994293c4c0d0779b68a85a99a3e2d4e0a792e1537c07b3a56fc0a637f1db059d9c2755625b778096d82240cfa6f9fe0dd38b9e597504a42e58cbeb7919c789a917450ee00e73fc355482abce20f5b06ae119b1a9ee79866d5b4854a389550c9d9580a0c0332260e7a1a1b7d1ec159e23065c342070603750ab953543978b4c8f018b2565948ff775ceffb1e33fabccc1c2c6ed03cc463645aa63019a0e9a8aae015cc11c6c4d370085df65a76eb880baf100b4c7401d011a1fa1648876798a747f9aa372d114a81db419e0c739689abf0882f2dc771d2b3f247cad2cff8b1630840b4892b8d5e902dd6b8d78a9a6e1651c21c2928db2b305d78290e51cfbb807ffbf93ceb6b1696b1e0d4200471f7d3b2dbab4c04349ddd829a29c58d75f91bdf56534369705c46523b29229840f9bf10916cad670f0d3727bd747cd737bfcc525bf923632dc5957c032aa800806617572612083635c080000000005617572610101164bf5fff13882bf1a85e593295939d26115f85d9539faff0d76d7be8b7eaa59ea1f86566b58362a71c19076c5ed65d72cb79994091851bd8811193bda730b8d070000002700000001db0700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f0c743a94cd1707ad4117ab058e336425f029d591d3a0d48e895c732013c7085053dc502546f00302e84fc41e74a7a310c556481c76ff18fc1bbb97c457866b972dc5d56deb74248c917228b08ea5774022ad965420fb4130a53edd44dbe883c91beac7ff23003ff41339e09470ba672bd4b9de9d51aa32ea9c4ca080c6f1f9820e698a3c6c6ffa8523ffd01d125f46795c6cc3c4047c60b36f44d1e89ee037319344d5b89f31dc79f5db9e5aca9ad7925ee5b750c1a78ce34ce503d4fa4f0d8dc38992403757e8242f70fa6e3898ee68b82eedcf933db38e0adfd352a1b6671ec7691f79817eeba28caf545200d0e9df486d47af1c5ac79ddd05aac62546f6f8723aa1c622a8d4a7757a130d4c7a24e2928acd073b425bab04c5e960b4ce22f7e9028813af0548dbba39f8e667b2679069aadf411ad3a194a5ca50253221be0784058e668c006cb6f4c83ba533b775447a77060f2a2fc873d79045329954a814221b7b6d52b96a216e614a1e963366a98b3e4fcb728ef8c2265b2a6a0b9fcce0ecaa41f094a00806617572612083635c080000000005617572610101d8d56b387d8b2b434c1c4d2186a48765e9ee91a39b214c4808eeb9f3461351196740631854c04bcda5980e50b7cb759df2f64ceae11f6f321ed05f8e6323aa80080000002800000001dc0700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f08fc371c2ce887eb3e132908c4dd533583177847b8434f5c7c0cfabd4852c24e6c7ae7aea674dd4bf6796cdc2d337782f885482bce24c9030e87b11bca54d7f5e50b6d38ff0fb29a9e76bdfe49f6f82140d112c80a357002c99a8941a08ec62190d3eda2b3ccfeed95db0ab62327db71f640621d80c753b4e69db9868dedfa7da60ce0b09e321b053d10dff1b04e435472170d760c41282be74154810f277b1360cc17225d667bf15fb35266c8cab27968b58813ea1ce41baede882d43bc1d83e973614a3dea1a30b67783af1e57f732e89eba9958b0fcb19fee08f9a3314eb9b44a574641d090ec0cf776b81af6ac92015443353b22a7db67f42bc7347a67aa8f846aa19dcdaf075ea1af3c7f1b0f66b33bf8505b7e94862bcc0c4f6e9eeabe890371dcb58c1c4fb0f5cad90c3ca6f598d596f8e47005536da04f424868b9963e9256e1d2002610491cd018b9dc0c88d6c9790b14b4b866b6e0600feffdc0c94c667a68c14f2264ecd4a757b0f14bf0b6d9ba93cb7e2e949191c86ee32c5402d158b85fadaf0c06617572612083635c08000000000466726f6e88016d1ffba7d4ede2a4c73b3740c4a775da3a0a1fb08e3b989ec5e49684d663be2700056175726101015adb821043ece09a08d82ae2cef89f37d7ca8f4f9350efeedc12db5884a0016763c60aa38c1abfccacd5f9ba16c4a8ab93f6b9177479100d0b6c975015a82280090000002900000001dd0700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fca221b74c8d32fa6a45bdc2271371109a97cdbe2c82fcbe452c548874b7dc1665af5692ca9a27a9d2c2f55affabdaa466fd435945f091b45944e0f0596984b664b7f5bee6c782d18ae489b645be2afffac110bc7bd97840260a807550ee056ca22a4b9f9c32f4c502ebe462083ec11fa689a7f3fd948fad1e3cb5ebdeb4737c7ce6a2e873ae8e530de6065dda3890084d568507ca5d9a69df19683805bf5d24ddf938c7c9decbab422bd00c4757dfe18489bff7e017b40ad9552871b9a7be88a4e315ffb734e34f9c25170faa579eebc5b74caa8cac8b1b87827e84055b256d35c644799d29d2e6f4f2c065ca606744598719d85f8c3e7c4d39abc3ea59be4a35ba689f6c0af4bcf5988a4580840e313618deb3b835c4e4a78cc56434b2ed6e6e90231a890359de5d57ef9864cf19f3a48c1fd1a63812b0765661c598856f956177cb62e8900830038feac3e62aff1f1333ad0ee8696a4558f6cc42b876e0603d703ab0d7f0eea80b0973cfd2d3a59268bf68211e1bc014bd94fd917b6e9475c3a78508c03fc0806617572612083635c080000000005617572610101b86bda32f157f1e933143a934f6505a6690c0a31dc52bda5f49d40e5e26073545851c15949664b7f55b8cb2163a3caf7f487d8b4cc87752aa3b4f40638c2ed850a0000002a00000001e30700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fac991d1eb1203b3ac74ce441fa62e97c8d7b99628a85cdab3806bc68ec92131490b0d19498a2d0ec54c48572abadcde6a5a2aa9ed3ab74314fb4dabe30ba6408d4894734e270af31d21c80c4e9e3fd540899669aed2f9bc7fe28891bc0fa6ff9a58ebfa37a19b1e8479d3d41c7b78204d9719c8ad1212ef0bae7f2cb82746ddb6816850f2467e875176ce398d1607d13d2f458526126a222d47f10c13396eb069577928443d3ca93d4582e1291c0c7caa7bef8c1f39c9863385e472ff1e9b38d85bafb3d2fe2ca2354c2fd1fd197b36bcff57292ec4e7b8ffba280227b13aa0247ba47d4824fd30328f70f64b056d4e36e3b75594d6f590e1b85e90a65f9a63ad6d42232ebb0628b11aa6a5bc31c1b8ac45f683b0290b14ea725b2b53ec8017ee90265f1561c96e26a3ac239745a0746ae6fac255bbc9a19aca4765900819e7a532c12d1760059118acf89d6a62c0f724e283081c8bcfd74a1db5435983c6cb67272699024dcc1f7c4089088d1fd7a4ece328c3611ab9219a8e0f43240841673317a5d3932a90806617572612083635c080000000005617572610101eccee865e2c48f93bf2d1ef99bf22549e3994e74b2f91cd00e742be73173cc40e5518cd480b62778a627e15268c8439cde21a27eb24fe0386645a5b11430c1820b0000000000000001ea0700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f0cd0c3ce103900b501fc82da6f8999bfd42354f8fc4db6230a764b7926c75118d570816caa86a63f7fdf4d0e0cae604060a1ca4d53a6bc0aeda87d1f5c28918b5f8233fecb446796519206c3b97eddddf86237cadeded98b041ef6ff4e474e53287a1697c8caf461dd58930f749d53af8832c488780a012590b005c36eda0d6e16c775b8eb42663d5517b3e9cc198e9658421ae7af2a76706d64e3324ce39240a32fc27041caffa0ead9c0064ae0ace86242f8074273018316dca50bc96a978da3883c3993b1a32b26325ff85be0b63084960a8b50132433176fdfb5c6ffc07bad1936428467f481ac53e71165512663839cf61457fba5ff17ed63e71b8c717680a10e1f33cda8f8d81c50c91152776a54b50195d6ec7f0d4bda452f9ac73012e902dc413d61ee372b7bc1305b79f10d3644a42c88cca1c4c8539760bdcced3d8c2b16819f00b2faf4d0fcf22d03a7cd76920487eeffed5b599f6cd32ccc9a64801fea2aa079feff2af002f0a71bd1fac90e9fea3a0573a5bfa2765a3a931bb5b8d55c9717290806617572612083635c08000000000561757261010166e2910c9513e1dbd2a6d3f938cd6a6253fbb2cfcf6a1f6cd76a8b7941b07b36824ead10eb5d99188c749988b49b9bb673571fac8b9317235b89b8d443a5fa860d0000000200000001f00700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f0e4c8eb3dc54e5c0d1b9402f44f89137c1c2d095bed90ecbb80e38c5cf245c4828efb3599bef57b84619255f76f0aa168ba42a0c7d3a7e519bcaa784bf0c66e38b0a4168467cc53f93902feb5d97440405d82d672f86e440c362301c6283263fe870ecde7baa2895003dec3d07836c095ae4f3e4a05347dbd7d7c4eee1f811e18628cc961b32b53e4cdd3a60be566fd3ab6a686041c07a8d4750133047a6907232365380453504c52c6b96438fdabd2d972e3dfe6bd1e9bf0f254bdc07583e80acfe461dc363714109b6e04784803614ee2b00967cb51eb899327193cf485260130b0b9ac802b3db074f304ddc428950e80ee3e2a14d9f009d8376fdff493b7da24d7edadba1d6f9a8bd642f426851bab186b7aae7eb8fe526b05e362b996704e9028625ec7da0150afcd6c1431ed2e8b56be1dfdbbbcc7e3eef1275b5bcdcf5f4091622a20013b839aa611349e4450391abb30a5e0bf166d201351ff2eeb43d5057714a5b6c959ed33b8bae5fa99ba5d0154e08ce851e94502c8cd069c78c0a0d748ba782160806617572612083635c08000000000561757261010112e21a778c7c5404d9cf20fb4ecfe2c06b14e3e5ce45c76619077a2d47920b5e2d605528db37f2d4724f811429bb3bd308c9a20ac0d05b91329bb6b49b3f5a80110000000600000001f20700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fa6703abcf0fc554a4cec1cb5677c287f06e3bfc3f74bbab462dcc1062c88ce353939f9e6aaad990f2c9837dafd9f5a22c17455389a73f05570a74aed86a63b1ca65a09155dcea6eb1d458cc292968124a0f9b2bf125cb797ab96086dd3ed40da478e5401533e30c628939b081e4e383a28a6890ff949d27913939b7bf5d9c4dc0419a23af2d8a14090a06412179f31a575a917c04973f1f645d777e5ec48c938ad24670905b57cb5e29d8e2b285fd25778a69930cb1e71a366be9594f58c1a80322a31962e767ce5d6aed172950bf0d2ace5df6050df675a4e6a58a4c308372d89fd8d101e7dc06f0ee9a515dd338f2d29098485ffa2c2eff98c6bcdb6805b483ebfb1a48db52c4f8bc5cd264c2746ab4034bad57e11be369e459fdda1bc9810e9025375b7d9d546b18adc6ed91373ba5ca0a1e0e2549458ed06d4a9ff337dc4b7f11a9e9700a5ac8cecafd414c670faa4c56cc84c1efdfbdd63620fb2afb77983d5bf004a11177c31f7cac28712d992f31156ff0627678961f8ac57d3a41afede86cb1e31d30806617572612083635c080000000005617572610101e8bc4a1acbda300bdc4da4c20caa87ec14e7d91c04718c20d677ba82a2adf422e72d52c66fdfa17eb1fa426428ea9f015f8ec3b4dafe3b59e9062f0e8a4e3580120000000700000001f30700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f12528c5a9d2d33f06fe0197fd72642634a924caa92f954c84d0b30a5af9f621282ef2d609cd65387717749105fc4e7266cc29d8fa13accb72282526ac1e28f285f18f4e449efb717850369313139477445e5f16e58c14d5fb079d1e95a4c15f9bb6baaef2e82cfb97712dcbcef475efdca635371ff35d5e300090491f05aab135cfff964324889a26d9a999c5644bfab36a5f2b900df8dd33260f70ff4685c212d423273981adabcb2d89ab22068042fda0beb26bffb730348f31d3d739eef8ec6b6f7e799362b07a29726bde1d02cd6fd231852f6d0df84695832f70b69df9f724160bbde1c8a99b1f8014f200f6752e90265361f4c1e55a576e58a0a1ada06c73c69f77b4076a93404329f96cc0d9fa7ff77bfd5e457f1caf71688c95cbef6e9022b8b83e8888c042d9e2f0fb1a9bbc456a8b07a9572f1e4fafdc8d699438a9bb296be96007d6837a96593348603378036a8638aa191f7164617e732af28d26929f83245736740228224f3d42f71eafba39c8f4faa94a01a589f51929cdee85a28b7895f760806617572612083635c080000000005617572610101a2f595ba846acda337e0a6891b3f517de907a15ae96920676e7aa300c603d903091eb30a8613fa54a684f1aa488272ebce844a5f9c44a0c56990c60f6149e882130000000800000001f50700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fba252d10741845a93492f9ad5fcb431e93c51be0f71ea6a8bd164093d4fc9f06df0b8c4567e4808dbbde228cad475e6060a923c214a5c4636f556eb42a7a4ba91f04569f398c5b44d0aa5cb46a0e95f958b5a3e8352da4092daab9cd8c4859bfcfd415dec6748a823e1141a8ea36b283cf6384420f9bfb21b22579a580402394ecccae790c40a892f3417def8c361ff5a86b33779c74316c77c548c4612fb838c362ebc4332cf2a926e56af72339da43cf00bfcf190ab013df11fc575919858136699ab86fbca58f8f5aed50b70c125329af294571a4e1172e3d25b8d1c0894079c48d94b4f1140f83e68d0517db11f4d29ed81a93ba725c0516ef4d81f2543579a1430ae725802df267fe527b925ac709ea5f238c6b87cfbe7ecaf751855e688903d303d17a8af293a6d14b5f06dd6ace4d5a465ebc84d372da88b1d00fba851237864c8b00f9c1ba4d0f26ecb08528e3c156de07ee51cdbf6fe50c332a727d46f303f688f8ef451f0175c1f5902c4200a374026f4e1b73706cf03f33dc1a1a2fc0877346cb0c06617572612083635c08000000000466726f6e88014968c2fd312c9100837a25718581fa2438136bfce75918d9974e85bab2d26a6300056175726101017a5c5bd08201a067464db34a431fdd575008542f500582eb799c010c99c28f325a1092ab8501da0a0e9dcb07dc7eb849c6ade998b512282ce5786fe3b84d168c140000000900000001f70700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fe0b5632409e78aba11d0d559c96acff0247194879afad0741ae3848f61e74a02397ee4075a31cdfdf00c080c5ae192e9a8921bf216eec48cd06a0ec54f3728e627e68a325aa980ab408e110b842221f262f65fb9c9ff3bf2526d7a454e1f5bb31c4957f9abac96eaee0b0e4592b60fc6abf498f2b216099590c593fd2d6956525001477ea73bc4f27bbf93246c8ed89995f7e69d9e003c4b80bafac51cba1b7a8669945a41c3693a006a1cb8a9000b443f45cc4db5b1ccdb50ea7c8d7330768b445c9a763b1e6309dbc77d14e1e27ddb362cce7ef9e4023660b6db4e520bde91088b3f2153fdc8dedb22d5984ca5773ab7e794d3a9a6f700d1fdac1e54ca83a9455b24f78b25d8dc601e67cb9c1a60ea2e7e907576b4579a091566ce062d7b9fe90294e33d043375a9630b8b43a8176a4b8620e3341e1e22e9ea344ca3bc8b6d989e86ea5900c0bac6b93a608c8ed70554e38d2a513b20072d9f2556008e9cd917066c54e371326902973308c86b18c426b4dfe19c571cd158847f719b5b23daead339f4889a0806617572612083635c0800000000056175726101018a4d75e4d4af8ea9dcd1ee5c8d18659be74f9e3dd4af5312f566705f7fe08f03b6453775880c2d6cc9552f0c99462de4845de92be70ec21d537143377fbf4e8b150000000a00000001f80700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f30cd84fed2904f8f55b7c41d0638af65a209ed81abf76645862f999e1900273d800e3bae3d3c043d319ff1b86f66954e336dfb77f4f62e2cba632ca321a19edf3f600c334e02e5f6ffb1b0a482e3d1ff6e8da744cb98ff90f4e32f87f038cc682462c29e53670de6368795f8e42b58b565b4b0edaee2dab8a1b9e7f2c5b485b4ece9de94fb93817787e0a41ec5703599be3d47882754fd9de578996f9c23f441060bc13cea84d987edb6859731738ee0046151a27ca5d150d89d09dfc33c7b8ba56677abec76be5329edd0100932d6ad28aa84ec94ee0f38fa28dcd7428370186c113fd2ca1d93c9c59e2047eea8b9d0c6d52e15c73398d58433ecaf2c85274ccdc35e3d74de2af6b8b3a2af12ca02f9f14c0a168519ed83cce058a4409204c5e9023d756ae602b1db65712f5c037388b6162cc31b73268aff4572f0b02f706d67145aa38d009ee4491a3c7e2d63f2f2bdc16c70244435ca7f4b86b282a82aec767fac5f02f7489304e192850975b4acfbd1825ee836b803947e5704b6dd1ff7c59c44adaf950806617572612083635c080000000005617572610101d8fd11466fd2a4ea198104ce9c2d273b49e32dcac2900be2313085c941ef415464e0f1fbd3431900521af212814b266900a8a95d2ba9fd74db69092891ef1681160000000b00000001fe0700002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f007a3643f15ab58061fe650dddb783e26ac918157f6414ceafa7027500721f43b043862b71d1dd71242ecaaeccbfb79a5d742557eb7ea49c6f6c9f8a4d0f3057c7b4ea49c745843988fc3e575d3afe2433ae789f87be82d6aaca8f8eca674e4c2c7d5e2ea06f6b1768ca7fd09ae4f4b40866331a4da0cde2d5092394f50c7f58500047700c9e0db25489c6f932e40c357e945914ee47ca84666410e7d5f4320731aa082085acca46517147f2925244a62cf6cdc12e0d1dba722ac51ebc034988c2d2ba2d6a33d8c373a69d92f154d8b404c16f73fa872a864488126f84d3168b3f0f212826e2b9c0c9e49fcdd3d099bf3ef8d93b11c71abd2601da09acf1f8fb139a332d9a5601f727c537d8edba21117d7be408a87d6b9279dad0d31da1dea289033d2444574e1a1ea77abf1d1183b53dd83993c39fe1c82e21a2c331e84673c2d10e7e0400e6bc71df504d3496437a3c4851298d686579cdcd6a0eb507d3bb3807b10b483abda468fe3b2e3f5c42ee4223b991b3a15f0f52c7e847691a692c36791391980d0c06617572612083635c08000000000466726f6e880127856dcce783370e0f2f8d245e6962e65da80ac30e97fb2fbe2fed1b290f77910005617572610101e6ee0eb043a6b5ef5dd2060a1280d0c51cb63c1cdd386c77f1b03fbfea95741e8e4f7f250c9e031120c116a69f5091690ac472b19bc082d2b8955711c28d508d180000000d00000001000800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f4c31488e9125c86f7c78b1174f11bc54e10e24d63b075bf9edb77d7a521b8a7c72aa802d70982fe9261673dfbde97a4e0085892f7e826784a62e974449168d280cc6780f3f88b07267c5bd28035c1ea8d3c7d57bd46c7aea9f3686d5dcbf0e5cd67ae5a926a93a34221dfa7965bb1bf5bdfe550eb4302b952a19747373223929defaaaac2a4312b976551770dd90b277a0fea279f4e133172053e186276ea502a4c071f88e10a9fea38468083ada172cf4c4b71fb676cc911d061b1182557483e9bf72a5cb0332c44eb9f1c9e7621ad6edb850cea38d14550d91a326c4501a9201ed82c8a883f444d0300947104879b646d005468d5cf853725170f9b358bd5ad59c197ab76485206c59f36a6036994dc543aa252b11a3c056814311d27f1404e9024f2df319184d0dfbb1db4f4259dd717a98a856adc2a8c8912efcc98fdf4abdfa2e002d00399fa53f7c1f185d8427c874a8f23b13afc475e5cee1f760d46974e71a54ccb2d353e4aa9bb6d3b4a641c9472a2faefe5adac9bd04b36beefb4168050b77a7f70806617572612083635c08000000000561757261010132730fe531d29230b22a9a0935ef83d80e40aff7efd8db34ce12604e23492d3ca79135c8bd2a4df2849c018629b7aa544222db8ad821d0f04720ee11e255a48c190000000e00000001030800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f48c1b5e26d4c749374e8f69ead4a5b953951cafcbc541903b9ff0496ce82796005d8114aa42d92f25e4aab90e7c0e5bcaf8e0b27364123b0a0442634aa351e0df5c0abdb50c18e3c20d85c1c0d85ffe9fc4b3d2fa6334a888f89fda67fd34125075303600635530959cfa99e3dab3e4e48746930e6c4b416c857d7974b5dfd703c0ad6dcc383bf87f8204dd9c4204e7586a5048bf43854682d5e6753e824c0358e2e245bd9ffa05c64f572f7ff97753222ace44ed6d7599503a4d5b0bb4674819ade986462caf64eb706b9dc9cac0f23d5a9ac8784f067d43663b936525e949c99823eaf56a8d5abc4a8efb4cdcf6f72330a94b423b765fb0267f6be2fd343d2e68724dfd57dc416a9e429bfea4c4265c30f8f3420f1dc6e7ee002b7d73edf0de9027005ab65c401c5eaafe10bac663bad41901d29b6303401251affd054103b251c22b24400e801ed65820b0cd6e8deebbac787b371c89060775a0decfc2a7112b01c51ef7c1e616da83f89c8305d421b535e08d1b33570743958df7c712c4adf9c96cddd330806617572612083635c08000000000561757261010146354b89f12715e609b5e0a272b1e790070fae3c06593211bcb1fc0c40d9806fa21be4cec44a86a847982f6598f92ea1895acbebe39ee857012ef88fc80eb9891a0000000f000000010a0800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fc00ba151fa23caf3a11590fe480f8713e26df35213521f26010a50d2da4d7c7356448cf00a57c405a2759052775a9ab72ef3975fbbfaa8c972b777958774db6c053837856a44c2ba69cb62bb627f28404b1de1f564f254f1a9b3d34d5b698f697ca16b1f5bb61e5b6cbbc9414a5bf0621ed419b43cc72ec90edd901a648d92f0b65504c69ee4a2793b9566effe4685816cc9e9b5afbeb574617a44181f56f84e9485fd594e5a541005820e3fbdaed7713dcedd4ba1c39ee31e386cdab27ea68b95d5c4474278dcea31832321ab93eb6b9d8f2cb5fb75a8e03cbcfba0dc6d495a900f07dfc7cfa4567f2f8ee94fb18793c79d46f233331913516c82a8ea379095c06172eb64373f8e6e622f61cc963278de4558aebbfcef14e1146d85964197e89501ef226b220d0a9f3ce159cd9fe3ce808391b3b90055c3960918277984e45f4c249e9a01006070e574c692886332a9cdfdd2bdbd972889c16e16354f1e470e52bf36db81d2bc7851c7622decd220c83b3668e028145434255ec44cd50cef9c25e5b868e92c001f0000001400000001260800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9fead4b15c40389b68b80da252f1c64ccdff8b7deda378cad3bc2040df93bc0b5e7252eaa94a54db1b10710b81df91791d58a801e17ac7e6c5be9558f6ddbd3c555b19f587cb3e1436049a1e54564e1539f558a5641e3822542ef04a8029f6c03c7cb08847b3ec399f336d9532afb4a312db3b0226c3e740d12b274eb378aec9f87c169bc67da0715fb92c783b5b8fa646489179f8bf7e4a4271fa7d6edee05e74605c69aee2333339e8a08abafeb357a0281a206a64d184dceec971d6d55f248b741ed705f509eb20a558dd1772cd34ff1359d80a4c176982169f7a7f44e8250f9027b2b9d48451380121cdfc5a041491c3fb5eefc07009d1aee921df7a0599bc01034b5e0124a5042e25c42f67aa1303afd59571d83ea865eb1a3aee512467d6e9025befd6bf43197109f64511896b4ab95b2bd221e6810459242b4053c8284686e822d3e200a8f49a3d18ac36932f3608c00c7e004147b3868fdb6d9fa0d431f2ac5c180bbff6f0a28f55b64489087c1e2e91989c0b886fa55e5ca22989fec8626754b531320806617572612083635c0800000000056175726101016c40c4eb4eba5a7917edb38b4813bd5160442f44c16f743b1ab83fbca0aab837686b92638dc57ff1280973721c79102295a43260c4dcd93a6b6dea614077ff812000000015000000012b0800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f7ec2f69737ac6304d9b45fc7bdad23a6e8b6b8c35c77226a9270a105205d5816f8f871b0e0903970117dac29c057cd2d5090f7381019703f20ef82e38de9484d4487ceaa35c2b85da43887dd27238c4a77a28d76326c0bb44244f2fd51fa8ec2298c0bd4ee975e581149da8c6b891bb511a5ae7f5b1794b58a133adba5beb7a75ec607a812b0ef58815e341279396158ae4d5234e03f750c7039bbf2b8b31c2ec4e675c6f7dd170b2fb31de633e340a39fba599e45cc13275825f7c505bc8088f059c6fb23e2e6148346053ee02154cfb8f1eae02f04419e59d9e23ec672d96610883519517ecee74ee2ca4c3a7d47ce8dea260ccfa1542ec8255ba9eca949a06f1c283dba5bef677dbbbe5b25fe5171ac3f48bebf64747f70bb7f689c57ecade90254ea986a37d66353e404a2f75415fe3470a1b41fd447a220d33953b9284c16d83ea543000d8737c8c55fcd4c37ab87b7558e052dfaeacdd7cb168de9064e583276e1a7046e68b68199397af29e1c83f02e4083890c222e4da2b762463e62a870cb327adb0806617572612083635c080000000005617572610101e6ab5f369c205dfda284a6744f7a74a83d94369526a4126c20161c5d273c81053f65482be2ac6e762d95a8a5f0e0474385b10e52bcff0b4e2116d9128108c6892200000017000000012c0800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f8eabe2f87cf524db949c7ce4a300c9943d5ce7cb61eaf056407256614567cf6808599ad0612f70b2011e286f46bb6499e08a50a483fd8f6dd0af19e1596945f877d87e4f34c7c2a362ad3ab7860336ef91c50037eb839a57418f6989b347c701e0b8c7d6d75a732aaa50c303c7c411e87bbb8479f3afc1e31fd7713002d275bb7a0ca5e2cb228b6e38d1806487741429d1dc70ee946a51c3ba3285432ccd0b3dd07663c15018b3931889fc09646632be942a7ec9e4764e9aaeb031663f6fc380401dfb396fe48d83b1d0d25eab92e1dcbc61365db4fc2178436ef2b7b9ab29b50cf12be580eb72539ac227b4fe392c8868b2feb21de9067224d77fae5f6769e46c526f8430531d3b87d1abd12f8df54b0fe7d2e5c6c68c7d45c88eb6a8266ad24903339562159a30ba82f5e54db05cfcf1b517c911492234e3698b3a25410f07a55beee0af0018bc9c144df2b89cf89269120ac2db451dca153e63509b970e86778f340bab743fa69abcba4b75ffe19442f27b7522589c8dca513b1bbfa2ceb2dea2358fefb908066e6d62738088197ec695687b3c54268b18d071a0ebb4323e84179ba664894124beefc56f52056e6d627301016609447513272ebcb0271154883e924a6165a990dbe10ac8f63c33dd71124432c2bb3550926a2c1edf8d3f392c22046af6f1e86347864de3465660969b55a3832300000018000000012d0800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f66064f3465604d64246bf810b9fe8e49a1cf470be7263b11a699edc83a9f352d4c87742fc24e8aeb858e82d633c5d21aaf645efce69123d14a5bf10cf2e87b829b622d5143362571270ca3f4f451bda43be9f870145afb34e3c3a25c97960a255c35ea9c5f11d8b5b87d7eca9b38db17bba211193fcb4df872dc755078e4bdf096a0b786e3efadc128eba8062eb163b018b1763689f1765872b64a88a4c3103c4796565257c6f24d0ee68e26627947ae8d033d2dbabd19d2f6796fab2a3488814af90dc6120f52e8b176ec45828b93ab9ed570537cb311f3006eb17a4b04add9aaf8b111859be1b3d33934ee9bc8a6fdc2bd674f5236099d42bf469cae8e9dde924db656504ab1637673c75be1d51e171906c6e8c12c51a06f6f89e712f7fa19e902f00e5727bb4eb555d4a4f5140b82f4d72ded6eff3506cef9f2eb8b57e36d13f732d41c00310b800ce61efabc623f074c2f728d6c7c2cfbd7c2be565b5bc4943a7a3bb2794ce3e5a2cae2afbc804926c92b104ba1e8eebb2fae209f6b244ced22de968a4e0806617572612083635c080000000005617572610101e81f196b8d3114ddeaf9eaf15d34be8e9ff43db5e25d1aad1774d6a819da913919f6f0b6a16dd668e2880547df0716338b7d0a8bf27203dd8b8d03c90adea88f2400000019000000012e0800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f723e5619fa775cfa0b85ed1f73557d87bfaf34c269c00141e20a8e4fede6c76c8148b38f39593ea9b1d336460e35f225c624104259c9e8338b6f7f74322dd1b8e18091df02039f421e89714802f6be2f2652d714893299798f6f059c5a37dda0fba4bcfd13dbee90229d0771f973af9995468146472ac6aeb912116b4d8bb5ea147bfe6f234a1d1f7bc9d4f10656648269075b7268ae6a243a1a745c7c85f077f2a0fc35b52911a0984a7d2d8641e25952ec16186995a251213ce8c932d3008a05530d6923ca084b8837f15cc4d4dc0bb3c63a1921fa15ec9556ae3ff9d359b1647a0094fc9493f16cf2deb282b73f53985acb3177cfcec28a6f455ef0b43a6eddd7bbebc1214a12baf9cc2bd7a46fbc1e5204d6c192fab5db3fb53c0304c3a6e902acc812718e754faa8f3b4979311ce24e7a9510bbc1160e124e881180d6f678c2debb20001d57ebbdc54d39ebc6c72bda6004b766154578ed841d357151fe9f7b7d80c42d43197f6f6983c8e53d10ce0384d8830a6f1b2121958dab34edff2a7849253d620806617572612083635c0800000000056175726101017e927e09f71f3a2af25d161aff972a98f588d4f42f84a28d295488cc2ebbe024223147f58a7afc121215ee03e9300d5e11058ae1e7f0e80f675967c1367e8e8b250000001a00000001350800002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f1c3964b2c71a0c5fb5c51687909a83cbaa4e94b907d2518bad392fc9a5a7ab7fe35190789786820fddab97684b50b4e8ce0f3595106ab256593fb1cae9ec8c9656a2f0d8e332d511e9b80f31f01eb57248a1a68fc87f8745b71b7d3c8f81982981df3189ee192b2a32bbd331fe28d70851d1d41462957f232206c470fe7c8e0be6133da394c2bb60467b7764e7648794626315f4f2ca7028fe235b0e9b47c33a974fd23cace0213c08cb20565b61ca4bcb3c0e5cdd4a20cf290b0e58d814688d0f34ff3fc478732698c85d8ddf6d0b5bd428df77b267a77999f6ab24b9728458cffa7727cd0ea5c03a212dabe51c377dbd53f0e7f0a1ddb79033fb3d7b73c1db5558455d3b6442a3da1f7827a645952460ccc5c3dd691bcae64a2d2668fe0097e9020425014b3839b51b99be80baa9dbf1a3ef01662a7ea51d6cb2da23814ae835357270a9006870fd9e022c9ac5b4de06bc198a481ae60f3909b8affe519a6c04ffed18438f3c163fa7c3cc25faf40c18eae7c40dbd5a27625fdc6646dce86a5773b6ab6d6b0806617572612083635c0800000000056175726101018c3417489b99706c04255a8b8648f2d6b6c701576dfd25ecd1de9d3b1e0cce4970616567b06ab1190b95a95572d61a4ae637ddbf7bf13bf648af9be585166785270000001c00000001050d00002245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f54de495b57c7c3505c62633f1ac3a2f22be74ed497a5884379e0821607d9173beb5dd269e10d71dc754ce3fac591364afce61b007925730544bf530068087c7d38e3e2cd8bdf7fef72ad23b076e0620ab5d97d3c0a98207dd8b6af8c3becff696cd2c59aefd1a3bd654df00febe502e9f6ba788b8584c7278c47a2983fd8e87bcea2f2c3fdf7cfe328e695f1a111ef699739c5054adcfd40969c673b39080e4dfa869839757fab6bf946cf5695e1a0d9335ad95e445cd5fe9988a1c82fb53c83fedc8cf6b2555199ecc95e7092742c5959f96bce04becaebfd9266f7642c23d76e25771c2505f549b5af7734c84213bd1fd39278fa8f07298ba69da55a5c70899ece96d300d33d733840cfd4035249b50618e4e81f1cd425bd304b0cffc13b8ee902e66008704ade0c28fa58a8abc788e7b1c3ef78710408947dd4fb2f400effb2344a581900c5c07c6729c92d2b6d2dd7b5e6a05c26adc79e138d57c1b74a2d42ff7c021eb3f127490b8eef8e2c1cff1b685fc398f2dfeef4554328f288259cd9de2280e3200806617572612083635c0800000000056175726101017e7b7d496aef40f4e44a8819b2652fe2d64d286288f21f25a08b31f937d82c7ad7a48673092c23d7a2aa5cb2ca8dead00a2ace14361537ef7f338dc51f34bb842a0000001f00000000d207000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc43216af2664480544590bc51a68be4ab955ba1e9d66544fb5ddd87455187539e427c0f0596d61ef285c22dc8d3e2f29bcfa542b00f0474045735c327e32b81a1e7fb4856020ebb9fb922899fe872508d3105d22f9749a0ed5cd6a2ee894140e1bc46ea832c2ab3e85936053ce6b203eb5396a696dc8d4977092004738ed02c4f5504a932e236c018a5ffcb5f0bed02fc84ee8a1d18fc7e02bd2550ea4606e043e4e6018a6818a0dc0acfb0e5ea67880a8e20dc6ef1f7399658c2fd4a7603eac9085a5574ce91797bfb2df82d4939ddd9f8aff952e1ce48b1d261c97088dcd5c86e532283647ac87830d1b136bce5fe3bad5acba67add2a89f005497d93fac9e5d2088e825d7969dad2cc6c04f8c4a6e65b5002877aaf6b7c32d0943efe5e127824089032a6b0cfbd22318888f75e0f1d6fd92d22de5cd8fb974579b68a848ee1a394979c675d0000e427f6dd8407605cc5ecb040ad322f127d040c84616ed6f41af284720d7c4e289ca21a7e551c98bed636f70900a463ebd20ae087491f8cab4b90668e7b3bc3c0c06617572612007c7b810000000000466726f6e8801b1adac139bde159ecba683741d3ecc31221e98eab5a429f3bb3f38c553018ecf00056175726101013c7952f659e18808ab2c763a25fb0c7532634085cec8d495f3030c6c3cb5b05722c5143a5dd1fad4f7d84eee22196d845e2e1309a48eba4859aba690c2e6a083030000002300000000d407000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc43228919c055a0f2137f13690e949df8f7d97d322fd990946b698f985989116ba7725269c36d48994939f136e4118d4344b4ec13a8c468ea71c52848177ee200ebab55f213795cf2769a0bc7d5ece59be01b8afc921529684f158ec31c4bc58eca35e4f74b2e3c8decba237f50ffbfe45753135891432ca98e1ce2cd1aafc02e67a80d7bd3ff76b4162eb32a149d0673960080c609ace1755ec2f60d6edcdd79309b9bcefd1b379e2291be3ec29e5429c11bb5269d085ef32e86996bb6086f9528541ab122ce2a5db613f84f3b2f96a7dc6bc53ceaa18f6985f08391a449a440c6332e9b1f9c0b02e36ec11b5167c7e7a00d24222061b4769e03517ab2e2ee6170c55a25e183e83a80c3d4968040681b1d15366669926dbf45ab4d7a84e553e426389065fd3dc2ad1fe4f17b0d3bb1bc2bbb7090363c52706ca6509f14a6a128d85ef337e52d5008cc9bdf131d19272e7143953afec22aa607608e1399048c920388988e7834c68139f2af193d65a5517524c47908fc3a6a6275d581dc9661a5594a853b0aacbe810066e6d6273801ec11d2d43768a4d5d45b5611db0b3298b77621d26742547fe41ce91c0d661260672616e648101ceaf24ecf730d7e039d2de8aa45733921ffba8cb0e7c6f75bd06d2591b901c77b1e807742745efcbd5a57bf08935eb21cc1fbeb85d80ed7c615ca7d35e249a07bf6d39c123ba9ba0bd7e75318040bc489985b106d6c98c5aa5a0811ede08f9060466726f6e890101ff0a22a78c0683078446688ddeebbc95a99b1884cd22a8201bf911cabb18059908309e9c2c6a18f7387da2ee66bf312501f6b7d5c7e5f0b9a17df6a084a87d334c3022239463cef94bfd88bf8317dd9b6cab00f087cb8df769cb6bb2c0d0f1d312056e6d62730101d6fa19a093846a11dbd3b9f26dca8a0dcdba8493d6b346b86486f3c54dd7ec72b46a8f8d7d17f1192788c741d069e068c08eb0c5cc6f6aae09d7548b6faabf82040000002400000000e507000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc432cc8a84b06644b0fd1e790c2de73d66c562f0cd3f6239d81b25cd278e5111266388a9171ad58bcbd15396dde9b68b6aeb9a7a6a5b5c7a47d77603f1889cfde56f1ac40ba6eb288239b7dd9c8af7f2b9fb0795cd5b28820c87180e3d735e7bcc0b51df051cf8f33419ca0f07f31f4e8dd2ad6f62bafdc1f0c47f9b727603cda6ae34a3726a20df66d660d77f1c01195a112535db1e046a4beaef91de906901911dc374d89fe1f805734c6f6144db3152748f9b0541cd6583e6c3d2894e04970082b98df5d914b951c9183bf25117b1a9072768e4df487f8e6ed918e5d6754319322d991a6158947113ff320990f38f5e37f0ece574395edfae743439824efbabd2d04f9ebc499a3c2040abe8fddc43943b8f123b71bb490e5f93c5f6093dad9f58e90231d5d69ba54e8052c9c50daa412c580711b273461ec198a7d5399fbfe614faa5ae8d9f004508b43b4a234f1cb724cce8e3e6b00c81f94bc0f321502794f4616392849994f67c842d1b63e4a18df453da38b88267d8389934b1f86bbb988e29162c71dd480806617572612083635c0800000000056175726101011c43c9dc41050f47d24f799941efe385c0b41380cb1b19fb9910ce4f2c55222a6e836b3aeb5c1c931f9c3e95dc9cd27e1d1585d69e18813f5cb889ecea1da3830c0000000100000000ee07000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc432021071f1207ce1d22e80903c737f4c89f2113bd1c7753de41092a70552dd6c17cffb003d8b5359ae9c274650df76ffdd14c52814ce9b5c17900bc52834e30a83ce59e7d5f163371f5d99dd3784d693dfac0e4eec4fa8accf45a881150ceebab3f42382ac9fa47baae525b7af81f2ba3125302b795ec05b0a3d9b4bca133d3a63e2c5da2612da3f1b3c8f42cd1522f6a7a48d2693d58088e5288694b90307074ef116265b6660b03112547414f40025cc4154c6eb7a6de2ad52e51a55af54b38aa1c54fa1bd03bcbd527584aa648fdd075dc0f8e79c247f5ffb1e6ac9c2be225ea60db84bb8d7ab17120d682d16696c329e958433a449faeb88b1c50464e2e4e2eaa75a54e13a8d890399906e24d00454ec4e709e4f7b939691771a23dab762bbe9023a3389663194f4d084a27298adb033bfd6a263f4d3c4c20a3e6f0e00863cb85196408b005a0aee92f8c6b9858c667067f971772855a47e11daf73a2aab1ae29e5db1dc095e8b6f463c0264370e99ac799f7b717e685218b437e4f6181e4abd56e1a0c0d60806617572612083635c08000000000561757261010122106da90907f90dfa17d639b60ab9d3aabdd7ea95daaeeccf3b4dee072fce038e61f86d7189ee821aec9a11d6ecc5c484d12229183553ca27a9d20f668af9860f0000000400000000ef07000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc4327ef0f295b0fdc0c650ffebced6be72c4dd0e1176da30930364fdb01a6713d036f0831c577230924e9326d19f346e64c95e84193c52352eda105704b09d804268b5a6b9e2d162fdb73841ea9372584d9699f321a111fe6f470b89b99b07d8431bafca0227eea00a532e1484c02ec006fa6495eba21abc438584d76f48802d0a910a508a2ee6c3f905a242e92b41e1bb438c7a083504fda57a1849ddd35896c47d78a5956fe6c7531a3c8ddefd9d17c1284ba60dac9040aca954dec18f365c9b8fbe455b67d90249d0096cd35843e3d71badc12c49381bce7312852dec8a1eb64498bbf023fc9e033deee0b54e1b2e1ca8e747d2dd9cb658cc07344df196fbd222e9520eab0f4b1fa37fc4afcb93a1c0ace0cde92aa0689ebcb0c5d1245bcd2c10e9028a0cc2de076fcc8655cd5cc9ae23351591616aded6d8ecab7f2a44e5d624a8fe422bb00056c4bf23b31c58b9773346f043fb8bf5373ded1962c0fec66853e9d754bb642036900eb42f10c8e6021345c5ca5bd1ab4e22e5b8022bf38b06f66425eebf4c5c0806617572612083635c0800000000056175726101011a3c100d29a795c4fbb068012b0291991aa5db384bc0d8c775ffadb60c9a955b4e259391b7f5c22a32ed066345b09aca2fb0b06fb2254ea3723b462ee7ce478a100000000500000000fb07000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc432740beb55375d7c8b84577fbe1c0743965ea389fe7585d6f68975111716c1ce24eb2dadcc3c91fcb5f23b9008ea9c6f934dc261a84b3d1b5b43ae437c6aa992452aea0a684700ff36d0a8b8ebd07fecc353bda87aad24b37c8944de3ce45efcb42fd6220dd68bba2bbc6a3756154fa5cf878797dacda5fe063c16fd8436a46bc160c9820134984676bc4e14204671db0b5494513b5a7fe4df5d0ea250e554a2080e31802525806814554d3a5677af77a3673b500d3539f04afdcfd720573b608ab0b09939c11cbff67c66f806f5221a61fdcfab5c4fa11c0d7ca12988135efa54ed1d69cd74ceee75480f46a7b15b926904d12cdea281aa28e908ec00a9d07cf8a82bcf585ae95386cc747fdf003d5802d9fe7484687bbbad0448af45553ff3b58903174881b8182392f9757efb143ba61d046a0208302a7a7b3df8041059ca538fef169b8d0012fd33199e57825afe600bbfff192f2b9e0f6b72f5da7e8f1f3147e859987cc129b23e32b2278bada2bc2795d7cd6034e3afa8e9a333d01b36b4e2250e583fe20c06617572612083635c08000000000466726f6e88014920420a08c71a4b56953bf3efbae9a45e737f4cea978bdbc470de907d64e1550005617572610101b491cb7862f8d14fb0e2f6cb8d1287b83a55ee7feb468577ca78d9c120fabe5f6b909448827d93de5eb8bd7afe6c9aea63fef46254208969cdab95fe6d02f18b170000000c000000000808000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc432c663d22aea82481985ee77a58f1e54bb8c64fe22c48df15aa43f69aad1c7276b53c400f7fb9008e65a12443807e49224efa495a9cfa03e6ce5298a88fc744478f3820cb625cff327e14fe2006713cce73555c4336a7d861715aa2429a7d349102f0b4f9017440cf5a06a286856c5795b974f5006aa07357854c3eb4c3892b7f5125cb08a3e444ff92b8868bb8d6d64947c84862353870b0e5121b002929c136db31e000378c0f3241a5408a49bc458bd75af5c0d16b35661e212a981ecf5be8d61707c8adc571e83812efa6aa3673e4423c1b56780c437c500583abf6839a68d096600b21370ef6b990d6922acbfc71de68d751ee2d62fbfc49125bd9709ffe33bbce32c9e23b266d24cc175dcc0d99e7c1182b6ce8824d724e9c34ad852a611e9021145925a3f018c9aaba6b3981aaa8e1e24f0c7d9b38f65643a16d3f55d77ff6ca60a4800e3f4f2125d650231ba3ecaa0a51aeda71beada9dad70479d4571b5f2ebfa480c16116166d9e35a66dcadb999da853bfcd1872563f5d0a785507bf67796e568ec0806617572612083635c080000000005617572610101de7241c718f67134d32b6b5656f8e8f3f31e8d29af28043a37c2b49429a082633c8c7abbe502084c28c4a548ac75f97732521c7e66ea9917e704e5469572558c1e00000013000000003808000026662318f64c51f6ed779aa7e32b1b7489bbdb6118812d1b0a2902b7405bc43278b266c6c3ba60a6e648093bcd62419d4ea68eec772f22b7dfc78c6b255dac57bd149cd2b1b166d73be2ed9409a0bebe842c1901d65b1fd3c5f8d98d82ea2a5c3ae10acc8770276d737e6334c59e468be878e6264c050f48ac3e4a6524b76b37c399c4a820e1a50b2bcbc805192c2609e8372e8ddd8432127f6200427dcb2d72386429023e01fcf193d96cc55f1d4d85db5b2f0a9e3a5fb6855c0e07b891d6581ea371a39cb30eaa0ea2d0b9bbe5734e8793d4db7c366277619bdff9611dfb8d4bba89ec371746a5e689ece6f2a117d2b4e7894776b7a96209e7177446c076874bde474bbe3f7bb78669eb48648bf238110f690c144c097988314989bf9b3d4ef410cd043b32cf71a25a00655ab6a0558e383e546c3728412be664273b51908f4903de6835c05f9da3c6ca7fc65846438da7947f93a152f77bb75a85e3d05385feb47e071300b797adcbfd7d3a94ba4b2e0ab454bfded856f759d2b674839d4adcacb72d2c1361edc442862859f46d384db20db351390b2d261be35211db5232e585f985291a08066e6d6273804852f3c0a603d7da194fbff174c24700f3f89d86b462760033bdcb1663c76c60056e6d62730101e2f5f706646cbfc613a98564ff9b4a958acead02fa488e1f3bbcee114d13594d75e9b79d5f5be257e9da81d3ab97cefe36a620a2384f6f1f8719ec2b23532684280000001d000000"
		resultBytes, err := common.HexToBytes(result)
		require.NoError(t, err)

		err = scale.Unmarshal(resultBytes, &candidateEvents)
		require.NoError(t, err)

		tCanEvents, err := parachaintypes.NewCandidateEvents()
		require.NoError(t, err)

		tCanEvents.Add(parachaintypes.CandidateBacked{
			CandidateReceipt: parachaintypes.CandidateReceipt{
				Descriptor: parachaintypes.CandidateDescriptor{
					ParaID:                      0xd05,
					RelayParent:                 common.MustHexToHash("0x2245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f"),
					Collator:                    parachaintypes.CollatorID{0x54, 0xde, 0x49, 0x5b, 0x57, 0xc7, 0xc3, 0x50, 0x5c, 0x62, 0x63, 0x3f, 0x1a, 0xc3, 0xa2, 0xf2, 0x2b, 0xe7, 0x4e, 0xd4, 0x97, 0xa5, 0x88, 0x43, 0x79, 0xe0, 0x82, 0x16, 0x7, 0xd9, 0x17, 0x3b},
					PersistedValidationDataHash: common.MustHexToHash("0xeb5dd269e10d71dc754ce3fac591364afce61b007925730544bf530068087c7d"),
					PovHash:                     common.MustHexToHash("0x38e3e2cd8bdf7fef72ad23b076e0620ab5d97d3c0a98207dd8b6af8c3becff69"),
					ErasureRoot:                 common.MustHexToHash("0x6cd2c59aefd1a3bd654df00febe502e9f6ba788b8584c7278c47a2983fd8e87b"),
					Signature: parachaintypes.CollatorSignature{206, 162, 242, 195, 253, 247, 207, 227, 40,
						230, 149, 241, 161, 17, 239, 105, 151, 57, 197, 5, 74, 220, 253, 64, 150, 156, 103, 59, 57,
						8, 14, 77, 250, 134, 152, 57, 117, 127, 171, 107, 249, 70, 207, 86, 149, 225, 160, 217, 51,
						90, 217, 94, 68, 92, 213, 254, 153, 136, 161, 200, 47, 181, 60, 131},
					ParaHead: common.MustHexToHash("0xfedc8cf6b2555199ecc95e7092742c5959f96bce04becaebfd9266f7642c23d7"),
					ValidationCodeHash: parachaintypes.ValidationCodeHash{110, 37, 119, 28, 37, 5, 245, 73, 181, 175,
						119, 52, 200, 66, 19, 189, 31, 211, 146, 120, 250, 143, 7, 41, 139, 166, 157, 165, 90, 92, 112, 137},
				},
				CommitmentsHash: common.MustHexToHash("0x9ece96d300d33d733840cfd4035249b50618e4e81f1cd425bd304b0cffc13b8e"),
			},
			HeadData: parachaintypes.HeadData{Data: []byte{230, 96, 8, 112, 74, 222, 12, 40, 250, 88, 168, 171,
				199, 136, 231, 177, 195, 239, 120, 113, 4, 8, 148, 125, 212, 251, 47, 64, 14, 255, 178, 52, 74, 88,
				25, 0, 197, 192, 124, 103, 41, 201, 45, 43, 109, 45, 215, 181, 230, 160, 92, 38, 173, 199, 158, 19,
				141, 87, 193, 183, 74, 45, 66, 255, 124, 2, 30, 179, 241, 39, 73, 11, 142, 239, 142, 44, 28, 255, 27,
				104, 95, 195, 152, 242, 223, 238, 244, 85, 67, 40, 242, 136, 37, 156, 217, 222, 34, 128, 227, 32, 8,
				6, 97, 117, 114, 97, 32, 131, 99, 92, 8, 0, 0, 0, 0, 5, 97, 117, 114, 97, 1, 1, 126, 123, 125, 73,
				106, 239, 64, 244, 228, 74, 136, 25, 178, 101, 47, 226, 214, 77, 40, 98, 136, 242, 31, 37, 160, 139,
				49, 249, 55, 216, 44, 122, 215, 164, 134, 115, 9, 44, 35, 215, 162, 170, 92, 178, 202, 141, 234,
				208, 10, 42, 206, 20, 54, 21, 55, 239, 127, 51, 141, 197, 31, 52, 187, 132},
			},
			CoreIndex:  parachaintypes.CoreIndex{Index: 42},
			GroupIndex: 31,
		})

		tCanEvents.Add(parachaintypes.CandidateIncluded{
			CandidateReceipt: parachaintypes.CandidateReceipt{
				Descriptor: parachaintypes.CandidateDescriptor{
					ParaID:                      0xd05,
					RelayParent:                 common.MustHexToHash("0x2245952bd39fbc912d3bcf6ab6d9c191b5c9df70a4e1da6f670f4761bf96cc9f"),
					Collator:                    parachaintypes.CollatorID{0x54, 0xde, 0x49, 0x5b, 0x57, 0xc7, 0xc3, 0x50, 0x5c, 0x62, 0x63, 0x3f, 0x1a, 0xc3, 0xa2, 0xf2, 0x2b, 0xe7, 0x4e, 0xd4, 0x97, 0xa5, 0x88, 0x43, 0x79, 0xe0, 0x82, 0x16, 0x7, 0xd9, 0x17, 0x3b},
					PersistedValidationDataHash: common.MustHexToHash("0xeb5dd269e10d71dc754ce3fac591364afce61b007925730544bf530068087c7d"),
					PovHash:                     common.MustHexToHash("0x38e3e2cd8bdf7fef72ad23b076e0620ab5d97d3c0a98207dd8b6af8c3becff69"),
					ErasureRoot:                 common.MustHexToHash("0x6cd2c59aefd1a3bd654df00febe502e9f6ba788b8584c7278c47a2983fd8e87b"),
					Signature: parachaintypes.CollatorSignature{206, 162, 242, 195, 253, 247, 207, 227, 40,
						230, 149, 241, 161, 17, 239, 105, 151, 57, 197, 5, 74, 220, 253, 64, 150, 156, 103, 59, 57,
						8, 14, 77, 250, 134, 152, 57, 117, 127, 171, 107, 249, 70, 207, 86, 149, 225, 160, 217, 51,
						90, 217, 94, 68, 92, 213, 254, 153, 136, 161, 200, 47, 181, 60, 131},
					ParaHead: common.MustHexToHash("0xfedc8cf6b2555199ecc95e7092742c5959f96bce04becaebfd9266f7642c23d7"),
					ValidationCodeHash: parachaintypes.ValidationCodeHash{110, 37, 119, 28, 37, 5, 245, 73, 181, 175,
						119, 52, 200, 66, 19, 189, 31, 211, 146, 120, 250, 143, 7, 41, 139, 166, 157, 165, 90, 92, 112, 137},
				},
				CommitmentsHash: common.MustHexToHash("0x9ece96d300d33d733840cfd4035249b50618e4e81f1cd425bd304b0cffc13b8e"),
			},
			HeadData: parachaintypes.HeadData{Data: []byte{230, 96, 8, 112, 74, 222, 12, 40, 250, 88, 168, 171,
				199, 136, 231, 177, 195, 239, 120, 113, 4, 8, 148, 125, 212, 251, 47, 64, 14, 255, 178, 52, 74, 88,
				25, 0, 197, 192, 124, 103, 41, 201, 45, 43, 109, 45, 215, 181, 230, 160, 92, 38, 173, 199, 158, 19,
				141, 87, 193, 183, 74, 45, 66, 255, 124, 2, 30, 179, 241, 39, 73, 11, 142, 239, 142, 44, 28, 255, 27,
				104, 95, 195, 152, 242, 223, 238, 244, 85, 67, 40, 242, 136, 37, 156, 217, 222, 34, 128, 227, 32, 8,
				6, 97, 117, 114, 97, 32, 131, 99, 92, 8, 0, 0, 0, 0, 5, 97, 117, 114, 97, 1, 1, 126, 123, 125, 73,
				106, 239, 64, 244, 228, 74, 136, 25, 178, 101, 47, 226, 214, 77, 40, 98, 136, 242, 31, 37, 160, 139,
				49, 249, 55, 216, 44, 122, 215, 164, 134, 115, 9, 44, 35, 215, 162, 170, 92, 178, 202, 141, 234,
				208, 10, 42, 206, 20, 54, 21, 55, 239, 127, 51, 141, 197, 31, 52, 187, 132},
			},
			CoreIndex:  parachaintypes.CoreIndex{Index: 42},
			GroupIndex: 31,
		})

		fmt.Printf("tCanEvents: %v\n", tCanEvents)
		for i, v := range tCanEvents.Types {
			logger.Infof("t candidateEvent %v %v", i, v)
		}

		inst.EXPECT().ParachainHostCandidateEvents().Return(&tCanEvents, err)

		msg2.Resp <- inst
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	time.Sleep(1000 * time.Millisecond)

	msgSenderQueryChan := make(chan availability_store.AvailableData)
	queryData := availability_store.QueryAvailableData{
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	fmt.Printf("msgQueryChan: %v\n", msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}
