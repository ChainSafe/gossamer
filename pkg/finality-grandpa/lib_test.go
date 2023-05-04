// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCommit(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A"})

	IDWeights := make([]IDWeight[int32], 0)
	for i := 1; i <= 100; i++ {
		IDWeights = append(IDWeights, IDWeight[int32]{int32(i), 1})
	}
	voters := NewVoterSet(IDWeights)

	makePrecommit := func(targetHash string, targetNumber uint, id int32) SignedPrecommit[string, uint, string, int32] {
		return SignedPrecommit[string, uint, string, int32]{
			Precommit: Precommit[string, uint]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
			},
			ID: id,
		}
	}

	var precommits []SignedPrecommit[string, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 67; i++ {
		ids = append(ids, int32(i))
	}
	for _, id := range ids {
		precommit := makePrecommit("C", 3, id)
		precommits = append(precommits, precommit)
	}

	// we have still not reached threshold with 66/100 votes, so the commit
	// is not valid.
	result, err := ValidateCommit[string, uint, string](
		Commit[string, uint, string, int32]{
			TargetHash:   "C",
			TargetNumber: 3,
			Precommits:   precommits,
		}, *voters, chain)
	assert.NoError(t, err)

	assert.False(t, result.Valid())

	// after adding one more commit targetting the same block we are over
	// the finalization threshold and the commit should be valid
	precommits = append(precommits, makePrecommit("C", 3, 67))

	result, err = ValidateCommit[string, uint, string](
		Commit[string, uint, string, int32]{
			TargetHash:   "C",
			TargetNumber: 3,
			Precommits:   precommits,
		}, *voters, chain)
	assert.NoError(t, err)

	assert.True(t, result.Valid())

	// the commit target must be the exact same as the round precommit ghost
	// that is calculated with the given precommits for the commit to be valid
	result, err = ValidateCommit[string, uint, string](
		Commit[string, uint, string, int32]{
			TargetHash:   "B",
			TargetNumber: 2,
			Precommits:   precommits,
		}, *voters, chain)
	assert.NoError(t, err)

	assert.False(t, result.Valid())
}

func TestValidateCommit_WithEquivocation(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C"})

	IDWeights := make([]IDWeight[int32], 0)
	for i := 1; i <= 100; i++ {
		IDWeights = append(IDWeights, IDWeight[int32]{int32(i), 1})
	}
	voters := NewVoterSet(IDWeights)

	makePrecommit := func(targetHash string, targetNumber uint, id int32) SignedPrecommit[string, uint, string, int32] {
		return SignedPrecommit[string, uint, string, int32]{
			Precommit: Precommit[string, uint]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
			},
			ID: id,
		}
	}

	// we add 66/100 precommits targeting block C
	var precommits []SignedPrecommit[string, uint, string, int32]
	ids := make([]int32, 0)
	for i := 1; i < 67; i++ {
		ids = append(ids, int32(i))
	}
	for _, id := range ids {
		precommit := makePrecommit("C", 3, id)
		precommits = append(precommits, precommit)
	}

	// we then add two equivocated votes targeting A and B
	// from the 67th validator
	precommits = append(precommits, makePrecommit("A", 1, 67))
	precommits = append(precommits, makePrecommit("B", 2, 67))

	// this equivocation is treated as "voting for all blocks", which means
	// that block C will now have 67/100 votes and therefore it can be
	// finalized.
	result, err := ValidateCommit[string, uint, string](
		Commit[string, uint, string, int32]{
			TargetHash:   "C",
			TargetNumber: 3,
			Precommits:   precommits,
		}, *voters, chain)
	assert.NoError(t, err)

	assert.True(t, result.Valid())
	assert.Equal(t, uint(1), result.NumEquiovcations())
}

func TestValidateCommit_PrecommitFromUnknownVoterIsIgnored(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C"})

	IDWeights := make([]IDWeight[int32], 0)
	for i := 1; i <= 100; i++ {
		IDWeights = append(IDWeights, IDWeight[int32]{int32(i), 1})
	}
	voters := NewVoterSet(IDWeights)

	makePrecommit := func(targetHash string, targetNumber uint, id int32) SignedPrecommit[string, uint, string, int32] {
		return SignedPrecommit[string, uint, string, int32]{
			Precommit: Precommit[string, uint]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
			},
			ID: id,
		}
	}

	var precommits []SignedPrecommit[string, uint, string, int32]

	// invalid vote from unknown voter should not influence the base
	precommits = append(precommits, makePrecommit("Z", 1, 1000))

	ids := make([]int32, 0)
	for i := 1; i <= 67; i++ {
		ids = append(ids, int32(i))
	}
	for _, id := range ids {
		precommit := makePrecommit("C", 3, id)
		precommits = append(precommits, precommit)
	}

	result, err := ValidateCommit[string, uint](
		Commit[string, uint, string, int32]{
			TargetHash:   "C",
			TargetNumber: 3,
			Precommits:   precommits,
		}, *voters, chain)
	assert.NoError(t, err)

	// we have threshold votes for block "C" so it should be valid
	assert.True(t, result.Valid())

	// there is one invalid voter in the commit
	assert.Equal(t, uint(1), result.NumInvalidVoters())
}
