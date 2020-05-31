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

package grandpa

import (
	"math/big"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

// testGenesisHeader is a test block header
var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

func newTestState(t *testing.T) *state.Service {
	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err := stateSrvc.Initialize(genesisData, testGenesisHeader, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	return stateSrvc
}

func newTestVoters(t *testing.T) []*Voter {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	voters := []*Voter{}
	for i, k := range kr.Keys {
		voters = append(voters, &Voter{
			key: k.Public().(*ed25519.PublicKey),
			id:  uint64(i),
		})
	}

	return voters
}

func TestGetDirectVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	voteA := &Vote{
		hash:   common.Hash{0xa},
		number: 1,
	}

	voteB := &Vote{
		hash:   common.Hash{0xb},
		number: 1,
	}

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
		}
	}

	directVotes := gs.getDirectVotes(prevote)
	require.Equal(t, 2, len(directVotes))
	require.Equal(t, uint64(5), directVotes[*voteA])
	require.Equal(t, uint64(4), directVotes[*voteB])
}

func TestGetVotesForBlock_NoDescendantVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	// 1/3 of voters equivocate; ie. vote for both blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForA)

	votesForB, err := gs.getVotesForBlock(voteB.hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForB)
}

func TestGetVotesForBlock_DescendantVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var branches []*types.Header
	var chain []*types.Header

	for {
		chain, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	// A is a descendant of B
	voteA := NewVoteFromHeader(chain[7])
	voteB := NewVoteFromHeader(chain[5])
	voteC := NewVoteFromHeader(branches[0])

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 5 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(3), votesForA)

	// votesForB should be # of votes for A + # of votes for B
	votesForB, err := gs.getVotesForBlock(voteB.hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForB)

	votesForC, err := gs.getVotesForBlock(voteC.hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForC)
}

func TestGetPossibleSelectedAncestors_SameAncestor(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with 3 branches all starting at depth 6
	branches := make(map[int]int)
	branches[6] = 2
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 3, len(leaves))

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint64)
	var blocks map[common.Hash]uint64

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote)
		require.NoError(t, err)
	}

	expected, err := common.HexToHash("0x32ed981734053dc565a1e224137d751f24917a1cb2aeea56fd44a06629550a23")
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b, c) with >=2/3 votes,
	// which is the node at depth 6.
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint64(6), blocks[expected])
}

func TestGetPossibleSelectedAncestors_VaryingAncestor(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := make(map[int]int)
	branches[6] = 1
	branches[7] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 3, len(leaves))

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint64)
	var blocks map[common.Hash]uint64

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote)
		require.NoError(t, err)
	}

	expectedAt6, err := common.HexToHash("0x32ed981734053dc565a1e224137d751f24917a1cb2aeea56fd44a06629550a23")
	require.NoError(t, err)

	expectedAt7, err := common.HexToHash("0x57508d4d2c5b01e6bd50dacee5d14979a6f23e41d4b4eb6464a8a29015549847")
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >=2/3 votes,
	// which are the nodes at depth 6 and 7.
	require.Equal(t, 2, len(blocks))
	require.Equal(t, uint64(6), blocks[expectedAt6])
	require.Equal(t, uint64(7), blocks[expectedAt7])
}

func TestGetPossibleSelectedAncestors_VaryingAncestor_MoreBranches(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with 1 branch starting at depth 6 and 2 branches starting at depth 7,
	branches := make(map[int]int)
	branches[6] = 1
	branches[7] = 2
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 4, len(leaves))

	t.Log(st.Block.BlocktreeAsString())

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)
	voteD, err := NewVoteFromHash(leaves[3], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else if i < 8 {
			gs.prevotes[voter] = voteC
		} else {
			gs.prevotes[voter] = voteD
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint64)
	var blocks map[common.Hash]uint64

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote)
		require.NoError(t, err)
	}

	expectedAt6, err := common.HexToHash("0x32ed981734053dc565a1e224137d751f24917a1cb2aeea56fd44a06629550a23")
	require.NoError(t, err)

	expectedAt7, err := common.HexToHash("0x57508d4d2c5b01e6bd50dacee5d14979a6f23e41d4b4eb6464a8a29015549847")
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >=2/3 votes,
	// which are the nodes at depth 6 and 7.
	require.Equal(t, 2, len(blocks))
	require.Equal(t, uint64(6), blocks[expectedAt6])
	require.Equal(t, uint64(7), blocks[expectedAt7])
}

