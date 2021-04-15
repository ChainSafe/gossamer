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
	//"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/scale"

	//"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

var testHeader = &types.Header{
	ParentHash: testGenesisHeader.Hash(),
	Number:     big.NewInt(1),
}

var testHash = testHeader.Hash()

func buildTestJustifications(t *testing.T, qty int, round, setID uint64, kr *keystore.Ed25519Keyring, subround subround) []*Justification {
	just := []*Justification{}
	for i := 0; i < qty; i++ {
		j := &Justification{
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
		Stage: precommit,
		Message: &SignedMessage{
			Hash:        common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			Number:      0x7777,
			Signature:   sig,
			AuthorityID: ed25519.PublicKeyBytes(common.MustHexToHash("0x34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691")),
		},
	}

	require.Equal(t, expected, msg)
}

func TestDecodeMessage_FinalizationMessage(t *testing.T) {
	cm := &ConsensusMessage{
		Data: common.MustHexToBytes("0x054d000000000000007db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a00000000040a0b0c0d00000000000000000000000000000000000000000000000000000000e70300000102030400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000034602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691"),
	}

	msg, err := decodeMessage(cm)
	require.NoError(t, err)

	expected := &FinalizationMessage{
		Round: 77,
		Vote: &Vote{
			hash:   common.MustHexToHash("0x7db9db5ed9967b80143100189ba69d9e4deab85ac3570e5df25686cabe32964a"),
			number: 0,
		},
		Justification: []*Justification{
			{
				Vote:        testVote,
				Signature:   testSignature,
				AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
			},
		},
	}

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
	v.number = 0x7777
	vm, err := gs.createVoteMessage(v, precommit, gs.keypair)
	require.NoError(t, err)

	cm, err := vm.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", cm)
	require.NoError(t, err)
	require.Nil(t, out)

	select {
	case vote := <-gs.in:
		require.Equal(t, vm, vote)
	case <-time.After(time.Second):
		t.Fatal("did not receive VoteMessage")
	}
}

func TestMessageHandler_NeighbourMessage(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	msg := &NeighbourMessage{
		Version: 1,
		Round:   2,
		SetID:   3,
		Number:  1,
	}

	cm, err := msg.ToConsensusMessage()
	require.NoError(t, err)

	// _, err = h.handleMessage("", cm)
	// require.True(t, errors.Is(err, chaindb.ErrKeyNotFound))

	block := &types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: st.Block.GenesisHash(),
		},
		Body: &types.Body{0},
	}

	err = st.Block.AddBlock(block)
	require.NoError(t, err)

	out, err := h.handleMessage("", cm)
	require.NoError(t, err)
	require.Nil(t, out)

	// finalized, err := st.Block.GetFinalizedHash(0, 0)
	// require.NoError(t, err)
	// require.Equal(t, block.Header.Hash(), finalized)
}

func TestMessageHandler_VerifyJustification_InvalidSig(t *testing.T) {
	gs, st := newTestService(t)
	gs.state.round = 77

	just := &Justification{
		Vote:        testVote,
		Signature:   [64]byte{0x1},
		AuthorityID: gs.publicKeyBytes(),
	}

	h := NewMessageHandler(gs, st.Block)
	err := h.verifyJustification(just, gs.state.round, gs.state.setID, precommit)
	require.Equal(t, err, ErrInvalidSignature)
}

func TestMessageHandler_FinalizationMessage_NoCatchUpRequest_ValidSig(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(77)
	gs.state.round = round
	gs.justification[round] = buildTestJustifications(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)

	fm := gs.newFinalizationMessage(gs.head, round)
	fm.Vote = NewVote(testHash, uint32(round))
	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", cm)
	require.NoError(t, err)
	require.Nil(t, out)

	hash, err := st.Block.GetFinalizedHash(0, 0)
	require.NoError(t, err)
	require.Equal(t, fm.Vote.hash, hash)

	hash, err = st.Block.GetFinalizedHash(fm.Round, gs.state.setID)
	require.NoError(t, err)
	require.Equal(t, fm.Vote.hash, hash)
}

func TestMessageHandler_FinalizationMessage_NoCatchUpRequest_MinVoteError(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(77)
	gs.state.round = round

	gs.justification[round] = buildTestJustifications(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)

	fm := gs.newFinalizationMessage(gs.head, round)
	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", cm)
	require.EqualError(t, err, ErrMinVotesNotMet.Error())
	require.Nil(t, out)
}

func TestMessageHandler_FinalizationMessage_WithCatchUpRequest(t *testing.T) {
	gs, st := newTestService(t)

	gs.justification[77] = []*Justification{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}

	fm := gs.newFinalizationMessage(gs.head, 77)
	cm, err := fm.ToConsensusMessage()
	require.NoError(t, err)
	gs.state.voters = gs.state.voters[:1]

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", cm)
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
	cm, err := req.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	_, err = h.handleMessage("", cm)
	require.Equal(t, ErrInvalidCatchUpRound, err)
}

func TestMessageHandler_CatchUpRequest_InvalidSetID(t *testing.T) {
	gs, st := newTestService(t)

	req := newCatchUpRequest(1, 77)
	cm, err := req.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	_, err = h.handleMessage("", cm)
	require.Equal(t, ErrSetIDMismatch, err)
}

