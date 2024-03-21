//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"

	"github.com/stretchr/testify/require"
)

func TestUpdateAuthorities(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, _ := newTestService(t, aliceKeyPair)

	err = gs.updateAuthorities()
	require.NoError(t, err)
	require.Equal(t, uint64(0), gs.state.setID)

	next := []Voter{
		{Key: *kr.Alice().Public().(*ed25519.PublicKey), ID: 0},
	}

	err = gs.grandpaState.(*state.GrandpaState).SetNextChange(next, 1)
	require.NoError(t, err)

	_, err = gs.grandpaState.(*state.GrandpaState).IncrementSetID()
	require.NoError(t, err)

	err = gs.updateAuthorities()
	require.NoError(t, err)

	require.Equal(t, uint64(1), gs.state.setID)
	require.Equal(t, next, gs.state.voters)
}

func TestGetDirectVotes(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, _ := newTestService(t, aliceKeyPair)

	voteA := Vote{
		Hash:   common.Hash{0xa},
		Number: 1,
	}

	voteB := Vote{
		Hash:   common.Hash{0xb},
		Number: 1,
	}

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: voteB,
			})
		}
	}

	directVotes := gs.getDirectVotes(prevote)
	require.Equal(t, 2, len(directVotes))
	require.Equal(t, uint64(5), directVotes[voteA])
	require.Equal(t, uint64(4), directVotes[voteB])
}

func TestGetVotesForBlock_NoDescendantVotes(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	// 1/3 of voters equivocate; ie. vote for both blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.Hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForA)

	votesForB, err := gs.getVotesForBlock(voteB.Hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForB)
}

func TestGetVotesForBlock_DescendantVotes(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	a, err := st.Block.GetHeader(leaves[0])
	require.NoError(t, err)

	// A is a descendant of B
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(a.ParentHash, st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 5 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.Hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(3), votesForA)

	// votesForB should be # of votes for A + # of votes for B
	votesForB, err := gs.getVotesForBlock(voteB.Hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForB)

	votesForC, err := gs.getVotesForBlock(voteC.Hash, prevote)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForC)
}

func TestGetPossibleSelectedAncestors_SameAncestor(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with 3 branches all starting at depth 6
	branches := map[uint]int{6: 2}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint32)
	var blocks map[common.Hash]uint32

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote, gs.state.threshold())
		require.NoError(t, err)
	}

	expected, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b, c) with >=2/3 votes,
	// which is the node at depth 6.
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expected])
}

func TestGetPossibleSelectedAncestors_VaryingAncestor(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := map[uint]int{6: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint32)
	var blocks map[common.Hash]uint32

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote, gs.state.threshold())
		require.NoError(t, err)
	}

	expectedAt6, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >2/3 votes,
	// which is the nodes at depth 6.
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expectedAt6])
}

func TestGetPossibleSelectedAncestors_VaryingAncestor_MoreBranches(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with 2 branches starting at depth 6 and 1 branch starting at depth 7,
	branches := map[uint]int{6: 2, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
	require.Equal(t, 4, len(leaves))

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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else if i < 8 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteD,
			})
		}
	}

	votes := gs.getVotes(prevote)
	prevoted := make(map[common.Hash]uint32)
	var blocks map[common.Hash]uint32

	for _, curr := range leaves {
		blocks, err = gs.getPossibleSelectedAncestors(votes, curr, prevoted, prevote, gs.state.threshold())
		require.NoError(t, err)
	}

	expectedAt6, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >2/3 votes,
	// which is the node at depth 6.
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expectedAt6])
}

func TestGetPossibleSelectedBlocks_OneBlock(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 7 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote, gs.state.threshold())
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, voteA.Number, blocks[voteA.Hash])
}

func TestGetPossibleSelectedBlocks_EqualVotes_SameAncestor(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with 3 branches all starting at depth 6
	branches := map[uint]int{6: 2}

	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote, gs.state.threshold())
	require.NoError(t, err)

	expected, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b, c)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expected])
}

func TestGetPossibleSelectedBlocks_EqualVotes_VaryingAncestor(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := map[uint]int{6: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote, gs.state.threshold())
	require.NoError(t, err)

	expectedAt6, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	// this should return the highest common ancestor of (a, b) and (b, c) with >2/3 votes,
	// which is the node at depth 6.
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expectedAt6])
}

func TestGetPossibleSelectedBlocks_OneThirdEquivocating(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	// 1/3 of voters equivocate; ie. vote for both blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	svA := &SignedVote{
		Vote: *voteA,
	}
	svB := &SignedVote{
		Vote: *voteB,
	}

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.prevotes.Store(voter, svA)
		} else if i < 6 {
			gs.prevotes.Store(voter, svB)
		} else {
			gs.pvEquivocations[voter] = []*SignedVote{svA, svB}
		}
	}

	expectedAt6, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	blocks, err := gs.getPossibleSelectedBlocks(prevote, gs.state.threshold())
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, uint32(6), blocks[expectedAt6])
}

