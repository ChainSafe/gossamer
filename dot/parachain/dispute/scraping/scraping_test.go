package scraping

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type expectedMessages struct {
	finalisedBlockRequests int
	ancestorRequests       int
}

type expectedRuntimeCalls struct {
	candidateEventsRequests int
	candidateVotesRequests  int
}

func getBlockNumberHash(blockNumber parachainTypes.BlockNumber) common.Hash {
	encodedBlockNumber, err := scale.Marshal(blockNumber)
	if err != nil {
		panic("failed to encode block number:" + err.Error())
	}

	blockHash, err := common.Blake2bHash(encodedBlockNumber)
	if err != nil {
		panic("failed to hash block number:" + err.Error())
	}

	return blockHash
}

func getNextLeaf(t *testing.T, chain *[]common.Hash) *overseer.ActivatedLeaf {
	t.Helper()
	nextBlockNumber := len(*chain)
	nextHash := getBlockNumberHash(parachainTypes.BlockNumber(nextBlockNumber))
	*chain = append(*(chain), nextHash)
	return dummyActivatedLeaf(parachainTypes.BlockNumber(nextBlockNumber))
}

func dummyActivatedLeaf(blockNumber parachainTypes.BlockNumber) *overseer.ActivatedLeaf {
	return &overseer.ActivatedLeaf{
		Hash:   getBlockNumberHash(blockNumber),
		Number: uint32(blockNumber),
	}
}

func dummyCandidateReceipt(relayParent common.Hash) parachainTypes.CandidateReceipt {
	descriptor := parachainTypes.CandidateDescriptor{
		ParaID:                      0,
		RelayParent:                 relayParent,
		Collator:                    parachainTypes.CollatorID{},
		PersistedValidationDataHash: common.Hash{},
		PovHash:                     common.Hash{},
		ErasureRoot:                 common.Hash{},
		Signature:                   parachainTypes.CollatorSignature{},
		ParaHead:                    common.Hash{},
		ValidationCodeHash:          parachainTypes.ValidationCodeHash{},
	}

	return parachainTypes.CandidateReceipt{
		Descriptor:      descriptor,
		CommitmentsHash: common.Hash{},
	}
}

func configureMockExpectations(
	expectedAncestries []int,
) (expectedMessages, expectedRuntimeCalls) {
	var (
		messages expectedMessages
		calls    expectedRuntimeCalls
	)

	// scraper initialisation calls
	messages.finalisedBlockRequests = 1
	messages.ancestorRequests = 0
	calls.candidateVotesRequests = 1
	calls.candidateEventsRequests = 1

	for _, expectedAncestryLength := range expectedAncestries {
		var ancestorRequests int
		switch expectedAncestryLength {
		case 1:
			ancestorRequests = 1
		default:
			ancestorRequests = (expectedAncestryLength + int(AncestryChunkSize) - 1) / int(AncestryChunkSize)
		}

		messages.finalisedBlockRequests += 1
		messages.ancestorRequests += ancestorRequests
		calls.candidateVotesRequests += expectedAncestryLength
		calls.candidateEventsRequests += expectedAncestryLength
	}

	return messages, calls
}

func configureMockOverseer(
	t *testing.T,
	sender *MockSender,
	chain *[]common.Hash,
	messages expectedMessages,
	finalisedBlock uint32,
) {
	var (
		finalisedBlockRequestCalls = 0
		ancestorRequestCalls       = 0
	)
	sender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
		switch message := msg.(type) {
		case overseer.FinalizedBlockNumberRequest:
			require.Less(t, finalisedBlockRequestCalls, messages.finalisedBlockRequests)
			result := finalisedBlock
			if finalisedBlockRequestCalls == 0 {
				result = 0
			}
			finalisedBlockRequestCalls++

			response := overseer.FinalizedBlockNumberResponse{
				Number: result,
				Err:    nil,
			}
			message.ResponseChannel <- response
		case overseer.AncestorsRequest:
			require.Less(t, ancestorRequestCalls, messages.ancestorRequests)
			ancestorRequestCalls++
			maybeBlockPosition := -1
			for idx, h := range *chain {
				if h == message.Hash {
					maybeBlockPosition = idx
					break
				}
			}

			var ancestors []common.Hash
			if maybeBlockPosition != -1 {
				ancestors = make([]common.Hash, 0)
				for i := maybeBlockPosition - 1; i >= 0 && i >= maybeBlockPosition-int(message.K); i-- {
					ancestors = append(ancestors, (*chain)[i])
				}
			}

			response := overseer.AncestorsResponse{
				Ancestors: ancestors,
				Error:     nil,
			}
			message.ResponseChannel <- response
		default:
			return fmt.Errorf("unknown message type")
		}
		return nil
	}).Times(messages.finalisedBlockRequests + messages.ancestorRequests)
}

