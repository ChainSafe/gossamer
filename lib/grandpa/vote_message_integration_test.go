//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
)

func TestCheckForEquivocation_NoEquivocation(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	for _, v := range newTestVoters(t) {
		err = gs.checkAndReportEquivocation(&v, &SignedVote{
			Vote: *vote,
		}, prevote)
		require.NoError(t, err)
	}
}

func TestCheckForEquivocation_WithEquivocation(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	vote1, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)

	voters := newTestVoters(t)
	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *vote1,
	})

	vote2, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	err = gs.checkAndReportEquivocation(&voter, &SignedVote{
		Vote: *vote2,
	}, prevote)
	require.ErrorIs(t, err, ErrEquivocation)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))
	require.Equal(t, 2, len(gs.pvEquivocations[voter.Key.AsBytes()]))
}

func TestCheckForEquivocation_WithExistingEquivocation(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	vote1, err := NewVoteFromHash(leaves[1], gs.blockState)
	require.NoError(t, err)

	voters := newTestVoters(t)
	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *vote1,
	})

	vote2, err := NewVoteFromHash(leaves[0], gs.blockState)
	require.NoError(t, err)

	err = gs.checkAndReportEquivocation(&voter, &SignedVote{
		Vote: *vote2,
	}, prevote)
	require.ErrorIs(t, err, ErrEquivocation)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))

	vote3 := vote1

	err = gs.checkAndReportEquivocation(&voter, &SignedVote{
		Vote: *vote3,
	}, prevote)
	require.ErrorIs(t, err, ErrEquivocation)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))
	require.Equal(t, 3, len(gs.pvEquivocations[voter.Key.AsBytes()]))
}

func TestValidateMessage_Valid(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(h), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	vote, err := gs.validateVoteMessage("", msg)
	require.NoError(t, err)
	require.Equal(t, h.Hash(), vote.Hash)
}

func TestValidateMessage_InvalidSignature(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(h), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	msg.Message.Signature[63] = 0

	const expectedErrString = "validating message signature: signature is not valid"
	_, err = gs.validateVoteMessage("", msg)
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.EqualError(t, err, expectedErrString)
}

func TestValidateMessage_SetIDMismatch(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(h), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	gs.state.setID = 1

	_, err = gs.validateVoteMessage("", msg)
	require.Equal(t, err, ErrSetIDMismatch)
}

func TestValidateMessage_Equivocation(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	voters := newTestVoters(t)
	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *voteA,
	})

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(voteB, prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	_, err = gs.validateVoteMessage("", msg)
	require.ErrorIs(t, err, ErrEquivocation)
}

func TestValidateMessage_BlockDoesNotExist(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)
	gs.tracker = newTracker(st.Block, gs.messageHandler)

	fake := &types.Header{
		Number: 77,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(fake), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	const expectedErrString = "validating vote: block does not exist"
	_, err = gs.validateVoteMessage("", msg)
	require.ErrorIs(t, err, ErrBlockDoesNotExist)
	require.EqualError(t, err, expectedErrString)
}

func TestValidateMessage_IsNotDescendant(t *testing.T) {
	t.Parallel()
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Network:      net,
		Interval:     time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	branches := map[uint]int{6: 1}
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches)
	leaves := gs.blockState.(*state.BlockState).Leaves()

	gs.head, err = gs.blockState.GetHeader(leaves[0])
	require.NoError(t, err)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	vote, err := NewVoteFromHash(leaves[1], gs.blockState)
	require.NoError(t, err)

	_, msg, err := gs.createSignedVoteAndVoteMessage(vote, prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	const expectedErrString = "validating vote: block in vote is not descendant of previously finalised block"
	_, err = gs.validateVoteMessage("", msg)

	require.ErrorIs(t, err, errVoteBlockMismatch)
	require.EqualError(t, err, expectedErrString)
}