func TestGetPossibleSelectedBlocks_MoreThanOneThirdEquivocating(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	// this tests a byzantine case where >1/3 of voters equivocate; ie. vote for multiple blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	svA := &SignedVote{
		Vote: *voteA,
	}
	svB := &SignedVote{
		Vote: *voteB,
	}

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 2 {
			// 2 votes for A
			gs.prevotes.Store(voter, svA)
		} else if i < 4 {
			// 2 votes for B
			gs.prevotes.Store(voter, svB)
		} else if i < 5 {
			// 1 vote for C
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		} else {
			// 4 equivocators
			gs.pvEquivocations[voter] = []*SignedVote{svA, svB}
		}
	}

	blocks, err := gs.getPossibleSelectedBlocks(prevote, gs.state.threshold())
	require.NoError(t, err)
	require.Equal(t, 2, len(blocks))
}

func TestGetPreVotedBlock_OneBlock(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 7 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, *voteA, block)
}

func TestGetPreVotedBlock_MultipleCandidates(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with branches starting at depth 6 and another branch starting at depth 7
	branches := map[uint]int{6: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	// expected block is that with the highest number ie. at depth 7
	expected, err := st.Block.GetHashByNumber(6)
	require.NoError(t, err)

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, expected, block.Hash)
	require.Equal(t, uint32(6), block.Number)
}

func TestGetPreVotedBlock_EvenMoreCandidates(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with 6 total branches, one each from depth 3 to 7
	branches := map[uint]int{3: 1, 4: 1, 5: 1, 6: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)

	leaves := gs.blockState.(*state.BlockState).Leaves()
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
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 4 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		} else if i < 7 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteD,
			})
		} else if i < 8 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteE,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteF,
			})
		}
	}

	// expected block is at depth 4
	expected, err := st.Block.GetHashByNumber(4)
	require.NoError(t, err)

	block, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, expected, block.Hash)
	require.Equal(t, uint32(4), block.Number)
}

func TestGetBestFinalCandidate_OneBlock(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	// this tests the case when the prevoted block and the precommited block are the same
	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 7 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
			gs.precommits.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
			gs.precommits.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, voteA, bfc)
}

func TestGetBestFinalCandidate_NoPrecommit(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	// this tests the case when no blocks have >=2/3 precommit votes
	// it should return the prevoted block
	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 7 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
			gs.precommits.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, voteA, bfc)
}

func TestGetBestFinalCandidate_PrecommitOnAnotherChain(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	// this tests the case when the precommited block is on another chain than the prevoted block
	// this should return their highest common ancestor
	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
			gs.precommits.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
			gs.precommits.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		}
	}

	pred, err := st.Block.LowestCommonAncestor(voteA.Hash, voteB.Hash)
	require.NoError(t, err)

	bfc, err := gs.getBestFinalCandidate()
	require.NoError(t, err)
	require.Equal(t, pred, bfc.Hash)
}

func TestDeterminePreVote_NoPrimaryPreVote(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	state.AddBlocksToState(t, st.Block, 3, false)
	pv, err := gs.determinePreVote()
	require.NoError(t, err)

	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, header.Hash(), pv.Hash)
}

func TestDeterminePreVote_WithPrimaryPreVote(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	state.AddBlocksToState(t, st.Block, 3, false)
	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 1, false)

	derivePrimary := gs.derivePrimary()
	primary := derivePrimary.PublicKeyBytes()
	gs.prevotes.Store(primary, &SignedVote{
		Vote: *NewVoteFromHeader(header),
	})

	pv, err := gs.determinePreVote()
	require.NoError(t, err)
	p, has := gs.prevotes.Load(primary)
	require.True(t, has)
	require.Equal(t, pv, &p.(*SignedVote).Vote)
}

func TestDeterminePreVote_WithInvalidPrimaryPreVote(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	state.AddBlocksToState(t, st.Block, 3, false)
	header, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	derivePrimary := gs.derivePrimary()
	primary := derivePrimary.PublicKeyBytes()
	gs.prevotes.Store(primary, &SignedVote{
		Vote: *NewVoteFromHeader(header),
	})

	state.AddBlocksToState(t, st.Block, 5, false)
	gs.head, err = st.Block.BestBlockHeader()
	require.NoError(t, err)

	pv, err := gs.determinePreVote()
	require.NoError(t, err)
	require.Equal(t, gs.head.Hash(), pv.Hash)
}

func TestGetGrandpaGHOST_CommonAncestor(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 4 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 5 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		}
	}

	pred, err := gs.blockState.LowestCommonAncestor(voteA.Hash, voteB.Hash)
	require.NoError(t, err)

	block, err := gs.getGrandpaGHOST()
	require.NoError(t, err)
	require.Equal(t, pred, block.Hash)
}