func mockBackedCandidateEvent(blockHash common.Hash) (*scale.VaryingDataTypeSlice, error) {
	candidateEvents, err := parachainTypes.NewCandidateEvents()
	if err != nil {
		return nil, fmt.Errorf("creating candidate events: %w", err)
	}
	candidateReceipt := dummyCandidateReceipt(blockHash)

	backedEvent := parachainTypes.CandidateBacked{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachainTypes.HeadData{},
		CoreIndex:        parachainTypes.CoreIndex{},
		GroupIndex:       0,
	}
	err = candidateEvents.Add(backedEvent)
	if err != nil {
		return nil, fmt.Errorf("adding candidate events: %w", err)
	}

	return &candidateEvents, nil
}

func mockBackedAndIncludedCandidateEvent(blockHash common.Hash) (*scale.VaryingDataTypeSlice, error) {
	candidateEvents, err := parachainTypes.NewCandidateEvents()
	if err != nil {
		return nil, fmt.Errorf("creating candidate events: %w", err)
	}
	candidateReceipt := dummyCandidateReceipt(blockHash)

	includedEvent := parachainTypes.CandidateIncluded{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachainTypes.HeadData{},
		CoreIndex:        parachainTypes.CoreIndex{},
		GroupIndex:       0,
	}
	backedEvent := parachainTypes.CandidateBacked{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachainTypes.HeadData{},
		CoreIndex:        parachainTypes.CoreIndex{},
		GroupIndex:       0,
	}
	err = candidateEvents.Add(includedEvent, backedEvent)
	if err != nil {
		return nil, fmt.Errorf("adding candidate events: %w", err)
	}

	return &candidateEvents, nil
}

func getMagicHash() (common.Hash, error) {
	toEncode := "abc"
	encoded, err := scale.Marshal(toEncode)
	if err != nil {
		return common.Hash{}, fmt.Errorf("encoding string: %w", err)
	}

	blockHash, err := common.Blake2bHash(encoded)
	if err != nil {
		return common.Hash{}, fmt.Errorf("hashing string: %w", err)
	}

	return blockHash, nil
}

func mockMagicCandidateEvents() (*scale.VaryingDataTypeSlice, error) {
	candidateEvents, err := parachainTypes.NewCandidateEvents()
	if err != nil {
		return nil, fmt.Errorf("creating candidate events: %w", err)
	}

	blockHash, err := getMagicHash()
	if err != nil {
		return nil, fmt.Errorf("getting magic hash: %w", err)
	}

	candidateReceipt := dummyCandidateReceipt(blockHash)
	includedEvent := parachainTypes.CandidateIncluded{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachainTypes.HeadData{},
		CoreIndex:        parachainTypes.CoreIndex{},
		GroupIndex:       0,
	}
	backedEvent := parachainTypes.CandidateBacked{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachainTypes.HeadData{},
		CoreIndex:        parachainTypes.CoreIndex{},
		GroupIndex:       0,
	}

	err = candidateEvents.Add(includedEvent, backedEvent)
	if err != nil {
		return nil, fmt.Errorf("adding candidate events: %w", err)
	}

	return &candidateEvents, nil
}

func mockCandidateEvents(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error) {
	maybeBlockNumber := -1
	for idx, h := range *chain {
		if h == blockHash {
			maybeBlockNumber = idx
			break
		}
	}

	candidateEvents, err := parachainTypes.NewCandidateEvents()
	if err != nil {
		return nil, fmt.Errorf("creating candidate events: %w", err)
	}

	if maybeBlockNumber != -1 {
		candidateReceipt := dummyCandidateReceipt(blockHash)
		candidateEvent1 := parachainTypes.CandidateIncluded{
			CandidateReceipt: candidateReceipt,
			HeadData:         parachainTypes.HeadData{},
			CoreIndex:        parachainTypes.CoreIndex{},
			GroupIndex:       0,
		}
		candidateEvent2 := parachainTypes.CandidateBacked{
			CandidateReceipt: candidateReceipt,
			HeadData:         parachainTypes.HeadData{},
			CoreIndex:        parachainTypes.CoreIndex{},
			GroupIndex:       0,
		}
		err = candidateEvents.Add(candidateEvent1, candidateEvent2)
		if err != nil {
			return nil, fmt.Errorf("adding candidate events: %w", err)
		}
		return &candidateEvents, nil
	}

	return &candidateEvents, nil
}

