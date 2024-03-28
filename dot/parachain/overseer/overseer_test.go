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
	db                database.Database
}

func newTestHarness(t *testing.T, seedDB bool) *testHarness {
	overseer := NewTestOverseer()
	harness := &testHarness{
		overseer:       overseer,
		broadcastIndex: 0,
		t:              t,
	}

	if seedDB {
		harness.db = availability_store.SetupTestDB(t)
	} else {
		harness.db = state.NewInMemoryDB(t)
	}

	testPruningConfig := availability_store.PruningConfig{
		KeepUnavailableFor: time.Second * 2,
		KeepFinalizedFor:   time.Second * 5,
		PruningInterval:    time.Second * 1,
	}

	availabilityStore, err := availability_store.CreateAndRegisterPruning(harness.overseer.GetSubsystemToOverseerChannel(),
		harness.db, testPruningConfig)

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

func (h *testHarness) printDB(caption string) {
	// print db
	fmt.Printf("db contents %v:\n", caption)
	iterator, err := h.db.NewIterator()
	require.NoError(h.t, err)
	defer iterator.Release()

	for iterator.First(); iterator.Valid(); iterator.Next() {
		fmt.Printf("key: %x, value: %x\n", iterator.Key(), iterator.Value())
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
		CandidateHash: parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash},
		Chunk:         availability_store.TestChunk1,
		Sender:        msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, chunkMsg)
	msgSenderQueryChan := make(chan availability_store.ErasureChunk)

	harness.broadcastMessages = append(harness.broadcastMessages, availability_store.QueryChunk{
		CandidateHash:  parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash},
		ValidatorIndex: 0,
		Sender:         msgSenderQueryChan,
	})

	err := harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()
	time.Sleep(100 * time.Millisecond)

	msgSenderChanResult := <-chunkMsg.Sender
	require.Nil(t, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availability_store.TestChunk1, msgQueryChan)
	time.Sleep(100 * time.Millisecond)

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
	require.Equal(t, nil, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availability_store.ErasureChunk{}, msgQueryChan)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}

func TestQueryChunkChecksMetadata(t *testing.T) {
	harness := newTestHarness(t, true)

	msgSenderChan := make(chan bool)

	queryChunkMsg := availability_store.QueryChunkAvailability{
		CandidateHash:  parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash},
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
	require.Equal(t, true, msgSenderChanResult)

	harness.triggerBroadcast()

	msgQueryChan := <-queryChunk2Msg.Sender
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
		NumValidators:       nValidators,
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
			ValidatorIndex: i,
			Sender:         msgSenderQueryChan,
		})
		harness.triggerBroadcast()
		msgQueryChan := <-msgSenderQueryChan
		require.Equal(t, chunksExpected[i], msgQueryChan.Chunk)
	}
}

func TestQueryAllChunksWorks(t *testing.T) {
	harness := newTestHarness(t, true)
	candidateHash := parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash}
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
		CandidateHash: parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash},
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
		NumValidators:       nValidators,
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
		NumValidators:       nValidators,
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
	harness := newTestHarness(t, false)
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
		NumValidators:       nValidators,
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

func importLeaf(t *testing.T, harness *testHarness, parentHash common.Hash,
	blockNumber parachaintypes.BlockNumber) common.Hash {
	header := types.Header{
		ParentHash: parentHash,
		Number:     uint(blockNumber),
	}
	aLeaf := header.Hash()

	harness.overseer.broadcast(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash:   aLeaf,
			Number: uint32(1),
		},
	})

	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(util.ChainAPIMessage[util.BlockHeader])
		msg2.ResponseChannel <- types.Header{
			ParentHash: parentHash,
			Number:     3,
		}
		require.Equal(t, aLeaf, msg2.Message.Hash)
	})

	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(util.ChainAPIMessage[util.Ancestors])
		msg2.ResponseChannel <- util.AncestorsResponse{
			Ancestors: []common.Hash{{0x01}, {0x02}},
		}
		require.Equal(t, aLeaf, msg2.Message.Hash)
	})

	harness.processes = append(harness.processes, func(msg any) {
		msg2, _ := msg.(parachain.RuntimeAPIMessage)
		require.Equal(t, aLeaf, msg2.Hash)
		ctrl := gomock.NewController(harness.t)
		inst := NewMockRuntimeInstance(ctrl)

		tCanEvents, err := parachaintypes.NewCandidateEvents()
		require.NoError(harness.t, err)

		tCanBacked := parachaintypes.CandidateBacked{
			CandidateReceipt: availability_store.TestCandidateReceipt,
		}
		tCanEvents.Add(tCanBacked)

		tCanIncluded := parachaintypes.CandidateIncluded{
			CandidateReceipt: availability_store.TestCandidateReceipt,
		}
		tCanEvents.Add(tCanIncluded)

		inst.EXPECT().ParachainHostCandidateEvents().Return(&tCanEvents, err)

		msg2.Resp <- inst
	})

	return aLeaf
}

