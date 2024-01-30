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

func mustHexTo64BArray(t *testing.T, inputHex string) (outputArray [64]byte) {
	t.Helper()
	copy(outputArray[:], common.MustHexToBytes(inputHex))
	return outputArray
}

func Test_OpaqueKeyOwnershipProof_ScaleCodec(t *testing.T) {
	t.Parallel()
	keyOwnershipProof := GrandpaOpaqueKeyOwnershipProof([]byte{64, 138, 252, 29, 127, 102, 189, 129, 207, 47, 157,
		60, 17, 138, 194, 121, 139, 92, 176, 175, 224, 16, 185, 93, 175, 251, 224, 81, 209, 61, 0, 71})
	encoded := scale.MustMarshal(keyOwnershipProof)
	var proof GrandpaOpaqueKeyOwnershipProof
	err := scale.Unmarshal(encoded, &proof)
	require.NoError(t, err)
	require.Equal(t, keyOwnershipProof, proof)
}

func TestInstance_GrandpaSubmitReportEquivocationUnsignedExtrinsicEncoding(t *testing.T) {
	t.Parallel()
	// source:
	// https://github.com/jimjbrettj/scale-encoding-generator/blob/a111e57d5103a7b5ba863cee09b83b89ba9c29e0/src/main.rs#L52
	expectedEncoding := common.MustHexToBytes("0x010000000000000000010000000000000088dc3417d5058ec4b4503e0c12ea" +
		"1a0a89be200fe98922423d4334014fa6b0ee4801b8e62d31167d30c893cc1970f6a0e289420282a4b245b75f2c46fb308af10a0000" +
		"00d7292caacc62504365f179892a7399f233944bf261f8a3f66260f70e0016f2db63922726b015c82dc7131f4730fbec61f71672a5" +
		"71453e51029bfb469070900fc314327941fdd924bc67fd72651c40aececd485ca3e878c21e02abb40feae5bd0a000000b3c408b749" +
		"05dfedfffa66f99f16fe8b938fd8df76a92225228a1ca075230b99a2d9e173c561952e1e378b701915ca188d2c832ef92a3fab8e455" +
		"f32570c0807")
	identity := common.MustHexToBytes("0x88dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee")
	identityPubKey, _ := ed25519.NewPublicKey(identity)
	firstVote := GrandpaVote{
		Hash:   common.MustHexToHash("0x4801b8e62d31167d30c893cc1970f6a0e289420282a4b245b75f2c46fb308af1"),
		Number: uint32(10),
	}
	secondVote := GrandpaVote{
		Hash:   common.MustHexToHash("0xc314327941fdd924bc67fd72651c40aececd485ca3e878c21e02abb40feae5bd"),
		Number: uint32(10),
	}

	firstSignatureArray := mustHexTo64BArray(t, "0xd7292caacc62504365f179892a7399f233944bf261f8a3f66260f70e0016f2d"+
		"b63922726b015c82dc7131f4730fbec61f71672a571453e51029bfb469070900f")

	secondSignatureArray := mustHexTo64BArray(t, "0xb3c408b74905dfedfffa66f99f16fe8b938fd8df76a92225228a1ca07523"+
		"0b99a2d9e173c561952e1e378b701915ca188d2c832ef92a3fab8e455f32570c0807")

	var authorityID [32]byte
	copy(authorityID[:], identityPubKey.Encode())

	grandpaEquivocation := GrandpaEquivocation{
		RoundNumber:     1,
		ID:              authorityID,
		FirstVote:       firstVote,
		FirstSignature:  firstSignatureArray,
		SecondVote:      secondVote,
		SecondSignature: secondSignatureArray,
	}

	preVoteEquivocation := PreVote(grandpaEquivocation)
	equivocationEnum := NewGrandpaEquivocation()
	err := equivocationEnum.SetValue(preVoteEquivocation)
	require.NoError(t, err)

	equivocationProof := GrandpaEquivocationProof{
		SetID:        1,
		Equivocation: *equivocationEnum,
	}

	actualEncoding := scale.MustMarshal(equivocationProof)
	require.Equal(t, expectedEncoding, actualEncoding)
}

func Test_GrandpaVote(t *testing.T) {
	t.Parallel()
	// source:
	// https://github.com/jimjbrettj/scale-encoding-generator/blob/a111e57d5103a7b5ba863cee09b83b89ba9c29e0/src/main.rs#L42
	expectedEncoding := []byte{10, 11, 12, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 231, 3, 0, 0}
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