func configureMockRuntime(
	runtime *MockRuntimeInstance,
	chain *[]common.Hash,
	calls expectedRuntimeCalls,
	eventGenerator func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error),
) {
	runtime.EXPECT().ParachainHostCandidateEvents(gomock.Any()).DoAndReturn(
		func(arg0 interface{},
		) (*scale.VaryingDataTypeSlice, error) {
			blockHash := arg0.(common.Hash)
			maybeBlockNumber := -1
			for idx, h := range *chain {
				if h == blockHash {
					maybeBlockNumber = idx
					break
				}
			}

			if maybeBlockNumber != -1 {
				return eventGenerator(blockHash, chain)
			}

			return nil, nil //nolint: nilnil
		}).Times(calls.candidateEventsRequests)

	runtime.EXPECT().ParachainHostOnChainVotes(gomock.Any()).Return(nil, nil).Times(calls.candidateVotesRequests)
}

func newTestState(
	t *testing.T,
	sender *MockSender,
	runtime *MockRuntimeInstance,
	messages expectedMessages,
	calls expectedRuntimeCalls,
	finalisedBlock uint32,
	eventGenerator func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error),
) (*ChainScraper, *[]common.Hash) {
	chain := []common.Hash{getBlockNumberHash(0), getBlockNumberHash(1)}
	configureMockOverseer(t, sender, &chain, messages, finalisedBlock)
	configureMockRuntime(runtime, &chain, calls, eventGenerator)

	scraper, _, err := NewChainScraper(sender, runtime, dummyActivatedLeaf(1))
	require.NoError(t, err)
	return scraper, &chain
}