func TestMessageHandler_CatchUpRequest_WithResponse(t *testing.T) {
	gs, st := newTestService(t)

	// set up needed info for response
	round := uint64(1)
	setID := uint64(0)
	gs.state.round = round + 1

	v := &Vote{
		hash:   testHeader.Hash(),
		number: 1,
	}

	err := gs.blockState.SetFinalizedHash(testHeader.Hash(), round, setID)
	require.NoError(t, err)
	err = gs.blockState.(*state.BlockState).SetHeader(testHeader)
	require.NoError(t, err)

	pvj := []*Justification{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pvjEnc, err := scale.Encode(pvj)
	require.NoError(t, err)

	pcj := []*Justification{
		{
			Vote:        testVote2,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pcjEnc, err := scale.Encode(pcj)
	require.NoError(t, err)

	err = gs.blockState.SetJustification(v.hash, append(pvjEnc, pcjEnc...))
	require.NoError(t, err)

	resp, err := gs.newCatchUpResponse(round, setID)
	require.NoError(t, err)

	expected, err := resp.ToConsensusMessage()
	require.NoError(t, err)

	// create and handle request
	req := newCatchUpRequest(round, setID)
	cm, err := req.ToConsensusMessage()
	require.NoError(t, err)

	h := NewMessageHandler(gs, st.Block)
	out, err := h.handleMessage("", cm)
	require.NoError(t, err)
	require.Equal(t, expected, out)
}

func TestVerifyJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	vote := NewVote(testHash, 123)
	just := &Justification{
		Vote:        vote,
		Signature:   createSignedVoteMsg(t, vote.number, 77, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
		AuthorityID: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
	}

	err := h.verifyJustification(just, 77, gs.state.setID, precommit)
	require.NoError(t, err)
}

func TestVerifyJustification_InvalidSignature(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	vote := NewVote(testHash, 123)
	just := &Justification{
		Vote: vote,
		// create signed vote with mismatched vote number
		Signature:   createSignedVoteMsg(t, vote.number+1, 77, gs.state.setID, kr.Alice().(*ed25519.Keypair), precommit),
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
	just := &Justification{
		Vote:        vote,
		Signature:   createSignedVoteMsg(t, vote.number, 77, gs.state.setID, fakeKey, precommit),
		AuthorityID: fakeKey.Public().(*ed25519.PublicKey).AsBytes(),
	}

	err = h.verifyJustification(just, 77, gs.state.setID, precommit)
	require.EqualError(t, err, ErrVoterNotFound.Error())
}

func TestMessageHandler_VerifyPreVoteJustification(t *testing.T) {
	gs, st := newTestService(t)
	h := NewMessageHandler(gs, st.Block)

	just := buildTestJustifications(t, int(gs.state.threshold()), 1, gs.state.setID, kr, prevote)
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
	just := buildTestJustifications(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
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

	pvJust := buildTestJustifications(t, int(gs.state.threshold()), round, gs.state.setID, kr, prevote)
	pcJust := buildTestJustifications(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &catchUpResponse{
		Round:                  round,
		SetID:                  gs.state.setID,
		PreVoteJustification:   pvJust,
		PreCommitJustification: pcJust,
		Hash:                   testHash,
		Number:                 uint32(round),
	}

	cm, err := msg.ToConsensusMessage()
	require.NoError(t, err)

	out, err := h.handleMessage("", cm)
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, round+1, gs.state.round)
}

func TestMessageHandler_VerifyBlockJustification(t *testing.T) {
	setID := uint64(450)
	// data received from network
	data := common.MustHexToBytes("0x3b1b0000000000002a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001d032a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d5be226e7e7b6b54eb6c830ab47d4fa29bc228f46f176be79bd9b687d01ad7c4441f0c5b7f489462f29ba1641672519a2bbfd9162fb11d646bf1990b0c858e0e026905dab6c71c2a664e9ca8e4f066bdee9265ec45b7885ab14a797ffe1bee362a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c393865ec446b236036e3c846577c930568ea67f1cfe34048cb360d3892d9da61c4b1bd7ee31df96907662e10dae18646ceec91181c5e3dd97605b15f0bfc20d02aabb29f640813f718f1e7495f42415f742457517d536ba5de50990d182df252a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000d68aa7ce3c3902cca498cf6c0051d5b0f901f22092a427faa23227a19bc55eda954191acba6b5c82ec03543633facf3f84176ba3b860428d0e160b5a3a2db0802c70dcaf367b35740713c0e6761d88a2e26f9762e9715fac8bbeabedd8c5eca2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600756e9f4187f078001a7368eedc194df7ac7b1cae92827dae36d345061afa3d49df2041a41387501c5fd1b282707303aa3a7c820cce1b44c081c0051e7fc229070339088379600f5bd507277eb894aba34d2a8f27ce1964f41e60ef2a4142dc6e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ea73d78ec38c271aebb2c9f7ebebb5d3a99edf327bbf4d03b5c245c08d0c2cceaf0030ecb5e04152255f32e4edfbee970f3ab05607f6d8b718826e559bc18b0e03d105a30087d96b5f0684f6ded76f826b01dab61e4136e1d851a24f0088b5ed2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600681cd145a0375e5f72a737bbf92bc57c1e2b9429cf8b66bda13d0096e1898385ada09ccad93971530349592a430aa61331efb46c04a1bd9af7c34d5be9f2d30b044a5968644cf9f9b5d1c7f657a65343af076730a89b07b4995a494aaa0c967a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600620c51368e57db2cbf7e1edee0a746dc950c6da129ae5dbf81f3171fbba241f564b847fe376bf7a6f62d9bcefeba69d7d8b23000c40b7c9ec4f2868e6d748408049603b355867a2d3a1978a771b71b2bee4e8ae1f344d9ee034f47b01a48f89b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000dec8ecfc90b708446f112ad2730149b124d1d4b891edcc7f192ad4f6634fb1529471e902b590049ee6049361537752f8b4780e1900f5134f101039d617a710e069a481209b07b48c1d548d4af68b7584f95b32eb374be0682b342cd64d3406c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008f6792c7ae77c3dbde17b000c86743bccd9f2247fb323044b12378a0b060f122674fc796d910fb2eaaf7512a5858ce467da7409a2e5fb14edc077c939efcc802088f8736e1cf2ea3d102f0a96ccf51222b8aa1f93d8e42947892f4395cb5477c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460043286aff05190884dc7be3f5b4e42728a1eb766c73cc9745f08322ed672ad944503ec837a379efef89b4c78a5aac7ffe8bd26e649eac482f5324280e259bcc030a790c3ba374fdcf1a1d457adcaa8fc01bb46eba8cae4e1940825364fe2d7ce02a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460025437e042850b2648ecd98937694a76d041c65b471a81344c55d9aae0650459daaa5f5e734ddf02ae030178ee8714c5f6cc241f8148eafeff450424fe62548040cd72d0e6daf0275f2eec96f6129243e01e255ad6b104ed500e614077d0710982a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600fecfa0a948e6cdb687849ea41dc46083ed8562ca4657d751f6c303ae0e35dba38e5c1550cfdd71011239872139770b8151eb4a04df948f666d43811009f12f0b0d49982b323ded139008e150fe97b0e85213a92b411c68f993aaa43ea62a7b282a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460063dc716d5616a3125fabc437ccc98b3e38bb2730a2564a11edf528552245f20657199cbc982288e2ff3127497a8a725852c35352ee3c0a2109d2e603c33c37000e1b3f7368c230b609c745aa7c2119b60218da3ade7e6a0dace3e337703c51992a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a07873b9fc9d6b71bba7749a8886a13c8d5f4c2b66907bcdf5fddbd88df3c6ece1d601799af6ffb7fb53cece0278df81bd34676ecd20a7494515eea9bb61af020ef0093d298f1adee0947ea81400adf0c27bd2e5c10b760d2c8be3b784e1fa002a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46006bc6c891010819529a51deedc4a7033de5e85aede0eb4103118110239cbfdad07327aeafc68acfa3b11872770653290f50c09658d416cb2534f0794ddc4b1f0c10d20c8996f5448cff82ded879bacf7534e4d499ec65f040024bd7c2402eba942a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460090c37f3814f7b66ad1008f43715aa91e3f7713285d44c9b82b88d7125a5a5be38f9c38675891de2455b276ea008321b742f420baa7b2ed8519918edbb8b9b90311addb386a5088eaafbe4c93e265b8c1459a8504f0d31a3799836224c6077d532a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460085114883f06b533bc6494706326c7191f224f0c086d0a21c8534269c3dd2d7a10c1beaffac80e05250d0c230aa3da0ebfbbd4fe47a8323766b09a8e3ba797b04124cb8ef23a2aa6f5d748070a84465b1cf4df73db94257d96c63df4b5ea8b80c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460037c148f4242ac604913f055679371442e9fdfd7f989ae9d73dbd5cd439a31f941e5c4a63f315a3b4c8faefe537dd51fed26ed948d5cc0110c8c89775e5a2a70914b432d81cead21854d70b19f9691b3d2a8ca7862b64a71f4f3172387078c4642a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d3acc5151d7a54944aa00350c835fed6f9dfb4f12b8124903d301a959bd63f087d19016a6f18b732d58d37f93ccdc5a9a27511054f54436dc8bf8a12e1a06c03155763a153e0c02b2eef1d8a9cd8a50a9eaac9d3af7ad1e559b1e2320b521e1c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46007fd97784bffbeb89e3fc4053e6667d73a24a15e5a864f8264d544f93b8d4938a5e57b079247499e1d0de9cfe038754c930d1427f491ba0f8a956c56b6abdc409172fa913b2ba5833a78ed51e13692e9d1045e7516b5ac738e6745c217f49fc1d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000b091d7bd11344b81890eaa9eb51f38fef522d5b38e05449ab04cf720c389c9a9669d8cf69411500ba2a78a4f7783d75c424b0ca60e5a564728a7908b90dec0918a1a6e0e81ae2cf403c328d1195eb73e9b78091bf1e33be4dc1e29736eaa0382a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a3aeb6a64836a35659a253533ca38c5e8f5dec712d764bbbfaaf079f671c4487345d232be14aead855a06873549f712ded1127217560fb315be740e2071812061c67ae7a3c999b95732bdfe95fa8670aac497a1027aba327a1438bc097fa700a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009ed7a0e3b08dafbdfa66248a9abe9f6bdaad694b2c8b1a85c2fb8ba387895d11890d482eea4078d1049373c9a7d95123093e2e63d4472c186c18ae7f47c310071d5ca234f8eeb85fedd882230378855b4e7390a78a802f0be7230156a8ce624c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a5b924123457be20ebb04eb12a248f693620331120a053e1837a343460769d6edb3316966f5784ed15e5e8f6105f8af34ad0bd6a8f11aef6f79eeec6e658090b1dc6da633d00a38d0d0b44056980bafbd624f2e1067027cd2b09267acfb70e9c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ed204b31f36a1c06fb6b69e206525afbb5449064e39aa477602ea878ac65a4e81e24fc35c1a8ba764ed1d75200a2c5681a43da66a8096349365a9302706d1d041e0695d7beee6c4a67c9a1a75f89f3f858a50a09ce1b96a08e685a49e20e7e442a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600124362e041e4e25522fef30f7267b7d52338dabb1a2bfa21d163b06df2261cf465067921371fa5be83279004e174d5cf5bc79102e2208139bf62892c64521e0b20d4e595c50bb9558dad9be6a8784492bbbfc754b9c5fae17edf4f8a84e8b4712a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46002d19070e68ddfa34d42b207919fcdf1583ffa6458a82e2680ee5c880ff145e4b6a18b00c1c683c511e196d7b24cf4abd508f6522ee3e8eb8b335a42a0f7dfc0121699d6780c6aa58e433f27d36b7f1ea5afd4fa404d8254bd62ed5b17c3f75672a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46002bd9ee43aaa4cd4691315b505af64f05fdb9d70e1fe5f8c7d9e536bcf723b67035239765e727d4e1146c786c97b1e900cd33d8c25440d9e38907a6d0af3d0602221a227686d900f6be260a84e81bab79adcac88ee2ceb1f4efe83e0be86fc88a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001b561a74a01dd3e3957d7930fae671d9a107a0e36b1ab1fe38a2acecf24ce281f0270b516c0d73ca05c1e7e7b6b5870e973ed04f8b4b2d8f486731e2871c5d012275aebac33745b7c0f47957b0f9c1dab14d6dc5cbece544e26830248b9b638d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600f394746cbe78e56e004009e2f374bc8fbca67b5228f33d328ba173673bc7ebdd31669b9272de648f7d19b7b79ea68937a6bcc395695c03c8afd36647bf20f70422b1625c123c72fa70ee375da98f15cf1326b254ef833a27ffaccbbe81e993d42a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460043359e329c56b481de900abf705e49957358bd8261d5215ac63a96b9098eb07d68b12aa67f2e4559fe6fa838233795e32e9f08b1680f89fa7db4da2cbf40010d23192a82e612c4b047a6efe2c0b20d6b8300a81639a6d1578bb94a66a721649f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ef4e7891ef4f0703d2ad1833ba02e1ebc743fd656dc4e0d5b2e9346694f142ab262a9143fda3c3a199e8f57820192a49648a0872e7b1de04bbc101bcbfd0ab0524796621f90c2d9f2d12a962e14f9852757f89de7b1f66c22506321ef38266bf2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d65939fa1869e5e2e2ed5800cac361424eb5449365fee6914719dcb768267f37be9efb0a21b1720308dd5961e3ad43b00eae969a117b8e1121ba36362f0b220225d5c2769f768059b45c6843d37949ab7866678eab000a5e6340e30d1dba306e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600aa0831e40a2cbc1ba04377e7cf9a0c6f5fb037c9516e4e9cd4d6ad0564953b99c8ab329e5f667f7fdaed70e9b8d3b4123e6ed9249b4e999b41dbb82ef04ce50d25ef09e15fe616e090335b55cf729832e2c9f562428fcb00fd3e3bed2dde843a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001396a84de1c3a492a93b29d74c536ea8231bd01657a1621a3af64b5993383781947c771948a7166dd9d21b4c34a36208dd102d77f3f6c99ef85037ccd4fb5b09271f0ce7fd9e0460b2b5f68afb847a80ea0821bf168dceeca396a1fe35754af52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600804dd690fd66f6c15a74a4c1c8be43b57f0eba4d45d75f66777a5204a506c6344e189d33c41ee6085420683807345324cf24602b2eac5a307603a1dfeb71040d27a748182c49d7c4ec091c9589da5c9c35fa99ac8ffa4d613129dd4d08e9f1122a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c91a2c346304d33ac0121707248a0560bb32b4446fb0f2d29c6c4f27f4073bf6455e662af07a4c693f199a1b850fa794d45d0640d33883e08681b525f99bfd0627e96e8217ce52d55d5cfe97580a03b8f9ffd41e72ad863300db45749088faed2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d91892fba8d2b41a62f81cc5dc225fd193aa215c9941dfacb197f7b11262609953c57726361b6c1458d500cdc90eed86c0d9ae4cb98ec1457a438af9e706d40e297b86aac8a8843ef944f9633404cf307ae264f031f6dc803bd93c7ea69889742a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46002c85b6e9e5c28ccc33e4d59857cdc0afd0e4f4d1c2218dd018123af7d96ec3b08e49ca35a5231431e2663db3a178dbbe8a1a4f4c0005014770c0cc85876a88062ae745fd0181edf42f9313bef7ddc5d62ae8289212df03f8f73e210e0ed907562a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e4ea8f7d9584724897151d5a1412a73e42c96aecbd84020e52770c1767e8f807befe8dd28f7fc8cb12addd1f1b72d6c58e418702111cd708ce7da9f61574d60d2ba3f6bd3084ac7d9448dd341d4c7ca8c991b659c85f27cc68e80fe7cbd73e422a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008c5d3fb36a8da0e1770f923bfbac75470ec6408c06f0cb1be5cdb5893a2c09eb1e370073aac480ee2ec8b9c3a9bea659869a4328e8e441851bfb846610ddaa072c3ce4d7102f4236e0df2e8f5797f6dc3d2e6f0e57d373c9a4b89b21d4d228682a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b6e743e03f526851e054ff4766ad6db181f83fbe2370a51431fb8c310f1cbf3788d99480e9fffd8164d31cad3db297d9b26c34523aa25d5c7f83e79b6c02390c2dad5b2212ee688f2eeb9ca1fb6a90574f006dc1c6680ac3a8523363a248940b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460020716a5266a471356b3780eeaf4ad7dc37f50352ece4e1416bbe47b5b2606bb59dbcb1d270aecdfa6eece9d3e8f0862966ea7c7643cf1c05945fc4a56ac65902304d970cf7e07987f8d7a500edaeb1c0973de1ca588512e5d9f268a9ec0874ad2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ed2d6b378c40cb99f357b0458085a0b0504c1622adf0316050396479b08ef1859c2978fa15b4e53fe911857add9ec253cfc8d6b949abaca4437824f324c1070e360e744fe76d5d471445ce6ad9587af67392b8d960d7715dc0efb43c698465a52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600013ddc65f819aa4c592cbbd59ab1f4a4bb27c3a4b717e19d409fde187da9e2e7dd6c9c72ecb1bb772c112f3229dc5140c202ab9341d04a14dc2c24a2b4e34a0936ea5662f48dd131b91defa20bde06049edcee2982714f78519fb64450d1b62d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600087672b184ff663524bfc5d05631172d3eff695a117cf6c8884053e97697b48dc255835918bcc10f915ff4d8fc3470ff5143eb433c00e2dfc8b46db2ff16360e37035dbee1de6bef71e401cd19d0a26cb8b69cf719ae340ea53c9677d6a9aaae2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a2d8a30c22e7f78ea1a477a0d7971533020276d5d10bbbf530f31ec5d9ae688108c2b4535dcf9204604406f21d104131e88d9c8399151bb67c14a790945ab30439712f967c4e06d284d6da4735cf2823c1f770674b668f7bc896071c3fd41eda2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46007770937b4a1d62bc1df63b1f80162728b198d26492e93765025f9c71dcf992482c9e85ef9100f7fb89931c0d2fd82d063e24f7dcb3d28e549d1d45d8a91f2c013a2f175229490c3169b5942260eb2572198b494e63c984dc364d3f48aaa7bf012a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600f7f152e001649cf1353540ca716715bad9fb6703c1f34b78673baccc8027c3885a6c9660cce85fbf2b2ab696220a2e417964c91912b030a0861ab7ca541daf073d75d8fd47f1074a78cc88248f1f6b9ea6cab42ccb676f94226e3fb5e16249c12a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600dc58eb9647b223730213a6f217010fc35556325166e03edd470b04b9b38f97a98acc1695a71120f06b2beda919889a5c2de43f40f9954894ba861cc2082393003db99a9882de9b7666591c1f9f2d87685a2569bd0c611358d6e144a014c5612d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460070846e878685dbe30ef68636052fcb92da91ff77ad2f7839edab14ff2f5a28cc27e0c464d175c995647ca3a0cedb2ec732e5732789ad3684ec4f08d94034c3023ded8aac5210831cbcc4d0cf1f96f2c711afc90a6f6f35a4ffb766be1dfbaee32a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600674883cb865697effe33d717b63eb6746fd144e8dce5df79892d453b51fa49cdce9eee1500be36a245e33a9254ce65631b0049218fa305c2e9d5ee1a8855b00b401def7965bb1e8373fbbb6561d5ca51832cbb92b67b0bb08c407d2c19cca96e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b4d95db0f1cd96e18f87a0c67e71cf7b8d2facf75d10671bbbc9f7e5642a4c2223100eae4e64a54ca179525b19a763fdf624d78e209f7268d7cf4cf70788200f406c4081abf0f3a151dcaecc6b7ca1a7489cfa1810cb02c8cb249bb67dae09302a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600bda3c89a9f0257273f01c786914fcac144189a20db45d05fb42ded836fe2067362fd1baa2ec4727a7d8d72530474f60a39070cb0f2af31c372a511c55ef2b20a41ef4a31eb7dc1e01f4630604e1908e644d7cdee3f66a60f98d6d59605326f8b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a8bec76286b65de5693a2cbc309bc3c301bd6812eb493113449e9d30d39b60ee58a4d918f127d54522e74d6c6167ed99dc3b10e8aa64f3e4e5a2a497e68d9e0542d0c88f0accb5117e31bc057d4f277a38e01f2325f02d0a9f647db09b67cf202a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a243df7ec7aea8df7944c4474fe09c1a2a006f697e29d15300da8a51ac7258244501ea6c87983d0ec4643c051ee1489087a3017a63c2ac3b7d06aaadb1b88a0943701217f2650ade985ac46e38e548bc1850bf3d895eb6c1ecf42d9e61b788c82a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005305ec8c29ed3ef51119c80c144ea8714321a31e533c0ed9a1c03a781e4401a279393f06e4e0f0dfd7f21b523a808e0aaf659aa587063d7017dab6ec8df7810543a9a2915f377cd4943f602be38c3bd6ef39e91562c09f18fb672884b4bf8eab2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600fc691469543554a61c6bfbf5d554ebf7cb596ac78579aa1b8d6b0c7e52f8295441a4bb450d8fe1d2579a51986d3f7a6de52a34690afe34fe5699a6bdc1fee103452185859ba92a24c1f836952da8d11c9425c9b35eec979744a345d2276de3ee2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600df5b15104aba6d1d4d70f19eee2d04e22056616f0b54011fc3f15449aadd958124dabcfac319655f72688f14130808f3dd12653e4f948034398589143c24fd0d4532b59911aec8842fd910a35fad7c6210b3c1ac73c7c9799963c635e2b562882a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b11b2845d75099b35ce971277fb82b0a8f2be06f40d2b759a291a095b3ae6f9655279b922667119308fc3d673e3d6f0063bce32f0b2c5370071214ef58b20106469947dc7cb086bd75216dcff8c8ddbbff5e0f112ba397d71b6bc2980bb6cf002a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c589c74c41ac4ba4e067df4a152dd5f34706f74828bbc1166a7ea1863e411058683a6d3b3c27db8540a8538fc50944dd9b3c8dfaa5f49fce6c5aeed834d8d504469e0a875597428e25f79157e441e933f3fff7d2afee478a9ac6e1903e4d4d1d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c0a2143bff781fde49d6989df90169579c588192fb669219f815f47117459f9a94ddadb68a90796f95186e7ec641e6973bc165d0d8b2fe651c9e563551bf960b472bdd18c1be48d5e3c40aa093db02e45c4bfe1a5b62a3884fd65be8eca3c6022a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a5f5af2141ecc0f6a323988e7d8ecdd8b609fb9379c20d108e218ff6c8aa7fc91ae54bde791fdde4d103c43299c033a649e69c8fc933db42b858a3730dc39a0a493c604e7f0a7cd6abf370c10b90eb8ff9d432bab420bdd9fe3da656d5e9b6752a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a1f8ef2a102fa613697a17e7135f7f7e429107d27a74ab15f0f2a46afed1e3257ce683c09017e25cc7abd447e06891350568aace957b76c5a2a8cbd9c0c8c800494d65e6b674eee86acf5ed70b45b37d7498b8c1d05cc0baa0d6473fdb596bfe2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600eda30d06498bcca078360129b8d46b6d8af3b8b9ddb252a00dced827e3b4c86373e299dae8720f622be9bab21766fc970ad786bc5733ec208b0105910b993c01496826b538d97906e14d417b48598d7d591a483ab5f4c6786cd0b96239c2f4cf2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009aed65d04eaf5a5c215e4f3cbf9144b83bcaa41c037e42ee788229ce622b28d3b2ec91c3b1b609aff0108bc920e0f6f43266e1b47598bc99ff5410e8af2aff024b35f8352f2aa4dac039d347d37488de305c7ef6e7aed2dbc3526f537efa2a3a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000c18a3ea31cd3830207a76b80da87192c61c608490c9c8826d4c860f8b39b2962d258dd43bcac5368fc2d9abb0793c188f6197f0a6d4c3346d2e53c6641086014e319a863a469525687eb7dfbc8924054c35d0599b6ba4dc94702edce01671a22a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005bdabb13bf58fbbe4553d4af8f8fe9ebd0a4aa99e0cf8271621c9975aefa2a5f35ce96bd2897eef6044c3eee4aa475af4288d660c1d64576a770985b9c74890350811bd4b7dd8a5ed9193d4e2d19248020c3d334249a809fb96234f058ff90fe2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600bb6766e42e7a117d09953e5b2505c6720b77655c557be72254b0d946dd0670b94a18a9529dd8c96404596e9b6161a0a5e94f5bce6c190e733da9d4b5504af60451237bd7fe895abb96631987f0f82a41ef6d56f217dd1ef8aef76e9af4e559862a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b1042015c7048dba6b6aec2a38267657d0f221d0a862514b91b4714dab337eb41a014683101b18d98bda3cb4860749f10e5bc3a1927946e1c96c3c937532800b51f980443e5fb020d8498ff88e49fc55f1581d7eed1c66bd7653a4f5a1ba18662a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e17cf09d6059f3140f99a05905c17f597bf7fd8685db6c61fdeb0f91f0d3686a1e218daff23f1964ca2e7f983efad0d0714ed9cb5b23dcc2d428c03cc10f1a00543a164d12ac3f4ce00b176cd41a7af343c56e4ff445a2634d74f3b182c9c7352a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46004c4a8f2ee83d7226b58b1415f57cf93574021a6e828b1852b30faaf97fd75325424bc67043a889d41b4f1010e40e2ce911d02ffb252dbb1f9b15f0936fd9cc0f56682332630f5dd42f160f1e3475d1881b2adab83023216892f538efa1e0e66f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460095af4f1a0b89f8e6458417dc977f40bd63d40cb16c2411f5305f3a12cb44997ddcca4c14c350296b4d2a30cafb1e272d22b772b0ba1d85ead112b93d82a9980f57a7cd79d0feac648df60b94b7eccde724eb7e473fa2368eb5b88181b030239b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005cfb9e1d0f208dcb64abae8d57bf8bbaeb4495e61da051a28d6247ee4eb02dfe0a5bcbe63a8c7869168caaa8da5ccad40e1d0c64b4c802cd9d41133cdd88f30557ae0c85ebaf333a6ec3251f577cab910cce072f238d1e50046322b83bbc0dd52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460051eb234aec184ded0063029767fcb3e32d46bfba803f5301e932a4ce8ebfaacbfda1480f48991d6259401a5962804e2949be93d9f96ab7303b7367c6dcf46c0f584b81d2c35f01525cb80a02c259565f0becfa7d1651dd4c313358d339f32d472a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460061ad5b7dcd93f5ac2673bc36633d2a612584685c1ce6f1bab11561a0135ec7b3c20e2c88ae6ca855d4109f6f9892c41ed14bc38b549f00f823486a3e81b22f0459522ea548446804370cb91e2c4eccd2599a018202b0e6c04b7643b143707a382a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600278975698f785c3012217bffdfd48882e43033099ad968e849a14a962e2d6e538d1dacac16df19a0831f52daefbd4ba7c6af77cb7b4a343c575382858d1bcb07596004e838b7d90408d46de62b01e13e631575f348bc59f926286aae4c88702a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d8aa4ba9183770f61c39668721066eabe70fd41d30110134ebcd703a7bb4ea5f0288e28ac58a12dbce2ace9546b0fc28f38c5c467a33cb0de07ce8c5ac14a30b59bba625d971d505a9c7d7c2f3ad69203e69e3e5e1fc1e4905fae7703fa19e032a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000d40e4163b524c69ddb59641228c09fbcb4bff1ed875161e54bb10bee1ee21e618ff0715d5fc9e9ed1f32dc49e01926b4728c8b2664f2aa535929ba72080c0055b29e3e31323bbb46b3d1c53639ef72499c58806c7f4cb6d2e8343e961bb6e3f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600bd7478f49bf6d8d87ef834bb9653a3f0bda54dc653fe7890c69882abaf429231fed0b99dc99d2bd4f504c32e2008b135004c60e2045c7560e0c0c21c144925085b379072ec1f3f70b4650979a47b24f9b080c03450f7e9587d92cb599fcf4d6b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600117c8995b21aab23931a24670b8a885ad04493aa804a80b51572f5d29cf04d6e7d8f9adebc0cfdb1df0e4414a199398fab565d55fa0e32d9f70b828de39fc3005cd0c7b21d4834c5a41a80f7d421c7d0297e42eb409a524d33aa7557df13adaa2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46003d5d559b960da2c238fdb2609b45d125f1940fc8c6e4934d2888080b5f2c7f7da8c34528130afbcdac99c0364b88b5a329c8b742bbcfe91c7e96556b03e70907607b38d0d0b1ce290ae681fd3a7fe09b4299117c3d2f9a0a52c8a3076e268c132a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008152931c4a5feea2e39f656662f3515ec32256c57aab6ab75c6f0b6f5bd50b508a743ff424f2eb046f1a923486a0566034023cb34413172ecb1e639634c9f00d6088164a2d219f2069d1d51e06c2b4eb70b85be9071afd6e99a92f1232ac645b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009226a52a4f9fd97a75c6c808a961744e731656160509b88038c3e7ba2e88f472b2f8fc5f865043e76558fabfd6b28dc0f8aad8a50d5744072a4bfa9a3d78200b62de44b8fae34cdd6d4df47fa7d420bf8513f35e3f63c66c4bf699675edc33e52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460014c6bb8be7effc1ee39f3cccad774fc1e71212f75ce887f1b4bbd7184de4691c9c1cea72845d9830bb86b24464fdf7286936ed6ab238adaf78773b335057b60c64bffa0b9f160a67beb033610656b80d6c9342797fd983375e27f91ebc7e6e0c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009dbeeab2dd1d70ae4993bda61bfede77c378865b264b9620d17da5320141d749bdbc6ae1b2251e68f79ebd445b5b93d8a5b2fc5a2257c307ebea37f1d45c740e65bef6e1f3f6609291199f6e0940c80e559588368e2031086afc1730e0584da22a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46004ff39ee2863945517eefde66d42c4ec96da046097b7efbfde7c3111a6c06ce97f552f33374f43a723709a4b3a0cbf6b226af61515805c74fce2527bbe828860b665da232ade423c7280c6552e6d0ec8782f9bc742c0b030fc08e9dbd3ca5d3862a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a363544def43bec4ab3cbf8aa4b637198f7cc7bd89c2a402d40c6c10aed6a9f231ec0ae946d8d21c4563ccbfc1e6d6ec73063aeace3397cfe34f6e78e378220666ed84a2c0bff828dcdd1d82463f642666dfeade46c020621490886912bf5e022a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c2ef4b28110bca12507a617a321e2e5e3abedfa0583e4bd9f320b02d5984f62caff3a28a627819a2f2d54f01f80713248a1e5f0bfaaf8b261a3ceee530b89b00677358fc648638cbd854d2a009dd39b8508dd3047d0c5f13bb403a64d053ad032a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009d76b8b8deb037178f7686154d9228969aa30d5751ad1c2e0097c97ca39f3686a91afba09d50e443f75a8ea25f41f1e63daa1a5e0026e099f9241ba7aa38600068f8fcc2977c5fdb4ee46f234fdcec2f60a22f9c63ac7091b8ded3ec441df9bd2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000b027d479cd33d8d57c998621de2cb6dfffd320f53d27585d3db9aa44c59bec3ecb584f003e5571e488cd92a3485ef2b740c27861a5d780571dfe10c07307a0e69470b335262280ce95a164f7963af49e41ad6173f8db9e3faaf3ab54a8c50322a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009bcb275c92fdc035bc43aada4277e97139993270fd0ab3661350adbb09304b03246c37b02500355654ced15912e9431858d85a741d29f5f9ff65ac597fb3700b6d14338abf210592babaddf4a584f2ddc06c0d333ba8a9f5e284c3be59c828512a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005c4b0f868037118206170a6b1d858ea81047028e10bddbaba24b1b6722ffadeac7a10dfd3cd1c2cdfd196ae3da5da59a0dd858f6ce15aeb884d39e100a6619096d28713cf7af6e13d24dc67d4540225f637f0384e58e2710d9e294e7473edc2f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600cad6e6d6beae1a9121a104e512d5d9d89f60d0aa1a14e89ad906e5a875cea4d6290956f0f6acfca1f0faa651886556d59ad2d5311f840eda172fdf0fb9ee360b6f7a1a4f9d8c054907f3b6b946a59525199dbcffe9fd3ba89612a5d4b548bede2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460096490d82522f60cceb6f64fe9d7be13cb9d3d5426f34d9c6dc7cc7fa10b51705ad8c0fb4cb8d1df8520af798f4cae294b765f4c918f75f8b80a9110ed4fe2e0674998004d06285d2b8d99a87a9fef38f0fc3109f4919006b8d1831b0dc59b3782a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460012d3db9c5db15a349f1152cfd3edecc7cbcc6b7d6e0170379bbf12e29cbed42c9cebd9bf55832bab8dfe4f761017aa068be5f84101c4d7bccfcb929b34c9ac0d74a47c40733f76b6d0502b67a709fb5e5cbb263b180fa0fba800d9d6758207362a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e1923130437dc6a03eccefc51c3717c7f95851aa9ce0ad76ba0a22b5dc4dfaefa9cf456eef9a10b9ffa1c33a98cae77d6b16df72b1ec7bd09b53d62dac15d40874a8906dd2888c9e7edea13ba3c42ad4d833f4a5de43d5ce0d9b12c654ab2e872a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600cc61c46e1babadb7667708e3ed03990398d9bf080ba41c1fdf67b4f51dcdb7e991b6918a6aebaf36df53f828f950a7698a21eeb1c640f809281a08424776cc0674db730277b3ea5a14e52fb0ebca202e56876c666aa160f623a4ca411b7ced702a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d9838ce8807f0de4ac9b15d648031b40e5ce436051756b3b4c852fe07c67715cab2b3cc237fb7f72e95f970760d9e36ece5107e07f41b43dafb771a42517d4027589ac4c910bd1e08bdb36f0236c05cd20349d62e07bd34d89afd21efe56b5fd2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c3136be04b86dbd5ea6d9151f51d16b47aab73e059569dc775951b16a71ac4fa890bba142e61bd67470cd1fac95ed42da2924bf82826caf7c70c3469acba540a7a92827270cfa82f16145a44f9bdd9ff5038ef1b665dd520a2e61db9749094962a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ffac47e7232095f25fdd02a1a5dfdbf3cb9f9c999881194fa50ffb9a6f6cb00db0897043b0d6542770710c727ae5dbbd231eba181ff1c097ade5be59d01389077be4c78d01b0569bdba6cdf39abb6e02f591942133aa1db230033aa48d18fd552a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c550613a2aa3703d95d343186fb5b9ea3e5dbc71389f842cca278d8a83f7735655df193eaa580de1a0f02082385ebf6a1148d93ce610a165135448a83e6a55017cb4b69ff1f333baa130c02d83fb533cb84fc47155a707b7978e8455431f4acf2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600113efdc304c9dedd0baf19c9b54c235fb707a91a7b6730fae65d3860e960d6f6e757109e78422f7d568864de98a9817fcd1d1c435c9a73f13680525da4694b027d32388dbca301421fe038986d97764ed933927b5b74b91ea5320371a31ef12f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600cd7aa2897e1d6d13af72130d66d65248a25ac7003734db33c3d5c59f344ba6bacdaaba9836f959a6ffd140153f8592a0a30146a9ddeb34b75184c17f8cfae10d7dea0d8ef4c3c5dccd46e8d2af6b2a85e8b36156cb97002a43ebee8229b82bff2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460097ba00cbc15f7d2e6c06f7113afeb36984a2322f18960c5fbb8d96833c6aeba2f7eb6f2b485aa1e470bddfe489527dd782b21d3d70c6a49321202f763bba82007e7ddc9a34f7d5e0acd418405ee795740b3a004c977a9ba4ee103755c05aa76a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c42a2e034b4fd25cd46ffe153c8ff0f6cb764ba36c36b2dc125ce88d8593e5e8ddd636b246e1852e3fd20dee2a7d5b026b89aad6cc9080b4042cdb101b05ae057f406fa8642d4d3834434465fe6b1815ecf25a5875e0334010bdfab4768fcbdf2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460059b47b5455aa16792cef813dafa11c7e7955b33fcf0c39a67b85a64e620403aac36542cea1ea2d2015bdc71c09b9c60c33d77bd28c35a69a656c83743b5e63087f4786d6fc06a89b65579200e5d1bd63caeab25893891f06597232389866b4152a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001f8980114bf02490af98132d5fd4e13b65eb990e39e0dff52f9e152754b83a19f750eb812a63a4255d621cbc095e504e54506c3a1399d421a36e77364d40c20f7f57f0e89ff23959dbdffc8f66bc433fe9849f7a9c335e0601dd59812db059862a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b90eacba52195a184b96f5f19948facb9d5cfff012278471ec8c2ff572e70fb3040cf595588bba91973fc2163e509e0686f56039fb15790ba172aa96ee39d80a818a546c630b881c2161588866965649678cda0f4110cccc3533d0f20e5e41202a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d679230a0317ac247bbbb605e6460aca06ff81c1dc65c5ea9b00d19b3df57265ce079a8be66d5d4047f1a5d81709dec9efa2f675536d3cdfdbe68fd426981d038930b6c9661fb81a752c137a40cc8091480743856cd77ac965f4f6979ca8f30f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46006869e6d4ca689eed4551201be9c4fd6024e38ec95ca0a6125e156b95f4f18ec4bd6b49e1ff7f5397a6d6ba77c27866d9d089eb660a0e7efdc61a8042dc4b8f028934b9a6c38dd420aefa1e115c839fad7a71147a6efc77ab593485c3b07576ac2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460038c61e1249a82da4c868ae51928fb4d21f980e96a9d32887fb9c4f41d2c6032c03c54e5e45914fa716c1e256af33758a5c7d333fa780c771e0e6f66cb0427b08895238f7e22e70f3f59aab49819d5237d6776eeb65c2d5a925b62ddc8d3f0c772a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b819531987ad0190e399e2d1c87559917590850870b8f938da3574953e6cfc7e0b6c6fdce91bdb0e0787c3bd2bdf1541eafb86da210ca85ae2bf6fb0a6fc69038c2e1ccb3645a2466ae46d3f3743ed89c0831a74720eb5c4d8490b1e04e5bd4d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460075c07804bad1b0a7f1211718a6d644cb500bc66416b6004f80903b51b3efc2617d1db5175485a2ede4d7bff6f0e8cf067630f11d62af1561032380150fef6b018d67a0c4b47eb0e087f2cc4c3973b59c9ecb729fc960775d76138ab09799466e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600bfeb58aeb94d24af2fa1fd94a4b9869d40982e3f21696fb70dc9f03a9b80f5d6389e4f4f11852e43900e505c282c8c8110545d547fd0b18cbd88707e8a5fbc058f02169e09026a2500b440eb43eccec58a6a32d9f5cd9644fcf26f5e0c9692632a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600601161377f0932fc08d9127736fbf0052b763f365aba81899e05c804a1c1fe945d13052e264c550e3524d6430d69700b016113b65b6e6e73b22918fac8cba7008f371abcaa2d51351b62e5b400f52086ef7803c1a1351563c5f405a6ac054fb52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460082af14bf589502d2963520eca5863fe7498a982e11941fe25b5da9349de59cd245ab8b510a09795ab701ebf008eeaa812c877afcebf54e70d95d45ede4839305908ad289176561ffd2c95f7cc8a2ab5c6a0effd161e3aeb1732140aa501edce12a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c19eeac39b52c944ff50ce5bc10e9726646881c798c6ee4a88de3dda17739fb11742e5e35b5c7222ccc4cf95869e91947b993e668471ab724c2c05fa3a37740b926ca2461d028a766de4efe3d8412bd08adc97c9bb3aa28c07a702ca82ac26a12a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c7eeccb5b57eaa678b568d72fd1459886758029cacf2d33f944d382f8945c56ebfc5adf4322f768116fcce9596469910f7fa214568e2ff581f2e583beb633d0a92c3f7a560edfc7927df1631ff35be263beee400f187c090dd02e4a5801a82102a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460019fe510b724fd4773b7fb1c739f1b0b9313d6ac2f6341f693dd1f898bb10bf53bd09fc398d305c55d598588be1f8b9727fa41a086c4d4f0835aeaf26ed50d30098077019fe554385e0cf6e0f3983b510cb37dea7d42f19ce0eccd86e3e147d352a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46003abdcc324544ba69a1b63ca3db2ac95b35e05d37fa96ae24d6a215db9943ea817a5eae59006784a8e1a6ed3ee554035edc0ca54ab4b04b7f95b48e3548ed36049944dcec71efeb5186d55e2d5acec943bb28d1ee9f84db3a2e2177c3d7ee9b842a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46002a4d4e13d1942c7f953c793731ab5c12a45d0a25826281c2d23eb23a1b74f7c40698bbae7c0f032b74ea44c71cd4f550b9295058f2f1dbd7b6f966b00bea110e998b7545adab70a645393ff37884688b9960c8c04989e8575602625d6cc344f12a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b70a4ca5731238626ccbe30d9e740afa075f92a8d217fd224301d1415dca12556a89e0a2797bb2098cd0ae62bba8c4ff98e7d2089092e729bb27c039b71c15039af39dbe3ec236efd64ad0c099648dac388e424ddaebb0cba1793b1ffb9a5e2a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600dc57933c5a5a2ac87b5b72d60ebb537a7bf9bfa2da6e0d45cd322d5408800d7cb1555298dfe74b847e69ffb237754897847afcc0dcafc80bae56209add6f760d9d10dbeb235ce888f8ed0279a2dea721df2f91c3809999301774fc2a5f4272ec2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b0cdd53676e135af460bc95ec069717e7b78747b262d5140a1a193b855d9fbb159cb499b7fda361fd80b7743d10dd383526b70feb8dbc0b06badc2b911fe22009ec49b1bf8a4c76bff00c0aa80adebe0e9249e37fd977babb7be36028b65b00d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d5cdd8a7eb544a9479ffbb0bc74072ba782768cb615ebaa2e39bed4784b4fab6a86c4db944a889ed7dda78b2c0b5cabe5f73c423376ac92c379202a6814fce069f107b3106c7f7570af198895d8314c9deb526a2311da652e5cc2c049212c0c92a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460033038e1fbc0dcaf9212abe66e29d423b659cca10395f78a86774250986079aed49f81381e1dd793a19cb7e0120b636ccc098604815be2e9e2950a9fc857a0f02a6c5e1d8748c139b8d98fc3bc12c3e3a37ac8e0b85847090093b99d0a3c4b9212a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c9730b655bbad75eedda6adc2f28a30882d8986f28bbb6d2f0eb8f367ae64a9eeb9ff25365ccc11c67db4763dd53f1eb6345169be20199da23559be607ccfd0da6d3f9c048fe3ba16d0edc38626afc398f2c032a6790690d1de77ecc6a65b42d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600feafbf9f8ca6a14c50f918d8fb35a0b296e9308ae8b6c4ed7147d2aad912aea13fe7d2639829a5ed64c309c573dba88894374fd480f595bc1d7018ec8b37ae0ea70673aa688422f1775915bc89d6ef922db7f9bef28218f11bb30996a406d9ed2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c21be32aa3713d12b85ad0f157e3877154384e4049d5ac28a98bf19b4e57e1c822e6350c5ca96658ad6f9887440dcf967085e2e4008346b3cb950d46964aad0caae2b6fd7dd2550a95d384a06011091f788cf7218762e59e686ca94fd09422ab2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460050ffac3d9c7f04901bd6dccb14601d1c6a9b0c80587e74eefbf625893e351cf0dc1d4aa08b3030ad5398a461c065186138e50ce56a92d5e46eab62701a036a0aac1daf1e3e59dae210f2b56a2b194bbad084b0c1cf492289b2f03289c635d68b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e7b07a35b9abac9905f77d01f690f01cadbd540fa75c8147808dc84f5655b17d20951de43322a74e3e9cafd48351d58ea0aab31695c3df00d5fa1bb33a997d0dacab0d24dd65b0df17ef62e2b6faca0662890de57163550c52416630850787292a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600881b5c1abdf8a9f8ef9d9d9b27bd9eed35f8930e8ec3bad5ba82490c139005cc78b856fb9c49eafa6581988f7fca348ad60a75723db53122c73d25e858284806accc6b239b9c0c62f0fd1c9cd9f90b823b41d955865bac175488ed4e7da408682a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009edc50b04aedd78e4771634a8333fcf75cb4519e3da0603bd6203475d78a3524dc9d5e6e1b671bf4f75e30f7e1c8bf044bd4577d94375d33123e365c0bb5bb08ae2734ab095cee7ac2ff74777a6d0293ee47303950df193932f2dad4d28526c22a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b987cb9c38a0a401cf03b2488fc73b7faaa5df95e8384d1153cbb57620d1313cd67a9a610414056b91b02127543d33399252687e6984d867cb1e2ea97d83d801af2d390a8e4d1e464e5abd80e9ef716ffb7e656631bc4778deb7834f4fc20dd32a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600bacd2480f5df4b871ac9d49d2e79c619e51ca03330c6e0eb517496a0f163614d24013930325ef5b76031574977e8747c04c6da8352f8c80a8f05444c18c85105b2bb7fd960d5d7ace8e36e0e4ccb7171e5c3740b2ca8a96d868a09cb5e17e32c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c73e0d10678f068c32248f2b015a3a0f415eeb404fc6f89e8589ca1e208ff5949f230d73d985f35920285cfa018d43158f0972b6588320c3756f3b9df9b5600fb38176139100069f1ff20156a180e55782796df79f4683020b06f2a5019da6dd2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46003b29752db4fed93ac5c20b344bce075851fa5cbafe02523c694bec4e3b4109fc0e88bdbd943f0d5325d62929e124a484cb24460dc06728d2039510b906928b01b7626f7e84ac48645871ac12e66bc747b9b1f64eff2352537811f9712c127c342a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c68b27c9db20c07fb3801dfae33f92407cf8e3b9c952f494687140d9a7514f0db75c71c2221b0960d74b23f1dfe643ba2f6c6fb429744e38b2f7f2ecd563d608b771cf172f891bcbbd1126e3354cdd4e324cd7202af62f5fcdcfd2ec01ef7d252a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b272ced92db8b46f6a4a9c0164fd24f66968b23b0dbbf52ebd736b1ce0a4c0512e4715cb0c0e063d580f4b4bea5811950837fc4882aaf5e59aec026ffb0f2300b7c13f1239888cda5c8e6ac9ea10675df17633368906e66a487f91ddd3268ca62a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e155d70b9ec48021bfb05f7aab92008e03f56a19222718aef3cee767a1882805166889d248053ed9f7b4d0c00b19d1931dac509709fe9cd0a577b71c8ba88c06b9186b95c90d2d00a31e7c68066bd37d73408271762604e3608e2f2c983f83092a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600438f0485ea2ab423a63142417de6065b4243e190cd326656e2ebb895489ea898950cbdc05ce8b99ec21c88a96c3953a2366970bf5a0f36c6d0ca34d121115f02b9404a5fb9f1723fc9fd4c5535dd9ef1d67c5237d9d322854697e9e245064e332a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460057b3433e3f0b68a8787e44e78c1ed5f27cfa4b155d3002193a452bd1351be719f47a4f445d635052945cc21df9e54613c7acec835c90a083e89276e7eefe3206bd81133a8d8a33ad2b084c54536d7a17ef0eda5e810a9994b13c2392fea208a92a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600f53a753bb53e0243dead9f8d43f93525d02a0c69b202ed0cafc6afdd50962cc8a0aaaacac45cbbb936748d75a2554173c94531a5bbe20f9664bcd925bb876007c0988a1b3c91b35a7c3722aba7d3a55f79ae07ac6a46d57b4a49a06eae20333e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600276a7f7126f933b4412dc4db264ac434134d77c174c3d2bca0fbca788d0327b24a2a1cd8134dd39cb9faed737ba496ce35b62f663955a88f69adf8f9f7061b06c3576342cbf99792896ee5329b04ff2eee2fc2bb6d53c5c03d52c8957ee793fb2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b5a298d2a97dbd7bf266ccfd61f20f739fd72f1cefac976ebb786e712643de23f8797aa0f160d327ed5210b6bcebb2429dcc673c4b9a39515cbe07dde7d80002c3ff25a1743a9df92af4ccd9a7aed5cbc90f64fd538c3df0a9539128f59652672a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000b87db26d1a4eeeef1f18331c234e6daeb8aea48568b4424e76478eaf1336e1cce3e3b72d136d638be075d9238c58dca48456e8587ce2ef2c46b1d7bbbadf50ac4025624fdf1544b90a61e85099d9e3ac235396d2d9d37f4921162fa688ba95d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600b472fc0441578ae962487a27c9903b0e8cef78c4c69521c759f359290290b1fd4ce33692f3726dc51dad68f8ff3cdc88dc6a137d2fa46ea4e10484605c4ed90bc4153342949e45f683cb6df646fe7aca71502b1139c1dfc929afaf73c0a6de822a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c58f80813694cdd8df0fe0586c341a1de7ff7f6dc7c093c2a7333d7e883354a5047ff31079cc00aacfe59aa12207c037e071f710a156247949099120a6afac0fc4374616443e809c1c763459d10d6bf6a2d999855b8d339a27d20d360ed5f1282a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46009bcc51ab91092800c4a9519ac8bd2f37c7a7079916a594cef09dcfff47978517c9b1c276749de14c21d010371cfa8da4d5f477787e0348fc71c7cc5c57322a06c49e3cd2c701bb845963c2870dcca12ed070c3f67ffc20144327a93aa6e896ec2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600c85a800ac1f59f0df206ad6fa5f26e2f208deda2c0af58ed8c6811c525a9fda21a781689446edfdc0b8a07062daffebe442699974b64cf1c31688fad59ce710bc61cb1b626fea15085663f7619a1769ffbe4fc1f8c63e6ece773acfe180806c92a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600fe7489b632220f8c23b1d66fd40bac34513f1eed3c9a343a59dc09d8d360517db73b05c2e33b0da1932a7cff38748645d8bcfe843290a561c63ebd7b98ec1e0cc63062763b78629518608de49f42c802ec2fc22f477e10a1a0023b237675d65d2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460018b4e42adfd1044bce65ff12455dda705b023cc0ffdef4c073aae2180cffac8375975214bb357988f37441c14e1c2913b2c7babed43a0b270920da6970566c08c676bdd3798340c7b35c624f2647f26816ba4b3a0821339f5a0d6c9f2d84ca3a2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460034a88c97937fabdc6e2ec582df7cef3829a2ba8b6e57795bb1832e08d07f12912c546f7452b4d0338ce197a9c0a376202387322f371495ebf891a3cb27551400c6a46dbbd2e87ecd0db7ffcffe24c1a82353bc4371d7c2cfb81e7c832d556fce2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460099d000bd6eb0abc52ab88474dc3e504f6e0b21a00cf196317e471926b93410177ae48f2860c98678769671d81b85ec462596ddb585b6e30f4e71e1a98ef80c01c70ad885a3f3ce3fa4041e1477cd801cbff8daa7c835d4463044fed653b3830b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000bef4c68d2fe10b9b34995633ab9dc2c76f43bc2e837252024d85417a12dcddec9cbe44c70891e541f0d67a19e1929de1b0ee5f610808feabb9d9741a0ac7201c8768f257eeaa2a6636fc68a00eb781941c26d2f5179455ac4949a320f958d6f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008d426d804b32a7615e207fc5fa9c119c9ace79782226bfb6dab21159f3950f18619faafa59caa9a1cd0e0401698137de6b454e45a74899e74681e7e2ee85ac07cb92ffeace78dbafe6fbf275741b4b38657fb81590712aa0bca7877931f6ad392a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008319290259d0047497e2825a6ce9166f285a7333ade0a970ad928fd1b259feea5c927aa2c742e7fc7b542c579e65979837a77437cf28370a44feef9b8fdb5003cbfefdf389bc341a7e17139f61146f9fb3a9d7bb84fe93c1f771eafd4d4d9d462a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d54d20eb70a33339ff29f8017a48a857c4772ada28e5a88be923045c7ca2194a5683d51472db38d9ee5295fa892963264055b15599226d1ad75aaf550caf9f00cca95e245ef1d3209f6707fec25036c0a1b93ae5613a5c93095e23520395c57b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46006fcea4a22b6b3b28a52fe41d07f4dfcb20f80b2d3308ed4b5b46ed8e4f982a6968bef3e34ac0657b1807f9f0203d905d8dc8caf4ff0c00fb651489984046510bd060d2b638be7c2ff45b20575a76c4acad1b9264607e1b71286e4e00a03266a22a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001c0549d687700803d27fa15f67df39a92975721e14adc62d267fc7b676c2bdf4d42c66eca0ca0b6fceaceb73d042a59bd88c040aa870c4636f5ee167776fca04d1c146d2429a5a827660008721c7a880e71f44feaa3dc75524c1a9281bac48cd2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460036ad739a304649cce8e4f18021379959c98bf50e9fa1abfd74a9c3778b35f3ee2c2401a0f51794c4c44fd99d167ec94a52529761c41182fc708a6bc25a32b50ed1deb1ed8d4e155f1b5bbc54610756cf2541f43b7c1776f97d404503878b96002a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46002e1b05424cd82fec9717f23563923e095c71221e13a284261e553617e94c2f4d053c28d9652abefb837ba0d9cedce01d40696794117fca8f2b232ba97d418802d2c1a8caace45c00d73f64f241162f85842c35e557e097fd1749040fedc94df92a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46001b1722198be15749cb2c16b9b0cd42dbb320c3f6ffaa880fa90a8a8db74ad322f69bdc2762796ae752ac4377637aa6e0d757320c78d8a2b3f47a8fd0a1214303d3adef5f9366e150e5a21d8837947fc2378997c7658245fdf0bca95a513f04f92a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d202e0bbf1b01acf647b11dd367019b763c8b37f1a9cf1d1a1fe663f5cf1aa6c3c9f4a6675515e9ff47255a6627e888de370f118abc6be342d624a6a83dd960fd416f2796d696faaeb34d2cdaf1004ede551b62a690f99d69b4ccd5f4a6c248b2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460028caec545fd07031c5e73f34814d463636013534942812417d2317c7a557af8d21c742280eeecf514916493746db6d6ee1e059170ea0123220a6ae57c3c8e102d4b350a3b25c27c9ab9366d130560326fe6bb7f3a820767736b8a061405e153e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e5a071915555df4404f783902b68cb3c44e4ae8485d043cec0e615ff63719e6eec2cab48ed142c04efffb4321724cedaa427d27e9bed99344dba219a533b6b09d67211ff6ded7cffc866c81e24a9f54f08cd1df7ab202ae796a01d72cd2d3ab32a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460029a7b0b7a6b4a80c7e89b1cc2bb8635bea1587ad214100da7b11add353cd3e69df0b8dfdf17264eae2753de1ef1d62d267cd7214768bda15f56f9e6b00612902d744af98bbb366011a0fe90c3cee81d21cb301d9bb72d5b40b28da348ecb81672a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d77c1a9378d17bbba63e854c9bfd0abab0b3650a74bd698d7b9feea4f588d5dfa2ce724f24f60d844a4feac5d9b5928f4c537caa9230ef2ac5d1d9953b1b330cd7be80fafe9de0570984b008636808811ccfd82f32639cf6806dfb86c4c4c4c72a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d373bbbb4349a52b00d00a3981a161ed96bd0fa969faf1dbb8b49879f86dd5839b0ab12856f912ebe2c8e8a12d82dc7688729a6449d3bb6b7a6d53f66d82d203d86e516b7564a8ab0b67b503c97976157c05fc2395068b27df5f38a1a9b29b6e2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005065b2652114169f82c0c1483c19f8fd2f0c72f8c98928fe419ce0b93b2ef1583cb684837009c4c97381eb83c880f62055e9068113d9e07d1b29c991521a5308d8a6314d0690c070c764863c7c25fb8e9d6cd462f9bd56b0b236558f66fd74fa2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460016e5038faa65afb9f179af818be6b65c1a93963effe329ecba6786891ac1d8956e3f209365e45bcaec3879ff27891202e219e8061228f7001446f532b0bc0d09d978b1073d9c88d84ece71a915d577b8b381938d07827447ec20a5bb250496d22a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d57606898de72fb911f85da1bb8fb3743b6edc0e1c61a61bc05d3b6d1d429d5b6d86e38ae7289b40efbb3df89d14f5704f8e61bf2bfe195470f8a8fe9146d009db1067e5c50401a17b926e54a98e103abd64c4b83c87b770a8312ff03bc29be72a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ef38ecce8f3b7f2253942c612b029e05a6b37f8b1c406f5d01479ec76f956cd040f3a679a9e77a037401185a665f3162531feab7221d828e4bf7662ea876000ddb231b388a5ba18b2568eb1ab9ec84a637a66d6c1286fc7e1c2351250db635c32a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46000d66f46119003f7476f2115373c0b88a81e3044ef83d9955fce655439adcaf22a12b74b5fd18db965e694d5635f4aa2418a263decf6c448b891462432ee7ae05db3aeeb826b44e7808e3865df0da2f479442e2f2fc895a46dffa8d908ec95f512a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460065ba17675d688b0fe1ae25984335d388c61f14a81ceddf6b6585b90a9a59bba0a4070d815018326acd57a88edbef65d60ba226554112f06c1f33b0c6b39b9401dc1aada86981424a634bbc637e58712bc5002a34bb6498487d89d83534b05da02a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46005565dd6e8a390aa49b5491cd0bce4798501a87884ef50ba120262b826ff28ba5741930e988241f4efcc0cd42f7c621033166193fc85b00b792d41135172d780fdcee76cd80cf2218de9e8c4ae1922fe1c26235f9cb479b2950b064025a1d69412a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600f796ea2ed7675e68b13fa0df5a873706de9c3490e006b4b7e330668ca62ba0eb7284b5087a09b9ae01e2ffa3aed1a545a1de99ea38b0bc183d46884315170502e2cc4424464983ada824bbef5aaa8995e80ef1017c15ef3b13902599841637ba2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46006abff33b091bfe069c92ba5bcd25d56b604a45cd1023080614e8e8578f9327c7025a9bb2ba92c4db3e699015ac4f46dacec10245d86b73d41b9e751bc7f6e203e58210689f52468a22c10566f3a2c6f870e61524298a3e781425861d40446b132a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d2a95ca51c964bb35183f2ae626a09a01b7b7e82a046af708da537389d68757b8a5d5c8f230523c5ec089277b7c0e072fcf4d9dca2ecd7bb6a3eec6e95919c01e6ebe15e9e2c9f0fe234241ee50e9c574774807c0d17f8dbc7fdd1802c5c79792a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600254314bb9821d205214651a8c292292089a320ccde7dd38acb21552a855eb0ace84b132f04317b591b84be06679d74df67b46abee0128089679c45e6f7836c00e7210714763e5bb3fab12067d0a55784ae0d70ef14ccf9e243bbfc6d329834102a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600e3b32fd44cc8a2dfd85b2717b54a4d21d93a8f40b0c01be21f81dbc482b383d2a4c32911a93fcc4c8a746dc75aa0925c6b266b397a17f05318e6cebf51661001e81206b483fdcf1fe42145f27d1efb8178a57ee24b196285374037da2a53b3232a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46008b0b5494d43206cf55b468597dbb9dcfc51cb59a5b94a2f2150d5bed423057867958b09d96d7b772d41d2ffb7f68125ea119f0d628f2ad338856a34fbdd8ef05e90f0dd5c3d8e23deb6b82b9d7335855f5887c3b77ac1fe30d9112b22319bc922a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d2a3355b20c0bb0286e1951b985ac51bea4439b0c70ede6f38c026495bdc948e2df2f72d00b1d3d7a2bedfabe8a57d6d4f0af59670b5e75eb713a1ec2cf7d802e9836cd1a96c0010f43161443a790d65ea2afdef21b05bd563ed55c6eecb00662a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460015d31429c5579f409756d92155519c4bd60ca1b095c8f5a5aac66459efa937f48989d599106e5749cb74e8718bb96c49119ec5aa11adb3251a9365456809ae0aeb0af0662538c295c85d7da7b1e0d929f8d887a55bf1a136a144f2f7cf3215802a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460083b2ce5b2c3df07146efcb19e6dd1d8755ca7ef52ba3b05e3a2830374a4f984a4a2946c0905d47f962bf14e51bb0aba4303806247135d579317349e421550505ee8f86db1c1d2b7cd11e962fcfd1a7e847204337c8893f1c1ddb75c6bff3f2bb2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600dd14dbdb4c43afa43d3da56daec9d48f09e356082e2317b505fc1a8520eb491d6fd344bc94479d9497be72d2dbc95d193c6bcaaaec8c5df928b37e8373cfb005eecc027b5c05b7158fa0614fd70df47412f181e808bb78cbb63e29adf14ec91c2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460053e0270ad201d8ccbd4253715af906a56e2f44c6d78f9344da12e465a5c755ba569b0f5db093834921b8ed11f4df2dd52497b387efdf3948ff632ae40f7da504f1020dda4f7e6af02b444bfcc70a0a18bd48042ad72b1955778bf4e56086a33f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600ff32188333a59ee7d640f5897f5415cdf08e97742f82d340687c63b71c0ba9ae42599d522a40233b772d6147edd01a7d5bc865714f95ff979c03e61b9d1a6a03f2a930984c2799a98d298521128eff1add9706c9e57e848d54de41f6b8dac4b42a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460039c0604b3ad216bbdfc17d641114a0d25e0865ff21cc945574cda12f7c41f7e19f3b65550b410b017e0e8f58c94d9abd29f01b7036193305fbecd341e7156509f5054136e1ea0e3956293c422a147eaa4950cfcf0ad9596dc6e6a287bdfb06602a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460029e0bf534f510a4cd43b5011d63057da67b36e2b03cf6087b3bb8ec80fdbb9e63b5e361313e2dee669d1d5644696e5c8b4e71548ec9c6daf4c85873318d9b708f8d4441b4ef1c2ee50272e1767ac4f773a249eb663c7e37bc075951f3b52ede52a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600753c2d7a7193f908ed808903b7cfbc3e2f5a8d9fb380c054f887c0fe21b206d81406d25bb1d0ef6cb6a0feaf015f9e96c0d703e9bc5a0e956feff40fdf0f0d0ef9887f6f0a8673aad0c10d1bf6ae3ae5f0089c8f05d54e180e16030949a8f7782a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d0c3538d97ce8594f8a113d01728ac4b13549646557385c6b5263f4a67a0ba96e2e5f0ee3ac9c85fff368b4013eb43900be5e672375a3f3860298342a0325f05fa786562ceec7e61851571e8eaae35c7f0f4b2ada3ddb4d577c66ab50a1d6ceb2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600eb8636c254b4cec68eea4ef00a3ae43b6dba9c8f85958b514042d0764f66bc9b22e4205ff4d8d4427395df3aeb15fc840356a9eabacee3a23c8cd6aff517db05faf968e9e1468d58947b9217ed7aecf19ce457036a2b46e47c44e5a1d67e5b1f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600a7038a3eb4b56c7414816925b26d1756706f2256596c99181ae8746a40cc32c5192c93a13997c9a11f31fa6d6263164e0c06034f0790400f1730c250f4054205fbcc3ce7dbd34cd50cb3563d2b2f62a5ff2bc847e6b6e79faee63257978f9c192a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600762a2c5d7849ffbd8c2560f7e6638ede211a0e59347bb1c7f629445e6af0d38826319366d3597f0773b49515907516f2e1894618f8517297b4624866c28e3e0ffc678679f11bc904720421273689e6826acb42b21e2c4c5c1d7bf532d89668412a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd46006dbd8fd05b77b7556cef50351310ca989db725fedbe17fbdc079043ba50dd86f7fe96f621e1eafb13aaf097fed1d441a90e6462d4919a752441455bf49698a0dfd9e354ed59f35c917c42471588142e715ed2151e89ecaf98bb5d837515d74772a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd460010d0c657b622c3927cb79d4e34d0d1608e0b4f12294277049528b4a3e4add5745ce5cf9048f16607fcba67325cf06c460d2766421210dbcbeff29b1f0dc39402ff0bf39c82ed573d48585448b0fc19ba3f6203806cb6b4a230848892c097a26f2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f702247bd4600d072ab3364ee35ffd547d78aba38e764805b6caeff708dcd806acd06ee2d684cccddbacd4c5d3ba73a6f6e03d3749635ee8eaff19137ec7dc686c46af8f76b07fff437ff18629bf1490e5c9b3ec6f1515d46bb9b2aeaa6e39e36611f2479b50d00")

	gs, _ := newTestService(t)
	gs.messageHandler.blockNumToSetID.Store(uint32(4635975), setID)
	err := gs.VerifyBlockJustification(data)
	require.NoError(t, err)
}
