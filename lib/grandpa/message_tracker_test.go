// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
)

func TestMessageTracker_ValidateMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, _, _, _ := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	fake := &types.Header{
		Number: 77,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(fake), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	expected := &networkVoteMessage{
		msg: msg,
	}

	_, err = gs.validateVoteMessage("", msg)
	require.Equal(t, err, ErrBlockDoesNotExist)
	require.Equal(t, expected, gs.tracker.voteMessages[fake.Hash()][kr.Alice().Public().(*ed25519.PublicKey).AsBytes()])
}

func TestMessageTracker_SendMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, in, _, _ := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)
	gs.tracker.start()
	defer gs.tracker.stop()

	parent, err := gs.blockState.BestBlockHeader()
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	next := &types.Header{
		ParentHash: parent.Hash(),
		Number:     4,
		Digest:     digest,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(next), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	expected := &networkVoteMessage{
		msg: msg,
	}

	_, err = gs.validateVoteMessage("", msg)
	require.Equal(t, err, ErrBlockDoesNotExist)
	require.Equal(t, expected, gs.tracker.voteMessages[next.Hash()][kr.Alice().Public().(*ed25519.PublicKey).AsBytes()])

	err = gs.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *next,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	const testTimeout = time.Second
	select {
	case v := <-in:
		require.Equal(t, msg, v.msg)
	case <-time.After(testTimeout):
		t.Errorf("did not receive vote message %v", msg)
	}
}

func TestMessageTracker_ProcessMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, _, _, _ := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	err = gs.Start()
	require.NoError(t, err)

	time.Sleep(time.Second) // wait for round to initiate

	parent, err := gs.blockState.BestBlockHeader()
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	next := &types.Header{
		ParentHash: parent.Hash(),
		Number:     4,
		Digest:     digest,
	}

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(next), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	expected := &networkVoteMessage{
		msg: msg,
	}

	_, err = gs.validateVoteMessage("", msg)
	require.Equal(t, ErrBlockDoesNotExist, err)
	require.Equal(t, expected, gs.tracker.voteMessages[next.Hash()][kr.Alice().Public().(*ed25519.PublicKey).AsBytes()])

	err = gs.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *next,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	time.Sleep(time.Second)
	expectedVote := &Vote{
		Hash:   msg.Message.BlockHash,
		Number: msg.Message.Number,
	}
	pv, has := gs.prevotes.Load(kr.Alice().Public().(*ed25519.PublicKey).AsBytes())
	require.True(t, has)
	require.Equal(t, expectedVote, &pv.(*SignedVote).Vote, gs.tracker.voteMessages)
}

func TestMessageTracker_MapInsideMap(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, _, _, _ := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 3, false)
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	header := &types.Header{
		Number: 77,
	}

	hash := header.Hash()
	_, ok := gs.tracker.voteMessages[hash]
	require.False(t, ok)

	gs.keypair = kr.Alice().(*ed25519.Keypair)
	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	_, msg, err := gs.createSignedVoteAndVoteMessage(NewVoteFromHeader(header), prevote)
	require.NoError(t, err)
	gs.keypair = kr.Bob().(*ed25519.Keypair)

	gs.tracker.addVote(&networkVoteMessage{
		msg: msg,
	})

	voteMsgs, ok := gs.tracker.voteMessages[hash]
	require.True(t, ok)

	_, ok = voteMsgs[authorityID]
	require.True(t, ok)
}

func TestMessageTracker_handleTick(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, in, _, _ := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))
	gs.tracker = newTracker(gs.blockState, gs.messageHandler)

	testHash := common.Hash{1, 2, 3}
	msg := &VoteMessage{
		Round: 100,
		Message: SignedMessage{
			BlockHash: testHash,
		},
	}
	gs.tracker.addVote(&networkVoteMessage{
		msg: msg,
	})

	gs.tracker.handleTick()

	const testTimeout = time.Second
	select {
	case v := <-in:
		require.Equal(t, msg, v.msg)
	case <-time.After(testTimeout):
		t.Errorf("did not receive vote message %v", msg)
	}

	// shouldn't be deleted as round in message >= grandpa round
	require.Equal(t, 1, len(gs.tracker.voteMessages[testHash]))

	gs.state.round = 1
	msg = &VoteMessage{
		Round: 0,
		Message: SignedMessage{
			BlockHash: testHash,
		},
	}
	gs.tracker.addVote(&networkVoteMessage{
		msg: msg,
	})

	gs.tracker.handleTick()

	select {
	case v := <-in:
		require.Equal(t, msg, v.msg)
	case <-time.After(testTimeout):
		t.Errorf("did not receive vote message %v", msg)
	}

	// should be deleted as round in message < grandpa round
	require.Empty(t, len(gs.tracker.voteMessages[testHash]))
}
