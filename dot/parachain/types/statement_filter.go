// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

type StatementKind uint8

const (
	SecondedKind StatementKind = iota
	ValidKind
)

// StatementFilter contains bitfields indicating the statements that are known or undesired about a candidate.
type StatementFilter struct {
	// seconded statements. '1' is known or undesired.
	SecondedInGroup BitVec
	// Valid statements. '1' is known or undesired.
	ValidatedInGroup BitVec
}

// NewStatementFilter creates a new statementFilter.
// If full is true, the statementFilter will be initialised with all bits set to 1.
func NewStatementFilter(groupSize uint, full bool) (*StatementFilter, error) {
	bits := make([]bool, groupSize)
	if full {
		for i := range bits {
			bits[i] = true
		}
	}

	secondedInGroup, err := NewBitVec(bits)
	if err != nil {
		return nil, err
	}

	validatedInGroup, err := NewBitVec(bits)
	if err != nil {
		return nil, err
	}

	return &StatementFilter{
		SecondedInGroup:  secondedInGroup,
		ValidatedInGroup: validatedInGroup,
	}, nil
}

// HasLen returns true if the statementFilter has the specified length in both groups.
func (s *StatementFilter) HasLen(len int) bool {
	return s.SecondedInGroup.Len() == len && s.ValidatedInGroup.Len() == len
}

// BackingValidators determines the number of backing validators in the statementFilter.
func (s *StatementFilter) BackingValidators() int {
	count := 0

	for i, seconded := range s.SecondedInGroup.Bits() {
		validated, err := s.ValidatedInGroup.Get(uint(i))
		if err != nil {
			panic("both groups were constructed with the same size. qed")
		}

		if seconded || validated { // no double-counting
			count++
		}
	}

	return count
}

// HasSeconded returns true if the statementFilter has at least one seconded statement.
func (s *StatementFilter) HasSeconded() bool {
	return s.SecondedInGroup.CountOnes() > 0
}

// MaskSeconded masks out seconded statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *StatementFilter) MaskSeconded(mask BitVec) {
	s.SecondedInGroup.Mask(mask)
}

// MaskValid masks out Valid statements in the filter according to the provided BitVec.
// Bits appearing in mask will not appear in the filter afterwards.
func (s *StatementFilter) MaskValid(mask BitVec) {
	s.ValidatedInGroup.Mask(mask)
}

// Clone returns a deep copy of the statement filter.
func (s *StatementFilter) Clone() StatementFilter {
	return StatementFilter{
		SecondedInGroup:  s.SecondedInGroup.Clone(),
		ValidatedInGroup: s.ValidatedInGroup.Clone(),
	}
}

func (s *StatementFilter) Contains(index uint, statementKind StatementKind) bool {
	switch statementKind {
	case SecondedKind:
		b, err := s.SecondedInGroup.Get(index)
		if err != nil {
			return false
		}
		return b
	case ValidKind:
		b, err := s.ValidatedInGroup.Get(index)
		if err != nil {
			return false
		}
		return b
	default:
		panic("unreachable")
	}
}

func (s *StatementFilter) Set(index uint, statementKind StatementKind) error {
	switch statementKind {
	case SecondedKind:
		return s.SecondedInGroup.Set(index, true)

	case ValidKind:
		return s.ValidatedInGroup.Set(index, true)
	default:
		panic("unreachable")
	}
}
