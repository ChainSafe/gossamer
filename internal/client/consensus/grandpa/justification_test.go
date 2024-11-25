// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"reflect"
	"testing"

	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePrecommit(t *testing.T,
	targetHash string,
	targetNumber uint64,
	round uint64, //nolint:unparam
	setID uint64,
	voter ed25519.Keyring,
) grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
	t.Helper()

	precommit := grandpa.Precommit[hash.H256, uint64]{
		TargetHash:   hash.H256(targetHash),
		TargetNumber: targetNumber,
	}
	msg := grandpa.NewMessage(precommit)
	encoded := primitives.NewLocalizedPayload(primitives.RoundNumber(round), primitives.SetID(setID), msg)
	signature := voter.Sign(encoded)

	return grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
		Precommit: grandpa.Precommit[hash.H256, uint64]{
			TargetHash:   hash.H256(targetHash),
			TargetNumber: targetNumber,
		},
		Signature: signature,
		ID:        voter.Pair().Public().(ced25519.Public),
	}
}

func TestJustificationEncoding(t *testing.T) {
	var hashA = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll
	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommit := makePrecommit(t, hashA, 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	expAncestries := make([]runtime.Header[uint64, hash.H256], 0)
	expAncestries = append(expAncestries, generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		hash.H256(hashA),
		runtime.Digest{}),
	)

	expected := primitives.GrandpaJustification[hash.H256, uint64]{
		Round: 2,
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash: hash.H256(
				"b\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", //nolint:lll
			),
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: expAncestries,
	}

	encodedJustification, err := scale.Marshal(expected)
	require.NoError(t, err)

	justification, err := DecodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encodedJustification)
	require.NoError(t, err)
	require.Equal(t, expected, justification.Justification)
}

