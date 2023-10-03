// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
	"reflect"
	"testing"
)

// Fulfills HashI interface
type testHash string

func (s testHash) IsEmpty() bool {
	return len(s) == 0
}

// Fulfills Header interface
type testHeader[H HashI, N constraints.Unsigned] struct {
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

// Fulfills HeaderBackend interface
type testBackend[H HashI, N constraints.Unsigned, Header testHeader[H, N]] struct {
	header *testHeader[H, N]
}

func (backend testBackend[H, N, Header]) Header(hash H) (*testHeader[H, N], error) {
	return backend.header, nil
}

func makePrecommit(t *testing.T, targetHash string, targetNumber uint, id int32) finalityGrandpa.SignedPrecommit[testHash, uint, string, int32] {
	t.Helper()
	return finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]{
		Precommit: finalityGrandpa.Precommit[testHash, uint]{
			TargetHash:   testHash(targetHash),
			TargetNumber: targetNumber,
		},
		ID: id,
	}
}

func TestJustificationEncoding(t *testing.T) {
	var precommits []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 4; i++ {
		ids = append(ids, int32(i))
	}
	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	expAncestries := make([]testHeader[testHash, uint], 0)
	expAncestries = append(expAncestries, testHeader[testHash, uint]{
		NumberField:     100,
		ParentHashField: "a",
	})

	justification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Round: 2,
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: expAncestries,
	}

	encodedJustification, err := scale.Marshal(justification)
	require.NoError(t, err)

	newJustificaiton := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{}
	err = scale.Unmarshal(encodedJustification, &newJustificaiton)
	require.NoError(t, err)
	require.Equal(t, justification, newJustificaiton)
}

func TestJustification_fromCommit(t *testing.T) {
	commit := finalityGrandpa.Commit[testHash, uint, string, int32]{}
	client := testBackend[testHash, uint, testHeader[testHash, uint]]{}
	_, err := fromCommit[testHash, uint, string, int32, testHeader[testHash, uint]](client, 2, commit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// nil header
	var precommits []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 4; i++ {
		ids = append(ids, int32(i))
	}

	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validCommit := finalityGrandpa.Commit[testHash, uint, string, int32]{
		TargetHash:   "a",
		TargetNumber: 1,
		Precommits:   precommits,
	}

	clientNil := testBackend[testHash, uint, testHeader[testHash, uint]]{}

	_, err = fromCommit[testHash, uint, string, int32, testHeader[testHash, uint]](clientNil, 2, validCommit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// currentHeader.Number() <= baseNumber
	_, err = fromCommit[testHash, uint, string, int32, testHeader[testHash, uint]](client, 2, validCommit)
	require.NotNil(t, err)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, "bad justification for header: invalid precommits for target commit", err.Error())

	// happy path
	clientLargeNum := testBackend[testHash, uint, testHeader[testHash, uint]]{
		header: &testHeader[testHash, uint]{
			NumberField:     100,
			ParentHashField: "a",
		},
	}
	expAncestries := make([]testHeader[testHash, uint], 0)
	expAncestries = append(expAncestries, testHeader[testHash, uint]{
		NumberField:     100,
		ParentHashField: "a",
	})
	expJustification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Round: 2,
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: expAncestries,
	}
	justification, err := fromCommit[testHash, uint, string, int32, testHeader[testHash, uint]](clientLargeNum, 2, validCommit)
	require.NoError(t, err)
	require.Equal(t, expJustification, justification)
}

func TestJustification_decodeAndVerifyFinalizes(t *testing.T) {
	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := decodeAndVerifyFinalizes[testHash, uint, string, int32, testHeader[testHash, uint]](invalidEncoding, hashNumber[testHash, uint]{}, 2, finalityGrandpa.VoterSet[int32]{})
	require.NotNil(t, err)

	// Invalid target
	justification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
		},
	}

	encWrongTarget, err := scale.Marshal(justification)
	require.NoError(t, err)
	_, err = decodeAndVerifyFinalizes[testHash, uint, string, int32, testHeader[testHash, uint]](encWrongTarget, hashNumber[testHash, uint]{}, 2, finalityGrandpa.VoterSet[int32]{})
	require.NotNil(t, err)
	require.Equal(t, "invalid commit target in grandpa justification", err.Error())

	// Happy path
	headerB := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[testHash, uint]{
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 4; i++ {
		ids = append(ids, int32(i))
	}

	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	encValid, err := scale.Marshal(validJustification)
	require.NoError(t, err)

	target := hashNumber[testHash, uint]{
		hash:   "a",
		number: 1,
	}

	IDWeights := make([]finalityGrandpa.IDWeight[int32], 0)
	for i := 1; i <= 4; i++ {
		IDWeights = append(IDWeights, finalityGrandpa.IDWeight[int32]{int32(i), 1})
	}
	voters := finalityGrandpa.NewVoterSet(IDWeights)

	newJustification, err := decodeAndVerifyFinalizes[testHash, uint, string, int32, testHeader[testHash, uint]](encValid, target, 2, *voters)
	require.NoError(t, err)
	require.Equal(t, validJustification, newJustification)

}

