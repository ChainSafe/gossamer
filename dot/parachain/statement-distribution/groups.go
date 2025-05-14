// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// groups tracks Validator groups within a session, plus some helpful indexing
// for looking up groups by validator indices or authority discovery ID.
type groups struct {
	groups           [][]parachaintypes.ValidatorIndex
	byValidatorIdx   map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex
	backingThreshold uint32
}

func newGroups(initGroups [][]parachaintypes.ValidatorIndex, backingThreshold uint32) *groups {
	if initGroups == nil {
		initGroups = [][]parachaintypes.ValidatorIndex{}
	}

	bvi := make(map[parachaintypes.ValidatorIndex]parachaintypes.GroupIndex)

	for groupIdx, group := range initGroups {
		for _, validatorIdx := range group {
			bvi[validatorIdx] = parachaintypes.GroupIndex(groupIdx)
		}
	}

	return &groups{
		groups:           initGroups,
		byValidatorIdx:   bvi,
		backingThreshold: backingThreshold,
	}
}

func (g *groups) all() [][]parachaintypes.ValidatorIndex {
	return g.groups
}

func (g *groups) get(groupIndex parachaintypes.GroupIndex) []parachaintypes.ValidatorIndex {
	if int(groupIndex) > len(g.groups)-1 {
		return nil
	}

	return g.groups[groupIndex]
}

func (g *groups) getSizeAndBackingThreshold(groupIndex parachaintypes.GroupIndex) (*uint32, *uint32) {
	group := g.get(groupIndex)
	if group == nil {
		return nil, nil
	}

	size := uint32(len(group))
	threshold := uint32(backing.EffectiveMinimumBackingVotes(uint(size), g.backingThreshold))
	return &size, &threshold
}

func (g *groups) byValidatorIndex(validatorIndex parachaintypes.ValidatorIndex) *parachaintypes.GroupIndex {
	groupIndex, ok := g.byValidatorIdx[validatorIndex]
	if !ok {
		return nil
	}

	return &groupIndex
}
