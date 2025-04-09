// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
)

// Any dispute that concluded more than a few minutes ago
// is not worth considering anymore.
const ActiveDurationSecs uint64 = 180

// DisputeIsInactive returns true if the dispute has concluded for longer than ActiveDurationSecs.
func DisputeIsInactive(status *DisputeStatus, now uint64) bool {
	at := status.ConcludedAt()
	return at != nil && *at+ActiveDurationSecs < now
}

// DisputeKey identifies a single dispute
// under polkadot-sdk the same representation is
// a tuple of (SessionIndex, CandidateHash)
type DisputeKey struct {
	SessionIndex  SessionIndex
	CandidateHash CandidateHash
}

// DisputeState is stored by the runtime
// and represents the entire dispute state
type DisputeState struct {
	// A bitfield indicating all validators for the candidate.
	ValidatorsFor BitVec

	// A bitfield indicating all validators against the candidate.
	ValidatorsAgainst BitVec

	// The block number at which the dispute started on-chain.
	Start BlockNumber

	// The block number at which the dispute concluded on-chain.
	ConcludedAt *BlockNumber
}

// ValidDisputeStatementKind Different kinds of statements of validity on a candidate.
type ValidDisputeStatementKind struct {
	inner any
}

type ValidDisputeStatementKindValues interface {
	ExplicitStatement | SecondedCandidateHash | Valid | ApprovalChecking | ApprovalCheckingMultipleCandidates
}

func setValidDisputeStatementKind[Value ValidDisputeStatementKindValues](
	mvdt *ValidDisputeStatementKind,
	value Value,
) {
	mvdt.inner = value
}

