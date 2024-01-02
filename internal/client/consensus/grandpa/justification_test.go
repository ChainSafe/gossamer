// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

func makePrecommit(t *testing.T,
	targetHash string,
	targetNumber uint,
	id app.Public,
) grandpa.SignedPrecommit[string, uint, string, string] {
	t.Helper()
	return grandpa.SignedPrecommit[string, uint, string, string]{
		Precommit: grandpa.Precommit[string, uint]{
			TargetHash:   targetHash,
			TargetNumber: targetNumber,
		},
		ID: id.String(),
	}
}

// func TestJustificationEncoding(t *testing.T) {
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

// 	encodedJustification, err := scale.Marshal(justification)
// 	require.NoError(t, err)

// 	newJustificaiton, err := decodeJustification[
// 		string,
// 		uint,
// 		string,
// 		dummyAuthID,
// 		testHeader[string, uint],
// 	](encodedJustification)
// 	require.NoError(t, err)
// 	require.Equal(t, justification, *newJustificaiton)
// }

// func TestJustification_fromCommit(t *testing.T) {
// 	commit := grandpa.Commit[string, uint, string, dummyAuthID]{}
// 	client := testHeaderBackend[string, uint]{}
// 	_, err := NewJustificationFromCommit[string, uint, string, dummyAuthID](client, 2, commit)
// 	require.NotNil(t, err)
// 	require.ErrorIs(t, err, errBadJustification)
// 	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

// 	// nil header
// 	var precommits []grandpa.SignedPrecommit[string, uint, string, dummyAuthID]
// 	precommit := makePrecommit(t, "a", 1, 1)
// 	precommits = append(precommits, precommit)

// 	precommit = makePrecommit(t, "b", 2, 3)
// 	precommits = append(precommits, precommit)

// 	validCommit := grandpa.Commit[string, uint, string, dummyAuthID]{
// 		TargetHash:   "a",
// 		TargetNumber: 1,
// 		Precommits:   precommits,
// 	}

// 	clientNil := testHeaderBackend[string, uint]{}

// 	_, err = NewJustificationFromCommit[string, uint, string, dummyAuthID](
// 		clientNil,
// 		2,
// 		validCommit,
// 	)
// 	require.NotNil(t, err)
// 	require.ErrorIs(t, err, errBadJustification)
// 	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

// 	// currentHeader.Number() <= baseNumber
// 	_, err = NewJustificationFromCommit[string, uint, string, dummyAuthID](
// 		client,
// 		2,
// 		validCommit,
// 	)
// 	require.NotNil(t, err)
// 	require.ErrorIs(t, err, errBadJustification)
// 	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

// 	// happy path
// 	header := Header[string, uint](testHeader[string, uint]{
// 		NumberField:     100,
// 		ParentHashField: "a",
// 	})
// 	clientLargeNum := testHeaderBackend[string, uint]{
// 		header: &header,
// 	}
// 	expAncestries := make([]Header[string, uint], 0)
// 	expAncestries = append(expAncestries, testHeader[string, uint]{
// 		NumberField:     100,
// 		ParentHashField: "a",
// 	})
// 	expJustification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Round: 2,
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 			Precommits:   precommits,
// 		},
// 		VotesAncestries: expAncestries,
// 	}
// 	justification, err := NewJustificationFromCommit[string, uint, string, dummyAuthID](
// 		clientLargeNum,
// 		2,
// 		validCommit)
// 	require.NoError(t, err)
// 	require.Equal(t, expJustification, justification)
// }

// func TestJustification_decodeAndVerifyFinalizes(t *testing.T) {
// 	// Invalid Encoding
// 	invalidEncoding := []byte{21}
// 	_, err := decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
// 		invalidEncoding,
// 		hashNumber[string, uint]{},
// 		2,
// 		grandpa.VoterSet[dummyAuthID]{})
// 	require.NotNil(t, err)

// 	// Invalid target
// 	justification := GrandpaJustification[string, uint, string, dummyAuthID]{
// 		Commit: grandpa.Commit[string, uint, string, dummyAuthID]{
// 			TargetHash:   "a",
// 			TargetNumber: 1,
// 		},
// 	}

// 	encWrongTarget, err := scale.Marshal(justification)
// 	require.NoError(t, err)
// 	_, err = decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
// 		encWrongTarget,
// 		hashNumber[string, uint]{},
// 		2,
// 		grandpa.VoterSet[dummyAuthID]{})
// 	require.NotNil(t, err)
// 	require.Equal(t, "invalid commit target in grandpa justification", err.Error())

// 	// Happy path
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

// 	encValid, err := scale.Marshal(validJustification)
// 	require.NoError(t, err)

// 	target := hashNumber[string, uint]{
// 		hash:   "a",
// 		number: 1,
// 	}

// 	IDWeights := make([]grandpa.IDWeight[dummyAuthID], 0)
// 	for i := 1; i <= 4; i++ {
// 		IDWeights = append(IDWeights, grandpa.IDWeight[dummyAuthID]{dummyAuthID(i), 1}) //nolint
// 	}
// 	voters := grandpa.NewVoterSet(IDWeights)

// 	newJustification, err := decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
// 		encValid,
// 		target,
// 		2,
// 		*voters)
// 	require.NoError(t, err)
// 	require.Equal(t, validJustification, newJustification)
// }

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
