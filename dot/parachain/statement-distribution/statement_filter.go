// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type StatementKind uint8

const (
	Seconded StatementKind = iota
	Validated
)

type FilterQuery interface {
	Contains(index uint, statementKind StatementKind) bool
	Set(index uint, statementKind StatementKind)
}

// StatementFilter contains bitfields indicating the statements that are known or undesired about a candidate.
type StatementFilter struct {
	// Seconded statements. '1' is known or undesired.
	secondedInGroup parachaintypes.BitVec
	// Valid statements. '1' is known or undesired.
	validatedInGroup parachaintypes.BitVec
}

var _ FilterQuery = (*StatementFilter)(nil)

// NewStatementFilter creates a new StatementFilter.
// If full is true, the StatementFilter will be initialised with all bits set to 1.
func NewStatementFilter(groupSize uint, full bool) (*StatementFilter, error) {
	bits := make([]bool, groupSize)
	if full {
		for i := range bits {
			bits[i] = true
		}
	}

	secondedInGroup, err := parachaintypes.NewBitVec(bits)
	if err != nil {
		return nil, err
	}

	validatedInGroup, err := parachaintypes.NewBitVec(bits)
	if err != nil {
		return nil, err
	}

	return &StatementFilter{
		secondedInGroup:  secondedInGroup,
		validatedInGroup: validatedInGroup,
	}, nil
}

// HasLen returns true if the StatementFilter has the specified length in both groups.
func (s *StatementFilter) HasLen(len int) bool {
	return s.secondedInGroup.Len() == len && s.validatedInGroup.Len() == len
}

// BackingValidators determines the number of backing validators in the StatementFilter.
func (s *StatementFilter) BackingValidators() int {
	count := 0

	for i, seconded := range s.secondedInGroup.Bits() {
		validated, err := s.validatedInGroup.Get(uint(i))
		if err != nil {
			panic("both groups were constructed with the same size. qed")
		}

		if seconded || validated { // no double-counting
			count++
		}
	}

	return count
}

// HasSeconded returns true if the StatementFilter has at least one seconded statement.
func (s *StatementFilter) HasSeconded() bool {
	return s.secondedInGroup.CountOnes() > 0
}

// MaskSeconded masks out Seconded statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *StatementFilter) MaskSeconded(mask parachaintypes.BitVec) {
	s.secondedInGroup.Mask(mask)
}

// MaskValid masks out Valid statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *StatementFilter) MaskValid(mask parachaintypes.BitVec) {
	s.validatedInGroup.Mask(mask)
}

// FilterQuery implementation

func (s *StatementFilter) Contains(index uint, statementKind StatementKind) bool {
	switch statementKind {
	case Seconded:
		b, err := s.secondedInGroup.Get(index)
		if err != nil {
			logger.Warnf("failed to access index %d in secondedInGroup: %v", index, err)
			return false
		}
		return b
	case Validated:
		b, err := s.validatedInGroup.Get(index)
		if err != nil {
			logger.Warnf("failed to access index %d in validatedInGroup: %v", index, err)
			return false
		}
		return b
	default:
		panic("unreachable")
	}
}

func (s *StatementFilter) Set(index uint, statementKind StatementKind) {
	switch statementKind {
	case Seconded:
		err := s.secondedInGroup.Set(index, true)
		if err != nil {
			logger.Warnf("failed to set index %d in secondedInGroup: %v", index, err)
		}
	case Validated:
		err := s.validatedInGroup.Set(index, true)
		if err != nil {
			logger.Warnf("failed to set index %d in validatedInGroup: %v", index, err)
		}
	default:
		panic("unreachable")
	}
}