func TestChainScraper(t *testing.T) {
	t.Parallel()

	t.Run("scraper_provides_included_state_when_initialised", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		candidate1, err := dummyCandidateReceipt(getBlockNumberHash(1)).Hash()
		require.NoError(t, err)
		candidate2, err := dummyCandidateReceipt(getBlockNumberHash(2)).Hash()
		require.NoError(t, err)

		expectedAncestryLength := 1
		finalisedBlock := uint32(0)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})

		scraper, chain := newTestState(t,
			mockSender,
			mockRuntime,
			messages,
			calls,
			finalisedBlock,
			mockCandidateEvents,
		)

		require.False(t, scraper.IsCandidateIncluded(candidate2))
		require.False(t, scraper.IsCandidateBacked(candidate2))
		require.True(t, scraper.IsCandidateIncluded(candidate1))
		require.True(t, scraper.IsCandidateBacked(candidate1))

		nextLeaf := getNextLeaf(t, chain)
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}

		_, err = scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)
		require.True(t, scraper.IsCandidateIncluded(candidate2))
		require.True(t, scraper.IsCandidateBacked(candidate2))
	})

	t.Run("scraper_requests_candidates_of_leaf_ancestors", func(t *testing.T) {
		t.Parallel()
		// How many blocks should we skip before sending a leaf update.
		const BlocksToSkip = 30

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(0)
		expectedAncestryLength := int(BlocksToSkip - finalisedBlock)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t, mockSender, mockRuntime, messages, calls, finalisedBlock, mockCandidateEvents)

		var nextLeaf *overseer.ActivatedLeaf
		for i := 0; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		nextBlockNumber := len(*chain)
		for i := 1; i < nextBlockNumber; i++ {
			candidateHash, err := dummyCandidateReceipt(getBlockNumberHash(parachainTypes.BlockNumber(i))).Hash()
			require.NoError(t, err)
			require.True(t, scraper.IsCandidateIncluded(candidateHash))
			require.True(t, scraper.IsCandidateBacked(candidateHash))
		}
	})

	t.Run("scraper_requests_candidates_of_non_cached_ancestors", func(t *testing.T) {
		t.Parallel()
		var BlocksToSkip = []int{30, 15}

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(0)
		messages, calls := configureMockExpectations(BlocksToSkip)
		scraper, chain := newTestState(t, mockSender, mockRuntime, messages, calls, finalisedBlock, mockCandidateEvents)

		var nextLeaf *overseer.ActivatedLeaf
		for i := 0; i < BlocksToSkip[0]; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		for i := 0; i < BlocksToSkip[1]; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate = overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err = scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)
	})

	t.Run("scraper_requests_candidates_of_non_finalized_ancestors", func(t *testing.T) {
		t.Parallel()
		// How many blocks should we skip before sending a leaf update.
		const BlocksToSkip = 30

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(17)
		expectedAncestryLength := int(BlocksToSkip - (finalisedBlock - DisputeCandidateLifetimeAfterFinalization))
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t, mockSender, mockRuntime, messages, calls, finalisedBlock, mockCandidateEvents)

		var nextLeaf *overseer.ActivatedLeaf
		// 1 because `TestState` starts at leaf 1.
		for i := 1; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)
	})

	t.Run("scraper_prunes_finalized_candidates", func(t *testing.T) {
		t.Parallel()
		const (
			TargetBlockNumber = 2
			BlocksToSkip      = 3
		)

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(1)
		expectedAncestryLength := BlocksToSkip - int(finalisedBlock)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t,
			mockSender,
			mockRuntime,
			messages,
			calls,
			finalisedBlock,
			func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error) {
				if blockHash == getBlockNumberHash(2) {
					return mockCandidateEvents(blockHash, chain)
				}
				candidateEvents, err := parachainTypes.NewCandidateEvents()
				if err != nil {
					return nil, fmt.Errorf("creating candidate events: %w", err)
				}
				return &candidateEvents, nil
			})

		var nextLeaf *overseer.ActivatedLeaf
		for i := 1; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		finalisedBlockNumber := TargetBlockNumber + DisputeCandidateLifetimeAfterFinalization
		scraper.ProcessFinalisedBlock(finalisedBlockNumber)

		candidate := dummyCandidateReceipt(getBlockNumberHash(TargetBlockNumber))
		candidateHash, err := candidate.Hash()
		require.NoError(t, err)

		require.False(t, scraper.IsCandidateBacked(candidateHash))
		require.False(t, scraper.IsCandidateIncluded(candidateHash))
	})

	t.Run("scraper_handles_backed_but_not_included_candidate", func(t *testing.T) {
		t.Parallel()
		const (
			TargetBlockNumber = 2
			BlocksToSkip      = 3
		)

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(1)
		expectedAncestryLength := BlocksToSkip - int(finalisedBlock)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t,
			mockSender,
			mockRuntime,
			messages,
			calls,
			finalisedBlock,
			func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error) {
				if blockHash == getBlockNumberHash(2) {
					return mockBackedCandidateEvent(blockHash)
				}
				candidateEvents, err := parachainTypes.NewCandidateEvents()
				if err != nil {
					return nil, fmt.Errorf("creating candidate events: %w", err)
				}
				return &candidateEvents, nil
			},
		)

		var nextLeaf *overseer.ActivatedLeaf
		for i := 1; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		finalisedBlock++
		scraper.ProcessFinalisedBlock(finalisedBlock)

		candidate := dummyCandidateReceipt(getBlockNumberHash(TargetBlockNumber))
		candidateHash, err := candidate.Hash()
		require.NoError(t, err)

		require.True(t, scraper.IsCandidateBacked(candidateHash))
		require.False(t, scraper.IsCandidateIncluded(candidateHash))
		require.True(t, finalisedBlock < TargetBlockNumber+DisputeCandidateLifetimeAfterFinalization)

		finalisedBlock += TargetBlockNumber + DisputeCandidateLifetimeAfterFinalization
		scraper.ProcessFinalisedBlock(finalisedBlock)

		require.False(t, scraper.IsCandidateBacked(candidateHash))
		require.False(t, scraper.IsCandidateIncluded(candidateHash))
	})

	t.Run("scraper_handles_the_same_candidate_included_in_two_different_block_heights", func(t *testing.T) {
		t.Parallel()
		testTarget1 := parachainTypes.BlockNumber(2)
		testTarget2 := parachainTypes.BlockNumber(3)
		const BlocksToSkip = 3

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(1)
		expectedAncestryLength := BlocksToSkip - int(finalisedBlock)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t, mockSender,
			mockRuntime,
			messages,
			calls,
			finalisedBlock,
			func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error) {
				if blockHash == getBlockNumberHash(1) {
					return mockBackedAndIncludedCandidateEvent(blockHash)
				}

				if blockHash == getBlockNumberHash(testTarget1) || blockHash == getBlockNumberHash(testTarget2) {
					return mockMagicCandidateEvents()
				}

				candidateEvents, err := parachainTypes.NewCandidateEvents()
				if err != nil {
					return nil, fmt.Errorf("creating candidate events: %w", err)
				}
				return &candidateEvents, nil
			})

		var nextLeaf *overseer.ActivatedLeaf
		for i := 1; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		// Finalize blocks to enforce pruning of scraped events.
		// The magic candidate was added twice, so it shouldn't be removed if we finalize two more blocks.
		finalisedBlock = uint32(testTarget1) + DisputeCandidateLifetimeAfterFinalization
		scraper.ProcessFinalisedBlock(finalisedBlock)

		magicHash, err := getMagicHash()
		require.NoError(t, err)
		magicCandidate := dummyCandidateReceipt(magicHash)
		magicCandidateHash, err := magicCandidate.Hash()
		require.NoError(t, err)

		require.True(t, scraper.IsCandidateBacked(magicCandidateHash))
		require.True(t, scraper.IsCandidateIncluded(magicCandidateHash))

		finalisedBlock += 1
		scraper.ProcessFinalisedBlock(finalisedBlock)

		require.False(t, scraper.IsCandidateBacked(magicCandidateHash))
		require.False(t, scraper.IsCandidateIncluded(magicCandidateHash))
	})

	t.Run("inclusions_per_candidate_properly_adds_and_prunes", func(t *testing.T) {
		t.Parallel()
		testTarget1 := parachainTypes.BlockNumber(2)
		testTarget2 := parachainTypes.BlockNumber(3)
		const BlocksToSkip = 4

		ctrl := gomock.NewController(t)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		mockSender := NewMockSender(ctrl)

		finalisedBlock := uint32(1)
		expectedAncestryLength := BlocksToSkip - int(finalisedBlock)
		messages, calls := configureMockExpectations([]int{expectedAncestryLength})
		scraper, chain := newTestState(t, mockSender,
			mockRuntime,
			messages,
			calls,
			finalisedBlock,
			func(blockHash common.Hash, chain *[]common.Hash) (*scale.VaryingDataTypeSlice, error) {
				if blockHash == getBlockNumberHash(1) {
					return mockBackedAndIncludedCandidateEvent(blockHash)
				}

				if blockHash == getBlockNumberHash(testTarget1) || blockHash == getBlockNumberHash(testTarget2) {
					return mockBackedAndIncludedCandidateEvent(getBlockNumberHash(testTarget1))
				}

				candidateEvents, err := parachainTypes.NewCandidateEvents()
				if err != nil {
					return nil, fmt.Errorf("creating candidate events: %w", err)
				}
				return &candidateEvents, nil
			})

		var nextLeaf *overseer.ActivatedLeaf
		for i := 1; i < BlocksToSkip; i++ {
			nextLeaf = getNextLeaf(t, chain)
		}
		nextUpdate := overseer.ActiveLeavesUpdate{Activated: nextLeaf}
		_, err := scraper.ProcessActiveLeavesUpdate(mockSender, nextUpdate)
		require.NoError(t, err)

		candidateHash, err := dummyCandidateReceipt(getBlockNumberHash(testTarget1)).Hash()
		require.NoError(t, err)
		inclusions := scraper.GetBlocksIncludingCandidate(candidateHash)
		require.Equal(t, 2, len(inclusions))
		require.Equal(t,
			Inclusion{
				BlockNumber: uint32(testTarget1),
				BlockHash:   getBlockNumberHash(testTarget1),
			},
			inclusions[0],
		)
		require.Equal(t,
			Inclusion{
				BlockNumber: uint32(testTarget2),
				BlockHash:   getBlockNumberHash(testTarget2),
			},
			inclusions[1],
		)

		finalisedBlock = uint32(testTarget1) + DisputeCandidateLifetimeAfterFinalization
		scraper.ProcessFinalisedBlock(finalisedBlock)

		inclusions = scraper.GetBlocksIncludingCandidate(candidateHash)
		require.Equal(t, 1, len(inclusions))
		require.Equal(t,
			Inclusion{
				BlockNumber: uint32(testTarget2),
				BlockHash:   getBlockNumberHash(testTarget2),
			},
			inclusions[0],
		)

		finalisedBlock = uint32(testTarget2) + DisputeCandidateLifetimeAfterFinalization
		scraper.ProcessFinalisedBlock(finalisedBlock)

		inclusions = scraper.GetBlocksIncludingCandidate(candidateHash)
		require.Equal(t, 0, len(inclusions))
	})
}