func TestGetPossibleSelectedBlocks_OneBlock(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote)
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, voteA.number, blocks[voteA.hash])
}

func TestGetPossibleSelectedBlocks_EqualVotes_SameAncestor(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with 3 branches all starting at depth 6
	branches := make(map[int]int)
	branches[6] = 2

	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.Leaves()
	require.Equal(t, 3, len(leaves))

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote)
	require.NoError(t, err)

	expected, err := common.HexToHash("0x32ed981734053dc565a1e224137d751f24917a1cb2aeea56fd44a06629550a23")
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b, c)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint64(6), blocks[expected])
}

func TestGetPossibleSelectedBlocks_EqualVotes_VaryingAncestor(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := make(map[int]int)
	branches[6] = 1
	branches[7] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 3, len(leaves))

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote)
	require.NoError(t, err)

	expectedAt6, err := common.HexToHash("0x32ed981734053dc565a1e224137d751f24917a1cb2aeea56fd44a06629550a23")
	require.NoError(t, err)

	expectedAt7, err := common.HexToHash("0x57508d4d2c5b01e6bd50dacee5d14979a6f23e41d4b4eb6464a8a29015549847")
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >=2/3 votes,
	// which are the nodes at depth 6 and 7.
	require.Equal(t, 2, len(blocks))
	require.Equal(t, uint64(6), blocks[expectedAt6])
	require.Equal(t, uint64(7), blocks[expectedAt7])
}

func TestGetPossibleSelectedBlocks_OneThirdEquivocating(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	// 1/3 of voters equivocate; ie. vote for both blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.pvEquivocations[voter] = []*Vote{voteA, voteB}
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote)
	require.NoError(t, err)
	require.Equal(t, 2, len(blocks))
}

func TestGetPossibleSelectedBlocks_MoreThanOneThirdEquivocating(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 2 {
			break
		}
	}

	// this tests a byzantine case where >1/3 of voters equivocate; ie. vote for multiple blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 2 {
			// 2 votes for A
			gs.prevotes[voter] = voteA
		} else if i < 4 {
			// 2 votes for B
			gs.prevotes[voter] = voteB
		} else if i < 5 {
			// 1 vote for C
			gs.prevotes[voter] = voteC
		} else {
			// 4 equivocators
			gs.pvEquivocations[voter] = []*Vote{voteA, voteB}
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote)
	require.NoError(t, err)
	require.Equal(t, 2, len(blocks))
}

func TestGetPreVotedBlock_OneBlock(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
		}
	}

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, *voteA, block)
}

func TestGetPreVotedBlock_MultipleCandidates(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := make(map[int]int)
	branches[6] = 1
	branches[7] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 3, len(leaves))

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes[voter] = voteA
		} else if i < 6 {
			gs.prevotes[voter] = voteB
		} else {
			gs.prevotes[voter] = voteC
		}
	}

	// expected block is that with the highest number ie. at depth 7
	expected, err := common.HexToHash("0x57508d4d2c5b01e6bd50dacee5d14979a6f23e41d4b4eb6464a8a29015549847")
	require.NoError(t, err)

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, expected, block.hash)
	require.Equal(t, uint64(7), block.number)
}

func TestGetPreVotedBlock_EvenMoreCandidates(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// this creates a tree with 6 total branches, one each from depth 3 to 7
	branches := make(map[int]int)
	branches[3] = 1
	branches[4] = 1
	branches[5] = 1
	branches[6] = 1
	branches[7] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.Leaves()
	require.Equal(t, 6, len(leaves))

	sort.Slice(leaves, func(i, j int) bool {
		return leaves[i][0] < leaves[j][0]
	})

	// voters vote for a blocks on a different chains
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)
	voteD, err := NewVoteFromHash(leaves[3], st.Block)
	require.NoError(t, err)
	voteE, err := NewVoteFromHash(leaves[4], st.Block)
	require.NoError(t, err)
	voteF, err := NewVoteFromHash(leaves[5], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 2 {
			gs.prevotes[voter] = voteA
		} else if i < 4 {
			gs.prevotes[voter] = voteB
		} else if i < 6 {
			gs.prevotes[voter] = voteC
		} else if i < 7 {
			gs.prevotes[voter] = voteD
		} else if i < 8 {
			gs.prevotes[voter] = voteE
		} else {
			gs.prevotes[voter] = voteF
		}
	}

	t.Log(st.Block.BlocktreeAsString())

	// expected block is at depth 5
	expected, err := common.HexToHash("0xd01a209a130af98b3375cf9a571e92b7c0fc8b61dee7917f852a673fcb57ac19")
	require.NoError(t, err)

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, expected, block.hash)
	require.Equal(t, uint64(5), block.number)
}

