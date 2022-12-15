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

func TestEncodeAndDecodeEquivocationPreVote(t *testing.T) {
	t.Parallel()
	// TODO refactor this test to use encoded bytes from substrate
	testFirstVote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	testSecondVote := GrandpaVote{
		Hash:   common.Hash{0xd, 0xc, 0xb, 0xa},
		Number: 999,
	}
	testSignature := [64]byte{1, 2, 3, 4}
	keypair, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	var authorityId [32]byte
	copy(authorityId[:], keypair.Public().Encode())

	equivocation := GrandpaEquivocation{
		RoundNumber:     0,
		ID:              authorityId,
		FirstVote:       testFirstVote,
		FirstSignature:  testSignature,
		SecondVote:      testSecondVote,
		SecondSignature: testSignature,
	}

	equivPreVote := PreVoteEquivocation(equivocation)
	equivVote := NewGrandpaEquivocation()
	err = equivVote.Set(equivPreVote)
	require.NoError(t, err)
	encoding, err := scale.Marshal(*equivVote)
	require.NoError(t, err)

	grandpaEquivocation := NewGrandpaEquivocation()
	err = scale.Unmarshal(encoding, grandpaEquivocation)
	require.NoError(t, err)
	require.Equal(t, equivVote, grandpaEquivocation)
}

func TestEncodeGrandpaVote(t *testing.T) {
	t.Parallel()
	expectedEncoding := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000")
	testVote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}

	encoding, err := scale.Marshal(testVote)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encoding)

	grandpaVote := GrandpaVote{}
	err = scale.Unmarshal(encoding, &grandpaVote)
	require.NoError(t, err)
	require.Equal(t, testVote, grandpaVote)

}

func TestEncodeSignedVote(t *testing.T) {
	t.Parallel()
	expectedEncoding := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	testVote := GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	testSignature := [64]byte{1, 2, 3, 4}
	testAuthorityID := [32]byte{5, 6, 7, 8}

	signedVote := GrandpaSignedVote{
		Vote:        testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}
	encoding, err := scale.Marshal(signedVote)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encoding)

	grandpaSignedVote := GrandpaSignedVote{}
	err = scale.Unmarshal(encoding, &grandpaSignedVote)
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

	encoding, err := scale.Marshal(auths)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encoding)

	grandpaAuthoritiesRaw := []GrandpaAuthoritiesRaw{}
	err = scale.Unmarshal(encoding, &grandpaAuthoritiesRaw)
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
