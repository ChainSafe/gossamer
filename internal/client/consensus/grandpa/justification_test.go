// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa/mocks"
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
	encoded := primitives.LocalizedPayload(primitives.Prevote, primitives.RoundNumber(round), primitives.SetID(setID), msg)
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

	justification, err := decodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encodedJustification)
	require.NoError(t, err)
	require.Equal(t, expected, justification.Justification)
}

func TestJustification_fromCommit(t *testing.T) {
	commit := primitives.Commit[hash.H256, uint64]{}
	client := mocks.NewHeaderBackend[hash.H256, uint64](t)
	_, err := NewJustificationFromCommit[hash.H256, uint64](client, 2, commit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// nil header
	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommit := makePrecommit(t, "a", 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	validCommit := primitives.Commit[hash.H256, uint64]{
		TargetHash:   "a",
		TargetNumber: 1,
		Precommits:   precommits,
	}

	clientNil := mocks.NewHeaderBackend[hash.H256, uint64](t)
	clientNil.EXPECT().Header(hash.H256("b")).Return(nil, nil)
	_, err = NewJustificationFromCommit[hash.H256, uint64](
		clientNil,
		2,
		validCommit,
	)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// currentHeader.Number() <= baseNumber
	var header runtime.Header[uint64, hash.H256] = generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		"",
		runtime.Digest{})

	client.EXPECT().Header(hash.H256("b")).Return(&header, nil)
	_, err = NewJustificationFromCommit[hash.H256, uint64](
		client,
		2,
		validCommit,
	)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// happy path
	header = generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		hash.H256("a"),
		runtime.Digest{})

	client = mocks.NewHeaderBackend[hash.H256, uint64](t)
	client.EXPECT().Header(hash.H256("b")).Return(&header, nil)

	expAncestries := make([]runtime.Header[uint64, hash.H256], 0)
	expAncestries = append(expAncestries, generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		hash.H256("a"),
		runtime.Digest{}),
	)
	expJustification := primitives.GrandpaJustification[hash.H256, uint64]{
		Round: 2,
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: expAncestries,
	}
	justification, err := NewJustificationFromCommit[hash.H256, uint64](
		client,
		2,
		validCommit)
	require.NoError(t, err)
	require.Equal(t, expJustification, justification.Justification)
}

func TestJustification_decodeAndVerifyFinalizes(t *testing.T) {
	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll

	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := DecodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
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
	_, err = DecodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
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

	newJustification, err := DecodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
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

func TestWriteJustification(t *testing.T) {
	store := newDummyStore()

	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(headerA.Hash()), 1, 1, 1, ed25519.Alice))

	expAncestries := make([]runtime.Header[uint64, hash.H256], 0)
	expAncestries = append(expAncestries, headerA)

	justification := GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   headerA.Hash(),
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: expAncestries,
			Round:          2,
		},
	}

	_, err := BestJustification[hash.H256, uint64, runtime.BlakeTwo256](store)
	require.ErrorIs(t, err, errValueNotFound)

	err = updateBestJustification[hash.H256, uint64](justification, write(store))
	require.NoError(t, err)

	bestJust, err := BestJustification[hash.H256, uint64, runtime.BlakeTwo256](store)
	require.NoError(t, err)
	require.NotNil(t, bestJust)
	require.Equal(t, justification, *bestJust)
}
