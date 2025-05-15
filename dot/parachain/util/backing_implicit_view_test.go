package util

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func chainOfBlock() []common.Hash {
	return []common.Hash{{4}, {5}, {6}, {7}, {8}, {9}}
}

func setupTest(t *testing.T) (*MockBlockState, *MockInstance, *BackingImplicitView) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockInstance := NewMockInstance(ctrl)

	view := NewBackingImplicitView(mockBlockState, nil)
	return mockBlockState, mockInstance, view
}

func TestBackingImplicitView_ActivateLeaf(t *testing.T) {
	t.Parallel()
	t.Run("activates_new_leaf_successfully", func(t *testing.T) {
		t.Parallel()
		chain := chainOfBlock()
		chainLen := len(chain)

		mockBlockState, _, view := setupTest(t)

		leafHash := chain[chainLen-1]
		parentHash := chain[chainLen-2]

		// Setup expectations
		mockBlockState.EXPECT().GetHeader(leafHash).Return(GetBlockHeader(t, chain, leafHash), nil)
		mockBlockState.EXPECT().GetHeader(parentHash).Return(GetBlockHeader(t, chain, parentHash), nil)

		ch := make(chan any)
		go func() {
			getMin := (<-ch).(messages.GetMinimumRelayParents)
			getMin.Sender <- []messages.ParaIDBlockNumber{{
				ParaId:      100,
				BlockNumber: 5,
			}}
		}()

		err := view.ActivateLeaf(leafHash, ch)
		require.NoError(t, err)

		// ensure the leaf is activated
		pruningInfo, exists := view.leaves[leafHash]
		require.True(t, exists)
		require.EqualValues(t, 4, pruningInfo.retainMinimum)

		// ensure the block info is stored
		info, exists := view.blockInfoStorage[leafHash]
		require.True(t, exists)
		require.EqualValues(t, 6, info.blockNumber)
		require.Equal(t, parentHash, info.parentHash)

		require.NotNil(t, info.allowedRelayParents)
		require.Len(t, info.allowedRelayParents.allowedRelayParentsContiguous, 2)

		minBlock, exists := info.allowedRelayParents.minimumRelayParent[100]
		require.True(t, exists)
		require.EqualValues(t, 5, minBlock)

	})

	t.Run("fails_when_leaf_already_exists", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		leafHash := common.Hash{1}
		view.leaves[leafHash] = activeLeafPruningInfo{retainMinimum: 0}

		err := view.ActivateLeaf(leafHash, make(chan any))
		require.Error(t, err)
		require.ErrorIs(t, err, errLeafAlreadyKnown)
	})

	t.Run("fails_when_cannot_get_header", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)

		leafHash := common.Hash{1}
		mockBlockState.EXPECT().GetHeader(leafHash).Return(nil, errors.New("header not found"))

		err := view.ActivateLeaf(leafHash, make(chan any))
		require.Error(t, err)
	})
}

func TestBackingImplicitView_DeactivateLeaf(t *testing.T) {
	t.Parallel()
	t.Run("deactivates_existing_leaf_and_prune_not_required_blockinfo", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		leafHash := common.Hash{1}
		view.leaves[leafHash] = activeLeafPruningInfo{retainMinimum: 5}
		view.leaves[common.Hash{2}] = activeLeafPruningInfo{retainMinimum: 7}

		view.blockInfoStorage[leafHash] = blockInfo{blockNumber: 10}      // Should not be pruned
		view.blockInfoStorage[common.Hash{3}] = blockInfo{blockNumber: 4} // Should be pruned
		view.blockInfoStorage[common.Hash{4}] = blockInfo{blockNumber: 6} // Should be pruned

		pruned := view.DeactivateLeaf(leafHash)
		require.Len(t, pruned, 2)
		require.Contains(t, pruned, common.Hash{3}) // because blockNumber-4 < minimumBlockNumber-7
		require.Contains(t, pruned, common.Hash{4}) // because blockNumber-6 < minimumBlockNumber-7
		require.NotContains(t, pruned, leafHash)    // because blockNumber-10 > minimumBlockNumber-7

		require.NotContains(t, view.leaves, leafHash)
	})

	t.Run("deactivates_leaf_and_prunes_everything", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		leafHash := common.Hash{1}
		view.leaves[leafHash] = activeLeafPruningInfo{retainMinimum: 5}

		// after removing leafHash, all these should be pruned
		view.blockInfoStorage[leafHash] = blockInfo{blockNumber: 10}
		view.blockInfoStorage[common.Hash{2}] = blockInfo{blockNumber: 4}
		view.blockInfoStorage[common.Hash{3}] = blockInfo{blockNumber: 6}

		pruned := view.DeactivateLeaf(leafHash)
		require.Len(t, pruned, 3)
		require.Contains(t, pruned, leafHash)
		require.Contains(t, pruned, common.Hash{2})
		require.Contains(t, pruned, common.Hash{3})

		require.NotContains(t, view.leaves, leafHash)
	})

	t.Run("does_nothing_for_non_existent_leaf", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		pruned := view.DeactivateLeaf(common.Hash{1})
		require.Empty(t, pruned)
	})
}