func TestGetGrandpaGHOST_MultipleCandidates(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// this creates a tree with branches starting at depth 3 and another branch starting at depth 7
	branches := map[uint]int{3: 1, 7: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()
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

		if i < 1 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteA,
			})
		} else if i < 2 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteB,
			})
		} else if i < 3 {
			gs.prevotes.Store(voter, &SignedVote{
				Vote: *voteC,
			})
		}
	}

	// expected block is that with the most votes ie. block 3
	expected, err := st.Block.GetHashByNumber(3)
	require.NoError(t, err)

	block, err := gs.getGrandpaGHOST()
	require.NoError(t, err)
	require.Equal(t, expected, block.Hash)
	require.Equal(t, uint32(3), block.Number)

	pv, err := gs.getPreVotedBlock()
	require.NoError(t, err)
	require.Equal(t, block, pv)
}

func TestGrandpaServiceCreateJustification_ShouldCountEquivocatoryVotes(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	// setup granpda service
	gs, st := newTestService(t, aliceKeyPair)
	now := time.Unix(1000, 0)

	const previousBlocksToAdd = 9
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number

	// create fake authorities
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	fakeAuthorities := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Eve().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),  // equivocatory
		ed25519Keyring.Dave().(*ed25519.Keypair), // equivocatory
	}

	equivocatories := make(map[ed25519.PublicKeyBytes][]*types.GrandpaSignedVote)
	prevotes := &sync.Map{}

	var totalLegitVotes int
	// voting on
	for _, v := range fakeAuthorities {
		vote := &SignedVote{
			AuthorityID: v.Public().(*ed25519.PublicKey).AsBytes(),
			Vote: types.GrandpaVote{
				Hash:   bfcHash,
				Number: uint32(bfcNumber),
			},
		}

		// to simulate the real world:
		// if the voter already has voted, then we remove
		// previous vote and add it on the equivocatories with the new vote
		previous, ok := prevotes.Load(vote.AuthorityID)
		if !ok {
			prevotes.Store(vote.AuthorityID, vote)
			totalLegitVotes++
		} else {
			prevotes.Delete(vote.AuthorityID)
			equivocatories[vote.AuthorityID] = []*types.GrandpaSignedVote{
				previous.(*types.GrandpaSignedVote),
				vote,
			}
			totalLegitVotes--
		}
	}

	gs.pvEquivocations = equivocatories
	gs.prevotes = prevotes

	justifications, err := gs.createJustification(bfcHash, prevote)
	require.NoError(t, err)

	var totalEqvVotes int
	// checks if the created justification contains all equivocatories votes
	for eqvPubKeyBytes, expectedVotes := range equivocatories {
		votesOnJustification := 0

		for _, justification := range justifications {
			if justification.AuthorityID == eqvPubKeyBytes {
				votesOnJustification++
			}
		}

		require.Equal(t, len(expectedVotes), votesOnJustification)
		totalEqvVotes += votesOnJustification
	}

	require.Len(t, justifications, totalLegitVotes+totalEqvVotes)
}

// addBlocksToState test helps adding previous blocks
func addBlocksToState(t *testing.T, blockState *state.BlockState, depth uint) {
	t.Helper()

	previousHash := blockState.BestBlockHash()
	rt, err := blockState.GetRuntime(previousHash)
	require.NoError(t, err)

	head, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	startNum := head.Number

	for i := startNum + 1; i <= depth; i++ {
		arrivalTime := time.Now()

		d, err := types.NewBabePrimaryPreDigest(0, uint64(i), [32]byte{}, [64]byte{}).ToPreRuntimeDigest()
		require.NoError(t, err)
		require.NotNil(t, d)
		digest := types.NewDigest()
		err = digest.Add(*d)
		require.NoError(t, err)

		block := &types.Block{
			Header: types.Header{
				ParentHash: previousHash,
				Number:     i,
				StateRoot:  inmemory_trie.EmptyHash,
				Digest:     digest,
			},
			Body: types.Body{},
		}

		hash := block.Header.Hash()
		err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.NoError(t, err)

		blockState.StoreRuntime(hash, rt)
		previousHash = hash
	}
}

func addBlocksAndReturnTheLastOne(
	t *testing.T, blockState *state.BlockState,
	depth uint,
	lastBlockArrivalTime time.Time,
) *types.Block {
	t.Helper()
	addBlocksToState(t, blockState, depth)

	// create a new fake block to fake authorities commit on
	previousHash := blockState.BestBlockHash()
	previousHead, err := blockState.BestBlockHeader()
	require.NoError(t, err)

	bfcNumber := previousHead.Number + 1

	d, err := types.NewBabePrimaryPreDigest(0, uint64(bfcNumber), [32]byte{}, [64]byte{}).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, d)
	digest := types.NewDigest()
	err = digest.Add(*d)
	require.NoError(t, err)

	bfcBlock := &types.Block{
		Header: types.Header{
			ParentHash: previousHash,
			Number:     bfcNumber,
			StateRoot:  inmemory_trie.EmptyHash,
			Digest:     digest,
		},
		Body: types.Body{},
	}

	err = blockState.AddBlockWithArrivalTime(bfcBlock, lastBlockArrivalTime)
	require.NoError(t, err)

	return bfcBlock
}
