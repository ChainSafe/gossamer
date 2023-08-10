// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// IDVoterInfo is tuple for ID and VoterInfo
type IDVoterInfo[ID constraints.Ordered] struct {
	ID ID
	VoterInfo
}

// A (non-empty) set of voters and associated weights.
//
// A `VoterSet` identifies all voters that are permitted to vote in a round
// of the protocol and their associated weights. A `VoterSet` is furthermore
// equipped with a total order, given by the ordering of the voter's IDs.
type VoterSet[ID constraints.Ordered] struct {
	voters      []IDVoterInfo[ID]
	threshold   VoterWeight
	totalWeight VoterWeight
}

type IDWeight[ID constraints.Ordered] struct {
	ID     ID
	Weight VoterWeight
}

// Create a voter set from a weight distribution produced by the given iterator.
//
// If the distribution contains multiple weights for the same voter ID, they are
// understood to be partial weights and are accumulated. As a result, the
// order in which the iterator produces the weights is irrelevant.
//
// Returns `None` if the iterator does not yield a valid voter set, which is
// the case if it either produced no non-zero weights or, i.e. the voter set
// would be empty, or if the total voter weight exceeds `u64::MAX`.
func NewVoterSet[ID constraints.Ordered](weights []IDWeight[ID]) *VoterSet[ID] {
	var totalWeight VoterWeight
	var voters = btree.NewMap[ID, VoterInfo](2)
	for _, iw := range weights {
		if iw.Weight != 0 {
			err := totalWeight.checkedAdd(iw.Weight)
			if err != nil {
				return nil
			}
			vi, has := voters.Get(iw.ID)
			if !has {
				voters.Set(iw.ID, VoterInfo{
					position: 0, // The total order is determined afterwards.
					weight:   iw.Weight,
				})
			} else {
				vi.weight = iw.Weight
				voters.Set(iw.ID, vi)
			}
		}
	}

	if voters.Len() == 0 {
		return nil
	}

	var orderedVoters = make([]IDVoterInfo[ID], voters.Len())
	var i uint
	voters.Scan(func(id ID, info VoterInfo) bool {
		info.position = i
		orderedVoters[i] = IDVoterInfo[ID]{id, info}
		i++
		return true
	})

	if totalWeight == 0 {
		panic("huh?")
	}

	return &VoterSet[ID]{
		voters:      orderedVoters,
		totalWeight: totalWeight,
		threshold:   threshold(totalWeight),
	}
}

// Get the voter info for the voter with the given ID, if any.
func (vs VoterSet[ID]) Get(id ID) *VoterInfo {
	idx, ok := slices.BinarySearchFunc(vs.voters, IDVoterInfo[ID]{ID: id}, func(a, b IDVoterInfo[ID]) int {
		switch {
		case a.ID == b.ID:
			return 0
		case a.ID > b.ID:
			return 1
		case b.ID > a.ID:
			return -1
		default:
			panic("huh?")
		}
	})
	if ok {
		return &vs.voters[idx].VoterInfo
	}
	return nil
}

// Get the size of the set.
func (vs VoterSet[ID]) Len() int {
	return len(vs.voters)
}

// Whether the set contains a voter with the given ID.
func (vs VoterSet[ID]) Contains(id ID) bool {
	return vs.Get(id) != nil
}

// Get the nth voter in the set, modulo the size of the set,
// as per the associated total order.
func (vs VoterSet[ID]) NthMod(n uint) IDVoterInfo[ID] {
	ivi := vs.Nth(n % uint(len(vs.voters)))
	if ivi == nil {
		panic("set is nonempty and n % len < len; qed")
	}
	return *ivi
}

// Get the nth voter in the set, if any.
//
// Returns `None` if `n >= len`.
func (vs VoterSet[ID]) Nth(n uint) *IDVoterInfo[ID] {
	if n >= uint(len(vs.voters)) {
		return nil
	}
	return &IDVoterInfo[ID]{
		vs.voters[n].ID,
		vs.voters[n].VoterInfo,
	}
}

// Get the threshold vote weight required for supermajority
// w.r.t. this set of voters.
func (vs VoterSet[ID]) Threshold() VoterWeight {
	return vs.threshold
}

// Get the total weight of all voters.
func (vs VoterSet[ID]) TotalWeight() VoterWeight {
	return vs.totalWeight
}

// Get an iterator over the voters in the set, as given by
// the associated total order.
func (vs VoterSet[ID]) Iter() []IDVoterInfo[ID] {
	return vs.voters
}

// Information about a voter in a `VoterSet`.
type VoterInfo struct {
	position uint
	weight   VoterWeight
}

func (vi VoterInfo) Position() uint {
	return vi.position
}

func (vi VoterInfo) Weight() VoterWeight {
	return vi.weight
}

// Compute the threshold weight given the total voting weight.
func threshold(totalWeight VoterWeight) VoterWeight { //skipcq: RVV-B0001
	// TODO: implement saturating sub
	// let faulty = total_weight.get().saturating_sub(1) / 3;
	var faulty = (totalWeight - 1) / 3
	// TODO: check that this computation is NonZero
	// VoterWeight::new(total_weight.get() - faulty).expect("subtrahend > minuend; qed")
	return totalWeight - faulty
}
