package grandpa

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/stretchr/testify/require"
)

var testVote = &Vote{
	hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	number: 999,
}

var testVote2 = &Vote{
	hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	number: 333,
}

var testSignature = [64]byte{1, 2, 3, 4}
var testAuthorityID = [32]byte{5, 6, 7, 8}

func TestVoteMessageToConsensusMessage(t *testing.T) {
	gs, st := newTestService(t)

	v, err := NewVoteFromHash(st.Block.BestBlockHash(), st.Block)
	require.NoError(t, err)

	gs.state.setID = 99
	gs.state.round = 77
	v.number = 0x7777

	// test precommit
	vm, err := gs.createVoteMessage(v, precommit, gs.keypair)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected := &VoteMessage{
		Round: gs.state.round,
		SetID: gs.state.setID,
		Message: &SignedMessage{
			Stage:       precommit,
			Hash:        v.hash,
			Number:      v.number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)

	// test prevote
	vm, err = gs.createVoteMessage(v, prevote, gs.keypair)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected = &VoteMessage{
		Round: gs.state.round,
		SetID: gs.state.setID,
		Message: &SignedMessage{
			Stage:       prevote,
			Hash:        v.hash,
			Number:      v.number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)
}

func TestCommitMessageToConsensusMessage(t *testing.T) {
	gs, _ := newTestService(t)
	gs.justification[77] = []*SignedPrecommit{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: gs.publicKeyBytes(),
		},
	}

	fm := gs.newCommitMessage(gs.head, 77)
	precommits, authData := justificationToCompact(gs.justification[77])

	expected := &CommitMessage{
		Round:      77,
		Vote:       NewVoteFromHeader(gs.head),
		Precommits: precommits,
		AuthData:   authData,
	}

	require.Equal(t, expected, fm)
}

func TestNewCatchUpResponse(t *testing.T) {
	gs, st := newTestService(t)

	round := uint64(1)
	setID := uint64(1)

	testHeader := &types.Header{
		Number: big.NewInt(1),
	}

	v := &Vote{
		hash:   testHeader.Hash(),
		number: 1,
	}

	err := st.Block.SetHeader(testHeader)
	require.NoError(t, err)

	err = gs.blockState.SetFinalizedHash(testHeader.Hash(), round, setID)
	require.NoError(t, err)
	err = gs.blockState.(*state.BlockState).SetHeader(testHeader)
	require.NoError(t, err)

	pvj := []*SignedPrecommit{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	pvjEnc, err := scale.Encode(pvj)
	require.NoError(t, err)

	pcj := []*SignedPrecommit{
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

	expected := &catchUpResponse{
		Round:                  round,
		SetID:                  setID,
		PreVoteJustification:   pvj,
		PreCommitJustification: pcj,
		Hash:                   v.hash,
		Number:                 v.number,
	}

	require.Equal(t, expected, resp)
}

func TestNeighbourMessageToConsensusMessage(t *testing.T) {
	msg := &NeighbourMessage{
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