func TestDecodeJustificationFromWestendBlock2323464(t *testing.T) {
	encodedJustification := []byte{
		101, 9, 0, 0, 0, 0, 0, 0, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 44, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 198, 201, 2, 183, 193, 46, 175, 3, 86, 192, 71, 8, 140, 159, 134, 111, 60, 19, 179, 232, 166, 52, 13, 83, 131, 67, 154, 255, 169, 77, 46, 164, 243, 101, 198, 120, 44, 34, 193, 252, 78, 116, 191, 19, 234, 3, 12, 73, 160, 191, 117, 195, 117, 189, 255, 154, 245, 228, 244, 56, 186, 175, 62, 5, 7, 217, 82, 218, 242, 208, 226, 97, 110, 83, 68, 166, 207, 249, 137, 163, 252, 197, 167, 154, 87, 153, 25, 140, 21, 255, 28, 6, 197, 26, 18, 128, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 176, 249, 251, 175, 109, 52, 88, 104, 18, 58, 131, 78, 60, 254, 31, 223, 108, 15, 5, 172, 89, 74, 87, 71, 224, 95, 205, 14, 140, 187, 209, 168, 191, 202, 222, 118, 236, 59, 136, 204, 25, 155, 54, 195, 41, 185, 185, 25, 220, 59, 102, 243, 78, 196, 207, 151, 235, 88, 18, 254, 235, 223, 213, 10, 22, 157, 169, 111, 232, 137, 254, 25, 242, 233, 70, 60, 76, 183, 48, 179, 52, 115, 165, 97, 240, 161, 93, 85, 129, 202, 124, 82, 54, 42, 37, 45, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 99, 120, 106, 150, 120, 143, 157, 182, 211, 188, 22, 39, 229, 213, 243, 215, 167, 136, 65, 250, 142, 19, 253, 68, 145, 41, 107, 23, 75, 214, 26, 218, 250, 117, 61, 126, 142, 16, 100, 37, 91, 52, 80, 139, 174, 188, 9, 16, 164, 118, 236, 64, 25, 60, 149, 184, 41, 98, 22, 130, 153, 247, 130, 14, 38, 165, 0, 130, 236, 99, 74, 108, 27, 250, 196, 212, 157, 16, 5, 85, 237, 246, 19, 223, 112, 61, 138, 10, 206, 62, 76, 149, 116, 94, 166, 153, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 213, 174, 223, 48, 80, 151, 122, 6, 20, 150, 237, 141, 18, 241, 40, 29, 108, 125, 140, 66, 118, 219, 205, 152, 120, 254, 234, 104, 178, 115, 48, 11, 203, 253, 199, 30, 145, 96, 190, 248, 119, 153, 70, 212, 109, 128, 253, 128, 218, 102, 237, 172, 20, 67, 150, 223, 33, 18, 180, 177, 236, 90, 246, 8, 49, 218, 231, 151, 187, 172, 14, 39, 185, 1, 53, 92, 201, 4, 70, 54, 157, 205, 21, 216, 249, 101, 147, 59, 87, 114, 62, 227, 137, 103, 13, 85, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 152, 158, 83, 232, 15, 138, 209, 77, 64, 107, 167, 37, 241, 208, 225, 167, 139, 4, 11, 198, 31, 54, 199, 4, 9, 37, 83, 81, 181, 3, 137, 105, 106, 8, 122, 244, 145, 242, 86, 170, 97, 164, 75, 149, 40, 184, 255, 144, 175, 167, 34, 239, 142, 209, 188, 254, 254, 172, 98, 151, 204, 104, 195, 3, 64, 75, 49, 166, 102, 51, 68, 198, 140, 126, 124, 42, 28, 244, 167, 103, 245, 58, 82, 95, 224, 34, 248, 13, 162, 131, 210, 200, 187, 19, 104, 109, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 125, 196, 88, 9, 14, 244, 127, 43, 91, 8, 3, 111, 60, 100, 68, 214, 232, 192, 227, 100, 3, 85, 96, 123, 39, 218, 144, 8, 105, 89, 91, 81, 247, 144, 105, 157, 52, 137, 232, 1, 147, 140, 235, 159, 111, 171, 42, 118, 132, 34, 246, 112, 201, 39, 14, 11, 246, 240, 61, 96, 179, 74, 214, 13, 90, 240, 22, 123, 223, 44, 19, 81, 145, 248, 255, 177, 85, 199, 95, 9, 123, 133, 115, 69, 104, 121, 209, 216, 156, 81, 148, 80, 144, 230, 69, 231, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 24, 143, 150, 165, 148, 107, 121, 215, 2, 248, 43, 157, 205, 237, 232, 215, 38, 4, 255, 171, 250, 235, 168, 112, 179, 227, 162, 65, 57, 103, 181, 78, 62, 138, 122, 170, 209, 197, 249, 15, 58, 60, 127, 131, 148, 23, 73, 104, 154, 68, 110, 187, 187, 81, 35, 7, 112, 40, 131, 72, 252, 159, 21, 0, 93, 5, 197, 56, 70, 126, 210, 89, 242, 82, 11, 86, 244, 159, 245, 136, 50, 204, 196, 64, 138, 105, 178, 138, 18, 187, 39, 63, 59, 65, 159, 44, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 8, 223, 196, 18, 152, 147, 171, 52, 220, 14, 7, 73, 92, 31, 114, 102, 49, 225, 193, 77, 208, 80, 122, 181, 225, 249, 241, 231, 111, 108, 168, 32, 195, 46, 167, 253, 161, 55, 210, 186, 77, 178, 44, 110, 183, 255, 177, 239, 51, 54, 7, 210, 244, 140, 118, 238, 166, 204, 115, 22, 91, 40, 200, 12, 198, 220, 66, 100, 134, 46, 17, 156, 132, 171, 46, 163, 191, 78, 170, 157, 123, 225, 4, 216, 141, 127, 62, 224, 142, 9, 139, 224, 1, 181, 171, 248, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 8, 116, 35, 0, 242, 86, 164, 127, 87, 169, 203, 130, 87, 220, 232, 163, 152, 74, 168, 1, 148, 28, 130, 172, 224, 187, 175, 110, 168, 210, 32, 192, 13, 239, 253, 128, 242, 61, 11, 147, 177, 71, 167, 232, 9, 164, 5, 55, 201, 23, 241, 38, 11, 183, 108, 222, 31, 205, 143, 241, 250, 113, 150, 153, 233, 157, 177, 2, 202, 111, 218, 104, 65, 158, 55, 67, 81, 194, 252, 131, 20, 178, 177, 182, 54, 147, 35, 144, 162, 50, 36, 210, 60, 48, 25, 122, 32, 236, 76, 178, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 92, 247, 208, 78, 40, 206, 67, 51, 161, 33, 242, 20, 96, 34, 29, 153, 173, 107, 81, 107, 174, 81, 70, 243, 152, 34, 56, 210, 233, 60, 104, 208, 249, 218, 76, 246, 143, 141, 145, 59, 93, 208, 11, 89, 238, 251, 144, 58, 21, 242, 241, 146, 191, 53, 87, 152, 2, 72, 147, 31, 112, 116, 86, 15, 203, 185, 105, 7, 239, 188, 63, 248, 237, 145, 209, 113, 202, 38, 101, 138, 44, 57, 5, 137, 2, 117, 55, 94, 218, 147, 126, 180, 86, 129, 98, 126, 61, 255, 19, 108, 231, 232, 166, 4, 100, 186, 92, 153, 57, 80, 3, 56, 96, 6, 5, 68, 151, 191, 177, 72, 88, 198, 80, 161, 92, 202, 155, 243, 9, 116, 35, 0, 204, 187, 70, 137, 74, 136, 59, 59, 152, 243, 175, 33, 2, 120, 172, 158, 205, 10, 95, 114, 45, 98, 202, 56, 6, 164, 248, 213, 112, 43, 254, 76, 1, 145, 133, 70, 249, 18, 164, 216, 44, 65, 107, 2, 252, 250, 196, 133, 81, 133, 128, 94, 88, 14, 82, 34, 40, 21, 53, 173, 146, 129, 123, 9, 207, 44, 219, 247, 170, 156, 134, 196, 172, 58, 196, 93, 17, 2, 49, 213, 206, 34, 254, 84, 173, 21, 246, 247, 139, 134, 102, 195, 145, 222, 62, 120, 4, 160, 39, 234, 163, 193, 179, 177, 30, 36, 107, 194, 191, 221, 25, 90, 49, 196, 99, 44, 212, 88, 240, 35, 236, 228, 218, 193, 52, 63, 124, 13, 205, 38, 208, 141, 0, 141, 209, 83, 5, 194, 62, 249, 11, 242, 116, 184, 9, 131, 144, 52, 213, 192, 28, 99, 214, 166, 50, 99, 255, 194, 162, 132, 89, 224, 209, 215, 18, 175, 115, 178, 154, 62, 213, 23, 253, 41, 157, 129, 228, 37, 15, 202, 20, 16, 146, 91, 190, 175, 232, 194, 152, 111, 207, 21, 204, 183, 86, 44, 6, 8, 6, 66, 65, 66, 69, 52, 2, 4, 0, 0, 0, 208, 224, 230, 15, 0, 0, 0, 0, 5, 66, 65, 66, 69, 1, 1, 50, 255, 71, 85, 126, 223, 168, 216, 206, 184, 250, 72, 32, 198, 203, 75, 199, 72, 221, 129, 46, 148, 138, 198, 131, 87, 171, 222, 179, 77, 17, 67, 253, 235, 143, 215, 161, 94, 64, 52, 22, 176, 169, 99, 87, 111, 63, 212, 235, 51, 27, 29, 171, 103, 11, 175, 236, 95, 31, 146, 12, 171, 224, 136,
	}

	justification, err := DecodeJustification[hash.H256, uint32, runtime.BlakeTwo256](encodedJustification)
	require.NoError(t, err)
	require.Equal(t, uint32(2323464), justification.Target().Number)
}

