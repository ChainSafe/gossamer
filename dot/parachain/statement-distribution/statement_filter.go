// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type statementKind uint8

const (
	seconded statementKind = iota
	validated
)

// statementFilter contains bitfields indicating the statements that are known or undesired about a candidate.
type statementFilter struct {
	// seconded statements. '1' is known or undesired.
	secondedInGroup parachaintypes.BitVec
	// Valid statements. '1' is known or undesired.
	validatedInGroup parachaintypes.BitVec
}

// newStatementFilter creates a new statementFilter.
// If full is true, the statementFilter will be initialised with all bits set to 1.
func newStatementFilter(groupSize uint, full bool) (*statementFilter, error) {
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

	return &statementFilter{
		secondedInGroup:  secondedInGroup,
		validatedInGroup: validatedInGroup,
	}, nil
}

// hasLen returns true if the statementFilter has the specified length in both groups.
func (s *statementFilter) hasLen(len int) bool {
	return s.secondedInGroup.Len() == len && s.validatedInGroup.Len() == len
}

// backingValidators determines the number of backing validators in the statementFilter.
func (s *statementFilter) backingValidators() int {
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

// hasSeconded returns true if the statementFilter has at least one seconded statement.
func (s *statementFilter) hasSeconded() bool {
	return s.secondedInGroup.CountOnes() > 0
}

// maskSeconded masks out seconded statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *statementFilter) maskSeconded(mask parachaintypes.BitVec) {
	s.secondedInGroup.Mask(mask)
}

// maskValid masks out Valid statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *statementFilter) maskValid(mask parachaintypes.BitVec) {
	s.validatedInGroup.Mask(mask)
}

// clone returns a deep copy of the statement filter.
func (s *statementFilter) clone() statementFilter {
	return statementFilter{
		secondedInGroup:  s.secondedInGroup.Clone(),
		validatedInGroup: s.validatedInGroup.Clone(),
	}
}

func (s *statementFilter) contains(index uint, statementKind statementKind) bool {
	switch statementKind {
	case seconded:
		b, err := s.secondedInGroup.Get(index)
		if err != nil {
			logger.Warnf("failed to access index %d in secondedInGroup: %v", index, err)
			return false
		}
		return b
	case validated:
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

func (s *statementFilter) set(index uint, statementKind statementKind) {
	switch statementKind {
	case seconded:
		err := s.secondedInGroup.Set(index, true)
		if err != nil {
			logger.Warnf("failed to set index %d in secondedInGroup: %v", index, err)
		}
	case validated:
		err := s.validatedInGroup.Set(index, true)
		if err != nil {
			logger.Warnf("failed to set index %d in validatedInGroup: %v", index, err)
		}
	default:
		panic("unreachable")
	}
}
