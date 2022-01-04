// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestEncodeGrandpaVote(t *testing.T) {
	exp := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000")
	var testVote = GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}

	enc, err := scale.Marshal(testVote)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	dec := GrandpaVote{}
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, testVote, dec)

}

func TestEncodeSignedVote(t *testing.T) {
	exp := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000") //nolint:lll
	var testVote = GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	var testSignature = [64]byte{1, 2, 3, 4}
	var testAuthorityID = [32]byte{5, 6, 7, 8}

	sv := GrandpaSignedVote{
		Vote:        testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}
	enc, err := scale.Marshal(sv)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	res := GrandpaSignedVote{}
	err = scale.Unmarshal(enc, &res)
	require.NoError(t, err)
	require.Equal(t, sv, res)
}

func TestGrandpaAuthoritiesRawToAuthorities(t *testing.T) {
	exp := common.MustHexToBytes("0x08eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640000000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000") //nolint:lll
	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	auths := []GrandpaAuthoritiesRaw{
		{Key: authA, ID: 0},
		{Key: authB, ID: 1},
	}

	enc, err := scale.Marshal(auths)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	dec := []GrandpaAuthoritiesRaw{}
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, auths, dec)

	authoritys, err := GrandpaAuthoritiesRawToAuthorities(dec)
	require.NoError(t, err)
	require.Equal(t, auths[0].ID, authoritys[0].Weight)

	a := Authority{}
	err = a.FromRawEd25519(dec[1])
	require.NoError(t, err)
	require.Equal(t, a, authoritys[1])
}
