//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var testHeader = &types.Header{
	ParentHash: testGenesisHeader.Hash(),
	Number:     1,
	Digest:     newTestDigest(),
}

var testHash = testHeader.Hash()

func newTestDigest() types.Digest {
	digest := types.NewDigest()
	prd, _ := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	digest.Add(*prd)
	return digest
}

func buildTestJustification(t *testing.T, qty int, round, setID uint64,
	kr *keystore.Ed25519Keyring, subround Subround) []SignedVote {
	t.Helper()
	var just []SignedVote
	for i := 0; i < qty; i++ {
		j := SignedVote{
			Vote:        *NewVote(testHash, uint32(round)),
			Signature:   createSignedVoteMsg(t, uint32(round), round, setID, kr.Keys[i%len(kr.Keys)], subround),
			AuthorityID: kr.Keys[i%len(kr.Keys)].Public().(*ed25519.PublicKey).AsBytes(),
		}
		just = append(just, j)
	}
	return just

}

func createSignedVoteMsg(t *testing.T, number uint32,
	round, setID uint64, pk *ed25519.Keypair, subround Subround) [64]byte {
	t.Helper()
	// create vote message
	msg, err := scale.Marshal(FullVote{
		Stage: subround,
		Vote:  *NewVote(testHash, number),
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
	t.Parallel()
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x004d000000000000006300000000000000017db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a7777000036e6eca85489bebbb0f687ca5404748d5aa2ffabee34e3ed272cc7b2f6d0a82c65b99bc7cd90dbc21bb528289ebf96705dbd7d96918d34d815509b4e0e2a030f34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691"), //nolint:lll
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	sigb := common.MustHexToBytes("0x36e6eca85489bebbb0f687ca5404748d5aa2ffabee34e3ed272cc7b2f6d0a82c65b99bc7cd90dbc21bb528289ebf96705dbd7d96918d34d815509b4e0e2a030f") //nolint:lll
	sig := [64]byte{}
	copy(sig[:], sigb)

	expected := &VoteMessage{
		Round: 77,
		SetID: 99,
		Message: SignedMessage{
			Stage:     precommit,
			BlockHash: common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			Number:    0x7777,
			Signature: sig,
			AuthorityID: ed25519.PublicKeyBytes(
				common.MustHexToHash("0x34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691"),
			),
		},
	}

	require.Equal(t, expected, msg)
}

func TestDecodeMessage_CommitMessage(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	expected := &CommitMessage{
		Round: 77,
		SetID: 1,
		Vote: Vote{
			Hash:   common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			Number: 99,
		},
		Precommits: []Vote{
			*testVote,
		},
		AuthData: []AuthData{
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
	t.Parallel()
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x020102000000000000000300000000000000ff000000"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	expected := &NeighbourPacketV1{
		Round:  2,
		SetID:  3,
		Number: 255,
	}
	require.Equal(t, expected, msg)
}

func TestDecodeMessage_CatchUpRequest(t *testing.T) {
	t.Parallel()
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x0311000000000000002200000000000000"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	expected := &CatchUpRequest{
		Round: 0x11,
		SetID: 0x22,
	}

	require.Equal(t, expected, msg)
}

func TestMessageHandler_VoteMessage(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	v, err := NewVoteFromHash(st.Block.BestBlockHash(), st.Block)
	require.NoError(t, err)
	v.Number = 0x7777

	charlieAuthority := kr.Charlie().(*ed25519.Keypair)

	const round = 99
	const setID = 77

	gs.state.round = round
	gs.state.setID = setID

	createdSigned, vm := createAndSignVoteMessage(t, charlieAuthority, round, setID, v, precommit)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	out, err := h.handleMessage("", vm)
	require.NoError(t, err)
	require.Nil(t, out)

	charlieAuthorityPublicKeyBytes := charlieAuthority.Public().(*ed25519.PublicKey).AsBytes()
	gotSignedVote, ok := gs.loadVote(charlieAuthorityPublicKeyBytes, precommit)
	require.True(t, ok)
	require.Equal(t, createdSigned, gotSignedVote)
}

func TestMessageHandler_NeighbourMessage(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)

	NeighbourPacketV1 := &NeighbourPacketV1{
		Round:  2,
		SetID:  3,
		Number: 1,
	}

	_, err = h.handleMessage("", NeighbourPacketV1)
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: types.Header{
			Number:     1,
			ParentHash: st.Block.GenesisHash(),
			Digest:     digest,
		},
		Body: *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	out, err := h.handleMessage("", NeighbourPacketV1)
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestMessageHandler_VerifyJustification_InvalidSig(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)
	gs.state.round = 77

	just := &SignedVote{
		Vote:        *testVote,
		Signature:   [64]byte{0x1},
		AuthorityID: gs.publicKeyBytes(),
	}

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	// scale encode the message to assert the wrapped error message
	expectedFullVote := FullVote{
		Stage: precommit,
		Vote:  just.Vote,
		Round: gs.state.round,
		SetID: gs.state.setID,
	}
	expectedErr := fmt.Errorf("%w: 0x%x for message {%v}", ErrInvalidSignature, just.Signature, expectedFullVote)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	err = verifyJustification(just, gs.state.round, gs.state.setID, precommit, h.grandpa.authorityKeySet())

	require.ErrorIs(t, err, ErrInvalidSignature)
	require.EqualError(t, expectedErr, err.Error())
}

func TestMessageHandler_CommitMessage_NoCatchUpRequest_ValidSig(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	round := uint64(1)
	gs.state.round = round
	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	err = st.Grandpa.SetPrecommits(round, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, round, gs.state.setID)
	require.NoError(t, err)
	fm.Vote = *NewVote(testHash, uint32(round))

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

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	out, err := gs.messageHandler.handleMessage("", fm)
	require.NoError(t, err)
	require.Nil(t, out)

	hash, err := st.Block.GetFinalisedHash(fm.Round, gs.state.setID)
	require.NoError(t, err)
	require.Equal(t, fm.Vote.Hash, hash)
}

func TestMessageHandler_CommitMessage_NoCatchUpRequest_MinVoteError(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	round := uint64(77)
	gs.state.round = round

	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	err = st.Grandpa.SetPrecommits(round, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(testGenesisHeader, round, gs.state.setID)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	out, err := h.handleMessage("", fm)

	const expectedErrString = "handling commit message: " +
		"verifying commit message justification: " +
		"minimum number of votes not met in a Justification: " +
		"for finalisation message; need 6 votes but received only 0 valid votes"

	require.EqualError(t, err, expectedErrString)
	require.ErrorIs(t, err, ErrMinVotesNotMet)
	require.Nil(t, out)
}

func TestMessageHandler_CommitMessage_WithCatchUpRequest(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	just := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}
	err = st.Grandpa.SetPrecommits(77, gs.state.setID, just)
	require.NoError(t, err)

	fm, err := gs.newCommitMessage(gs.head, 77, gs.state.setID)
	require.NoError(t, err)

	gs.state.voters = gs.state.voters[:1]

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	_, err = h.handleMessage("", fm)
	require.NoError(t, err)
}

func TestMessageHandler_CatchUpRequest_InvalidRound(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)
	req := newCatchUpRequest(77, 0)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	_, err = h.handleMessage("", req)
	require.Equal(t, ErrInvalidCatchUpRound, err)
}

