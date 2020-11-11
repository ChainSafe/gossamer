// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

// branch tree randomly
type testBranch struct {
	hash  common.Hash
	depth int
}

// AddBlocksToState adds blocks to a BlockState up to depth, with random branches
func AddBlocksToState(t *testing.T, blockState *BlockState, depth int) ([]*types.Header, []*types.Header) {
	previousHash := blockState.BestBlockHash()

	branches := []testBranch{}
	r := *rand.New(rand.NewSource(rand.Int63()))

	arrivalTime := uint64(1)
	currentChain := []*types.Header{}
	branchChains := []*types.Header{}

	head, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	// create base tree
	startNum := int(head.Number.Int64())
	for i := startNum + 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				StateRoot:  trie.EmptyHash,
			},
			Body: &types.Body{},
		}

		currentChain = append(currentChain, block.Header)

		hash := block.Header.Hash()
		err := blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.Nil(t, err)

		previousHash = hash

		isBranch := r.Intn(2)
		if isBranch == 1 {
			branches = append(branches, testBranch{
				hash:  hash,
				depth: i,
			})
		}

		arrivalTime++
	}

	// create tree branches
	for _, branch := range branches {
		previousHash = branch.hash

		for i := branch.depth; i < depth; i++ {
			block := &types.Block{
				Header: &types.Header{
					ParentHash: previousHash,
					Number:     big.NewInt(int64(i) + 1),
					StateRoot:  trie.EmptyHash,
					Digest:     [][]byte{{byte(i)}},
				},
				Body: &types.Body{},
			}

			branchChains = append(branchChains, block.Header)

			hash := block.Header.Hash()
			err := blockState.AddBlockWithArrivalTime(block, arrivalTime)
			require.Nil(t, err)

			previousHash = hash

			arrivalTime++
		}
	}

	return currentChain, branchChains
}

// AddBlocksToStateWithFixedBranches adds blocks to a BlockState up to depth, with fixed branches
// branches are provided with a map of depth -> # of branches
func AddBlocksToStateWithFixedBranches(t *testing.T, blockState *BlockState, depth int, branches map[int]int, r byte) {
	previousHash := blockState.BestBlockHash()
	tb := []testBranch{}
	arrivalTime := uint64(1)

	head, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	// create base tree
	startNum := int(head.Number.Int64())
	for i := startNum + 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				StateRoot:  trie.EmptyHash,
			},
			Body: &types.Body{},
		}

		hash := block.Header.Hash()
		err := blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.Nil(t, err)

		previousHash = hash

		isBranch := branches[i] > 0
		if isBranch {
			for j := 0; j < branches[i]; j++ {
				tb = append(tb, testBranch{
					hash:  hash,
					depth: i,
				})
			}
		}

		arrivalTime++
	}

	// create tree branches
	for j, branch := range tb {
		previousHash = branch.hash

		for i := branch.depth; i < depth; i++ {
			block := &types.Block{
				Header: &types.Header{
					ParentHash: previousHash,
					Number:     big.NewInt(int64(i)),
					StateRoot:  trie.EmptyHash,
					Digest:     [][]byte{{byte(i), byte(j), r}},
				},
				Body: &types.Body{},
			}

			hash := block.Header.Hash()
			err := blockState.AddBlockWithArrivalTime(block, arrivalTime)
			require.Nil(t, err)

			previousHash = hash

			arrivalTime++
		}
	}
}

func generateBlockWithRandomTrie(t *testing.T, serv *Service) (*types.Block, *TrieState) {
	trieState, err := serv.Storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	// Generate random data for trie state.
	rand := time.Now().UnixNano()
	key := []byte("testKey" + fmt.Sprint(rand))
	value := []byte("testValue" + fmt.Sprint(rand))
	err = trieState.Set(key, value)
	require.NoError(t, err)

	trieStateRoot, err := trieState.Root()
	require.NoError(t, err)

	// Generate a block with the above StateRoot.
	block := &types.Block{
		Header: &types.Header{
			ParentHash: serv.Block.BestBlockHash(),
			Number:     big.NewInt(rand),
			StateRoot:  trieStateRoot,
		},
		Body: types.NewBody([]byte{}),
	}
	return block, trieState
}
