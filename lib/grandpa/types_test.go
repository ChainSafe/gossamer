// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa/testdata"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestPubkeyToVoter(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	state := NewState(voters, 0, 0)
	voter, err := state.pubkeyToVoter(kr.Alice().Public().(*ed25519.PublicKey))
	require.NoError(t, err)
	require.Equal(t, voters[0], *voter)
}

func TestSignedVoteEncoding(t *testing.T) {
	exp := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	just := SignedVote{
		Vote:        *testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	enc, err := scale.Marshal(just)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	dec := SignedVote{}
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, just, dec)
}

func TestSignedVoteArrayEncoding(t *testing.T) {
	exp := common.MustHexToBytes("0x040a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	just := []SignedVote{
		{
			Vote:        *testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	enc, err := scale.Marshal(just)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	dec := []SignedVote{}
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, just, dec)
}

func TestJustification(t *testing.T) {
	exp := common.MustHexToBytes("0x6300000000000000000000000000000000000000000000000000000000000000000000000000000000000000040a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	just := SignedVote{
		Vote:        *testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	fj := Justification{
		Round: 99,
		Commit: Commit{
			Precommits: []SignedVote{just},
		},
	}
	enc, err := scale.Marshal(fj)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	res := Justification{}
	err = scale.Unmarshal(enc, &res)
	require.NoError(t, err)
	require.Equal(t, fj, res)
}

func TestJustification_Decode(t *testing.T) {
	// data received from network
	data := testdata.Data3b1b0(t)
	fj := Justification{}

	err := scale.Unmarshal(data, &fj)
	require.NoError(t, err)
	require.Equal(t, uint64(6971), fj.Round)
	require.Equal(t, uint32(4635975), fj.Commit.Number)
	require.Equal(t,
		common.MustHexToHash("0x2a82146e771968df054c8036040dea584339df52d8cbac6970d4c22ed59f7022"),
		fj.Commit.Hash)
	require.Equal(t, 199, len(fj.Commit.Precommits))
}
