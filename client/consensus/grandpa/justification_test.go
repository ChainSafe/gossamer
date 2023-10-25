// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"reflect"
	"testing"

	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
)

// Fulfils Header interface
type testHeader[H constraints.Ordered, N constraints.Unsigned] struct {
	ParentHashField H
	NumberField     N
	StateRoot       H
	ExtrinsicsRoot  H
	HashField       H
}

func (s testHeader[H, N]) ParentHash() H {
	return s.ParentHashField
}

func (s testHeader[H, N]) Hash() H {
	return s.HashField
}

func (s testHeader[H, N]) Number() N {
	return s.NumberField
}

// Fulfils HeaderBackend interface
type testBackend[H constraints.Ordered, N constraints.Unsigned, Header testHeader[H, N]] struct {
	header *testHeader[H, N]
}

func (backend testBackend[H, N, Header]) Header(hash H) (*testHeader[H, N], error) {
	return backend.header, nil
}

func makePrecommit(t *testing.T,
	targetHash string,
	targetNumber uint, id dummyAuthID) finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID] {
	t.Helper()
	return finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]{
		Precommit: finalityGrandpa.Precommit[string, uint]{
			TargetHash:   targetHash,
			TargetNumber: targetNumber,
		},
		ID: id,
	}
}

func TestJustificationEncoding(t *testing.T) {
	var precommits []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	expAncestries := make([]testHeader[string, uint], 0)
	expAncestries = append(expAncestries, testHeader[string, uint]{
		NumberField:     100,
		ParentHashField: "a",
	})

	justification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Round: 2,
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: expAncestries,
	}

	encodedJustification, err := scale.Marshal(justification)
	require.NoError(t, err)

	newJustificaiton := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{}
	err = scale.Unmarshal(encodedJustification, &newJustificaiton)
	require.NoError(t, err)
	require.Equal(t, justification, newJustificaiton)
}

func TestJustification_fromCommit(t *testing.T) {
	commit := finalityGrandpa.Commit[string, uint, string, dummyAuthID]{}
	client := testBackend[string, uint, testHeader[string, uint]]{}
	_, err := NewJustificationFromCommit[string, uint, string, dummyAuthID, testHeader[string, uint]](client, 2, commit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// nil header
	var precommits []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validCommit := finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
		TargetHash:   "a",
		TargetNumber: 1,
		Precommits:   precommits,
	}

	clientNil := testBackend[string, uint, testHeader[string, uint]]{}

	_, err = NewJustificationFromCommit[string, uint, string, dummyAuthID, testHeader[string, uint]](clientNil, 2, validCommit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// currentHeader.Number() <= baseNumber
	_, err = NewJustificationFromCommit[string, uint, string, dummyAuthID, testHeader[string, uint]](client, 2, validCommit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// happy path
	clientLargeNum := testBackend[string, uint, testHeader[string, uint]]{
		header: &testHeader[string, uint]{
			NumberField:     100,
			ParentHashField: "a",
		},
	}
	expAncestries := make([]testHeader[string, uint], 0)
	expAncestries = append(expAncestries, testHeader[string, uint]{
		NumberField:     100,
		ParentHashField: "a",
	})
	expJustification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Round: 2,
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: expAncestries,
	}
	justification, err := NewJustificationFromCommit[string, uint, string, dummyAuthID, testHeader[string, uint]](
		clientLargeNum,
		2,
		validCommit)
	require.NoError(t, err)
	require.Equal(t, expJustification, justification)
}

func TestJustification_decodeAndVerifyFinalizes(t *testing.T) {
	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
		invalidEncoding,
		hashNumber[string, uint]{},
		2,
		finalityGrandpa.VoterSet[dummyAuthID]{})
	require.NotNil(t, err)

	// Invalid target
	justification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
		},
	}

	encWrongTarget, err := scale.Marshal(justification)
	require.NoError(t, err)
	_, err = decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
		encWrongTarget,
		hashNumber[string, uint]{},
		2,
		finalityGrandpa.VoterSet[dummyAuthID]{})
	require.NotNil(t, err)
	require.Equal(t, "invalid commit target in grandpa justification", err.Error())

	// Happy path
	headerB := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[string, uint]{
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	encValid, err := scale.Marshal(validJustification)
	require.NoError(t, err)

	target := hashNumber[string, uint]{
		hash:   "a",
		number: 1,
	}

	IDWeights := make([]finalityGrandpa.IDWeight[dummyAuthID], 0)
	for i := 1; i <= 4; i++ {
		IDWeights = append(IDWeights, finalityGrandpa.IDWeight[dummyAuthID]{dummyAuthID(i), 1}) //nolint
	}
	voters := finalityGrandpa.NewVoterSet(IDWeights)

	newJustification, err := decodeAndVerifyFinalizes[string, uint, string, dummyAuthID, testHeader[string, uint]](
		encValid,
		target,
		2,
		*voters)
	require.NoError(t, err)
	require.Equal(t, validJustification, newJustification)
}