func hasAllChunks(harness *testHarness, candidateHash parachaintypes.CandidateHash, nValidators uint,
	expectPresent bool) bool {
	for i := uint(0); i < nValidators; i++ {
		msgQueryChan := make(chan availability_store.ErasureChunk)
		queryChunk := availability_store.QueryChunk{
			CandidateHash:  candidateHash,
			ValidatorIndex: i,
			Sender:         msgQueryChan,
		}
		harness.broadcastMessages = append(harness.broadcastMessages, queryChunk)
		harness.triggerBroadcast()

		msgQueryChanResult := <-queryChunk.Sender
		if msgQueryChanResult.Chunk == nil && expectPresent {
			return false
		}
	}
	return true
}

func TestStoredDataKeptUntilFinalized(t *testing.T) {
	harness := newTestHarness(t, false)
	candidateHash := parachaintypes.CandidateHash{Value: availability_store.TestCandidateReceiptHash}
	nValidators := uint(10) // TODO(ed): simulate nValidators call

	pov := parachaintypes.PoV{BlockData: parachaintypes.BlockData{4, 5, 6}}

	availableData := availability_store.AvailableData{
		PoV: pov,
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte{}},
		},
	}
	parent := common.Hash{0x02, 0x02, 0x02, 0x02}
	blockNumber := parachaintypes.BlockNumber(3)

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
		NumValidators:       nValidators,
		AvailableData:       availableData,
		ExpectedErasureRoot: branchHash,
		Sender:              msgSenderChan,
	}

	harness.broadcastMessages = append(harness.broadcastMessages, blockMsg)

	err = harness.overseer.Start()
	require.NoError(t, err)

	go harness.processMessages()

	harness.triggerBroadcast()

	// result from seeding data
	msgSenderChanResult := <-blockMsg.Sender
	require.Equal(t, nil, msgSenderChanResult)

	// check that the data is there
	msgSenderQueryChan := make(chan availability_store.AvailableData)
	queryData := availability_store.QueryAvailableData{
		CandidateHash: candidateHash,
		Sender:        msgSenderQueryChan,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan := <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)
	harness.printDB("before import leaf")

	// trigger import leaf
	aLeaf := importLeaf(t, harness, parent, blockNumber)

	time.Sleep(500 * time.Millisecond)
	harness.printDB("after import leaf")

	// check that the data is still there
	// queryAvailabeData, hasAllChunks
	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	require.Equal(t, availableData, msgQueryChan)
	harness.printDB("after queryData")

	// check that the chunks are there
	hasChunks := hasAllChunks(harness, candidateHash, nValidators, true)
	require.True(t, hasChunks)

	// trigger block finalized
	blockFinalizedSignal := parachaintypes.BlockFinalizedSignal{
		Hash:        aLeaf,
		BlockNumber: blockNumber,
	}
	harness.broadcastMessages = append(harness.broadcastMessages, blockFinalizedSignal)
	harness.triggerBroadcast()

	// wait for pruning to occur and check that the data is gone
	time.Sleep(5000 * time.Millisecond)
	harness.printDB("after block finalized")

	harness.broadcastMessages = append(harness.broadcastMessages, queryData)

	harness.triggerBroadcast()
	msgQueryChan = <-msgSenderQueryChan
	expectedResult := availability_store.AvailableData{}
	require.Equal(t, expectedResult, msgQueryChan)
	harness.printDB("queryData after pruning")

	// check that the chunks are gone
	hasChunks = hasAllChunks(harness, candidateHash, nValidators, false)
	require.True(t, hasChunks)

	err = harness.overseer.Stop()
	require.NoError(t, err)
}
