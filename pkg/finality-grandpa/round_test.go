// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVoteMultiplicity_Contains(t *testing.T) {
	type headerNumber struct {
		Header string
		Number uint
	}
	type signature string
	var (
		headerNumber1 = headerNumber{"header1", 1}
		signature1    = signature("sig1")
		headerNumber2 = headerNumber{"header2", 2}
		signature2    = signature("sig2")
	)
	tests := []struct {
		name  string
		value interface{}
		args  voteSignature[headerNumber, signature]
		want  bool
	}{
		{
			name: "Single",
			value: Single[headerNumber, signature]{
				headerNumber1,
				signature1,
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber1,
				signature1,
			},
			want: true,
		},
		{
			name: "Single",
			value: Single[headerNumber, signature]{
				headerNumber1,
				signature1,
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber2,
				signature2,
			},
			want: false,
		},
		{
			name: "Equivocated",
			value: Equivocated[headerNumber, signature]{
				{headerNumber1, signature1},
				{headerNumber2, signature2},
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber1,
				signature1,
			},
			want: true,
		},
		{
			name: "Equivocated",
			value: Equivocated[headerNumber, signature]{
				{headerNumber1, signature1},
				{headerNumber2, signature2},
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber2,
				signature2,
			},
			want: true,
		},
		{
			name: "Equivocated",
			value: Equivocated[headerNumber, signature]{
				{headerNumber1, signature1},
				{headerNumber2, signature2},
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber1,
				signature1,
			},
			want: true,
		},
		{
			name: "Equivocated",
			value: Equivocated[headerNumber, signature]{
				{headerNumber1, signature1},
				{headerNumber2, signature2},
			},
			args: voteSignature[headerNumber, signature]{
				headerNumber{"bleh", 99},
				signature("bleh"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := VoteMultiplicity[headerNumber, signature]{}
			vm.MustSet(tt.value)
			got := vm.Contains(tt.args.Vote, tt.args.Signature)
			if got != tt.want {
				t.Errorf("VoteMultiplicity.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_EstimateIsValid(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	chain.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	chain.PushBlocks("F", []string{"FA", "FB", "FC"})
	voters := NewVoterSet([]IDWeight[string]{{"Alice", 4}, {"Bob", 7}, {"Eve", 3}})

	round := NewRound[string, string, uint32, string](RoundParams[string, string, uint32]{
		RoundNumber: 1,
		Voters:      *voters,
		Base:        HashNumber[string, uint32]{"C", 4},
	})

	_, err := round.importPrevote(chain, Prevote[string, uint32]{"FC", 10}, "Alice", "Alice")
	assert.NoError(t, err)

	_, err = round.importPrevote(chain, Prevote[string, uint32]{"ED", 10}, "Bob", "Bob")
	assert.NoError(t, err)

	assert.Equal(t, HashNumber[string, uint32]{"E", 6}, *round.prevoteGhost)
	assert.Equal(t, HashNumber[string, uint32]{"E", 6}, *round.estimate)
	assert.False(t, round.completable)

	_, err = round.importPrevote(chain, Prevote[string, uint32]{"F", 7}, "Eve", "Eve")
	assert.NoError(t, err)

	assert.Equal(t, HashNumber[string, uint32]{"E", 6}, *round.prevoteGhost)
	assert.Equal(t, HashNumber[string, uint32]{"E", 6}, *round.estimate)
}

func TestRound_Finalisation(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	chain.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	chain.PushBlocks("F", []string{"FA", "FB", "FC"})

	voters := NewVoterSet([]IDWeight[string]{{"Alice", 4}, {"Bob", 7}, {"Eve", 3}})
	round := NewRound[string, string, uint32, string](RoundParams[string, string, uint32]{
		RoundNumber: 1,
		Voters:      *voters,
		Base:        HashNumber[string, uint32]{"C", 4},
	})

	ir1, err := round.importPrecommit(chain, Precommit[string, uint32]{"FC", 10}, "Alice", "Alice")
	assert.NoError(t, err)
	assert.NotNil(t, ir1)

	ir1, err = round.importPrecommit(chain, Precommit[string, uint32]{"ED", 10}, "Bob", "Bob")
	assert.NoError(t, err)
	assert.NotNil(t, ir1)

	assert.Nil(t, round.finalized)

	// import some prevotes.
	{
		ir, err := round.importPrevote(chain, Prevote[string, uint32]{"FC", 10}, "Alice", "Alice")
		assert.NoError(t, err)
		assert.NotNil(t, ir)

		ir, err = round.importPrevote(chain, Prevote[string, uint32]{"ED", 10}, "Bob", "Bob")
		assert.NoError(t, err)
		assert.NotNil(t, ir)

		ir, err = round.importPrevote(chain, Prevote[string, uint32]{"EA", 7}, "Eve", "Eve")
		assert.NoError(t, err)
		assert.NotNil(t, ir)

		assert.Equal(t, &HashNumber[string, uint32]{"E", 6}, round.finalized)
	}

	ir1, err = round.importPrecommit(chain, Precommit[string, uint32]{"EA", 7}, "Eve", "Eve")
	assert.NoError(t, err)
	assert.NotNil(t, ir1)

	assert.Equal(t, &HashNumber[string, uint32]{"EA", 7}, round.finalized)
}

func TestRound_EquivocateDoesNotDoubleCount(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	chain.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	chain.PushBlocks("F", []string{"FA", "FB", "FC"})

	voters := NewVoterSet([]IDWeight[string]{{"Alice", 4}, {"Bob", 7}, {"Eve", 3}})
	round := NewRound[string, string, uint32, string](RoundParams[string, string, uint32]{
		RoundNumber: 1,
		Voters:      *voters,
		Base:        HashNumber[string, uint32]{"C", 4},
	})

	// first prevote by eve
	ir, err := round.importPrevote(chain, Prevote[string, uint32]{"FC", 10}, "Eve", "Eve-1")
	assert.NoError(t, err)
	assert.NotNil(t, ir)
	assert.Nil(t, ir.Equivocation)

	assert.Nil(t, round.prevoteGhost)

	// second prevote by eve: comes with equivocation proof
	ir, err = round.importPrevote(chain, Prevote[string, uint32]{"ED", 10}, "Eve", "Eve-2")
	assert.NoError(t, err)
	assert.NotNil(t, ir)
	assert.NotNil(t, ir.Equivocation)

	// third prevote: returns nothing.
	ir, err = round.importPrevote(chain, Prevote[string, uint32]{"F", 7}, "Eve", "Eve-2")
	assert.NoError(t, err)
	assert.NotNil(t, ir)
	assert.Nil(t, ir.Equivocation)

	// three eves together would be enough.
	assert.Nil(t, round.prevoteGhost)

	ir, err = round.importPrevote(chain, Prevote[string, uint32]{"FA", 8}, "Bob", "Bob-1")
	assert.NoError(t, err)
	assert.NotNil(t, ir)
	assert.Nil(t, ir.Equivocation)

	assert.Equal(t, &HashNumber[string, uint32]{"FA", 8}, round.prevoteGhost)
}

func TestRound_HistoricalVotesWorks(t *testing.T) {
	chain := NewDummyChain()
	chain.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	chain.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	chain.PushBlocks("F", []string{"FA", "FB", "FC"})

	voters := NewVoterSet([]IDWeight[string]{{"Alice", 4}, {"Bob", 7}, {"Eve", 3}})
	round := NewRound[string, string, uint32, string](RoundParams[string, string, uint32]{
		RoundNumber: 1,
		Voters:      *voters,
		Base:        HashNumber[string, uint32]{"C", 4},
	})

	ir, err := round.importPrevote(chain, Prevote[string, uint32]{"FC", 10}, "Alice", "Alice")
	assert.NoError(t, err)
	assert.NotNil(t, ir)

	round.historicalVotes.SetPrevotedIdx()

	ir, err = round.importPrevote(chain, Prevote[string, uint32]{"EA", 7}, "Eve", "Eve")
	assert.NoError(t, err)
	assert.NotNil(t, ir)

	ir1, err := round.importPrecommit(chain, Precommit[string, uint32]{"EA", 7}, "Eve", "Eve")
	assert.NoError(t, err)
	assert.NotNil(t, ir1)

	ir, err = round.importPrevote(chain, Prevote[string, uint32]{"EC", 10}, "Alice", "Alice")
	assert.NoError(t, err)
	assert.NotNil(t, ir)

	round.historicalVotes.SetPrecommittedIdx()

	var newUint32 = func(ui uint64) *uint64 {
		return &ui
	}
	assert.Equal(t, HistoricalVotes[string, uint32, string, string]{
		seen: []SignedMessage[string, uint32, string, string]{
			{
				Message: Message[string, uint32]{
					value: Prevote[string, uint32]{
						TargetHash:   "FC",
						TargetNumber: 10,
					},
				},
				Signature: "Alice",
				ID:        "Alice",
			},
			{
				Message: Message[string, uint32]{
					value: Prevote[string, uint32]{
						TargetHash:   "EA",
						TargetNumber: 7,
					},
				},
				Signature: "Eve",
				ID:        "Eve",
			},
			{
				Message: Message[string, uint32]{
					value: Precommit[string, uint32]{
						TargetHash:   "EA",
						TargetNumber: 7,
					},
				},
				Signature: "Eve",
				ID:        "Eve",
			},
			{
				Message: Message[string, uint32]{
					value: Prevote[string, uint32]{
						TargetHash:   "EC",
						TargetNumber: 10,
					},
				},
				Signature: "Alice",
				ID:        "Alice",
			},
		},
		prevoteIdx:   newUint32(1),
		precommitIdx: newUint32(4),
	}, round.historicalVotes)
}