func TestBackingImplicitView_PathsViaRelayParent(t *testing.T) {
	t.Parallel()
	t.Run("finds_paths_to_relay_parent", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		// Setup a simple chain: leaf -> middle -> relayParent
		leafHash := common.Hash{3}
		middleHash := common.Hash{2}
		relayParentHash := common.Hash{1}

		view.leaves[leafHash] = activeLeafPruningInfo{}
		view.blockInfoStorage[leafHash] = blockInfo{
			blockNumber: 3,
			parentHash:  middleHash,
		}
		view.blockInfoStorage[middleHash] = blockInfo{
			blockNumber: 2,
			parentHash:  relayParentHash,
		}
		view.blockInfoStorage[relayParentHash] = blockInfo{
			blockNumber: 1,
		}

		paths := view.PathsViaRelayParent(relayParentHash)
		require.Len(t, paths, 1)
		require.Equal(t, []common.Hash{relayParentHash, middleHash, leafHash}, paths[0])
	})

	t.Run("returns_empty_when_no_paths_exist", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		paths := view.PathsViaRelayParent(common.Hash{1})
		require.Empty(t, paths)
	})

	t.Run("handles_multiple_paths", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		// Setup two paths to relay parent:
		// leaf1 -> middle -> relayParent
		// leaf2 -> relayParent
		relayParentHash := common.Hash{1}
		middleHash := common.Hash{2}
		leaf1Hash := common.Hash{3}
		leaf2Hash := common.Hash{4}

		view.leaves[leaf1Hash] = activeLeafPruningInfo{}
		view.leaves[leaf2Hash] = activeLeafPruningInfo{}

		view.blockInfoStorage[leaf1Hash] = blockInfo{
			blockNumber: 3,
			parentHash:  middleHash,
		}
		view.blockInfoStorage[middleHash] = blockInfo{
			blockNumber: 2,
			parentHash:  relayParentHash,
		}
		view.blockInfoStorage[leaf2Hash] = blockInfo{
			blockNumber: 2,
			parentHash:  relayParentHash,
		}
		view.blockInfoStorage[relayParentHash] = blockInfo{
			blockNumber: 1,
		}

		paths := view.PathsViaRelayParent(relayParentHash)
		require.Len(t, paths, 2)
	})
}

func TestBackingImplicitView_KnownAllowedRelayParentsUnder(t *testing.T) {
	t.Parallel()
	t.Run("returns_relay_parents_for_parachain", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		blockHash := common.Hash{1}
		paraID := parachaintypes.ParaID(100)

		ancestors := []common.Hash{{1}, {2}, {3}}
		allowed := &allowedRelayParents{
			minimumRelayParent: map[parachaintypes.ParaID]parachaintypes.BlockNumber{
				paraID: 1,
			},
			allowedRelayParentsContiguous: ancestors,
		}

		view.blockInfoStorage[blockHash] = blockInfo{
			blockNumber:         3,
			allowedRelayParents: allowed,
		}

		relayParents := view.KnownAllowedRelayParentsUnder(blockHash, &paraID)
		require.Equal(t, ancestors[:3], relayParents)
	})

	t.Run("returns_all_relay_parents_when_paraID_is_nil", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		blockHash := common.Hash{1}
		ancestors := []common.Hash{{1}, {2}, {3}}
		allowed := &allowedRelayParents{
			allowedRelayParentsContiguous: ancestors,
		}

		view.blockInfoStorage[blockHash] = blockInfo{
			blockNumber:         3,
			allowedRelayParents: allowed,
		}

		relayParents := view.KnownAllowedRelayParentsUnder(blockHash, nil)
		require.Equal(t, ancestors, relayParents)
	})

	t.Run("returns_nil_when_block_not_found", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)

		relayParents := view.KnownAllowedRelayParentsUnder(common.Hash{1}, nil)
		require.Nil(t, relayParents)
	})
}