func TestDecodeGrandpaJustificationVerifyFinalizes(t *testing.T) {
	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll

	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		invalidEncoding,
		HashNumber[hash.H256, uint64]{},
		2,
		grandpa.VoterSet[string]{})
	require.Error(t, err)

	// Invalid target
	justification := primitives.GrandpaJustification[hash.H256, uint64]{
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
		},
	}

	encWrongTarget, err := scale.Marshal(justification)
	require.NoError(t, err)
	_, err = DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encWrongTarget,
		HashNumber[hash.H256, uint64]{},
		2,
		grandpa.VoterSet[string]{})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid commit target in grandpa justification")

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		a,
		runtime.Digest{})

	hederList := []runtime.Header[uint64, hash.H256]{headerB}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 1, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 1, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 1, ed25519.Charlie))

	expectedJustification := primitives.GrandpaJustification[hash.H256, uint64]{
		Round: 1,
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: hederList,
	}

	encodedJustification, err := scale.Marshal(expectedJustification)
	require.NoError(t, err)

	target := HashNumber[hash.H256, uint64]{
		Hash:   a,
		Number: 1,
	}

	idWeights := make([]grandpa.IDWeight[string], 0)
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		idWeights = append(idWeights, grandpa.IDWeight[string]{
			ID: string(id[:]), Weight: 1,
		})
	}
	voters := grandpa.NewVoterSet(idWeights)

	newJustification, err := DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encodedJustification,
		target,
		1,
		*voters)
	require.NoError(t, err)
	require.Equal(t, expectedJustification, newJustification.Justification)
}