func TestJustification_verify(t *testing.T) {
	// Nil voter case
	IDWeights := make([]finalityGrandpa.IDWeight[int32], 0)
	justification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{}
	err := justification.verify(2, IDWeights)
	require.ErrorIs(t, err, errInvalidAuthoritiesSet)

	// happy path
	for i := 1; i <= 4; i++ {
		IDWeights = append(IDWeights, finalityGrandpa.IDWeight[int32]{int32(i), 1})
	}

	headerB := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[testHash, uint]{
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 4; i++ {
		ids = append(ids, int32(i))
	}

	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	err = validJustification.verify(2, IDWeights)
	require.NoError(t, err)
}

func TestJustification_verifyWithVoterSet(t *testing.T) {
	// 1) invalid commit
	IDWeights := make([]finalityGrandpa.IDWeight[int32], 0)
	for i := 1; i <= 4; i++ {
		IDWeights = append(IDWeights, finalityGrandpa.IDWeight[int32]{int32(i), 1})
	}
	voters := finalityGrandpa.NewVoterSet(IDWeights)

	invalidJustification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "B",
			TargetNumber: 2,
			Precommits:   []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]{},
		},
	}

	err := invalidJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: invalid commit in grandpa justification")

	// 2) visitedHashes != ancestryHashes
	headerA := testHeader[testHash, uint]{
		HashField: "a",
	}

	headerB := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerList := []testHeader[testHash, uint]{
		headerA,
		headerB,
	}

	var precommits []finalityGrandpa.SignedPrecommit[testHash, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 4; i++ {
		ids = append(ids, int32(i))
	}

	precommit := makePrecommit(t, "a", 1, 1)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "a", 1, 2)
	precommits = append(precommits, precommit)

	precommit = makePrecommit(t, "b", 2, 3)
	precommits = append(precommits, precommit)

	validJustification := Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
			TargetHash:   "a",
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VotesAncestries: headerList,
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: invalid precommit ancestries in grandpa justification with unused headers")

	// Valid case
	headerList = []testHeader[testHash, uint]{
		headerB,
	}

	validJustification = Justification[testHash, uint, string, int32, testHeader[testHash, uint]]{
		Commit: finalityGrandpa.Commit[testHash, uint, string, int32]{
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
	dummyHeader := testHeader[testHash, uint]{
		HashField: "a",
	}
	expAncestryMap := make(map[testHash]testHeader[testHash, uint])
	hash := dummyHeader.Hash()
	expAncestryMap[hash] = dummyHeader
	type testCase[H HashI, N constraints.Unsigned] struct {
		name    string
		headers []testHeader[H, N]
		want    ancestryChain[H, N, testHeader[H, N]]
	}
	tests := []testCase[testHash, uint]{
		{
			name:    "no input headers",
			headers: []testHeader[testHash, uint]{},
			want: ancestryChain[testHash, uint, testHeader[testHash, uint]]{
				ancestry: make(map[testHash]testHeader[testHash, uint]),
			},
		},
		{
			name: "valid input",
			headers: []testHeader[testHash, uint]{
				dummyHeader,
			},
			want: ancestryChain[testHash, uint, testHeader[testHash, uint]]{
				ancestry: expAncestryMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newAncestryChain[testHash, uint](tt.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newAncestryChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAncestryChain_Ancestry(t *testing.T) {
	headerA := testHeader[testHash, uint]{
		HashField: "a",
	}

	headerB := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerC := testHeader[testHash, uint]{
		HashField:       "c",
		ParentHashField: "b",
	}

	invalidParentHeader := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "",
	}

	headerList := []testHeader[testHash, uint]{
		headerA,
		headerB,
		headerC,
	}
	invalidHeaderList := []testHeader[testHash, uint]{
		invalidParentHeader,
	}
	validAncestryMap := newAncestryChain[testHash, uint](headerList)
	invalidAncestryMap := newAncestryChain[testHash, uint](invalidHeaderList)
	type testCase[H HashI, N constraints.Unsigned] struct {
		name      string
		chain     ancestryChain[H, N, testHeader[H, N]]
		base      H
		block     H
		want      []H
		expErr    error
		expErrMsg string
	}
	tests := []testCase[testHash, uint]{
		{
			name:  "base equals block",
			chain: validAncestryMap,
			base:  "a",
			block: "a",
			want:  []testHash{},
		},
		{
			name:   "base equals block",
			chain:  validAncestryMap,
			base:   "a",
			block:  "d",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:   "invalid parent HashField",
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
			want:  []testHash{"b"},
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
	headerA := testHeader[testHash, uint]{
		HashField: "a",
	}

	headerB := testHeader[testHash, uint]{
		HashField:       "b",
		ParentHashField: "a",
	}

	headerC := testHeader[testHash, uint]{
		HashField:       "c",
		ParentHashField: "b",
	}

	headerList := []testHeader[testHash, uint]{
		headerA,
		headerB,
		headerC,
	}

	validAncestryMap := newAncestryChain[testHash, uint](headerList)
	type testCase[H HashI, N constraints.Unsigned] struct {
		name  string
		chain ancestryChain[H, N, testHeader[H, N]]
		base  H
		block H
		want  bool
	}
	tests := []testCase[testHash, uint]{
		{
			name:  "base equals block",
			chain: validAncestryMap,
			base:  "a",
			block: "a",
			want:  true,
		},
		{
			name:  "base equals block",
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
