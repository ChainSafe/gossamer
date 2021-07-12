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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/stretchr/testify/require"
)

var testHeader = &types.Header{
	ParentHash: testGenesisHeader.Hash(),
	Number:     big.NewInt(1),
	Digest: types.Digest{
		types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest(),
	},
}

var testBlock = &types.Block{
	Header: testHeader,
	Body:   &types.Body{},
}

var testHash = testHeader.Hash()

func buildTestJustification(t *testing.T, qty int, round, setID uint64, kr *keystore.Ed25519Keyring, subround subround) []*SignedVote {
	just := []*SignedVote{}
	for i := 0; i < qty; i++ {
		j := &SignedVote{
			Vote:        NewVote(testHash, uint32(round)),
			Signature:   createSignedVoteMsg(t, uint32(round), round, setID, kr.Keys[i%len(kr.Keys)], subround),
			AuthorityID: kr.Keys[i%len(kr.Keys)].Public().(*ed25519.PublicKey).AsBytes(),
		}
		just = append(just, j)
	}
	return just

}

func createSignedVoteMsg(t *testing.T, number uint32, round, setID uint64, pk *ed25519.Keypair, subround subround) [64]byte {
	// create vote message
	msg, err := scale.Encode(&FullVote{
		Stage: subround,
		Vote:  NewVote(testHash, number),
		Round: round,
		SetID: setID,
	})
	require.NoError(t, err)

	var sMsgArray [64]byte
	sMsg, err := pk.Sign(msg)
	require.NoError(t, err)
	copy(sMsgArray[:], sMsg)
	return sMsgArray
}

func TestDecodeMessage_VoteMessage(t *testing.T) {
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x004d000000000000006300000000000000017db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a7777000036e6eca85489bebbb0f687ca5404748d5aa2ffabee34e3ed272cc7b2f6d0a82c65b99bc7cd90dbc21bb528289ebf96705dbd7d96918d34d815509b4e0e2a030f34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	sigb := common.MustHexToBytes("0x36e6eca85489bebbb0f687ca5404748d5aa2ffabee34e3ed272cc7b2f6d0a82c65b99bc7cd90dbc21bb528289ebf96705dbd7d96918d34d815509b4e0e2a030f")
	sig := [64]byte{}
	copy(sig[:], sigb)

	expected := &VoteMessage{
		Round: 77,
		SetID: 99,
		Message: &SignedMessage{
			Stage:       precommit,
			Hash:        common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			Number:      0x7777,
			Signature:   sig,
			AuthorityID: ed25519.PublicKeyBytes(common.MustHexToHash("0x34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691")),
		},
	}

	require.Equal(t, expected, msg)
}

func TestDecodeMessage_CommitMessage(t *testing.T) {
	expected := &CommitMessage{
		Round: 77,
		SetID: 1,
		Vote: &Vote{
			Hash:   common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			Number: 99,
		},
		Precommits: []*Vote{
			testVote,
		},
		AuthData: []*AuthData{
			{
				Signature:   testSignature,
				AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
			},
		},
	}
	cm, err := expected.ToConsensusMessage()
	require.NoError(t, err)

	msg, err := decodeMessage(cm)
	require.NoError(t, err)
	require.Equal(t, expected, msg)
}

func TestDecodeMessage_NeighbourMessage(t *testing.T) {
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x020102000000000000000300000000000000ff000000"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	expected := &NeighbourMessage{
		Version: 1,
		Round:   2,
		SetID:   3,
		Number:  255,
	}
	require.Equal(t, expected, msg)
}

func TestDecodeMessage_CatchUpRequest(t *testing.T) {
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x0311000000000000002200000000000000"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	expected := &catchUpRequest{
		Round: 0x11,
		SetID: 0x22,
	}

	require.Equal(t, expected, msg)
}

func TestMessageHandler_VoteMessage(t *testing.T) {
	gs, st := newTestService(t)

	v, err := NewVoteFromHash(st.Block.BestBlockHash(), st.Block)
	require.NoError(t, err)

	gs.state.setID = 99
	gs.state.round = 77
	v.Number = 0x7777
	_, vm, err := gs.createSignedVoteAndVoteMessage(v, precommit)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", vm)
	require.NoError(t, err)
	require.Nil(t, out)

	select {
	case vote := <-gs.in:
		require.Equal(t, vm, vote.msg)
	case <-time.After(time.Second):
		t.Fatal("did not receive VoteMessage")
	}
}

