// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

var inc, _ = time.ParseDuration("1s")

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB(t *testing.T) database.Database {
	testDatadirPath := t.TempDir()

	db, err := utils.SetupDatabase(testDatadirPath, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func createPrimaryBABEDigest(t testing.TB) scale.VaryingDataTypeSlice {
	babeDigest := types.NewBabeDigest()
	err := babeDigest.Set(types.BabePrimaryPreDigest{AuthorityIndex: 0})
	require.NoError(t, err)

	bdEnc, err := scale.Marshal(babeDigest)
	require.NoError(t, err)

	digest := types.NewDigest()
	err = digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc,
	})
	require.NoError(t, err)
	return digest
}

// branch tree randomly
type testBranch struct {
	hash  common.Hash
	depth uint
}

// AddBlocksToState adds `depth` number of blocks to the BlockState, optionally with random branches
func AddBlocksToState(t *testing.T, blockState *BlockState, depth uint,
	withBranches bool) ([]*types.Header, []*types.Header) {
	var (
		currentChain, branchChains []*types.Header
		branches                   []testBranch
	)

	arrivalTime := time.Now()
	head, err := blockState.BestBlockHeader()
	require.NoError(t, err)
	previousHash := head.Hash()

	// create base tree
	startNum := head.Number
	for i := startNum + 1; i <= depth+startNum; i++ {
		d := types.NewBabePrimaryPreDigest(0, uint64(i), [32]byte{}, [64]byte{})
		digest := types.NewDigest()
		prd, err := d.ToPreRuntimeDigest()
		require.NoError(t, err)
		err = digest.Add(*prd)
		require.NoError(t, err)

		block := &types.Block{
			Header: types.Header{
				ParentHash: previousHash,
				Number:     i,
				StateRoot:  trie.EmptyHash,
				Digest:     digest,
			},
			Body: types.Body{},
		}
		currentChain = append(currentChain, &block.Header)

		hash := block.Header.Hash()
		err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.NoError(t, err)

		previousHash = hash

		isBranch, err := rand.Int(rand.Reader, big.NewInt(2))
		require.NoError(t, err)
		if isBranch.Cmp(big.NewInt(1)) == 0 {
			branches = append(branches, testBranch{
				hash:  hash,
				depth: i,
			})
		}

		arrivalTime = arrivalTime.Add(inc)
	}

	if !withBranches {
		return currentChain, nil
	}

	// create tree branches
	for _, branch := range branches {
		previousHash = branch.hash

		for i := branch.depth; i < depth; i++ {
			digest := types.NewDigest()
			_ = digest.Add(types.PreRuntimeDigest{
				Data: []byte{byte(i)},
			})

			block := &types.Block{
				Header: types.Header{
					ParentHash: previousHash,
					Number:     i + 1,
					StateRoot:  trie.EmptyHash,
					Digest:     digest,
				},
				Body: types.Body{},
			}

			branchChains = append(branchChains, &block.Header)

			hash := block.Header.Hash()
			err := blockState.AddBlockWithArrivalTime(block, arrivalTime)
			require.NoError(t, err)

			previousHash = hash
			arrivalTime = arrivalTime.Add(inc)
		}
	}

	return currentChain, branchChains
}

// AddBlocksToStateWithFixedBranches adds blocks to a BlockState up to depth, with fixed branches
// branches are provided with a map of depth -> # of branches
func AddBlocksToStateWithFixedBranches(t *testing.T, blockState *BlockState, depth uint, branches map[uint]int) {
	bestBlockHash := blockState.BestBlockHash()
	tb := []testBranch{}
	arrivalTime := time.Now()

	rt, err := blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	head, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	// create base tree
	startNum := head.Number
	for i := startNum + 1; i <= depth; i++ {
		d, err := types.NewBabePrimaryPreDigest(0, uint64(i), [32]byte{}, [64]byte{}).ToPreRuntimeDigest()
		require.NoError(t, err)
		require.NotNil(t, d)
		digest := types.NewDigest()
		_ = digest.Add(*d)

		block := &types.Block{
			Header: types.Header{
				ParentHash: bestBlockHash,
				Number:     i,
				StateRoot:  trie.EmptyHash,
				Digest:     digest,
			},
			Body: types.Body{},
		}

		hash := block.Header.Hash()
		err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.NoError(t, err)

		blockState.StoreRuntime(hash, rt)

		bestBlockHash = hash

		isBranch := branches[i] > 0
		if isBranch {
			for j := 0; j < branches[i]; j++ {
				tb = append(tb, testBranch{
					hash:  hash,
					depth: i,
				})
			}
		}

		arrivalTime = arrivalTime.Add(inc)
	}

	// create tree branches
	for j, branch := range tb {
		bestBlockHash = branch.hash

		for i := branch.depth; i < depth; i++ {
			d, err := types.NewBabePrimaryPreDigest(0, uint64(i+uint(j)+99), [32]byte{}, [64]byte{}).ToPreRuntimeDigest()
			require.NoError(t, err)
			require.NotNil(t, d)
			digest := types.NewDigest()
			_ = digest.Add(*d)

			block := &types.Block{
				Header: types.Header{
					ParentHash: bestBlockHash,
					Number:     i + 1,
					StateRoot:  trie.EmptyHash,
					Digest:     digest,
				},
				Body: types.Body{},
			}

			hash := block.Header.Hash()
			err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
			require.NoError(t, err)

			blockState.StoreRuntime(hash, rt)

			bestBlockHash = hash
			arrivalTime = arrivalTime.Add(inc)
		}
	}
}

func generateBlockWithRandomTrie(t *testing.T, serv *Service,
	parent *common.Hash, bNum uint) (*types.Block, *runtime.TrieState) {
	trieState, err := serv.Storage.TrieState(nil)
	require.NoError(t, err)

	// Generate random data for trie state.
	rand := time.Now().UnixNano()
	key := []byte("testKey" + fmt.Sprint(rand))
	value := []byte("testValue" + fmt.Sprint(rand))
	err = trieState.Put(key, value)
	require.NoError(t, err)

	trieStateRoot, err := trieState.Root()
	require.NoError(t, err)

	if parent == nil {
		bb := serv.Block.BestBlockHash()
		parent = &bb
	}

	body, err := types.NewBodyFromBytes([]byte{})
	require.NoError(t, err)

	block := &types.Block{
		Header: types.Header{
			ParentHash: *parent,
			Number:     bNum,
			StateRoot:  trieStateRoot,
			Digest:     createPrimaryBABEDigest(t),
		},
		Body: *body,
	}
	return block, trieState
}
