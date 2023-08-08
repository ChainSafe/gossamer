// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// The context of a `Round` in which vote weights are calculated.
type context[ID constraints.Ordered] struct {
	voters        VoterSet[ID]
	equivocations bitfield
}

// newContext will create a new context for a round with the given set of voters.
func newContext[ID constraints.Ordered](voters VoterSet[ID]) context[ID] {
	return context[ID]{
		voters:        voters,
		equivocations: newBitfield(),
	}
}

// Voters will return the set of voters.
func (c context[ID]) Voters() VoterSet[ID] {
	return c.voters
}

// EquivocationWeight returns the weight of observed equivocations in phase `p`.
func (c context[ID]) EquivocationWeight(p Phase) VoteWeight {
	switch p {
	case PrevotePhase:
		return weight(c.equivocations.Iter1sEven(), c.voters)
	case PrecommitPhase:
		return weight(c.equivocations.Iter1sOdd(), c.voters)
	default:
		panic("wtf?")
	}
}

// Equivocated will record voter `v` as an equivocator in phase `p`.
func (c *context[ID]) Equivocated(v VoterInfo, p Phase) {
	c.equivocations.SetBit(newVote[ID](v, p).bit.position)
}

// Weight computes the vote weight on node `n` in phase `p`, taking into account
// equivocations.
func (c context[ID]) Weight(n voteNode[ID], p Phase) VoteWeight {
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

// A single vote that can be incorporated into a `voteNode`.
type vote[ID constraints.Ordered] struct {
	bit bit1
}

// NewVote will create a new vote cast by voter `v` in phase `p`.
func newVote[ID constraints.Ordered](v VoterInfo, p Phase) vote[ID] {
	switch p {
	case PrevotePhase:
		return vote[ID]{
			bit: bit1{
				position: v.position * 2,
			},
		}
	case PrecommitPhase:
		return vote[ID]{
			bit: bit1{
				position: v.position*2 + 1,
			},
		}
	default:
		panic("wtf?")
	}
}

// Get the voter who cast the vote from the given voter set,
// if it is contained in that set.
func (v vote[ID]) voter(vs VoterSet[ID]) *idVoterInfo[ID] {
	return vs.nth(v.bit.position / 2)
}

func weight[ID constraints.Ordered](bits []bit1, voters VoterSet[ID]) (total VoteWeight) { //skipcq: RVV-B0001
	for _, bit := range bits {
		vote := vote[ID]{bit}
		ivi := vote.voter(voters)
		if ivi != nil {
			total = total + VoteWeight(ivi.VoterInfo.weight)
		}
	}
	return
}

type voteNodeI[voteNode, Vote any] interface {
	add(other voteNode)
	addVote(other Vote)
	copy() voteNode
}

type voteNode[ID constraints.Ordered] struct {
	bits bitfield
}

func (vn *voteNode[ID]) add(other *voteNode[ID]) {
	vn.bits.Merge(other.bits)
}

func (vn *voteNode[ID]) addVote(vote vote[ID]) {
	vn.bits.SetBit(vote.bit.position)
}

func (vn *voteNode[ID]) copy() *voteNode[ID] {
	copiedBits := newBitfield()
	copiedBits.bits = make([]uint64, len(vn.bits.bits))
	copy(copiedBits.bits, vn.bits.bits)
	return &voteNode[ID]{
		bits: copiedBits,
	}
}
