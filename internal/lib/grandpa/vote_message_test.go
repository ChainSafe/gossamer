// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
)

func TestCheckForEquivocation_NoEquivocation(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	for _, v := range voters {
		equivocated := gs.checkForEquivocation(&v, &SignedVote{
			Vote: *vote,
		}, prevote)
		require.False(t, equivocated)
	}
}

func TestCheckForEquivocation_WithEquivocation(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := make(map[int]int)
	branches[6] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches, 0)
	leaves := gs.blockState.Leaves()

	vote1, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)

	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *vote1,
	})

	vote2, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	equivocated := gs.checkForEquivocation(&voter, &SignedVote{
		Vote: *vote2,
	}, prevote)
	require.True(t, equivocated)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))
	require.Equal(t, 2, len(gs.pvEquivocations[voter.Key.AsBytes()]))
}

func TestCheckForEquivocation_WithExistingEquivocation(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := make(map[int]int)
	branches[6] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches, 0)
	leaves := gs.blockState.Leaves()

	vote1, err := NewVoteFromHash(leaves[1], gs.blockState)
	require.NoError(t, err)

	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *vote1,
	})

	vote2, err := NewVoteFromHash(leaves[0], gs.blockState)
	require.NoError(t, err)

	equivocated := gs.checkForEquivocation(&voter, &SignedVote{
		Vote: *vote2,
	}, prevote)
	require.True(t, equivocated)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))

	vote3 := vote1

	equivocated = gs.checkForEquivocation(&voter, &SignedVote{
		Vote: *vote3,
	}, prevote)
	require.True(t, equivocated)

	require.Equal(t, 0, gs.lenVotes(prevote))
	require.Equal(t, 1, len(gs.pvEquivocations))
	require.Equal(t, 3, len(gs.pvEquivocations[voter.Key.AsBytes()]))
}

func TestValidateMessage_Valid(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
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

	vote, err := gs.validateMessage("", msg)
	require.NoError(t, err)
	require.Equal(t, h.Hash(), vote.Hash)
}

func TestValidateMessage_InvalidSignature(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
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

	_, err = gs.validateMessage("", msg)
	require.Equal(t, err, ErrInvalidSignature)
}

func TestValidateMessage_SetIDMismatch(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
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

	_, err = gs.validateMessage("", msg)
	require.Equal(t, err, ErrSetIDMismatch)
}

func TestValidateMessage_Equivocation(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	branches := make(map[int]int)
	branches[6] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches, 0)
	leaves := gs.blockState.Leaves()

	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	voter := voters[0]

	gs.prevotes.Store(voter.Key.AsBytes(), &SignedVote{
		Vote: *voteA,
	})

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(voteB, prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	_, err = gs.validateMessage("", msg)
	require.Equal(t, ErrEquivocation, err, gs.prevotes)
}

func TestValidateMessage_BlockDoesNotExist(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3, false)
	gs.tracker = newTracker(st.Block, gs.messageHandler)

	fake := &types.Header{
		Number: big.NewInt(77),
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(fake), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	_, err = gs.validateMessage("", msg)
	require.Equal(t, err, ErrBlockDoesNotExist)
}

func TestValidateMessage_IsNotDescendant(t *testing.T) {
	st := newTestState(t)
	net := newTestNetwork(t)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	cfg := &Config{
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: NewMockDigestHandler(),
		Voters:        voters,
		Keypair:       kr.Bob().(*ed25519.Keypair),
		Network:       net,
		Interval:      time.Second,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	branches := make(map[int]int)
	branches[6] = 1
	state.AddBlocksToStateWithFixedBranches(t, st.Block, 8, branches, 0)
	leaves := gs.blockState.Leaves()

	gs.head, err = gs.blockState.GetHeader(leaves[0])
	require.NoError(t, err)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	vote, err := NewVoteFromHash(leaves[1], gs.blockState)
	require.NoError(t, err)

	_, msg, err := gs.createSignedVoteAndVoteMessage(vote, prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	_, err = gs.validateMessage("", msg)
	require.Equal(t, errInvalidVoteBlock, err, gs.prevotes)
}
