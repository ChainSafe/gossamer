// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

var sampleBlockBody = *types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})

var testGenesisHeader = &types.Header{
	Number:    0,
	StateRoot: trie.EmptyHash,
	Digest:    types.NewDigest(),
}

func newTestBlockState(t *testing.T) *BlockState {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	db := NewInMemoryDB(t)
	header := testGenesisHeader

	trieDBTable := database.NewTable(db, "storage")
	trieDB := NewTrieDB(trieDBTable)

	bs, err := NewBlockStateFromGenesis(db, trieDB, header, telemetryMock)
	require.NoError(t, err)

	// loads in-memory tries with genesis state root, should be deleted
	// after another block is finalised
	tr := trie.NewEmptyTrie()
	err = tr.Load(bs.db, header.StateRoot)
	require.NoError(t, err)
	bs.trieDB.Put(tr)

	return bs
}

func TestSetAndGetHeader(t *testing.T) {
	bs := newTestBlockState(t)

	header := &types.Header{
		Number:    0,
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	err := bs.SetHeader(header)
	require.NoError(t, err)

	res, err := bs.GetHeader(header.Hash())
	require.NoError(t, err)
	require.Equal(t, header, res)
}

func TestHasHeader(t *testing.T) {
	bs := newTestBlockState(t)

	header := &types.Header{
		Number:    0,
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	err := bs.SetHeader(header)
	require.NoError(t, err)

	has, err := bs.HasHeader(header.Hash())
	require.NoError(t, err)
	require.Equal(t, true, has)
}

func TestGetBlockByNumber(t *testing.T) {
	bs := newTestBlockState(t)

	blockHeader := &types.Header{
		ParentHash: testGenesisHeader.Hash(),
		Number:     1,
		Digest:     createPrimaryBABEDigest(t),
	}

	block := &types.Block{
		Header: *blockHeader,
		Body:   sampleBlockBody,
	}

	err := bs.AddBlock(block)
	require.NoError(t, err)

	retBlock, err := bs.GetBlockByNumber(blockHeader.Number)
	require.NoError(t, err)
	require.Equal(t, block, retBlock, "Could not validate returned retBlock as expected")
}

func TestAddBlock(t *testing.T) {
	bs := newTestBlockState(t)

	// Create header
	header0 := &types.Header{
		Number:     1,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: testGenesisHeader.Hash(),
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	block0 := &types.Block{
		Header: *header0,
		Body:   sampleBlockBody,
	}

	// Add the block0 to the DB
	err := bs.AddBlock(block0)
	require.NoError(t, err)

	// Create header & blockData for block 2
	header1 := &types.Header{
		Number:     2,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: blockHash0,
	}
	blockHash1 := header1.Hash()

	block1 := &types.Block{
		Header: *header1,
		Body:   sampleBlockBody,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	require.NoError(t, err)

	// Get the blocks & check if it's the same as the added blocks
	retBlock, err := bs.GetBlockByHash(blockHash0)
	require.NoError(t, err)

	require.Equal(t, block0, retBlock, "Could not validate returned block0 as expected")

	retBlock, err = bs.GetBlockByHash(blockHash1)
	require.NoError(t, err)

	// this will panic if not successful, so catch and fail it so
	func() {
		hash := retBlock.Header.Hash()
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("got panic when processing retBlock.Header.Hash() ", r)
			}
		}()
		require.False(t, hash.IsEmpty())
	}()

	require.Equal(t, block1, retBlock, "Could not validate returned block1 as expected")

	// Check if latestBlock is set correctly
	require.Equal(t, block1.Header.Hash(), bs.BestBlockHash(), "Latest Header Block Check Fail")
}

func TestGetSlotForBlock(t *testing.T) {
	bs := newTestBlockState(t)
	expectedSlot := uint64(77)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, expectedSlot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	res, err := bs.GetSlotForBlock(block.Header.Hash())
	require.NoError(t, err)
	require.Equal(t, expectedSlot, res)
}

func TestGetHashesByNumber(t *testing.T) {
	t.Parallel()

	// create two blocks with the same block number and test if GetHashesByNumber gets us
	// both the blocks
	bs := newTestBlockState(t)
	slot := uint64(77)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, slot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	babeHeader2 := types.NewBabeDigest()
	err = babeHeader2.Set(*types.NewBabePrimaryPreDigest(1, slot+1, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data2, err := scale.Marshal(babeHeader2)
	require.NoError(t, err)
	preDigest2 := types.NewBABEPreRuntimeDigest(data2)

	digest2 := types.NewDigest()
	err = digest2.Add(*preDigest2)
	require.NoError(t, err)
	block2 := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest2,
		},
		Body: types.Body{},
	}
	err = bs.AddBlock(block2)
	require.NoError(t, err)

	blocks, err := bs.GetHashesByNumber(1)
	require.NoError(t, err)
	require.ElementsMatch(t, blocks, []common.Hash{block.Header.Hash(), block2.Header.Hash()})
}

func TestGetAllDescendants(t *testing.T) {
	t.Parallel()

	bs := newTestBlockState(t)
	slot := uint64(77)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, slot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest,
		},
		Body: sampleBlockBody,
	}

	err = bs.AddBlockWithArrivalTime(block, time.Now())
	require.NoError(t, err)

	babeHeader2 := types.NewBabeDigest()
	err = babeHeader2.Set(*types.NewBabePrimaryPreDigest(1, slot+1, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data2, err := scale.Marshal(babeHeader2)
	require.NoError(t, err)
	preDigest2 := types.NewBABEPreRuntimeDigest(data2)

	digest2 := types.NewDigest()
	err = digest2.Add(*preDigest2)
	require.NoError(t, err)
	block2 := &types.Block{
		Header: types.Header{
			ParentHash: block.Header.Hash(),
			Number:     2,
			Digest:     digest2,
		},
		Body: sampleBlockBody,
	}
	err = bs.AddBlockWithArrivalTime(block2, time.Now())
	require.NoError(t, err)

	err = bs.SetFinalisedHash(block2.Header.Hash(), 1, 1)
	require.NoError(t, err)

	// can't fetch given block's descendants since the given block get removed from memory after
	// being finalised, using blocktree.GetAllDescendants
	_, err = bs.bt.GetAllDescendants(block.Header.Hash())
	require.ErrorIs(t, err, blocktree.ErrNodeNotFound)

	// can fetch given finalised block's descendants using disk, using using blockstate.GetAllDescendants
	blockHashes, err := bs.GetAllDescendants(block.Header.Hash())
	require.NoError(t, err)
	require.ElementsMatch(t, blockHashes, []common.Hash{block.Header.Hash(), block2.Header.Hash()})
}

func TestGetBlockHashesBySlot(t *testing.T) {
	t.Parallel()

	// create two block in the same slot and test if GetBlockHashesBySlot gets us
	// both the blocks
	bs := newTestBlockState(t)
	slot := uint64(77)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, slot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	babeHeader2 := types.NewBabeDigest()
	err = babeHeader2.Set(*types.NewBabePrimaryPreDigest(1, slot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data2, err := scale.Marshal(babeHeader2)
	require.NoError(t, err)
	preDigest2 := types.NewBABEPreRuntimeDigest(data2)

	digest2 := types.NewDigest()
	err = digest2.Add(*preDigest2)
	require.NoError(t, err)
	block2 := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest2,
		},
		Body: types.Body{},
	}
	err = bs.AddBlock(block2)
	require.NoError(t, err)

	blocks, err := bs.GetBlockHashesBySlot(slot)
	require.NoError(t, err)
	require.ElementsMatch(t, blocks, []common.Hash{block.Header.Hash(), block2.Header.Hash()})
}

func TestIsBlockOnCurrentChain(t *testing.T) {
	bs := newTestBlockState(t)
	currChain, branchChains := AddBlocksToState(t, bs, 3, false)

	for _, header := range currChain {
		onChain, err := bs.isBlockOnCurrentChain(header)
		require.NoError(t, err)

		if !onChain {
			t.Fatalf("Fail: expected block %s to be on current chain", header.Hash())
		}
	}

	for _, header := range branchChains {
		onChain, err := bs.isBlockOnCurrentChain(header)
		require.NoError(t, err)

		if onChain {
			t.Fatalf("Fail: expected block %s not to be on current chain", header.Hash())
		}
	}
}

func TestAddBlock_BlockNumberToHash(t *testing.T) {
	bs := newTestBlockState(t)
	currChain, branchChains := AddBlocksToState(t, bs, 8, false)

	bestHash := bs.BestBlockHash()
	bestHeader, err := bs.BestBlockHeader()
	require.NoError(t, err)

	var resBlock *types.Block
	for _, header := range currChain {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		require.NoError(t, err)

		if resBlock.Header.Hash() != header.Hash() {
			t.Fatalf("Fail: got %s expected %s for block %d", resBlock.Header.Hash(), header.Hash(), header.Number)
		}
	}

	for _, header := range branchChains {
		resBlock, err = bs.GetBlockByNumber(header.Number)
		require.NoError(t, err)

		if resBlock.Header.Hash() == header.Hash() {
			t.Fatalf("Fail: should not have gotten block %s for branch block num=%d", header.Hash(), header.Number)
		}
	}

	newBlock := &types.Block{
		Header: types.Header{
			ParentHash: bestHash,
			Number:     bestHeader.Number + 1,
			Digest:     createPrimaryBABEDigest(t),
		},
		Body: types.Body{},
	}

	err = bs.AddBlock(newBlock)
	require.NoError(t, err)

	resBlock, err = bs.GetBlockByNumber(newBlock.Header.Number)
	require.NoError(t, err)

	if resBlock.Header.Hash() != newBlock.Header.Hash() {
		t.Fatalf("Fail: got %s expected %s for block %d",
			resBlock.Header.Hash(), newBlock.Header.Hash(), newBlock.Header.Number)
	}
}

func TestFinalization_DeleteBlock(t *testing.T) {
	bs := newTestBlockState(t)
	AddBlocksToState(t, bs, 5, false)

	btBefore := bs.bt.DeepCopy()
	before := bs.bt.GetAllBlocks()
	leaves := bs.Leaves()

	// pick block to finalise
	fin := leaves[len(leaves)-1]
	err := bs.SetFinalisedHash(fin, 1, 1)
	require.NoError(t, err)

	after := bs.bt.GetAllBlocks()

	isIn := func(arr []common.Hash, b common.Hash) bool {
		for _, a := range arr {
			if b == a {
				return true
			}
		}
		return false
	}

	// assert that every block except finalised has been deleted
	for _, b := range before {
		if b == fin {
			continue
		}

		if isIn(after, b) {
			continue
		}

		isFinalised, err := btBefore.IsDescendantOf(b, fin)
		require.NoError(t, err)

		has, err := bs.HasHeader(b)
		require.NoError(t, err)
		if isFinalised {
			require.True(t, has)
		} else {
			require.False(t, has)
		}

		has, err = bs.HasBlockBody(b)
		require.NoError(t, err)
		if isFinalised {
			require.True(t, has)
		} else {
			require.False(t, has)
		}
	}
}

func TestGetHashByNumber(t *testing.T) {
	bs := newTestBlockState(t)

	res, err := bs.GetHashByNumber(0)
	require.NoError(t, err)
	require.Equal(t, bs.genesisHash, res)

	header := &types.Header{
		Number:     1,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: testGenesisHeader.Hash(),
	}

	block := &types.Block{
		Header: *header,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	res, err = bs.GetHashByNumber(1)
	require.NoError(t, err)
	require.Equal(t, header.Hash(), res)
}

func TestAddBlock_WithReOrg(t *testing.T) {
	t.Skip() // TODO: this should be fixed after state refactor PR
	bs := newTestBlockState(t)

	header1a := &types.Header{
		Number:     1,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: testGenesisHeader.Hash(),
	}

	blockbody1a := types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})
	block1a := &types.Block{
		Header: *header1a,
		Body:   *blockbody1a,
	}

	err := bs.AddBlock(block1a)
	require.NoError(t, err)

	block1hash, err := bs.GetHashByNumber(1)
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	header1b := &types.Header{
		Number:         1,
		Digest:         createPrimaryBABEDigest(t),
		ParentHash:     testGenesisHeader.Hash(),
		ExtrinsicsRoot: common.Hash{99},
	}

	block1b := &types.Block{
		Header: *header1b,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block1b)
	require.NoError(t, err)

	// should still be hash 1a since it arrived first
	block1hash, err = bs.GetHashByNumber(1)
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	header2b := &types.Header{
		Number:         2,
		Digest:         createPrimaryBABEDigest(t),
		ParentHash:     header1b.Hash(),
		ExtrinsicsRoot: common.Hash{99},
	}

	block2b := &types.Block{
		Header: *header2b,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block2b)
	require.NoError(t, err)

	// should now be hash 1b since it's on the longer chain
	block1hash, err = bs.GetHashByNumber(1)
	require.NoError(t, err)
	require.Equal(t, header1b.Hash(), block1hash)

	block2hash, err := bs.GetHashByNumber(2)
	require.NoError(t, err)
	require.Equal(t, header2b.Hash(), block2hash)

	header2a := &types.Header{
		Number:     2,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: header1a.Hash(),
	}

	block2a := &types.Block{
		Header: *header2a,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block2a)
	require.NoError(t, err)

	header3a := &types.Header{
		Number:     3,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: header2a.Hash(),
	}

	block3a := &types.Block{
		Header: *header3a,
		Body:   sampleBlockBody,
	}

	err = bs.AddBlock(block3a)
	require.NoError(t, err)

	// should now be hash 1a since it's on the longer chain
	block1hash, err = bs.GetHashByNumber(1)
	require.NoError(t, err)
	require.Equal(t, header1a.Hash(), block1hash)

	// should now be hash 2a since it's on the longer chain
	block2hash, err = bs.GetHashByNumber(2)
	require.NoError(t, err)
	require.Equal(t, header2a.Hash(), block2hash)

	block3hash, err := bs.GetHashByNumber(3)
	require.NoError(t, err)
	require.Equal(t, header3a.Hash(), block3hash)
}

func TestAddBlockToBlockTree(t *testing.T) {
	bs := newTestBlockState(t)

	header := &types.Header{
		Number:     1,
		Digest:     createPrimaryBABEDigest(t),
		ParentHash: testGenesisHeader.Hash(),
	}

	err := bs.setArrivalTime(header.Hash(), time.Now())
	require.NoError(t, err)

	err = bs.AddBlockToBlockTree(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)
	require.Equal(t, bs.BestBlockHash(), header.Hash())
}

func TestNumberIsFinalised(t *testing.T) {
	bs := newTestBlockState(t)
	fin, err := bs.NumberIsFinalised(0)
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(1)
	require.NoError(t, err)
	require.False(t, fin)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	digest2 := types.NewDigest()
	prd, err = types.NewBabeSecondaryPlainPreDigest(0, 100).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest2.Add(*prd)
	require.NoError(t, err)

	header1 := types.Header{
		Number:     1,
		Digest:     digest,
		ParentHash: testGenesisHeader.Hash(),
	}

	header2 := types.Header{
		Number:     2,
		Digest:     digest2,
		ParentHash: header1.Hash(),
	}

	err = bs.AddBlock(&types.Block{
		Header: header1,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	err = bs.AddBlock(&types.Block{
		Header: header2,
		Body:   types.Body{},
	})
	require.NoError(t, err)
	err = bs.SetFinalisedHash(header2.Hash(), 1, 1)
	require.NoError(t, err)

	fin, err = bs.NumberIsFinalised(0)
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(1)
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(2)
	require.NoError(t, err)
	require.True(t, fin)

	fin, err = bs.NumberIsFinalised(100)
	require.NoError(t, err)
	require.False(t, fin)
}

func TestRange(t *testing.T) {
	t.Parallel()

	loadHeaderFromDiskErr := errors.New("[mocked] cannot read, database closed ex")
	testcases := map[string]struct {
		blocksToCreate        int
		blocksToPersistAtDisk int

		newBlockState func(t *testing.T, ctrl *gomock.Controller,
			genesisHeader *types.Header) *BlockState
		wantErr   error
		stringErr string

		expectedHashes   func(hashesCreated []common.Hash) (expected []common.Hash)
		executeRangeCall func(blockState *BlockState,
			hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error)
	}{
		"all_blocks_stored_in_disk": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 128,
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any()).Times(2)

				db := NewInMemoryDB(t)

				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return hashesCreated
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[0]
				endHash := hashesCreated[len(hashesCreated)-1]

				return blockState.Range(startHash, endHash)
			},
		},

		"all_blocks_persisted_in_blocktree": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 0,
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any())

				db := NewInMemoryDB(t)

				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return hashesCreated
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[0]
				endHash := hashesCreated[len(hashesCreated)-1]

				return blockState.Range(startHash, endHash)
			},
		},

		"half_blocks_placed_in_blocktree_half_stored_in_disk": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 64,
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any()).Times(2)

				db := NewInMemoryDB(t)

				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},
			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return hashesCreated
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[0]
				endHash := hashesCreated[len(hashesCreated)-1]

				return blockState.Range(startHash, endHash)
			},
		},

		"error_while_loading_header_from_disk": {
			blocksToCreate:        2,
			blocksToPersistAtDisk: 0,
			wantErr:               loadHeaderFromDiskErr,
			stringErr: "retrieving end hash from database: " +
				"querying database: [mocked] cannot read, database closed ex",
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any())

				db := NewInMemoryDB(t)
				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)

				mockedDb := NewMockBlockStateDatabase(ctrl)
				// cannot assert the exact hash type since the block header
				// hash is generate by the running test case
				mockedDb.EXPECT().Get(gomock.AssignableToTypeOf([]byte{})).
					Return(nil, loadHeaderFromDiskErr)
				blockState.db = mockedDb

				require.NoError(t, err)
				return blockState
			},
			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return nil
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[0]
				endHash := hashesCreated[len(hashesCreated)-1]

				return blockState.Range(startHash, endHash)
			},
		},

		"using_same_hash_as_parameters": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 0,
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any())

				db := NewInMemoryDB(t)
				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return []common.Hash{hashesCreated[0]}
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[0]
				endHash := hashesCreated[0]

				return blockState.Range(startHash, endHash)
			},
		},

		"start_hash_greater_than_end_hash_in_database": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 128,
			wantErr:               ErrStartGreaterThanEnd,
			stringErr:             "start greater than end",
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any()).Times(2)

				db := NewInMemoryDB(t)
				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return nil
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[10]
				endHash := hashesCreated[0]

				return blockState.Range(startHash, endHash)
			},
		},

		"start_hash_greater_than_end_hash_in_memory": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 0,
			wantErr:               blocktree.ErrStartGreaterThanEnd,
			stringErr: "retrieving range from in-memory blocktree: " +
				"getting blocks in range: start greater than end",
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any())

				db := NewInMemoryDB(t)
				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return nil
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[10]
				endHash := hashesCreated[0]

				return blockState.Range(startHash, endHash)
			},
		},

		"start_hash_in_memory_while_end_hash_in_database": {
			blocksToCreate:        128,
			blocksToPersistAtDisk: 64,
			wantErr:               database.ErrNotFound,
			stringErr: "range start should be in database: " +
				"querying database: pebble: not found",
			newBlockState: func(t *testing.T, ctrl *gomock.Controller,
				genesisHeader *types.Header) *BlockState {
				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Any()).Times(2)

				db := NewInMemoryDB(t)
				trieDBTable := database.NewTable(db, "storage")
				trieDB := NewTrieDB(trieDBTable)

				blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
				require.NoError(t, err)

				return blockState
			},

			// execute the Range call. All the values returned must
			// match the hashes we previsouly created
			expectedHashes: func(hashesCreated []common.Hash) (expected []common.Hash) {
				return nil
			},
			executeRangeCall: func(blockState *BlockState,
				hashesCreated []common.Hash) (retrievedHashes []common.Hash, err error) {
				startHash := hashesCreated[len(hashesCreated)-1]
				// since we finalized 64 of 128 blocks the end hash is one of
				// those blocks persisted at database, while start hash is
				// one of those blocks that keeps in memory
				endHash := hashesCreated[0]

				return blockState.Range(startHash, endHash)
			},
		},
	}

	for tname, tt := range testcases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			require.LessOrEqualf(t, tt.blocksToPersistAtDisk, tt.blocksToCreate,
				"blocksToPersistAtDisk should be lower or equal blocksToCreate")

			ctrl := gomock.NewController(t)
			genesisHeader := &types.Header{
				Number:    0,
				StateRoot: trie.EmptyHash,
				Digest:    types.NewDigest(),
			}

			blockState := tt.newBlockState(t, ctrl, genesisHeader)

			testBlockBody := *types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})
			hashesCreated := make([]common.Hash, 0, tt.blocksToCreate)
			previousHeaderHash := genesisHeader.Hash()
			for blockNumber := 1; blockNumber <= tt.blocksToCreate; blockNumber++ {
				currentHeader := &types.Header{
					Number:     uint(blockNumber),
					Digest:     createPrimaryBABEDigest(t),
					ParentHash: previousHeaderHash,
				}

				block := &types.Block{
					Header: *currentHeader,
					Body:   testBlockBody,
				}

				err := blockState.AddBlock(block)
				require.NoError(t, err)

				hashesCreated = append(hashesCreated, currentHeader.Hash())
				previousHeaderHash = currentHeader.Hash()
			}

			if tt.blocksToPersistAtDisk > 0 {
				hashIndexToSetAsFinalized := tt.blocksToPersistAtDisk - 1
				selectedHash := hashesCreated[hashIndexToSetAsFinalized]

				err := blockState.SetFinalisedHash(selectedHash, 0, 0)
				require.NoError(t, err)
			}

			expectedHashes := tt.expectedHashes(hashesCreated)
			retrievedHashes, err := tt.executeRangeCall(blockState, hashesCreated)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.stringErr != "" {
				require.EqualError(t, err, tt.stringErr)
			}

			require.Equal(t, expectedHashes, retrievedHashes)
		})
	}
}

