// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa/mocks"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func makePrecommit(t *testing.T,
	targetHash string,
	targetNumber uint64,
	round uint64,
	setID uint64,
	voter ed25519.Keyring,
) grandpa.SignedPrecommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID] {
	t.Helper()

	precommit := grandpa.Precommit[hash.H256, uint64]{
		TargetHash:   hash.H256(targetHash),
		TargetNumber: targetNumber,
	}
	msg := grandpa.NewMessage(precommit)
	encoded := pgrandpa.LocalizedPayload(pgrandpa.RoundNumber(round), pgrandpa.SetID(setID), msg)
	signature := voter.Sign(encoded)

	return grandpa.SignedPrecommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]{
		Precommit: grandpa.Precommit[hash.H256, uint64]{
			TargetHash:   hash.H256(targetHash),
			TargetNumber: targetNumber,
		},
		Signature: signature,
		ID:        voter.Pair().Public().(ced25519.Public),
	}
}

func TestJustificationEncoding(t *testing.T) {
	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]
	precommit := makePrecommit(t, "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	expAncestries := make([]runtime.Header[uint64, hash.H256], 0)
	expAncestries = append(expAncestries, generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		"a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		runtime.Digest{}),
	)

	expected := pgrandpa.GrandpaJustification[hash.H256, uint64]{
		Round: 2,
		Commit: pgrandpa.Commit[hash.H256, uint64]{
			TargetHash:   hash.H256("b\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
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
	commit := pgrandpa.Commit[hash.H256, uint64]{}
	client := mocks.NewHeaderBackend[hash.H256, uint64](t)
	_, err := NewJustificationFromCommit[hash.H256, uint64](client, 2, commit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// nil header
	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]
	precommit := makePrecommit(t, "a", 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	validCommit := pgrandpa.Commit[hash.H256, uint64]{
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
	expJustification := pgrandpa.GrandpaJustification[hash.H256, uint64]{
		Round: 2,
		Commit: pgrandpa.Commit[hash.H256, uint64]{
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
	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := decodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		invalidEncoding,
		hashNumber[hash.H256, uint64]{},
		2,
		grandpa.VoterSet[string]{})
	require.Error(t, err)

	// Invalid target
	justification := pgrandpa.GrandpaJustification[hash.H256, uint64]{
		Commit: pgrandpa.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
		},
	}

	encWrongTarget, err := scale.Marshal(justification)
	require.NoError(t, err)
	_, err = decodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encWrongTarget,
		hashNumber[hash.H256, uint64]{},
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

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]
	precommit := makePrecommit(t, string(a), 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)
	precommit = makePrecommit(t, string(a), 1, 1, 1, ed25519.Bob)
	precommits = append(precommits, precommit)
	precommit = makePrecommit(t, string(headerB.Hash()), 2, 1, 1, ed25519.Charlie)
	precommits = append(precommits, precommit)

	expectedJustification := pgrandpa.GrandpaJustification[hash.H256, uint64]{
		Round: 1,
		Commit: pgrandpa.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: hederList,
	}

	encodedJustification, err := scale.Marshal(expectedJustification)
	require.NoError(t, err)

	target := hashNumber[hash.H256, uint64]{
		hash:   a,
		number: 1,
	}

	idWeights := make([]grandpa.IDWeight[string], 0)
	for i := 1; i <= 3; i++ {
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

	newJustification, err := decodeAndVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encodedJustification,
		target,
		1,
		*voters)
	require.NoError(t, err)
	require.Equal(t, expectedJustification, newJustification.Justification)
}

// func TestJustification_verify(t *testing.T) {
// 	// Nil voter case
// 	auths := make(AuthorityList[dummyAuthID], 0)
// 	justification := GrandpaJustification[string, uint, string, dummyAuthID]{}
// 	err := justification.Verify(2, auths)
// 	require.ErrorIs(t, err, errInvalidAuthoritiesSet)

// 	// happy path
// 	for i := 1; i <= 4; i++ {
// 		auths = append(auths, Authority[dummyAuthID]{
// 			dummyAuthID(i),
// 			1,
// 		})
// 	}

// 	headerB := Header[string, uint](testHeader[string, uint]{
// 		HashField:       "b",
// 		ParentHashField: "a",
// 	})

// 	headerList := []Header[string, uint]{
// 		headerB,
// 	}

// 	var precommits []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]
// 	precommit := makePrecommit(t, "a", 1, 1)
// 	precommits = append(precommits, precommit)

// 	precommit = makePrecommit(t, "a", 1, 2)
// 	precommits = append(precommits, precommit)

// 	precommit = makePrecommit(t, "b", 2, 3)
// 	precommits = append(precommits, precommit)

// 	validJustification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 			Precommits:   precommits,
// 		},
// 		VotesAncestries: headerList,
// 	}

// 	err = validJustification.Verify(2, auths)
// 	require.NoError(t, err)
// }

// func TestJustification_verifyWithVoterSet(t *testing.T) {
// 	// 1) invalid commit
// 	IDWeights := make([]grandpa.IDWeight[dummyAuthID], 0)
// 	for i := 1; i <= 4; i++ {
// 		IDWeights = append(IDWeights, grandpa.IDWeight[dummyAuthID]{dummyAuthID(i), 1}) //nolint
// 	}
// 	voters := grandpa.NewVoterSet(IDWeights)

// 	invalidJustification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "B",
// 			TargetNumber: 2,
// 			Precommits:   []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]{},
// 		},
// 	}

// 	err := invalidJustification.verifyWithVoterSet(2, *voters)
// 	require.ErrorIs(t, err, errBadJustification)
// 	require.Equal(t, err.Error(), "bad justification for header: invalid commit in grandpa justification")

// 	// 2) visitedHashes != ancestryHashes
// 	headerA := Header[string, uint](testHeader[string, uint]{
// 		HashField: "a",
// 	})

// 	headerB := Header[string, uint](testHeader[string, uint]{
// 		HashField:       "b",
// 		ParentHashField: "a",
// 	})

// 	headerList := []Header[string, uint]{
// 		headerA,
// 		headerB,
// 	}

// 	var precommits []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]
// 	precommit := makePrecommit(t, "a", 1, 1)
// 	precommits = append(precommits, precommit)

// 	precommit = makePrecommit(t, "a", 1, 2)
// 	precommits = append(precommits, precommit)

// 	precommit = makePrecommit(t, "b", 2, 3)
// 	precommits = append(precommits, precommit)

// 	validJustification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 			Precommits:   precommits,
// 		},
// 		VotesAncestries: headerList,
// 	}

// 	err = validJustification.verifyWithVoterSet(2, *voters)
// 	require.ErrorIs(t, err, errBadJustification)
// 	require.Equal(t, err.Error(), "bad justification for header: "+
// 		"invalid precommit ancestries in grandpa justification with unused headers")

// 	// Valid case
// 	headerList = []Header[string, uint]{
// 		headerB,
// 	}

// 	validJustification = GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 			Precommits:   precommits,
// 		},
// 		VotesAncestries: headerList,
// 	}

// 	err = validJustification.verifyWithVoterSet(2, *voters)
// 	require.NoError(t, err)
// }

// func Test_newAncestryChain(t *testing.T) {
// 	dummyHeader := testHeader[string, uint]{
// 		HashField: "a",
// 	}
// 	expAncestryMap := make(map[string]testHeader[string, uint])
// 	hash := dummyHeader.Hash()
// 	expAncestryMap[hash] = dummyHeader
// 	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
// 		name    string
// 		headers []testHeader[H, N]
// 		want    ancestryChain[H, N, testHeader[H, N]]
// 	}
// 	tests := []testCase[string, uint]{
// 		{
// 			name:    "noInputHeaders",
// 			headers: []testHeader[string, uint]{},
// 			want: ancestryChain[string, uint, testHeader[string, uint]]{
// 				ancestry: make(map[string]testHeader[string, uint]),
// 			},
// 		},
// 		{
// 			name: "validInput",
// 			headers: []testHeader[string, uint]{
// 				dummyHeader,
// 			},
// 			want: ancestryChain[string, uint, testHeader[string, uint]]{
// 				ancestry: expAncestryMap,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := newAncestryChain[string, uint](tt.headers); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("newAncestryChain() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestAncestryChain_Ancestry(t *testing.T) {
// 	headerA := testHeader[string, uint]{
// 		HashField: "a",
// 	}

// 	headerB := testHeader[string, uint]{
// 		HashField:       "b",
// 		ParentHashField: "a",
// 	}

// 	headerC := testHeader[string, uint]{
// 		HashField:       "c",
// 		ParentHashField: "b",
// 	}

// 	invalidParentHeader := testHeader[string, uint]{
// 		HashField:       "b",
// 		ParentHashField: "",
// 	}

// 	headerList := []testHeader[string, uint]{
// 		headerA,
// 		headerB,
// 		headerC,
// 	}
// 	invalidHeaderList := []testHeader[string, uint]{
// 		invalidParentHeader,
// 	}
// 	validAncestryMap := newAncestryChain[string, uint](headerList)
// 	invalidAncestryMap := newAncestryChain[string, uint](invalidHeaderList)
// 	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
// 		name   string
// 		chain  ancestryChain[H, N, testHeader[H, N]]
// 		base   H
// 		block  H
// 		want   []H
// 		expErr error
// 	}
// 	tests := []testCase[string, uint]{
// 		{
// 			name:  "baseEqualsBlock",
// 			chain: validAncestryMap,
// 			base:  "a",
// 			block: "a",
// 			want:  []string{},
// 		},
// 		{
// 			name:   "baseEqualsBlock",
// 			chain:  validAncestryMap,
// 			base:   "a",
// 			block:  "d",
// 			expErr: errBlockNotDescendentOfBase,
// 		},
// 		{
// 			name:   "invalidParentHashField",
// 			chain:  invalidAncestryMap,
// 			base:   "a",
// 			block:  "b",
// 			expErr: errBlockNotDescendentOfBase,
// 		},
// 		{
// 			name:  "validRoute",
// 			chain: validAncestryMap,
// 			base:  "a",
// 			block: "c",
// 			want:  []string{"b"},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.chain.Ancestry(tt.base, tt.block)
// 			assert.ErrorIs(t, err, tt.expErr)
// 			assert.Equal(t, tt.want, got)
// 		})
// 	}
// }

// func TestAncestryChain_IsEqualOrDescendantOf(t *testing.T) {
// 	headerA := testHeader[string, uint]{
// 		HashField: "a",
// 	}

// 	headerB := testHeader[string, uint]{
// 		HashField:       "b",
// 		ParentHashField: "a",
// 	}

// 	headerC := testHeader[string, uint]{
// 		HashField:       "c",
// 		ParentHashField: "b",
// 	}

// 	headerList := []testHeader[string, uint]{
// 		headerA,
// 		headerB,
// 		headerC,
// 	}

// 	validAncestryMap := newAncestryChain[string, uint](headerList)
// 	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
// 		name  string
// 		chain ancestryChain[H, N, testHeader[H, N]]
// 		base  H
// 		block H
// 		want  bool
// 	}
// 	tests := []testCase[string, uint]{
// 		{
// 			name:  "baseEqualsBlock",
// 			chain: validAncestryMap,
// 			base:  "a",
// 			block: "a",
// 			want:  true,
// 		},
// 		{
// 			name:  "baseEqualsBlock",
// 			chain: validAncestryMap,
// 			base:  "a",
// 			block: "d",
// 			want:  false,
// 		},
// 		{
// 			name:  "validRoute",
// 			chain: validAncestryMap,
// 			base:  "a",
// 			block: "c",
// 			want:  true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := tt.chain.IsEqualOrDescendantOf(tt.base, tt.block)
// 			assert.Equal(t, tt.want, got)
// 		})
// 	}
// }

// func TestWriteJustification(t *testing.T) {
// 	store := newDummyStore()

// 	var precommits []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]
// 	precommit := makePrecommit(t, "a", 1, 1)
// 	precommits = append(precommits, precommit)

// 	expAncestries := make([]Header[string, uint], 0)
// 	expAncestries = append(expAncestries, testHeader[string, uint]{
// 		NumberField:     100,
// 		ParentHashField: "a",
// 	})

// 	justification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round: 2,
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 			Precommits:   precommits,
// 		},
// 		VotesAncestries: expAncestries,
// 	}

// 	_, err := BestJustification[string, uint, string, dummyAuthID, testHeader[string, uint]](store)
// 	require.ErrorIs(t, err, errValueNotFound)

// 	err = updateBestJustification[string, uint, string, dummyAuthID](justification, write(store))
// 	require.NoError(t, err)

// 	bestJust, err := BestJustification[string, uint, string, dummyAuthID, testHeader[string, uint]](store)
// 	require.NoError(t, err)
// 	require.NotNil(t, bestJust)
// 	require.Equal(t, justification, *bestJust)
// }