func TestJustification_verify(t *testing.T) {
	// Nil voter case
	auths := make(primitives.AuthorityList, 0)
	justification := GrandpaJustification[hash.H256, uint64]{}
	err := justification.Verify(2, auths)
	require.ErrorIs(t, err, errInvalidAuthoritiesSet)

	// happy path
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		auths = append(auths, primitives.AuthorityIDWeight{
			AuthorityID:     id,
			AuthorityWeight: 1,
		})
	}

	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll
	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		a,
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{headerB}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 2, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 2, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 2, ed25519.Charlie))

	validJustification := GrandpaJustification[hash.H256, uint64]{
		Justification: primitives.GrandpaJustification[hash.H256, uint64]{
			Round: 1,
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   a,
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
		},
	}

	err = validJustification.Verify(2, auths)
	require.NoError(t, err)
}

func TestJustification_verifyWithVoterSet(t *testing.T) {
	// 1) invalid commit
	idWeights := make([]grandpa.IDWeight[string], 0)
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		idWeights = append(idWeights, grandpa.IDWeight[string]{
			ID: string(id[:]), Weight: 1,
		})
	}
	voters := grandpa.NewVoterSet(idWeights)

	invalidJustification := GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   "B",
				TargetNumber: 2,
			},
		},
	}

	err := invalidJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: invalid commit in grandpa justification")

	// 2) visitedHashes != ancestryHashes
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
	}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(headerA.Hash()), 1, 1, 2, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(headerA.Hash()), 1, 1, 2, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 2, ed25519.Charlie))

	validJustification := GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   headerA.Hash(),
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
			Round:          1,
		},
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: "+
		"invalid precommit ancestries in grandpa justification with unused headers")

	// Valid case
	headerList = []runtime.Header[uint64, hash.H256]{
		headerB,
	}

	validJustification = GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   headerA.Hash(),
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
			Round:          1,
		},
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.NoError(t, err)
}

func Test_newAncestryChain(t *testing.T) {
	dummyHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	expAncestryMap := make(map[hash.H256]runtime.Header[uint64, hash.H256])
	expAncestryMap[dummyHeader.Hash()] = dummyHeader
	type testCase struct {
		name    string
		headers []runtime.Header[uint64, hash.H256]
		want    ancestryChain[hash.H256, uint64]
	}
	tests := []testCase{
		{
			name:    "noInputHeaders",
			headers: []runtime.Header[uint64, hash.H256]{},
			want: ancestryChain[hash.H256, uint64]{
				ancestry: make(map[hash.H256]runtime.Header[uint64, hash.H256]),
			},
		},
		{
			name: "validInput",
			headers: []runtime.Header[uint64, hash.H256]{
				dummyHeader,
			},
			want: ancestryChain[hash.H256, uint64]{
				ancestry: expAncestryMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newAncestryChain[hash.H256, uint64](tt.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newAncestryChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAncestryChain_Ancestry(t *testing.T) {
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerC := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		3,
		hash.H256(""),
		hash.H256(""),
		headerB.Hash(),
		runtime.Digest{})

	invalidParentHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		hash.H256("invalid"),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
		headerC,
	}
	invalidHeaderList := []runtime.Header[uint64, hash.H256]{
		invalidParentHeader,
	}
	validAncestryMap := newAncestryChain[hash.H256, uint64](headerList)
	invalidAncestryMap := newAncestryChain[hash.H256, uint64](invalidHeaderList)

	type testCase struct {
		name   string
		chain  ancestryChain[hash.H256, uint64]
		base   hash.H256
		block  hash.H256
		want   []hash.H256
		expErr error
	}
	tests := []testCase{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerA.Hash(),
			want:  []hash.H256{},
		},
		{
			name:   "baseEqualsBlock",
			chain:  validAncestryMap,
			base:   headerA.Hash(),
			block:  "notDescendant",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:   "invalidParentHashField",
			chain:  invalidAncestryMap,
			base:   headerA.Hash(),
			block:  "notDescendant",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerC.Hash(),
			want:  []hash.H256{headerB.Hash()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.chain.Ancestry(tt.base, tt.block)
			assert.ErrorIs(t, err, tt.expErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAncestryChain_IsEqualOrDescendantOf(t *testing.T) {
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerC := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		3,
		hash.H256(""),
		hash.H256(""),
		headerB.Hash(),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
		headerC,
	}

	validAncestryMap := newAncestryChain[hash.H256, uint64](headerList)

	type testCase struct {
		name  string
		chain ancestryChain[hash.H256, uint64]
		base  hash.H256
		block hash.H256
		want  bool
	}
	tests := []testCase{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerA.Hash(),
			want:  true,
		},
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: "someInvalidBLock",
			want:  false,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerC.Hash(),
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.chain.IsEqualOrDescendantOf(tt.base, tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}
