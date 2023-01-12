// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func Test_OpaqueKeyOwnershipProof_ScaleCodec(t *testing.T) {
	t.Parallel()
	keyOwnershipProof := OpaqueKeyOwnershipProof([]byte{64, 138, 252, 29, 127, 102, 189, 129, 207, 47, 157,
		60, 17, 138, 194, 121, 139, 92, 176, 175, 224, 16, 185, 93, 175, 251, 224, 81, 209, 61, 0, 71})
	encoded := scale.MustMarshal(keyOwnershipProof)
	var proof OpaqueKeyOwnershipProof
	err := scale.Unmarshal(encoded, &proof)
	require.NoError(t, err)
	require.Equal(t, keyOwnershipProof, proof)
}

func Test_PreVoteEquivocation_ScaleCodec(t *testing.T) {
	t.Parallel()
	firstVote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	secondVote := GrandpaVote{
		Hash:   common.Hash{0xd, 0xc, 0xb, 0xa},
		Number: 999,
	}
	signature := [64]byte{1, 2, 3, 4}
	keypair, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	var authorityID [32]byte
	copy(authorityID[:], keypair.Public().Encode())

	equivocation := GrandpaEquivocation{
		RoundNumber:     0,
		ID:              authorityID,
		FirstVote:       firstVote,
		FirstSignature:  signature,
		SecondVote:      secondVote,
		SecondSignature: signature,
	}

	equivPreVote := PreVoteEquivocation(equivocation)
	equivVote := NewGrandpaEquivocation()
	err = equivVote.Set(equivPreVote)
	require.NoError(t, err)
	encoding := scale.MustMarshal(*equivVote)

	grandpaEquivocation := NewGrandpaEquivocation()
	err = scale.Unmarshal(encoding, grandpaEquivocation)
	require.NoError(t, err)
	require.Equal(t, equivVote, grandpaEquivocation)
}

func TestEncodeGrandpaVote(t *testing.T) {
	t.Parallel()
	expectedEncoding := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000")
	vote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}

	encoding := scale.MustMarshal(vote)
	require.Equal(t, expectedEncoding, encoding)

	grandpaVote := GrandpaVote{}
	err := scale.Unmarshal(encoding, &grandpaVote)
	require.NoError(t, err)
	require.Equal(t, vote, grandpaVote)

}

func TestEncodeSignedVote(t *testing.T) {
	t.Parallel()
	expectedEncoding := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	vote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	signature := [64]byte{1, 2, 3, 4}
	authorityID := [32]byte{5, 6, 7, 8}

	signedVote := GrandpaSignedVote{
		Vote:        vote,
		Signature:   signature,
		AuthorityID: authorityID,
	}
	encoding := scale.MustMarshal(signedVote)
	require.Equal(t, expectedEncoding, encoding)

	grandpaSignedVote := GrandpaSignedVote{}
	err := scale.Unmarshal(encoding, &grandpaSignedVote)
	require.NoError(t, err)
	require.Equal(t, signedVote, grandpaSignedVote)
}

func TestGrandpaAuthoritiesRawToAuthorities(t *testing.T) {
	t.Parallel()
	expectedEncoding := common.MustHexToBytes("0x08eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640000000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000") //nolint:lll
	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	auths := []GrandpaAuthoritiesRaw{
		{Key: authA, ID: 0},
		{Key: authB, ID: 1},
	}

	encoding := scale.MustMarshal(auths)
	require.Equal(t, expectedEncoding, encoding)

	var grandpaAuthoritiesRaw []GrandpaAuthoritiesRaw
	err := scale.Unmarshal(encoding, &grandpaAuthoritiesRaw)
	require.NoError(t, err)
	require.Equal(t, auths, grandpaAuthoritiesRaw)

	authorities, err := GrandpaAuthoritiesRawToAuthorities(grandpaAuthoritiesRaw)
	require.NoError(t, err)
	require.Equal(t, auths[0].ID, authorities[0].Weight)

	authority := Authority{}
	err = authority.FromRawEd25519(grandpaAuthoritiesRaw[1])
	require.NoError(t, err)
	require.Equal(t, authority, authorities[1])
}