func TestMessageHandler_NeighbourMessage(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	err := st.Block.AddBlock(testBlock)
	require.NoError(t, err)

	msg := &NeighbourMessage{
		Version: 1,
		Round:   2,
		SetID:   3,
		Number:  1,
	}

	_, err = h.handleMessage("", msg)
	require.NoError(t, err)

	out, err := h.handleMessage("", msg)
	require.NoError(t, err)
	require.Nil(t, out)

	// check if request for justification was sent out
	expected := &testJustificationRequest{
		to:  "",
		num: 1,
	}
	require.Equal(t, expected, gs.network.(*testNetwork).justificationRequest)
}

func TestMessageHandler_VerifyJustification_InvalidSig(t *testing.T) {
	gs, st := newTestService(t)
	gs.state.round = 77

	just := &SignedVote{
		Vote:        testVote,
		Signature:   [64]byte{0x1},
		AuthorityID: gs.publicKeyBytes(),
	}

	h := NewMessageHandler(gs, st.Block)
	err := h.verifyJustification(just, gs.state.round, gs.state.setID, precommit)
	require.Equal(t, err, ErrInvalidSignature)
}

func TestMessageHandler_CommitMessage_NoCatchUpRequest_ValidSig(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(77)
	gs.state.round = round
	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	err := st.Grandpa.SetPrecommits(round, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, round)
	require.NoError(t, err)
	fm.Vote = NewVote(testHash, uint32(round))

	err = st.Block.AddBlock(testBlock)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", fm)
	require.NoError(t, err)
	require.Nil(t, out)

	hash, err := st.Block.GetFinalisedHash(0, 0)
	require.NoError(t, err)
	require.Equal(t, fm.Vote.Hash, hash)

	hash, err = st.Block.GetFinalisedHash(fm.Round, gs.state.setID)
	require.NoError(t, err)
	require.Equal(t, fm.Vote.Hash, hash)
}

func TestMessageHandler_CommitMessage_NoCatchUpRequest_MinVoteError(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(77)
	gs.state.round = round

	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	err := st.Grandpa.SetPrecommits(round, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, round)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", fm)
	require.EqualError(t, err, ErrMinVotesNotMet.Error())
	require.Nil(t, out)
}

func TestMessageHandler_CommitMessage_WithCatchUpRequest(t *testing.T) {
	gs, st := newTestService(t)

	just := []*SignedVote{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err := st.Grandpa.SetPrecommits(77, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77)
	require.NoError(t, err)

	gs.state.voters = gs.state.voters[:1]

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", fm)
	require.NoError(t, err)
	require.NotNil(t, out)

	req := newCatchUpRequest(77, gs.state.setID)
	expected, err := req.ToConsensusMessage()
	require.NoError(t, err)
	require.Equal(t, expected, out)
}

func TestMessageHandler_CatchUpRequest_InvalidRound(t *testing.T) {
	gs, st := newTestService(t)
	req := newCatchUpRequest(77, 0)

	h := NewMessageHandler(gs, st.Block)
	_, err := h.handleMessage("", req)
	require.Equal(t, ErrInvalidCatchUpRound, err)
}

func TestMessageHandler_CatchUpRequest_InvalidSetID(t *testing.T) {
	gs, st := newTestService(t)
	req := newCatchUpRequest(1, 77)

	h := NewMessageHandler(gs, st.Block)
	_, err := h.handleMessage("", req)
	require.Equal(t, ErrSetIDMismatch, err)
}

func TestMessageHandler_CatchUpRequest_WithResponse(t *testing.T) {
	gs, st := newTestService(t)

	// set up needed info for response
	round := uint64(1)
	setID := uint64(0)
	gs.state.round = round + 1

	err := st.Block.AddBlock(testBlock)
	require.NoError(t, err)

	err = gs.blockState.SetFinalisedHash(testHeader.Hash(), round, setID)
	require.NoError(t, err)
	err = gs.blockState.(*state.BlockState).SetHeader(testHeader)
	require.NoError(t, err)

	pvj := []*SignedVote{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pcj := []*SignedVote{
		{
			Vote:        testVote2,
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

	expected, err := resp.ToConsensusMessage()
	require.NoError(t, err)

	// create and handle request
	req := newCatchUpRequest(round, setID)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", req)
	require.NoError(t, err)
	require.Equal(t, expected, out)
}

func TestVerifyJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote:        vote,
		Signature:   createSignedVoteMsg(t, vote.Number, 77, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
		AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
	}

	err := h.verifyJustification(just, 77, gs.state.setID, precommit)
	require.NoError(t, err)
}

func TestVerifyJustification_InvalidSignature(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote: vote,
		// create signed vote with mismatched vote number
		Signature:   createSignedVoteMsg(t, vote.Number+1, 77, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
		AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
	}

	err := h.verifyJustification(just, 77, gs.state.setID, precommit)
	require.EqualError(t, err, ErrInvalidSignature.Error())
}

func TestVerifyJustification_InvalidAuthority(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)
	// sign vote with key not in authority set
	fakeKey, err := ed25519.NewKeypairFromPrivateKeyString("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	require.NoError(t, err)

	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote:        vote,
		Signature:   createSignedVoteMsg(t, vote.Number, 77, gs.state.setID, fakeKey, precommit),
		AuthorityID: fakeKey.Public().(*ed25519.PublicKey).AsBytes(),
	}

	err = h.verifyJustification(just, 77, gs.state.setID, precommit)
	require.EqualError(t, err, ErrVoterNotFound.Error())
}

func TestMessageHandler_VerifyPreVoteJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	just := buildTestJustification(t, int(gs.state.threshold()), 1, gs.state.setID, kr, prevote)
	msg := &catchUpResponse{
		Round:                1,
		SetID:                gs.state.setID,
		PreVoteJustification: just,
	}

	prevote, err := h.verifyPreVoteJustification(msg)
	require.NoError(t, err)
	require.Equal(t, testHash, prevote)
}

func TestMessageHandler_VerifyPreCommitJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	round := uint64(1)
	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &catchUpResponse{
		Round:                  round,
		SetID:                  gs.state.setID,
		PreCommitJustification: just,
		Hash:                   testHash,
		Number:                 uint32(round),
	}

	err := h.verifyPreCommitJustification(msg)
	require.NoError(t, err)
}