func TestMessageHandler_CatchUpRequest_InvalidSetID(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)
	req := newCatchUpRequest(1, 77)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	_, err = h.handleMessage("", req)
	require.Equal(t, ErrSetIDMismatch, err)
}

func TestMessageHandler_CatchUpRequest_WithResponse(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	// set up needed info for response
	round := uint64(1)
	setID := uint64(0)
	gs.state.round = round + 1

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

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	err = gs.blockState.SetFinalisedHash(testGenesisHeader.Hash(), round, setID)
	require.NoError(t, err)
	err = gs.blockState.(*state.BlockState).SetHeader(&block.Header)
	require.NoError(t, err)

	pvj := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pcj := []SignedVote{
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

	expected, err := resp.ToConsensusMessage()
	require.NoError(t, err)

	// create and handle request
	req := newCatchUpRequest(round, setID)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)
	out, err := h.handleMessage("", req)
	require.NoError(t, err)
	require.Equal(t, expected, out)
}

func TestVerifyJustification(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)
	h := NewMessageHandler(gs, st.Block, telemetryMock)

	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote:        *vote,
		Signature:   createSignedVoteMsg(t, vote.Number, 77, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
		AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
	}

	err = verifyJustification(just, 77, gs.state.setID, precommit, h.grandpa.authorityKeySet())
	require.NoError(t, err)
}

