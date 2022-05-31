// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa/models"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

var testVote = &models.Vote{
	Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	Number: 999,
}

var testVote2 = &models.Vote{
	Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	Number: 333,
}

var testSignature = [64]byte{1, 2, 3, 4}
var testAuthorityID = [32]byte{5, 6, 7, 8}

func TestCommitMessageEncode(t *testing.T) {
	exp := common.MustHexToBytes("0x4d0000000000000000000000000000007db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a00000000040a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000040102030400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee") //nolint:lll
	gs, st := newTestService(t)
	just := []models.SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err := st.Grandpa.SetPrecommits(77, gs.state.SetID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77)
	require.NoError(t, err)
	precommits, authData := justificationToCompact(just)

	expected := models.CommitMessage{
		Round:      77,
		Vote:       *models.NewVoteFromHeader(gs.head),
		Precommits: precommits,
		AuthData:   authData,
	}

	enc, err := scale.Marshal(*fm)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	msg := models.CommitMessage{}
	err = scale.Unmarshal(enc, &msg)
	require.NoError(t, err)
	require.Equal(t, expected, msg)
}

func TestVoteMessageToConsensusMessage(t *testing.T) {
	gs, st := newTestService(t)

	v, err := models.NewVoteFromHash(st.Block.BestBlockHash(), st.Block)
	require.NoError(t, err)

	gs.state.SetID = 99
	gs.state.Round = 77
	v.Number = 0x7777

	// test precommit
	_, vm, err := gs.createSignedVoteAndVoteMessage(v, models.Precommit)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected := &models.VoteMessage{
		Round: gs.state.Round,
		SetID: gs.state.SetID,
		Message: models.SignedMessage{
			Stage:       models.Precommit,
			BlockHash:   v.Hash,
			Number:      v.Number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)

	// test prevote
	_, vm, err = gs.createSignedVoteAndVoteMessage(v, models.Prevote)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected = &models.VoteMessage{
		Round: gs.state.Round,
		SetID: gs.state.SetID,
		Message: models.SignedMessage{
			Stage:       models.Prevote,
			BlockHash:   v.Hash,
			Number:      v.Number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)
}

func TestCommitMessageToConsensusMessage(t *testing.T) {
	gs, st := newTestService(t)
	just := []models.SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err := st.Grandpa.SetPrecommits(77, gs.state.SetID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77)
	require.NoError(t, err)
	precommits, authData := justificationToCompact(just)

	expected := &models.CommitMessage{
		Round:      77,
		Vote:       *models.NewVoteFromHeader(gs.head),
		Precommits: precommits,
		AuthData:   authData,
	}

	require.Equal(t, expected, fm)
}

func TestNewCatchUpResponse(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(1)
	setID := uint64(1)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     1,
			Digest:     digest,
		},
		Body: types.Body{},
	}

	hash := block.Header.Hash()
	v := &models.Vote{
		Hash:   hash,
		Number: 1,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	err = gs.blockState.SetFinalisedHash(hash, round, setID)
	require.NoError(t, err)
	err = gs.blockState.(*state.BlockState).SetHeader(testHeader)
	require.NoError(t, err)

	pvj := []models.SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pcj := []models.SignedVote{
		{
			Vote:        *testVote2,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	err = gs.grandpaState.SetPrevotes(round, setID, pvj)
	require.NoError(t, err)
	err = gs.grandpaState.SetPrecommits(round, setID, pcj)
	require.NoError(t, err)

	resp, err := gs.newCatchUpResponse(round, setID)
	require.NoError(t, err)

	expected := &models.CatchUpResponse{
		Round:                  round,
		SetID:                  setID,
		PreVoteJustification:   pvj,
		PreCommitJustification: pcj,
		Hash:                   v.Hash,
		Number:                 v.Number,
	}

	require.Equal(t, expected, resp)
}

func TestNeighbourMessageToConsensusMessage(t *testing.T) {
	msg := &models.NeighbourMessage{
		Version: 1,
		Round:   2,
		SetID:   3,
		Number:  255,
	}

	cm, err := msg.ToConsensusMessage()
	require.NoError(t, err)

	expected := &ConsensusMessage{
		Data: common.MustHexToBytes("0x020102000000000000000300000000000000ff000000"),
	}

	require.Equal(t, expected, cm)
}
