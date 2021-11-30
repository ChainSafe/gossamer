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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func Test_VerifyPreCommitJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := newCatchUp(true, gs, newTestNetwork(t), nil)

	const previousBlocksToAdd = 7
	now := time.Unix(1000, 0)
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number.Int64()

	// many of equivocatory votes
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	// Alice, Charlie, David - Equivocatory
	// Bob, Eve, Ferdie, Ian - Legit
	// total of votes 4 legit + 3 equivocatory
	// the threshold for testing is 9, so 2/3 of 9 = 6
	fakeAuthorities := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Eve().(*ed25519.Keypair),
		ed25519Keyring.Ferdie().(*ed25519.Keypair),
		ed25519Keyring.Ian().(*ed25519.Keypair),
	}

	prevotesJustification := make([]SignedVote, len(fakeAuthorities))
	for idx, fakeAuthority := range fakeAuthorities {
		vote := types.GrandpaVote{
			Hash:   bfcHash,
			Number: uint32(bfcNumber),
		}

		sig := signFakeFullVote(
			t, fakeAuthority, precommit, vote, gs.state.round, gs.state.setID)

		prevotesJustification[idx] = SignedVote{
			Vote:        vote,
			Signature:   sig,
			AuthorityID: fakeAuthority.Public().(*ed25519.PublicKey).AsBytes(),
		}
	}

	testCatchUpResponse := &CatchUpResponse{
		SetID:                  gs.state.setID,
		Round:                  gs.state.round,
		PreCommitJustification: prevotesJustification,
		Hash:                   bfcHash,
		Number:                 uint32(bfcNumber),
	}

	err = h.verifyPreCommitJustification(testCatchUpResponse)
	require.NoError(t, err)
}

func Test_VerifyPrevoteJustification_CountEquivocatoryVoters(t *testing.T) {
	gs, st := newTestService(t)
	h := newCatchUp(true, gs, newTestNetwork(t), nil)

	const previousBlocksToAdd = 9
	now := time.Unix(1000, 0)
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number.Int64()

	// many of equivocatory votes
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	fakeAuthorities := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Eve().(*ed25519.Keypair),
		ed25519Keyring.Ferdie().(*ed25519.Keypair),
		ed25519Keyring.Ian().(*ed25519.Keypair),
	}

	prevotesJustification := make([]SignedVote, len(fakeAuthorities))
	for idx, fakeAuthority := range fakeAuthorities {
		var vote types.GrandpaVote

		// put one vote on a different hash
		if idx == 1 {
			vote = types.GrandpaVote{
				Hash:   bfcBlock.Header.ParentHash,
				Number: uint32(bfcNumber - 1),
			}
		} else {
			vote = types.GrandpaVote{
				Hash:   bfcHash,
				Number: uint32(bfcNumber),
			}
		}

		sig := signFakeFullVote(
			t, fakeAuthority, prevote, vote, gs.state.round, gs.state.setID)

		prevotesJustification[idx] = SignedVote{
			Vote:        vote,
			Signature:   sig,
			AuthorityID: fakeAuthority.Public().(*ed25519.PublicKey).AsBytes(),
		}
	}

	testCatchUpResponse := &CatchUpResponse{
		SetID:                gs.state.setID,
		Round:                gs.state.round,
		PreVoteJustification: prevotesJustification,
		Hash:                 bfcHash,
		Number:               uint32(bfcNumber),
	}

	hash, err := h.verifyPreVoteJustification(testCatchUpResponse)
	require.NoError(t, err)
	require.Equal(t, hash, bfcHash)
}

func signFakeFullVote(
	t *testing.T, auth *ed25519.Keypair,
	stage Subround, v types.GrandpaVote,
	round, setID uint64) [64]byte {
	msg, err := scale.Marshal(FullVote{
		Stage: stage,
		Vote:  v,
		Round: round,
		SetID: setID,
	})
	require.NoError(t, err)

	var sig [64]byte
	privSig, err := auth.Private().Sign(msg)
	require.NoError(t, err)

	copy(sig[:], privSig)

	return sig
}