func TestVerifyJustification_InvalidSignature(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)
	h := NewMessageHandler(gs, st.Block, telemetryMock)

	const round = 77
	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote: *vote,
		// create signed vote with mismatched vote number
		Signature:   createSignedVoteMsg(t, vote.Number+1, round, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
		AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
	}

	expectedFullVote := FullVote{
		Stage: precommit,
		Vote:  just.Vote,
		Round: round,
		SetID: gs.state.setID,
	}

	expectedErr := fmt.Errorf("%w: 0x%x for message {%v}", ErrInvalidSignature, just.Signature, expectedFullVote)
	err = verifyJustification(just, round, gs.state.setID, precommit, h.grandpa.authorityKeySet())
	require.ErrorIs(t, err, ErrInvalidSignature)
	require.EqualError(t, err, expectedErr.Error())
}

func TestVerifyJustification_InvalidAuthority(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)
	h := NewMessageHandler(gs, st.Block, telemetryMock)
	// sign vote with key not in authority set
	fakeKey, err := ed25519.NewKeypairFromPrivateKeyString(
		"0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	require.NoError(t, err)

	vote := NewVote(testHash, 123)
	just := &SignedVote{
		Vote:        *vote,
		Signature:   createSignedVoteMsg(t, vote.Number, 77, gs.state.setID, fakeKey, precommit),
		AuthorityID: fakeKey.Public().(*ed25519.PublicKey).AsBytes(),
	}

	encodedAuthorityID, err := just.AuthorityID.Encode()
	require.NoError(t, err)

	expectedErrMessage := fmt.Sprintf("%s: authority ID 0x%x", ErrVoterNotFound, encodedAuthorityID)
	err = verifyJustification(just, 77, gs.state.setID, precommit, h.grandpa.authorityKeySet())
	require.ErrorIs(t, err, ErrVoterNotFound)
	require.EqualError(t, err, expectedErrMessage)
}

func TestMessageHandler_VerifyPreVoteJustification(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block, telemetryMock)

	just := buildTestJustification(t, int(gs.state.threshold()), 1, gs.state.setID, kr, prevote)
	msg := &CatchUpResponse{
		Round:                1,
		SetID:                gs.state.setID,
		PreVoteJustification: just,
	}

	prevote, err := h.verifyPreVoteJustification(msg)
	require.NoError(t, err)
	require.Equal(t, testHash, prevote)
}

func TestMessageHandler_VerifyPreCommitJustification(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block, telemetryMock)

	round := uint64(1)
	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &CatchUpResponse{
		Round:                  round,
		SetID:                  gs.state.setID,
		PreCommitJustification: just,
		Hash:                   testHash,
		Number:                 uint32(round),
	}

	err = h.verifyPreCommitJustification(msg)
	require.NoError(t, err)
}