func Test_loadHeaderFromDisk_WithGenesisBlock(t *testing.T) {
	ctrl := gomock.NewController(t)

	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any())

	db := NewInMemoryDB(t)

	genesisHeader := &types.Header{
		Number:    0,
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}

	trieDBTable := database.NewTable(db, "storage")
	trieDB := NewTrieDB(trieDBTable)

	blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
	require.NoError(t, err)

	header, err := blockState.loadHeaderFromDatabase(genesisHeader.Hash())
	require.NoError(t, err)
	require.Equal(t, genesisHeader.Hash(), header.Hash())
}

func Test_GetRuntime_StoreRuntime(t *testing.T) {
	ctrl := gomock.NewController(t)

	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	db := NewInMemoryDB(t)

	genesisHeader := &types.Header{
		Number:    0,
		StateRoot: trie.EmptyHash,
		Digest:    types.NewDigest(),
	}
	genesisHash := genesisHeader.Hash()

	trieDBTable := database.NewTable(db, "storage")
	trieDB := NewTrieDB(trieDBTable)

	blockState, err := NewBlockStateFromGenesis(db, trieDB, genesisHeader, telemetryMock)
	require.NoError(t, err)

	runtimeInstance := NewMockInstance(nil)
	blockState.StoreRuntime(genesisHash, runtimeInstance)

	genesisRuntimeInstance, err := blockState.GetRuntime(genesisHash)
	require.NoError(t, err)
	require.Equal(t, runtimeInstance, genesisRuntimeInstance)

	chain, _ := AddBlocksToState(t, blockState, 5, false)
	for _, hashInChain := range chain {
		genesisRuntimeInstance, err := blockState.GetRuntime(hashInChain.Hash())
		require.NoError(t, err)
		require.Equal(t, runtimeInstance, genesisRuntimeInstance)
	}

	lastElementOnChain := chain[len(chain)-1]
	err = blockState.SetFinalisedHash(lastElementOnChain.Hash(), 1, 0)
	require.NoError(t, err)

	sameRuntimeOnDiffHash, err := blockState.GetRuntime(lastElementOnChain.Hash())
	require.NoError(t, err)
	require.Equal(t, runtimeInstance, sameRuntimeOnDiffHash)
}
