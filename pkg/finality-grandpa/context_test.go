// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func (Phase) Generate(rand *rand.Rand, _ int) reflect.Value {
	index := rand.Intn(2)
	return reflect.ValueOf([]Phase{PrevotePhase, PrecommitPhase}[index])
}

func (context[ID]) Generate(rand *rand.Rand, size int) reflect.Value {
	vs := VoterSet[ID]{}.Generate(rand, size).Interface().(VoterSet[ID])

	n := rand.Int() % len(vs.voters)
	equivocators := make([]VoterInfo, n+1)
	for i := 0; i <= n; i++ {
		ivi := vs.nthMod(uint(rand.Uint64()))
		equivocators[i] = ivi.VoterInfo
	}

	c := context[ID]{
		voters: vs,
	}
	for _, v := range equivocators {
		c.Equivocated(v, Phase(0).Generate(rand, size).Interface().(Phase))
	}
	return reflect.ValueOf(c)
}

func TestVote_voter(t *testing.T) {
	f := func(vs VoterSet[uint], phase Phase) bool {
		for _, idv := range vs.iter() {
			id := idv.ID
			v := idv.VoterInfo
			eq := assert.Equal(t, &idVoterInfo[uint]{id, v}, newVote[uint](v, phase).voter(vs))
			if !eq {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestWeights(t *testing.T) {
	f := func(ctx context[uint], phase Phase, voters []uint) bool {
		ew := ctx.EquivocationWeight(phase)
		tw := ctx.voters.TotalWeight()

		// The equivocation weight must never be larger than the total
		// voter weight.
		if !assert.True(t, uint64(ew) <= uint64(tw)) {
			return false
		}

		// Let a random subset of voters cast a vote, whether already
		// an equivocator or not.
		n := voteNode[uint]{}
		expected := ew
		for _, v := range voters {
			idvi := ctx.voters.nthMod(v)
			vote := newVote[uint](idvi.VoterInfo, phase)

			// We only expect the weight to increase if the voter did not
			// start out as an equivocator and did not yet vote.
			if !ctx.equivocations.testBit(vote.bit.position) && !n.bits.testBit(vote.bit.position) {
				expected = expected + VoteWeight(idvi.VoterInfo.weight)
			}
			n.addVote(vote)
		}

		// Let the context compute the weight.
		w := ctx.Weight(n, phase)

		// A vote-node weight must never be greater than the total voter weight.
		if !assert.True(t, uint64(w) <= uint64(tw)) {
			return false
		}

		return assert.Equal(t, expected, w)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