func TestMessageHandler_HandleCatchUpResponse(t *testing.T) {
	gs, st := newTestService(t)

	err := st.Block.SetHeader(testHeader)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)

	round := uint64(77)
	gs.state.round = round + 1

	pvJust := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, prevote)
	pcJust := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &catchUpResponse{
		Round:                  round,
		SetID:                  gs.state.setID,
		PreVoteJustification:   pvJust,
		PreCommitJustification: pcJust,
		Hash:                   testHash,
		Number:                 uint32(round),
	}

	out, err := h.handleMessage("", msg)
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, round+1, gs.state.round)
}

func TestMessageHandler_VerifyBlockJustification(t *testing.T) {
	auths := []*types.GrandpaVoter{
		{
			Key: kr.Alice().Public().(*ed25519.PublicKey),
		},
		{
			Key: kr.Bob().Public().(*ed25519.PublicKey),
		},
		{
			Key: kr.Charlie().Public().(*ed25519.PublicKey),
		},
	}

	gs, st := newTestService(t)
	err := st.Grandpa.SetNextChange(auths, big.NewInt(1))
	require.NoError(t, err)

	err = st.Grandpa.IncrementSetID()
	require.NoError(t, err)

	setID, err := st.Grandpa.GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	round := uint64(2)
	number := uint32(2)
	precommits := buildTestJustification(t, 2, round, setID, kr, precommit)
	just := newJustification(round, testHash, number, precommits)
	data, err := just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.NoError(t, err)

	// use wrong hash, shouldn't verify
	just = newJustification(round, common.Hash{}, number, precommits)
	data, err = just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.NotNil(t, err)
	require.Equal(t, ErrJustificationHashMismatch, err)

	// use wrong number, shouldn't verify
	just = newJustification(round, testHash, number+1, precommits)
	data, err = just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.NotNil(t, err)
	require.Equal(t, ErrJustificationNumberMismatch, err)

	// use wrong round, shouldn't verify
	just = newJustification(round+1, testHash, number, precommits)
	data, err = just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.NotNil(t, err)
	require.Equal(t, ErrInvalidSignature, err)

	// add authority not in set, shouldn't verify
	precommits = buildTestJustification(t, len(auths)+1, round, setID, kr, precommit)
	just = newJustification(round, testHash, number, precommits)
	data, err = just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.Equal(t, ErrAuthorityNotInSet, err)

	// not enough signatures, shouldn't verify
	precommits = buildTestJustification(t, 1, round, setID, kr, precommit)
	just = newJustification(round, testHash, number, precommits)
	data, err = just.Encode()
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(data)
	require.Equal(t, ErrMinVotesNotMet, err)
}