func TestMessageHandler_HandleCatchUpResponse(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	gs, st := newTestService(t, aliceKeyPair)

	err = st.Block.SetHeader(testHeader)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)

	round := uint64(1)
	gs.state.round = round + 1

	pvJust := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, prevote)
	pcJust := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &CatchUpResponse{
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

func TestMessageHandler_VerifyBlockJustification_WithEquivocatoryVotes(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	auths := []types.GrandpaVoter{
		{
			Key: *kr.Alice().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Bob().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Charlie().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Dave().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Eve().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Ferdie().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.George().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Heather().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Ian().Public().(*ed25519.PublicKey),
		},
	}

	gs, st := newTestService(t, aliceKeyPair)
	err = st.Grandpa.SetNextChange(auths, 0)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	setID, err := st.Grandpa.IncrementSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	round := uint64(1)
	number := uint32(1)
	precommits := buildTestJustification(t, 18, round, setID, kr, precommit)
	just := newJustification(round, testHash, number, precommits)
	data, err := scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.NoError(t, err)
}

func TestMessageHandler_VerifyBlockJustification(t *testing.T) {

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	auths := []types.GrandpaVoter{
		{
			Key: *kr.Alice().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Bob().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Charlie().Public().(*ed25519.PublicKey),
		},
	}

	gs, st := newTestService(t, aliceKeyPair)
	err = st.Grandpa.SetNextChange(auths, 0)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	digest2 := types.NewDigest()
	prd2, _ := types.NewBabeSecondaryPlainPreDigest(0, 2).ToPreRuntimeDigest()
	digest2.Add(*prd2)

	testHeader2 := types.Header{
		ParentHash: testGenesisHeader.Hash(),
		Number:     1,
		Digest:     digest2,
	}

	block2 := &types.Block{
		Header: testHeader2,
		Body:   *body,
	}

	err = st.Block.AddBlock(block2)
	require.NoError(t, err)

	err = st.Block.SetHeader(&testHeader2)
	require.NoError(t, err)

	setID, err := st.Grandpa.IncrementSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	round := uint64(1)
	number := uint32(1)
	precommits := buildTestJustification(t, 2, round, setID, kr, precommit)
	just := newJustification(round, testHash, number, precommits)
	data, err := scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.NoError(t, err)

	// use wrong hash, shouldn't verify
	precommits = buildTestJustification(t, 2, round+1, setID, kr, precommit)
	just = newJustification(round+1, testHash, number, precommits)
	just.Commit.Precommits[0].Vote.Hash = testHeader2.Hash()
	data, err = scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.Equal(t, ErrPrecommitBlockMismatch, err)
}

func TestMessageHandler_VerifyBlockJustification_invalid(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	auths := []types.GrandpaVoter{
		{
			Key: *kr.Alice().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Bob().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Charlie().Public().(*ed25519.PublicKey),
		},
	}

	gs, st := newTestService(t, aliceKeyPair)
	err = st.Grandpa.SetNextChange(auths, 1)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	setID, err := st.Grandpa.IncrementSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	genhash := st.Block.GenesisHash()
	round := uint64(2)
	number := uint32(2)

	// use wrong hash, shouldn't verify
	precommits := buildTestJustification(t, 2, round+1, setID, kr, precommit)
	just := newJustification(round+1, testHash, number, precommits)
	just.Commit.Precommits[0].Vote.Hash = genhash
	data, err := scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.Equal(t, ErrPrecommitBlockMismatch, err)

	// use wrong round, shouldn't verify
	precommits = buildTestJustification(t, 2, round+1, setID, kr, precommit)
	just = newJustification(round+2, testHash, number, precommits)
	data, err = scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.Equal(t, ErrInvalidSignature, err)

	// add authority not in set, shouldn't verify
	precommits = buildTestJustification(t, len(auths)+1, round+1, setID, kr, precommit)
	just = newJustification(round+1, testHash, number, precommits)
	data, err = scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.Equal(t, ErrAuthorityNotInSet, err)

	// not enough signatures, shouldn't verify
	precommits = buildTestJustification(t, 1, round+1, setID, kr, precommit)
	just = newJustification(round+1, testHash, number, precommits)
	data, err = scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.Equal(t, ErrMinVotesNotMet, err)

	// mismatch justification header and block header
	precommits = buildTestJustification(t, 1, round+1, setID, kr, precommit)
	just = newJustification(round+1, testHash, number, precommits)
	data, err = scale.Marshal(*just)
	require.NoError(t, err)
	otherHeader := types.NewEmptyHeader()
	err = gs.VerifyBlockJustification(otherHeader.Hash(), data)
	require.ErrorIs(t, err, ErrJustificationMismatch)

	expectedErr := fmt.Sprintf("%s: justification %s and block hash %s", ErrJustificationMismatch,
		testHash.Short(), otherHeader.Hash().Short())
	assert.ErrorIs(t, err, ErrJustificationMismatch)
	require.EqualError(t, err, expectedErr)
}

func TestMessageHandler_VerifyBlockJustification_ErrFinalisedBlockMismatch(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	auths := []types.GrandpaVoter{
		{
			Key: *kr.Alice().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Bob().Public().(*ed25519.PublicKey),
		},
		{
			Key: *kr.Charlie().Public().(*ed25519.PublicKey),
		},
	}

	gs, st := newTestService(t, aliceKeyPair)
	err = st.Grandpa.SetNextChange(auths, 1)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{0})
	require.NoError(t, err)

	block := &types.Block{
		Header: *testHeader,
		Body:   *body,
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	setID := uint64(0)
	round := uint64(1)
	number := uint32(1)

	err = st.Block.SetFinalisedHash(block.Header.Hash(), round, setID)
	require.NoError(t, err)

	var testHeader2 = &types.Header{
		ParentHash: testHeader.Hash(),
		Number:     2,
		Digest:     newTestDigest(),
	}

	testHash = testHeader2.Hash()
	block2 := &types.Block{
		Header: *testHeader2,
		Body:   *body,
	}
	err = st.Block.AddBlock(block2)
	require.NoError(t, err)

	// justification fails since there is already a block finalised in this round and set id
	precommits := buildTestJustification(t, 18, round, setID, kr, precommit)
	just := newJustification(round, testHash, number, precommits)
	data, err := scale.Marshal(*just)
	require.NoError(t, err)
	err = gs.VerifyBlockJustification(testHash, data)
	require.ErrorIs(t, err, errFinalisedBlocksMismatch)
}

func Test_getEquivocatoryVoters(t *testing.T) {
	t.Parallel()

	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	tests := map[string]struct {
		votes []AuthData
		want  map[ed25519.PublicKeyBytes]struct{}
	}{
		"no_votes": {
			votes: []AuthData{},
			want:  map[ed25519.PublicKeyBytes]struct{}{},
		},
		"one_vote": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{},
		},
		"two_votes_different_authorities": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{},
		},
		"duplicate_votes": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{},
		},
		"equivocatory_vote": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{
				ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(): {},
			},
		},
		"equivocatory_vote_with_duplicate": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{
				ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(): {},
			},
		},
		"three_voters_one_equivocatory": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{
				ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(): {},
			},
		},
		"three_voters_one_equivocatory_one_duplicate": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{
				ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(): {},
			},
		},
		"three_voters_two_equivocatory": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
				{
					AuthorityID: ed25519Keyring.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{
				ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(): {},
				ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes():   {},
			},
		},
		"three_voters_two_duplicate": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{},
		},
		"three_voters": {
			votes: []AuthData{
				{
					AuthorityID: ed25519Keyring.Alice().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Bob().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{1, 2, 3, 4},
				},
				{
					AuthorityID: ed25519Keyring.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
					Signature:   [64]byte{5, 6, 7, 8},
				},
			},
			want: map[ed25519.PublicKeyBytes]struct{}{},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := getEquivocatoryVoters(tt.votes)
			assert.Equalf(t, tt.want, got, "getEquivocatoryVoters(%v)", tt.votes)
		})
	}
}

