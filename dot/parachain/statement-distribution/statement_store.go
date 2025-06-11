package statementdistribution

import (
	"errors"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var errValidatorUnknown = errors.New("validator unknown")

type statementOrigin byte

func (o statementOrigin) isLocal() bool {
	return o == statementOriginLocal
}

const (
	// The statement originated locally.
	statementOriginLocal statementOrigin = iota
	// The statement originated from a remote peer.
	statementOriginRemote
)

type storedStatement struct {
	stmt           *parachaintypes.SignedStatement
	knownByBacking bool
}

type fingerprintKind byte

const (
	// CompactSeconded is a fingerprint for a Seconded statement.
	fingerprintKindCompactSeconded fingerprintKind = iota
	// CompactValid is a fingerprint for a Valid statement.
	fingerprintKindCompactValid
)

type fingerprint struct {
	validator     parachaintypes.ValidatorIndex
	kind          fingerprintKind
	candidateHash parachaintypes.CandidateHash
}

type validatorMeta struct {
	groupIdx       parachaintypes.GroupIndex
	withinGroupIdx uint
	secondedCount  uint
}

type groupStatements struct {
	seconded parachaintypes.BitVec
	valid    parachaintypes.BitVec
}

func newGroupStatements(size int) (*groupStatements, error) {
	seconded, err := parachaintypes.NewBitVec(slices.Repeat([]bool{false}, size))
	if err != nil {
		return nil, err
	}

	valid, err := parachaintypes.NewBitVec(slices.Repeat([]bool{false}, size))
	if err != nil {
		return nil, err
	}

	return &groupStatements{
		seconded: seconded,
		valid:    valid,
	}, nil
}

func (g *groupStatements) noteSeconded(index uint) error {
	return g.seconded.Set(index, true)
}

func (g *groupStatements) noteValidated(index uint) error {
	return g.valid.Set(index, true)
}

type groupAndCandidateHash struct {
	groupIdx      parachaintypes.GroupIndex
	candidateHash parachaintypes.CandidateHash
}

// Storage for statements. Intended to be used for statements signed under
// the same relay-parent.
type statements struct {
	validatorMeta map[parachaintypes.ValidatorIndex]*validatorMeta

	// we keep statements per-group because even though only one group _should_ be
	// producing statements about a candidate, until we have the candidate receipt
	// itself, we can't tell which group that is.
	groupStmts map[groupAndCandidateHash]*groupStatements
	knownStmts map[fingerprint]*storedStatement
}

func newStatementStore(groups *groups) *statements {
	meta := map[parachaintypes.ValidatorIndex]*validatorMeta{}

	for gIdx, validators := range groups.all() {
		for vIdx, v := range validators {
			meta[v] = &validatorMeta{
				groupIdx:       parachaintypes.GroupIndex(gIdx),
				withinGroupIdx: uint(vIdx),
				secondedCount:  0,
			}
		}
	}

	return &statements{
		validatorMeta: meta,
		groupStmts:    make(map[groupAndCandidateHash]*groupStatements),
		knownStmts:    make(map[fingerprint]*storedStatement),
	}
}

// Insert adds a statement. Returns true if it was not known already, false if it was.
// Ignores statements by unknown validators and returns an error.
func (s *statements) insert(
	groups *groups,
	statement *parachaintypes.SignedStatement,
	origin statementOrigin,
) (bool, error) { //nolint:unparam
	validatorIndex := statement.ValidatorIndex
	validatorMeta, ok := s.validatorMeta[validatorIndex]
	if !ok {
		return false, errValidatorUnknown
	}

	compact, err := statement.Payload.ToCompact()
	if err != nil {
		return false, err
	}

	kind := fingerprintKindCompactSeconded // default to Seconded
	if _, ok := compact.(*parachaintypes.CompactValid); ok {
		kind = fingerprintKindCompactValid
	}

	fp := fingerprint{
		validator:     validatorIndex,
		kind:          kind,
		candidateHash: compact.CandidateHash(),
	}

	if stored, exists := s.knownStmts[fp]; exists {
		if origin.isLocal() {
			stored.knownByBacking = true
		}
		return false, nil
	} else {
		s.knownStmts[fp] = &storedStatement{
			stmt:           statement,
			knownByBacking: origin.isLocal(),
		}
	}

	candidateHash := compact.CandidateHash()

	_, seconded := compact.(*parachaintypes.CompactSeconded)

	// cross-reference updates
	groupIndex := validatorMeta.groupIdx
	group := groups.get(groupIndex)
	if len(group) == 0 {
		// log error: groups passed into insert differ from those used at store creation
		logger.Errorf("groups passed into `insert` differ "+
			"from those used at store creation, group index: %d", groupIndex)
		return false, errValidatorUnknown
	}

	key := groupAndCandidateHash{
		groupIdx:      groupIndex,
		candidateHash: candidateHash,
	}
	groupStmts, ok := s.groupStmts[key]
	if !ok {
		gs, err := newGroupStatements(len(group))
		if err != nil {
			return false, err
		}
		groupStmts = gs
		s.groupStmts[key] = groupStmts
	}

	if seconded {
		validatorMeta.secondedCount++
		err = groupStmts.noteSeconded(validatorMeta.withinGroupIdx)
		if err != nil {
			return false, err
		}
	} else {
		err = groupStmts.noteValidated(validatorMeta.withinGroupIdx)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// fillStatementFilter fills a StatementFilter with all statements already known for the given group and candidate hash.
func (s *statements) fillStatementFilter( //nolint:unused
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	statementFilter *parachaintypes.StatementFilter,
) {
	key := groupAndCandidateHash{
		groupIdx:      groupIndex,
		candidateHash: candidateHash,
	}
	if statements, ok := s.groupStmts[key]; ok {
		statementFilter.SecondedInGroup = statementFilter.SecondedInGroup.Or(statements.seconded)
		statementFilter.ValidatedInGroup = statementFilter.ValidatedInGroup.Or(statements.valid)
	}
}

// groupStatements returns all stored signed statements by the group conforming to the given filter.
// Seconded statements are provided first.
func (s *statements) groupStatements( //nolint:unused
	groups *groups,
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	filter *parachaintypes.StatementFilter,
) []*parachaintypes.SignedStatement {
	var result []*parachaintypes.SignedStatement

	groupValidators := groups.get(groupIndex)

	// Seconded statements first
	for _, i := range filter.SecondedInGroup.Ones() {
		if i < len(groupValidators) {
			v := groupValidators[i]
			fp := fingerprint{
				validator:     v,
				kind:          fingerprintKindCompactSeconded,
				candidateHash: candidateHash,
			}
			if sst, ok := s.knownStmts[fp]; ok {
				result = append(result, sst.stmt)
			}
		}
	}

	// Then validated statements
	for _, i := range filter.ValidatedInGroup.Ones() {
		if i < len(groupValidators) {
			v := groupValidators[i]
			fp := fingerprint{
				validator:     v,
				kind:          fingerprintKindCompactValid,
				candidateHash: candidateHash,
			}
			if sst, ok := s.knownStmts[fp]; ok {
				result = append(result, sst.stmt)
			}
		}
	}

	return result
}

// validatorStatement returns the full statement of this kind issued by this validator, if it is known.
func (s *statements) validatorStatement( //nolint:unused
	validatorIndex parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) (*parachaintypes.SignedStatement, bool) {
	kind := fingerprintKindCompactSeconded // default to Seconded
	if _, ok := statement.(*parachaintypes.CompactValid); ok {
		kind = fingerprintKindCompactValid
	}

	fp := fingerprint{
		validator:     validatorIndex,
		kind:          kind,
		candidateHash: statement.CandidateHash(),
	}
	sst, ok := s.knownStmts[fp]
	return sst.stmt, ok
}

// freshStatementsForBacking returns all statements for the given candidate hash
// and validators that are not yet known by backing.
// Seconded statements are provided before Valid statements.
func (s *statements) freshStatementsForBacking(
	validators []parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) []*parachaintypes.SignedStatement {
	var result []*parachaintypes.SignedStatement

	// First, fresh seconded
	for _, v := range validators {
		fp := fingerprint{validator: v, kind: fingerprintKindCompactSeconded, candidateHash: candidateHash}
		if stored, ok := s.knownStmts[fp]; ok && !stored.knownByBacking {
			result = append(result, stored.stmt)
		}
	}

	// Then, fresh valid
	for _, v := range validators {
		fp := fingerprint{validator: v, kind: fingerprintKindCompactValid, candidateHash: candidateHash}
		if stored, ok := s.knownStmts[fp]; ok && !stored.knownByBacking {
			result = append(result, stored.stmt)
		}
	}

	return result
}

// secondedCount returns the amount of known Seconded statements by the given validator index.
func (s *statements) secondedCount( //nolint:unused
	validatorIndex parachaintypes.ValidatorIndex,
) uint {
	if meta, ok := s.validatorMeta[validatorIndex]; ok {
		return meta.secondedCount
	}
	return 0
}

// noteKnownByBacking marks a statement as known by the backing subsystem.
func (s *statements) noteKnownByBacking( //nolint:unused
	validatorIndex parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) {
	kind := fingerprintKindCompactSeconded // default to Seconded
	if _, ok := statement.(*parachaintypes.CompactValid); ok {
		kind = fingerprintKindCompactValid
	}

	fp := fingerprint{
		validator:     validatorIndex,
		kind:          kind,
		candidateHash: statement.CandidateHash(),
	}

	if stored, ok := s.knownStmts[fp]; ok {
		stored.knownByBacking = true
	}
}
