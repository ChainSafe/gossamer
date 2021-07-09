package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"

	"github.com/stretchr/testify/require"
)

var testVote = &Vote{
	Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	Number: 999,
}

var testVote2 = &Vote{
	Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
	Number: 333,
}

var testSignature = [64]byte{1, 2, 3, 4}
var testAuthorityID = [32]byte{5, 6, 7, 8}

func TestVoteMessageToConsensusMessage(t *testing.T) {
	gs, st := newTestService(t)

	v, err := NewVoteFromHash(st.Block.BestBlockHash(), st.Block)
	require.NoError(t, err)

	gs.state.setID = 99
	gs.state.round = 77
	v.Number = 0x7777

	// test precommit
	_, vm, err := gs.createSignedVoteAndVoteMessage(v, precommit)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected := &VoteMessage{
		Round: gs.state.round,
		SetID: gs.state.setID,
		Message: &SignedMessage{
			Stage:       precommit,
			Hash:        v.Hash,
			Number:      v.Number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)

	// test prevote
	_, vm, err = gs.createSignedVoteAndVoteMessage(v, prevote)
	require.NoError(t, err)
	vm.Message.Signature = [64]byte{}

	expected = &VoteMessage{
		Round: gs.state.round,
		SetID: gs.state.setID,
		Message: &SignedMessage{
			Stage:       prevote,
			Hash:        v.Hash,
			Number:      v.Number,
			AuthorityID: gs.keypair.Public().(*ed25519.PublicKey).AsBytes(),
		},
	}

	require.Equal(t, expected, vm)
}

func TestCommitMessageToConsensusMessage(t *testing.T) {
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
	precommits, authData := justificationToCompact(just)

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

	v := &Vote{
		Hash:   testHeader.Hash(),
		Number: 1,
	}

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

	expected := &catchUpResponse{
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