func TestJustification_verify(t *testing.T) {
	// Nil voter case
	auths := make(AuthorityList[dummyAuthID], 0)
	justification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{}
	err := justification.Verify(2, auths)
	require.ErrorIs(t, err, errInvalidAuthoritiesSet)

	// happy path
	for i := 1; i <= 4; i++ {
		auths = append(auths, Authority[dummyAuthID]{
			dummyAuthID(i),
			1,
		})
	}

	headerB := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[string, uint]{
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	err = validJustification.Verify(2, auths)
	require.NoError(t, err)
}

func TestJustification_verifyWithVoterSet(t *testing.T) {
	// 1) invalid commit
	IDWeights := make([]finalityGrandpa.IDWeight[dummyAuthID], 0)
	for i := 1; i <= 4; i++ {
		IDWeights = append(IDWeights, finalityGrandpa.IDWeight[dummyAuthID]{dummyAuthID(i), 1}) //nolint
	}
	voters := finalityGrandpa.NewVoterSet(IDWeights)

	invalidJustification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "B",
			TargetNumber: 2,
			Precommits:   []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]{},
		},
	}

	err := invalidJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: invalid commit in grandpa justification")

	// 2) visitedHashes != ancestryHashes
	headerA := testHeader[string, uint]{
		HashField: "a",
	}

	headerB := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[string, uint]{
		headerA,
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[string, uint, string, dummyAuthID]
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: "+
		"invalid precommit ancestries in grandpa justification with unused headers")

	// Valid case
	headerList = []testHeader[string, uint]{
		headerB,
	}

	validJustification = Justification[string, uint, string, dummyAuthID, testHeader[string, uint]]{
		Commit: finalityGrandpa.Commit[string, uint, string, dummyAuthID]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.NoError(t, err)
}

func Test_newAncestryChain(t *testing.T) {
	dummyHeader := testHeader[string, uint]{
		HashField: "a",
	}
	expAncestryMap := make(map[string]testHeader[string, uint])
	hash := dummyHeader.Hash()
	expAncestryMap[hash] = dummyHeader
	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
		name    string
		headers []testHeader[H, N]
		want    ancestryChain[H, N, testHeader[H, N]]
	}
	tests := []testCase[string, uint]{
		{
			name:    "noInputHeaders",
			headers: []testHeader[string, uint]{},
			want: ancestryChain[string, uint, testHeader[string, uint]]{
				ancestry: make(map[string]testHeader[string, uint]),
			},
		},
		{
			name: "validInput",
			headers: []testHeader[string, uint]{
				dummyHeader,
			},
			want: ancestryChain[string, uint, testHeader[string, uint]]{
				ancestry: expAncestryMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newAncestryChain[string, uint](tt.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newAncestryChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAncestryChain_Ancestry(t *testing.T) {
	headerA := testHeader[string, uint]{
		HashField: "a",
	}

	headerB := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerC := testHeader[string, uint]{
		HashField:       "c",
		ParentHashField: "b",
	}

	invalidParentHeader := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "",
	}

	headerList := []testHeader[string, uint]{
		headerA,
		headerB,
		headerC,
	}
	invalidHeaderList := []testHeader[string, uint]{
		invalidParentHeader,
	}
	validAncestryMap := newAncestryChain[string, uint](headerList)
	invalidAncestryMap := newAncestryChain[string, uint](invalidHeaderList)
	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
		name   string
		chain  ancestryChain[H, N, testHeader[H, N]]
		base   H
		block  H
		want   []H
		expErr error
	}
	tests := []testCase[string, uint]{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  "a",
			block: "a",
			want:  []string{},
		},
		{
			name:   "baseEqualsBlock",
			chain:  validAncestryMap,
			base:   "a",
			block:  "d",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:   "invalidParentHashField",
			chain:  invalidAncestryMap,
			base:   "a",
			block:  "b",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  "a",
			block: "c",
			want:  []string{"b"},
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
	headerA := testHeader[string, uint]{
		HashField: "a",
	}

	headerB := testHeader[string, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerC := testHeader[string, uint]{
		HashField:       "c",
		ParentHashField: "b",
	}

	headerList := []testHeader[string, uint]{
		headerA,
		headerB,
		headerC,
	}

	validAncestryMap := newAncestryChain[string, uint](headerList)
	type testCase[H constraints.Ordered, N constraints.Unsigned] struct {
		name  string
		chain ancestryChain[H, N, testHeader[H, N]]
		base  H
		block H
		want  bool
	}
	tests := []testCase[string, uint]{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  "a",
			block: "a",
			want:  true,
		},
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  "a",
			block: "d",
			want:  false,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  "a",
			block: "c",
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