func Test_VerifyCommitMessageJustification_ShouldRemoveEquivocatoryVotes(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	const fakeRound = 2

	gs, st := newTestService(t, aliceKeyPair)
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	h := NewMessageHandler(gs, st.Block, telemetryMock)

	const previousBlocksToAdd = 8
	now := time.Unix(1000, 0)
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number

	// many of equivocatory votes
	ed25519Keyring, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	fakeAuthorities := []*ed25519.Keypair{
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Alice().(*ed25519.Keypair),
		ed25519Keyring.Bob().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Charlie().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Dave().(*ed25519.Keypair),
		ed25519Keyring.Eve().(*ed25519.Keypair),
		ed25519Keyring.Ferdie().(*ed25519.Keypair),
	}

	authData := make([]AuthData, len(fakeAuthorities))
	precommits := make([]Vote, len(fakeAuthorities))

	for i, auth := range fakeAuthorities {
		vote := types.GrandpaVote{
			Hash:   bfcHash,
			Number: uint32(bfcNumber),
		}

		sig := signFakeFullVote(
			t, auth, precommit, vote, fakeRound, gs.state.setID,
		)

		authData[i] = AuthData{
			Signature:   sig,
			AuthorityID: auth.Public().(*ed25519.PublicKey).AsBytes(),
		}
		precommits[i] = Vote{Hash: bfcHash, Number: uint32(bfcNumber)}
	}

	// Charlie has an equivocatory vote
	testCommitData := &CommitMessage{
		Round: fakeRound,
		Vote: Vote{
			Hash:   bfcHash,
			Number: uint32(bfcNumber),
		},
		Precommits: precommits,
		AuthData:   authData,
	}

	err = verifyCommitMessageJustification(*testCommitData, h.grandpa.state.setID,
		h.grandpa.state.threshold(), h.grandpa.authorityKeySet(), h.blockState)

	require.NoError(t, err)
}