func (mvdt *ValidDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ExplicitStatement:
		setValidDisputeStatementKind(mvdt, value)
		return
	case SecondedCandidateHash:
		setValidDisputeStatementKind(mvdt, value)
		return
	case Valid:
		setValidDisputeStatementKind(mvdt, value)
		return
	case ApprovalChecking:
		setValidDisputeStatementKind(mvdt, value)
		return
	case ApprovalCheckingMultipleCandidates:
		setValidDisputeStatementKind(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ValidDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ExplicitStatement:
		return 0, mvdt.inner, nil
	case SecondedCandidateHash:
		return 1, mvdt.inner, nil
	case Valid:
		return 2, mvdt.inner, nil
	case ApprovalChecking:
		return 3, mvdt.inner, nil
	case ApprovalCheckingMultipleCandidates:
		return 4, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ValidDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ValidDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return ExplicitStatement{}, nil
	case 1:
		return SecondedCandidateHash{}, nil
	case 2:
		return Valid{}, nil
	case 3:
		return ApprovalChecking{}, nil
	case 4:
		return ApprovalCheckingMultipleCandidates{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// ExplicitStatement an explicit statement issued as part of a dispute.
type ExplicitStatement struct{}

// ApprovalChecking An approval vote from the approval checking phase.
type ApprovalChecking struct{}

// ApprovalCheckingMultipleCandidates An approval vote from the new version.
type ApprovalCheckingMultipleCandidates struct {
	CandidateHashes []CandidateHash
}

// InvalidDisputeStatementKind Different kinds of statements of invalidity on a candidate.
type InvalidDisputeStatementKind struct {
	inner any
}

type InvalidDisputeStatementKindValues interface {
	ExplicitStatement
}

func setInvalidDisputeStatementKind[Value InvalidDisputeStatementKindValues](
	mvdt *InvalidDisputeStatementKind, value Value,
) {
	mvdt.inner = value
}

func (mvdt *InvalidDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ExplicitStatement:
		setInvalidDisputeStatementKind(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt InvalidDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ExplicitStatement:
		return 0, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt InvalidDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt InvalidDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return ExplicitStatement{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// CandidateVotes Tracked votes on candidates, for the purposes of dispute resolution.
type CandidateVotes struct {
	// The receipt of the candidate itself.
	CandidateReceipt CandidateReceipt
	// Votes of validity, sorted by validator index.
	Valid *btree.Map[ValidatorIndex, Vote[ValidDisputeStatementKind]]
	// Votes of invalidity, sorted by validator index.
	Invalid *btree.Map[ValidatorIndex, Vote[InvalidDisputeStatementKind]]
}

type Vote[Kind any] struct {
	Kind      Kind
	Signature ValidatorSignature
}

func (cv *CandidateVotes) VotedIndices() (keys btree.Set[ValidatorIndex]) {
	cv.Valid.Ascend(
		ValidatorIndex(0),
		func(k ValidatorIndex, _ Vote[ValidDisputeStatementKind]) bool {
			keys.Insert(k)
			return true
		},
	)

	cv.Invalid.Ascend(
		ValidatorIndex(0),
		func(k ValidatorIndex, _ Vote[InvalidDisputeStatementKind]) bool {
			keys.Insert(k)
			return true
		},
	)

	return keys
}

// InsertValidVote inserts a valid vote, replacing any already existing vote.
// Returns true if the insert had any effect.
func (cv *CandidateVotes) InsertValidVote(
	validatorIndex ValidatorIndex,
	kind ValidDisputeStatementKind,
	sig ValidatorSignature,
) bool {
	entry, ok := cv.Valid.Get(validatorIndex)
	if !ok {
		cv.Valid.Set(validatorIndex, Vote[ValidDisputeStatementKind]{Kind: kind, Signature: sig})
		return true
	}

	entryKindVariant, err := entry.Kind.Value()
	if err != nil {
		panic("unexpected error: variant should not be missing")
	}

	switch entryKindVariant.(type) {
	case Valid, SecondedCandidateHash:
		return false
	default:
		cv.Valid.Set(validatorIndex, Vote[ValidDisputeStatementKind]{Kind: kind, Signature: sig})
		return true
	}
}

// DisputeStatus represents the status of a dispute.
type DisputeStatus struct {
	inner any
}

type DisputeStatusValues interface {
	Active | ConcludedFor | ConcludedAgainst | Confirmed
}

func setDisputeStatus[Value DisputeStatusValues](ds *DisputeStatus, value Value) {
	ds.inner = value
}

func (ds *DisputeStatus) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Active:
		setDisputeStatus(ds, value)
		return
	case ConcludedFor:
		setDisputeStatus(ds, value)
		return
	case ConcludedAgainst:
		setDisputeStatus(ds, value)
		return
	case Confirmed:
		setDisputeStatus(ds, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (ds DisputeStatus) IndexValue() (index uint, value any, err error) {
	switch ds.inner.(type) {
	case Active:
		return 0, ds.inner, nil
	case ConcludedFor:
		return 1, ds.inner, nil
	case ConcludedAgainst:
		return 2, ds.inner, nil
	case Confirmed:
		return 3, ds.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (ds DisputeStatus) Value() (value any, err error) {
	_, value, err = ds.IndexValue()
	return
}

func (ds DisputeStatus) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return Active{}, nil
	case 1:
		return ConcludedFor{}, nil
	case 2:
		return ConcludedAgainst{}, nil
	case 3:
		return Confirmed{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// Active represents an active dispute.
type Active struct{}

// ConcludedFor represents a dispute concluded in favour of the candidate.
type ConcludedFor struct {
	Timestamp uint64
}

// ConcludedAgainst represents a dispute concluded against the candidate.
type ConcludedAgainst struct {
	Timestamp uint64
}

// Confirmed represents a confirmed dispute.
type Confirmed struct{}

// NewDisputeStatusActive initialises the status to the active state.
func NewDisputeStatusActive() DisputeStatus {
	return DisputeStatus{inner: Active{}}
}

// Confirm moves the status to confirmed status, if not yet concluded/confirmed already.
func (ds DisputeStatus) Confirm() DisputeStatus {
	switch ds.inner.(type) {
	case Active, Confirmed:
		return DisputeStatus{inner: Confirmed{}}
	default:
		return ds
	}
}

// IsConfirmedConcluded checks whether the dispute is not a spam dispute.
func (ds DisputeStatus) IsConfirmedConcluded() bool {
	switch ds.inner.(type) {
	case Confirmed, ConcludedFor, ConcludedAgainst:
		return true
	default:
		return false
	}
}

// HasConcludedFor checks if the dispute has concluded in favour of the candidate.
func (ds DisputeStatus) HasConcludedFor() bool {
	_, ok := ds.inner.(ConcludedFor)
	return ok
}

// HasConcludedAgainst checks if the dispute has concluded against the candidate.
func (ds DisputeStatus) HasConcludedAgainst() bool {
	_, ok := ds.inner.(ConcludedAgainst)
	return ok
}

// ConcludeFor transitions the status to a new status after
// observing the dispute has concluded for the candidate.
func (ds DisputeStatus) ConcludeFor(now uint64) DisputeStatus {
	switch inner := ds.inner.(type) {
	case Active, Confirmed:
		return DisputeStatus{inner: ConcludedFor{Timestamp: now}}
	case ConcludedFor:
		return DisputeStatus{inner: ConcludedFor{Timestamp: min(inner.Timestamp, now)}}
	default:
		return ds
	}
}

// ConcludeAgainst transitions the status to a new status after
// observing the dispute has concluded against the candidate.
func (ds DisputeStatus) ConcludeAgainst(now uint64) DisputeStatus {
	switch inner := ds.inner.(type) {
	case Active, Confirmed:
		return DisputeStatus{inner: ConcludedAgainst{Timestamp: now}}
	case ConcludedFor:
		return DisputeStatus{inner: ConcludedAgainst{Timestamp: min(inner.Timestamp, now)}}
	case ConcludedAgainst:
		return DisputeStatus{inner: ConcludedAgainst{Timestamp: min(inner.Timestamp, now)}}
	default:
		return ds
	}
}

// IsPossiblyInvalid checks whether the disputed candidate is possibly invalid.
func (ds DisputeStatus) IsPossiblyInvalid() bool {
	switch ds.inner.(type) {
	case Active, Confirmed, ConcludedAgainst:
		return true
	case ConcludedFor:
		return false
	}
	return false
}

// ConcludedAt yields the timestamp this dispute concluded at, if any.
func (ds DisputeStatus) ConcludedAt() *uint64 {
	switch inner := ds.inner.(type) {
	case ConcludedFor:
		return &inner.Timestamp
	case ConcludedAgainst:
		return &inner.Timestamp
	default:
		return nil
	}
}

// DisputeStatement represents a statement about a candidate, to be used within the dispute resolution process.
type DisputeStatement struct {
	inner any
}

type DisputeStatementValues interface {
	ValidDisputeStatement | InvalidDisputeStatement
}

func setDisputeStatement[Value DisputeStatementValues](ds *DisputeStatement, value Value) {
	ds.inner = value
}

func (ds *DisputeStatement) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ValidDisputeStatement:
		setDisputeStatement(ds, value)
		return
	case InvalidDisputeStatement:
		setDisputeStatement(ds, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (ds DisputeStatement) IndexValue() (index uint, value any, err error) {
	switch ds.inner.(type) {
	case ValidDisputeStatement:
		return 0, ds.inner, nil
	case InvalidDisputeStatement:
		return 1, ds.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (ds DisputeStatement) Value() (value any, err error) {
	_, value, err = ds.IndexValue()
	return
}

func (ds DisputeStatement) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return ValidDisputeStatement{}, nil
	case 1:
		return InvalidDisputeStatement{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

type ValidDisputeStatement struct {
	Kind ValidDisputeStatementKind
}

type InvalidDisputeStatement struct {
	Kind InvalidDisputeStatementKind
}

type MultiDisputeStatementSet = []DisputeStatementSet

// DisputeStatementSet represents a set of statements about a specific candidate.
type DisputeStatementSet struct {
	// The candidate referenced by this set.
	CandidateHash CandidateHash
	// The session index of the candidate.
	Session SessionIndex
	// Statements about the candidate.
	Statements []DisputeStatementEntry
}

// DisputeStatementEntry represents a single statement about a candidate.
type DisputeStatementEntry struct {
	Statement DisputeStatement
	Index     ValidatorIndex
	Signature ValidatorSignature
}
