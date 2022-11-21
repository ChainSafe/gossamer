// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

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

func TestMessageTracker_SendMessage(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs := setupGrandpa(t, kr.Bob().(*ed25519.Keypair))

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

	aliceAuthority := kr.Alice().(*ed25519.Keypair)
	aliceSignedVote, aliceVoteMessage := createAndSignVoteMessage(t, aliceAuthority, gs.state.round,
		gs.state.setID, NewVoteFromHeader(next), prevote)

	const expectedErr = "validating vote: block does not exist"
	_, err = gs.validateVoteMessage("", aliceVoteMessage)
	require.ErrorIs(t, err, ErrBlockDoesNotExist)
	require.EqualError(t, err, expectedErr)

	authorityID := kr.Alice().Public().(*ed25519.PublicKey).AsBytes()
	voteMessage := getMessageFromVotesMapping(gs.tracker.votes.mapping, next.Hash(), authorityID)
	require.Equal(t, aliceVoteMessage, voteMessage)

	err = gs.blockState.(*state.BlockState).AddBlock(&types.Block{
		Header: *next,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	// grandpa tracker check every second if the block
	// was included in the block tree
	time.Sleep(2 * time.Second)

	aliceAuthorityPublicBytes := aliceAuthority.Public().(*ed25519.PublicKey).AsBytes()
	gotSignedVote, ok := gs.loadVote(aliceAuthorityPublicBytes, prevote)
	require.True(t, ok)
	require.Equal(t, aliceSignedVote, gotSignedVote)
}