func TestIsCompletable(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
		}
	}

	completable, err := gs.isCompletable()
	require.NoError(t, err)
	require.True(t, completable)
}

func TestFindParentWithNumber(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	// no branches needed
	branches := make(map[int]int)
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.Leaves()

	v, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)

	p, err := gs.findParentWithNumber(v, 1)
	require.NoError(t, err)
	t.Log(st.Block.BlocktreeAsString())

	expected, err := st.Block.GetBlockByNumber(big.NewInt(1))
	require.NoError(t, err)

	require.Equal(t, expected.Header.Hash(), p.hash)
}

func TestGetBestFinalCandidate_OneBlock(t *testing.T) {
	// this tests the case when the prevoted block and the precommited block are the same
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
			gs.precommits[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteB
		}
	}

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, voteA, bfc)
}

func TestGetBestFinalCandidate_PrecommitAncestor(t *testing.T) {
	// this tests the case when the highest precommited block is an ancestor of the prevoted block
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash

	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	// in precommit round, 2/3 voters will vote for ancestor of A
	voteC, err := gs.findParentWithNumber(voteA, 6)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
			gs.precommits[voter] = voteC
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteB
		}
	}

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, voteC, bfc)
}

func TestGetBestFinalCandidate_NoPrecommit(t *testing.T) {
	// this tests the case when no blocks have >=2/3 precommit votes
	// it should return the prevoted block
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash

	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteB
		}
	}

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, voteA, bfc)
}

func TestGetBestFinalCandidate_PrecommitOnAnotherChain(t *testing.T) {
	// this tests the case when the precommited block is on another chain than the prevoted block
	// this should return their highest common ancestor
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash

	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
			gs.precommits[voter] = voteB
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteA
		}
	}

	pred, err := st.Block.HighestCommonAncestor(voteA.hash, voteB.hash)
	require.NoError(t, err)

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, pred, bfc.hash)
}

func TestDeterminePreVote_NoPrimaryPreVote(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	state.AddBlocksToState(t, st.Block, 3)
	pv, err := gs.determinePreVote()
	require.NoError(t, err)

	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, header.Hash(), pv.hash)
}

func TestDeterminePreVote_WithPrimaryPreVote(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	state.AddBlocksToState(t, st.Block, 3)
	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 1)

	primary := gs.derivePrimary().PublicKeyBytes()
	gs.prevotes[primary] = NewVoteFromHeader(header)

	pv, err := gs.determinePreVote()
	require.NoError(t, err)
	require.Equal(t, gs.prevotes[primary], pv)
}

func TestDeterminePreVote_WithInvalidPrimaryPreVote(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	state.AddBlocksToState(t, st.Block, 3)
	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	primary := gs.derivePrimary().PublicKeyBytes()
	gs.prevotes[primary] = NewVoteFromHeader(header)

	state.AddBlocksToState(t, st.Block, 5)
	gs.head, err = st.Block.BestBlockHeader()
	require.NoError(t, err)

	pv, err := gs.determinePreVote()
	require.NoError(t, err)
	require.Equal(t, gs.head.Hash(), pv.hash)
}

func TestIsFinalizable_True(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 3)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
			gs.precommits[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteB
		}
	}

	finalizable, err := gs.isFinalizable()
	require.NoError(t, err)
	require.True(t, finalizable)
}

func TestIsFinalizable_False(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 3)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes[voter] = voteA
			gs.precommits[voter] = voteA
		} else {
			gs.prevotes[voter] = voteB
			gs.precommits[voter] = voteB
		}
	}

	// previous round has finalized block # higher than current, so round is not finalizable
	gs.state.round = 1
	gs.bestFinalCandidate[0] = &Vote{
		number: 4,
	}

	finalizable, err := gs.isFinalizable()
	require.NoError(t, err)
	require.False(t, finalizable)
}
