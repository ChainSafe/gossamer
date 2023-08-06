// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// The context of a `Round` in which vote weights are calculated.
type Context[ID constraints.Ordered] struct {
	voters        VoterSet[ID]
	equivocations Bitfield
}

// Create a new context for a round with the given set of voters.
func NewContext[ID constraints.Ordered](voters VoterSet[ID]) Context[ID] {
	return Context[ID]{
		voters:        voters,
		equivocations: NewBitfield(),
	}
}

// Get the set of voters.
func (c Context[ID]) Voters() VoterSet[ID] {
	return c.voters
}

// Get the weight of observed equivocations in phase `p`.
func (c Context[ID]) EquivocationWeight(p Phase) VoteWeight {
	switch p {
	case PrevotePhase:
		return weight(c.equivocations.Iter1sEven(), c.voters)
	case PrecommitPhase:
		return weight(c.equivocations.Iter1sOdd(), c.voters)
	default:
		panic("wtf?")
	}
}

// Record voter `v` as an equivocator in phase `p`.
func (c *Context[ID]) Equivocated(v VoterInfo, p Phase) {
	c.equivocations.SetBit(NewVote[ID](v, p).bit.Position)
}

// Compute the vote weight on node `n` in phase `p`, taking into account
// equivocations.
func (c Context[ID]) Weight(n VoteNode[ID], p Phase) VoteWeight {
	if c.equivocations.IsBlank() {
		switch p {
		case PrevotePhase:
			return weight(n.bits.Iter1sEven(), c.voters)
		case PrecommitPhase:
			return weight(n.bits.Iter1sOdd(), c.voters)
		default:
			panic("wtf?")
		}
	} else {
		switch p {
		case PrevotePhase:
			bits := n.bits.Iter1sMergedEven(c.equivocations)
			return weight(bits, c.voters)
		case PrecommitPhase:
			bits := n.bits.Iter1sMergedOdd(c.equivocations)
			return weight(bits, c.voters)
		default:
			panic("wtf?")
		}
	}
}

// A single vote that can be incorporated into a `VoteNode`.
type Vote[ID constraints.Ordered] struct {
	bit Bit1
}

// Create a new vote cast by voter `v` in phase `p`.
func NewVote[ID constraints.Ordered](v VoterInfo, p Phase) Vote[ID] {
	switch p {
	case PrevotePhase:
		return Vote[ID]{
			bit: Bit1{
				Position: v.position * 2,
			},
		}
	case PrecommitPhase:
		return Vote[ID]{
			bit: Bit1{
				Position: v.position*2 + 1,
			},
		}
	default:
		panic("wtf?")
	}

}

// Get the voter who cast the vote from the given voter set,
// if it is contained in that set.
func (v Vote[ID]) voter(vs VoterSet[ID]) *idVoterInfo[ID] {
	return vs.nth(v.bit.Position / 2)
}

func weight[ID constraints.Ordered](bits []Bit1, voters VoterSet[ID]) (total VoteWeight) { //skipcq: RVV-B0001
	for _, bit := range bits {
		vote := Vote[ID]{bit}
		ivi := vote.voter(voters)
		if ivi != nil {
			total = total + VoteWeight(ivi.VoterInfo.weight)
		}
	}
	return
}

type voteNodeI[VoteNode, Vote any] interface {
	Add(other VoteNode)
	AddVote(other Vote)
	Copy() VoteNode
}

type VoteNode[ID constraints.Ordered] struct {
	bits Bitfield
}

func (vn *VoteNode[ID]) Add(other *VoteNode[ID]) {
	vn.bits.Merge(other.bits)
}

func (vn *VoteNode[ID]) AddVote(vote Vote[ID]) {
	vn.bits.SetBit(vote.bit.Position)
}

func (vn *VoteNode[ID]) Copy() *VoteNode[ID] {
	copiedBits := NewBitfield()
	copiedBits.bits = make([]uint64, len(vn.bits.bits))
	copy(copiedBits.bits, vn.bits.bits)
	return &VoteNode[ID]{
		bits: copiedBits,
	}
}