func Test_VerifyPrevoteJustification_CountEquivocatoryVoters(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	gs, st := newTestService(t, aliceKeyPair)
	h := NewMessageHandler(gs, st.Block, telemetryMock)

	const previousBlocksToAdd = 9
	now := time.Unix(1000, 0)
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number

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

func Test_VerifyPreCommitJustification(t *testing.T) {
	t.Parallel()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	aliceKeyPair := kr.Alice().(*ed25519.Keypair)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).
		AnyTimes()

	gs, st := newTestService(t, aliceKeyPair)
	h := NewMessageHandler(gs, st.Block, telemetryMock)

	const previousBlocksToAdd = 7
	now := time.Unix(1000, 0)
	bfcBlock := addBlocksAndReturnTheLastOne(t, st.Block, previousBlocksToAdd, now)

	bfcHash := bfcBlock.Header.Hash()
	bfcNumber := bfcBlock.Header.Number

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
		ed25519Keyring.Bob().(*ed25519.Keypair),
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

func signFakeFullVote(
	t *testing.T, auth *ed25519.Keypair,
	stage Subround, v types.GrandpaVote,
	round, setID uint64) [64]byte {
	t.Helper()
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

func TestService_VerifyBlockJustification(t *testing.T) { //nolint
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	precommits := buildTestJustification(t, 2, 1, 0, kr, precommit)
	justification := newJustification(1, testHash, 1, precommits)
	justificationBytes, err := scale.Marshal(*justification)
	require.NoError(t, err)

	type fields struct {
		blockStateBuilder   func(ctrl *gomock.Controller) BlockState
		grandpaStateBuilder func(ctrl *gomock.Controller) GrandpaState
	}
	type args struct {
		hash          common.Hash
		justification []byte
	}
	tests := map[string]struct {
		fields  fields
		args    args
		want    []byte
		wantErr error
	}{
		"invalid_justification": {
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					return nil
				},
				grandpaStateBuilder: func(ctrl *gomock.Controller) GrandpaState {
					return nil
				},
			},
			args: args{
				hash:          common.Hash{},
				justification: []byte{1, 2, 3},
			},
			want: nil,
			wantErr: errors.New("decoding struct: unmarshalling field at index 1: decoding struct: unmarshalling" +
				" field at index 0: EOF"),
		},
		"valid_justification": {
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().HasFinalisedBlock(uint64(1), uint64(0)).Return(false, nil)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(testHeader, nil)
					mockBlockState.EXPECT().IsDescendantOf(testHash, testHash).
						Return(true, nil).Times(3)
					mockBlockState.EXPECT().GetHeader(testHash).Return(testHeader, nil).Times(3)
					mockBlockState.EXPECT().SetFinalisedHash(testHash, uint64(1),
						uint64(0)).Return(nil)
					return mockBlockState
				},
				grandpaStateBuilder: func(ctrl *gomock.Controller) GrandpaState {
					mockGrandpaState := NewMockGrandpaState(ctrl)
					mockGrandpaState.EXPECT().GetSetIDByBlockNumber(uint(1)).Return(uint64(0), nil)
					mockGrandpaState.EXPECT().GetAuthorities(uint64(0)).Return([]types.GrandpaVoter{
						{Key: *kr.Alice().Public().(*ed25519.PublicKey), ID: 1},
						{Key: *kr.Bob().Public().(*ed25519.PublicKey), ID: 2},
						{Key: *kr.Charlie().Public().(*ed25519.PublicKey), ID: 3},
					}, nil)
					return mockGrandpaState
				},
			},
			args: args{
				hash:          testHash,
				justification: justificationBytes,
			},
			want: justificationBytes,
		},
		"valid_justification_extra_bytes": {
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().HasFinalisedBlock(uint64(1), uint64(0)).Return(false, nil)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(testHeader, nil)
					mockBlockState.EXPECT().IsDescendantOf(testHash, testHash).
						Return(true, nil).Times(3)
					mockBlockState.EXPECT().GetHeader(testHash).Return(testHeader, nil).Times(3)
					mockBlockState.EXPECT().SetFinalisedHash(testHash, uint64(1),
						uint64(0)).Return(nil)
					return mockBlockState
				},
				grandpaStateBuilder: func(ctrl *gomock.Controller) GrandpaState {
					mockGrandpaState := NewMockGrandpaState(ctrl)
					mockGrandpaState.EXPECT().GetSetIDByBlockNumber(uint(1)).Return(uint64(0), nil)
					mockGrandpaState.EXPECT().GetAuthorities(uint64(0)).Return([]types.GrandpaVoter{
						{Key: *kr.Alice().Public().(*ed25519.PublicKey), ID: 1},
						{Key: *kr.Bob().Public().(*ed25519.PublicKey), ID: 2},
						{Key: *kr.Charlie().Public().(*ed25519.PublicKey), ID: 3},
					}, nil)
					return mockGrandpaState
				},
			},
			args: args{
				hash:          testHash,
				justification: append(justificationBytes, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}...),
			},
			want: justificationBytes,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			s := &Service{
				blockState:   tt.fields.blockStateBuilder(ctrl),
				grandpaState: tt.fields.grandpaStateBuilder(ctrl),
			}
			err := s.VerifyBlockJustification(tt.args.hash, tt.args.justification)
			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