func TestBackingImplicitView_FetchAncestorsUpToMinBlockNumber(t *testing.T) {
	t.Parallel()

	t.Run("fetches_and_stores_ancestor_blocks", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		chain := chainOfBlock()

		nextAncestorNumber := parachaintypes.BlockNumber(2)
		minBlockNumber := parachaintypes.BlockNumber(1)

		nextHash := chain[2]
		header := GetBlockHeader(t, chain, nextHash)

		parentHash := chain[1]
		parentHeader := GetBlockHeader(t, chain, parentHash)

		mockBlockState.EXPECT().GetHeader(nextHash).Return(header, nil)
		mockBlockState.EXPECT().GetHeader(parentHash).Return(parentHeader, nil)

		ancestry, err := view.fetchAncestorsUpToMinBlockNumber(nextHash, nextAncestorNumber, minBlockNumber)
		require.NoError(t, err)
		require.Len(t, ancestry, 2)
		require.Equal(t, nextHash, ancestry[0])
		require.Equal(t, parentHash, ancestry[1])

		// Verify block info was stored
		info, exists := view.blockInfoStorage[nextHash]
		require.True(t, exists)
		require.Equal(t, parachaintypes.BlockNumber(2), info.blockNumber)
		require.Equal(t, header.ParentHash, info.parentHash)
		require.Nil(t, info.allowedRelayParents)

		info, exists = view.blockInfoStorage[parentHash]
		require.True(t, exists)
		require.Equal(t, parachaintypes.BlockNumber(1), info.blockNumber)
		require.Equal(t, parentHeader.ParentHash, info.parentHash)
		require.Nil(t, info.allowedRelayParents)
	})

	t.Run("stops_at_genesis", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)

		nextHash := common.Hash{1}
		header := &types.Header{Number: 0} // Genesis block

		mockBlockState.EXPECT().GetHeader(nextHash).Return(header, nil)

		ancestry, err := view.fetchAncestorsUpToMinBlockNumber(nextHash, 0, 0)
		require.NoError(t, err)
		require.Len(t, ancestry, 1)
	})
}

func TestBackingImplicitView_FetchFreshLeafAndInsertAncestry(t *testing.T) {
	t.Parallel()
	t.Run("successfully_fetches_leaf_and_ancestry", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		chain := chainOfBlock()
		leafHash := chain[3] // block 4
		header := GetBlockHeader(t, chain, leafHash)

		// Mock getting block header
		mockBlockState.EXPECT().GetHeader(leafHash).Return(header, nil)

		// Mock headers for ancestors
		mockBlockState.EXPECT().GetHeader(chain[2]).Return(GetBlockHeader(t, chain, chain[2]), nil)
		mockBlockState.EXPECT().GetHeader(chain[1]).Return(GetBlockHeader(t, chain, chain[1]), nil)

		// Set up channel to handle prospective parachains request
		ch := make(chan any)
		go func() {
			getMin := (<-ch).(messages.GetMinimumRelayParents)
			getMin.Sender <- []messages.ParaIDBlockNumber{{
				ParaId:      100,
				BlockNumber: 2, // This will set minAncestorNumber to 2
			}}
		}()

		summary, err := view.fetchFreshLeafAndInsertAncestry(leafHash, ch)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Equal(t, parachaintypes.BlockNumber(2), summary.minimumAncestorNumber)
		require.Equal(t, parachaintypes.BlockNumber(4), summary.leafNumber)

		// Verify block info was stored correctly
		info, exists := view.blockInfoStorage[leafHash]
		require.True(t, exists)
		require.Equal(t, parachaintypes.BlockNumber(4), info.blockNumber)
		require.Equal(t, chain[2], info.parentHash)
		require.NotNil(t, info.allowedRelayParents)
		require.Len(t, info.allowedRelayParents.allowedRelayParentsContiguous, 3) // leaf + 2 ancestors
	})

	t.Run("handles_error_getting_block_header", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		leafHash := common.Hash{1}

		mockBlockState.EXPECT().GetHeader(leafHash).Return(nil, errors.New("header not found"))

		summary, err := view.fetchFreshLeafAndInsertAncestry(leafHash, make(chan any))
		require.Error(t, err)
		require.Nil(t, summary)
		require.Contains(t, err.Error(), "getting block header for leaf")
	})

	t.Run("handles_genesis_block", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		genesisHeader := &types.Header{Number: 0}

		// Mock getting block header
		mockBlockState.EXPECT().GetHeader(genesisHash).Return(genesisHeader, nil)

		// Set up channel to handle prospective parachains request
		ch := make(chan any)
		go func() {
			getMin := (<-ch).(messages.GetMinimumRelayParents)
			getMin.Sender <- []messages.ParaIDBlockNumber{{
				ParaId:      100,
				BlockNumber: 0,
			}}
		}()

		summary, err := view.fetchFreshLeafAndInsertAncestry(genesisHash, ch)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Equal(t, parachaintypes.BlockNumber(0), summary.minimumAncestorNumber)
		require.Equal(t, parachaintypes.BlockNumber(0), summary.leafNumber)

		// Verify block info was stored correctly
		info, exists := view.blockInfoStorage[genesisHash]
		require.True(t, exists)
		require.Equal(t, parachaintypes.BlockNumber(0), info.blockNumber)
		require.NotNil(t, info.allowedRelayParents)
		require.Len(t, info.allowedRelayParents.allowedRelayParentsContiguous, 1) // just genesis
	})

	t.Run("handles_empty_min_relay_parents", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		chain := chainOfBlock()
		leafHash := chain[1] // block 2

		// Mock getting block header
		mockBlockState.EXPECT().GetHeader(leafHash).Return(GetBlockHeader(t, chain, leafHash), nil)

		// Set up channel to handle prospective parachains request with empty response
		ch := make(chan any)
		go func() {
			getMin := (<-ch).(messages.GetMinimumRelayParents)
			getMin.Sender <- []messages.ParaIDBlockNumber{}
		}()

		summary, err := view.fetchFreshLeafAndInsertAncestry(leafHash, ch)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Equal(t, parachaintypes.BlockNumber(2), summary.minimumAncestorNumber)
		require.Equal(t, parachaintypes.BlockNumber(2), summary.leafNumber)

		// Verify block info was stored correctly
		info, exists := view.blockInfoStorage[leafHash]
		require.True(t, exists)
		require.Equal(t, parachaintypes.BlockNumber(2), info.blockNumber)
		require.Equal(t, chain[0], info.parentHash)
		require.NotNil(t, info.allowedRelayParents)
		require.Len(t, info.allowedRelayParents.allowedRelayParentsContiguous, 1) // leaf
	})
}

func TestBackingImplicitView_FindMinRelayParents(t *testing.T) {
	t.Parallel()

	t.Run("gets_min_relay_parents_from_prospective_parachains_when_not_collating", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)
		chain := chainOfBlock()
		leafHash := chain[5]
		header := GetBlockHeader(t, chain, leafHash)

		ch := make(chan any)
		go func() {
			getMin := (<-ch).(messages.GetMinimumRelayParents)
			getMin.Sender <- []messages.ParaIDBlockNumber{{
				ParaId:      100,
				BlockNumber: 5,
			}}
		}()

		minParents, err := view.findMinRelayParents(leafHash, header, ch)
		require.NoError(t, err)
		require.Len(t, minParents, 1)
		require.EqualValues(t, parachaintypes.ParaID(100), minParents[0].ParaId)
		require.EqualValues(t, parachaintypes.BlockNumber(5), minParents[0].BlockNumber)
	})

	t.Run("handles_timeout_from_prospective_parachains", func(t *testing.T) {
		t.Parallel()
		_, _, view := setupTest(t)
		chain := chainOfBlock()
		leafHash := chain[5]
		header := GetBlockHeader(t, chain, leafHash)

		ch := make(chan any)
		// Consume messages but don't respond to simulate timeout
		go func() {
			<-ch
		}()

		minParents, err := view.findMinRelayParents(leafHash, header, ch)
		require.Error(t, err)
		require.Contains(t, err.Error(), "timeout while waiting for relay parents")
		require.Nil(t, minParents)
	})

	t.Run("handles_collator_with_same_session_index", func(t *testing.T) {
		t.Parallel()
		mockBlockState, mockInstance, view := setupTest(t)
		paraID := parachaintypes.ParaID(100)
		view.collatingFor = &paraID

		lookahead := 3

		chain := chainOfBlock()
		chainLen := len(chain)
		lastBlockIndex := chainLen - 1

		leafHash := chain[chainLen-1]
		leafNumber := chainLen

		mockBlockState.EXPECT().GetRuntime(leafHash).Return(mockInstance, nil)
		mockInstance.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil)
		mockInstance.EXPECT().ParachainHostSchedulingLookAhead().Return(uint32(lookahead), nil)

		// Set up ancestors with same session index
		for i := 0; i < lookahead; i++ {
			mockBlockState.EXPECT().GetHeader(chain[lastBlockIndex-i]).
				Return(GetBlockHeader(t, chain, chain[lastBlockIndex-i]), nil)

			// mock runtime instance for a parent block, except genesis block
			if lastBlockIndex-i > 1 {
				// mock runtime instance for a parent block
				mockBlockState.EXPECT().GetRuntime(chain[lastBlockIndex-1-i]).Return(mockInstance, nil)
				mockInstance.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil)
			}

		}

		minParents, err := view.findMinRelayParents(leafHash, GetBlockHeader(t, chain, leafHash), nil)
		require.NoError(t, err)
		require.Len(t, minParents, 1)
		require.EqualValues(t, paraID, minParents[0].ParaId)
		require.EqualValues(t, leafNumber-lookahead, minParents[0].BlockNumber)
	})

	t.Run("handles_runtime_error_for_collator", func(t *testing.T) {
		t.Parallel()
		mockBlockState, _, view := setupTest(t)
		paraID := parachaintypes.ParaID(100)
		view.collatingFor = &paraID

		chain := chainOfBlock()
		leafHash := chain[5]
		header := GetBlockHeader(t, chain, leafHash)

		mockBlockState.EXPECT().GetRuntime(leafHash).Return(nil, errors.New("runtime error"))

		minParents, err := view.findMinRelayParents(leafHash, header, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "getting runtime instance")
		require.Nil(t, minParents)
	})

	t.Run("handles_session_index_error_for_collator", func(t *testing.T) {
		t.Parallel()
		mockBlockState, mockInstance, view := setupTest(t)
		paraID := parachaintypes.ParaID(100)
		view.collatingFor = &paraID

		chain := chainOfBlock()
		leafHash := chain[5]
		header := GetBlockHeader(t, chain, leafHash)

		mockBlockState.EXPECT().GetRuntime(leafHash).Return(mockInstance, nil)
		mockInstance.EXPECT().ParachainHostSessionIndexForChild().
			Return(parachaintypes.SessionIndex(0), errors.New("session error"))

		minParents, err := view.findMinRelayParents(leafHash, header, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "getting session index")
		require.Nil(t, minParents)
	})

	t.Run("handles_scheduling_lookahead_error_for_collator", func(t *testing.T) {
		t.Parallel()
		mockBlockState, mockInstance, view := setupTest(t)
		paraID := parachaintypes.ParaID(100)
		view.collatingFor = &paraID

		chain := chainOfBlock()
		leafHash := chain[5]
		header := GetBlockHeader(t, chain, leafHash)

		mockBlockState.EXPECT().GetRuntime(leafHash).Return(mockInstance, nil)
		mockInstance.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(2), nil)
		mockInstance.EXPECT().ParachainHostSchedulingLookAhead().Return(uint32(0), errors.New("lookahead error"))

		minParents, err := view.findMinRelayParents(leafHash, header, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "getting scheduling lookahead")
		require.Nil(t, minParents)
	})
}

func TestBackingImplicitView_ActivateLeafFromProspectiveParachains(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                     string
		activeLeaves             map[common.Hash]activeLeafPruningInfo
		leaf                     *BlockInfoProspectiveParachains
		ancestors                []*BlockInfoProspectiveParachains
		expectedBlockInfoStorage map[common.Hash]blockInfo
	}{
		{
			name:         "activates_new_leaf_without_ancestors",
			activeLeaves: make(map[common.Hash]activeLeafPruningInfo),
			leaf: &BlockInfoProspectiveParachains{
				Hash:       common.Hash{1},
				ParentHash: common.Hash{0},
				Number:     5,
			},
			ancestors: nil,
			expectedBlockInfoStorage: map[common.Hash]blockInfo{
				{1}: {
					blockNumber: 5,
					parentHash:  common.Hash{0},
					allowedRelayParents: &allowedRelayParents{
						minimumRelayParent:            nil,
						allowedRelayParentsContiguous: []common.Hash{},
					},
				},
			},
		},
		{
			name:         "activates_new_leaf_with_ancestors",
			activeLeaves: make(map[common.Hash]activeLeafPruningInfo),
			leaf: &BlockInfoProspectiveParachains{
				Hash:       common.Hash{3},
				ParentHash: common.Hash{2},
				Number:     3,
			},
			ancestors: []*BlockInfoProspectiveParachains{
				{
					Hash:       common.Hash{1},
					ParentHash: common.Hash{0},
					Number:     1,
				},
				{
					Hash:       common.Hash{2},
					ParentHash: common.Hash{1},
					Number:     2,
				},
			},
			expectedBlockInfoStorage: map[common.Hash]blockInfo{
				{1}: {
					blockNumber:         1,
					parentHash:          common.Hash{0},
					allowedRelayParents: nil,
				},
				{2}: {
					blockNumber:         2,
					parentHash:          common.Hash{1},
					allowedRelayParents: nil,
				},
				{3}: {
					blockNumber: 3,
					parentHash:  common.Hash{2},
					allowedRelayParents: &allowedRelayParents{
						minimumRelayParent:            nil,
						allowedRelayParentsContiguous: []common.Hash{{1}, {2}},
					},
				},
			},
		},
		{
			name:         "no_op_for_known_leaf",
			activeLeaves: map[common.Hash]activeLeafPruningInfo{{1}: {}},
			leaf: &BlockInfoProspectiveParachains{
				Hash:       common.Hash{1},
				ParentHash: common.Hash{0},
				Number:     1,
			},
			ancestors:                nil,
			expectedBlockInfoStorage: map[common.Hash]blockInfo{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			view := NewBackingImplicitView(nil, nil)
			view.leaves = tc.activeLeaves

			view.ActivateLeafFromProspectiveParachains(tc.leaf, tc.ancestors)

			// Verify leaf is activated
			require.Contains(t, view.leaves, tc.leaf.Hash)

			// Verify block info storage
			require.EqualValues(t, tc.expectedBlockInfoStorage, view.blockInfoStorage)

		})
	}
}

func TestAllowedRelayParents_AllowedFor(t *testing.T) {
	t.Parallel()

	paraID := parachaintypes.ParaID(100)
	ancestors := []common.Hash{{1}, {2}, {3}}

	testCases := []struct {
		name        string
		paraID      *parachaintypes.ParaID
		baseNumber  parachaintypes.BlockNumber
		minBlockNum map[parachaintypes.ParaID]parachaintypes.BlockNumber
		expected    []common.Hash
	}{
		{
			name:        "returns_full_ancestry_when_paraID_is_nil",
			paraID:      nil,
			baseNumber:  3,
			minBlockNum: make(map[parachaintypes.ParaID]parachaintypes.BlockNumber),
			expected:    ancestors,
		},
		{
			name:        "returns_nil_when_paraID_not_found",
			paraID:      &paraID,
			baseNumber:  3,
			minBlockNum: make(map[parachaintypes.ParaID]parachaintypes.BlockNumber),
			expected:    nil,
		},
		{
			name:       "returns_nil_when_base_number_less_than_minimum",
			paraID:     &paraID,
			baseNumber: 4,
			minBlockNum: map[parachaintypes.ParaID]parachaintypes.BlockNumber{
				paraID: 5,
			},
			expected: nil,
		},
		{
			name:       "returns_slice_based_on_difference_from_minimum",
			paraID:     &paraID,
			baseNumber: 2,
			minBlockNum: map[parachaintypes.ParaID]parachaintypes.BlockNumber{
				paraID: 1,
			},
			expected: ancestors[:2],
		},
		{
			name:       "returns_full_ancestry_when_difference_exceeds_length",
			paraID:     &paraID,
			baseNumber: 10,
			minBlockNum: map[parachaintypes.ParaID]parachaintypes.BlockNumber{
				paraID: 1,
			},
			expected: ancestors,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allowed := &allowedRelayParents{
				minimumRelayParent:            tc.minBlockNum,
				allowedRelayParentsContiguous: ancestors,
			}
			result := allowed.allowedFor(tc.paraID, tc.baseNumber)
			require.Equal(t, tc.expected, result)
		})
	}
}
