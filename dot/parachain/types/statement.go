// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var backingStatementMagic = [4]byte{'B', 'K', 'N', 'G'}

// Statement is a result of candidate validation. It could be either `Valid` or `Seconded`.
type StatementVDTValues interface {
	Valid | Seconded
}

type StatementVDT struct {
	inner any
}

func setStatement[Value StatementVDTValues](mvdt *StatementVDT, value Value) {
	mvdt.inner = value
}

func (mvdt *StatementVDT) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Valid:
		setStatement(mvdt, value)
		return

	case Seconded:
		setStatement(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt StatementVDT) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Valid:
		return 2, mvdt.inner, nil

	case Seconded:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt StatementVDT) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt StatementVDT) ValueAt(index uint) (value any, err error) {
	switch index {
	case 2:
		return *new(Valid), nil

	case 1:
		return *new(Seconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewStatement returns a new statement varying data type
func NewStatementVDT() StatementVDT {
	return StatementVDT{}
}

// Seconded represents a statement that a validator seconds a candidate.
type Seconded CommittedCandidateReceiptV2

// SecondedCandidateHash represents just the seconded candidate hash
type SecondedCandidateHash CandidateHash

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash

// CompactStatement returns a compact representation of the statement.
func (s StatementVDT) CompactStatement() (CompactStatement, error) {
	switch s := s.inner.(type) {
	case Valid:
		return NewCompactValid(CandidateHash(s)), nil
	case Seconded:
		hash, err := GetCandidateHash(CommittedCandidateReceiptV2(s))
		if err != nil {
			return nil, fmt.Errorf("getting candidate hash: %w", err)
		}
		return NewCompactSeconded(CandidateHash(hash)), nil
	}
	return nil, fmt.Errorf("unsupported type")
}

func (s *StatementVDT) Sign(validator Validator, keystore keystore.Keystore) (*SignedFullStatement, error) {
	compact, err := s.CompactStatement()
	if err != nil {
		return nil, fmt.Errorf("getting compact statement: %w", err)
	}

	bytes, err := scale.Marshal(compact)
	if err != nil {
		return nil, fmt.Errorf("marshalling statementVDT: %w", err)
	}

	valSign, err := validator.Sign(keystore, bytes)
	if err != nil {
		return nil, fmt.Errorf("signing encoded statementVDT: %w", err)
	}

	return &SignedFullStatement{
		Payload:        *s,
		ValidatorIndex: validator.Index,
		Signature:      *valSign,
	}, nil
}

// VerifySignature verifies the validator signature for the statement.
func (s *StatementVDT) VerifySignature(validator Validator, validatorSignature ValidatorSignature) (bool, error) {
	compact, err := s.CompactStatement()
	if err != nil {
		return false, fmt.Errorf("getting compact statement: %w", err)
	}

	bytes, err := scale.Marshal(compact)
	if err != nil {
		return false, fmt.Errorf("marshalling statementVDT: %w", err)
	}

	return validator.VerifySignature(bytes, validatorSignature)
}

// UncheckedSignedFullStatement is a Variant of `SignedFullStatement` where the signature has not yet been verified.
type UncheckedSignedFullStatement struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload StatementVDT `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// SigningContext is a type returned by runtime with current session index and a parent hash.
type SigningContext struct {
	/// current session index.
	SessionIndex SessionIndex `scale:"1"`
	/// hash of the parent.
	ParentHash common.Hash `scale:"2"`
}

// SignedFullStatement represents a statement along with its corresponding signature
// and the index of the sender. The signing context and validator set should be
// apparent from context. This statement is "full" as the `Seconded` variant includes
// the candidate receipt. Only the compact `SignedStatement` is suitable for submission
// to the chain.
type SignedFullStatement UncheckedSignedFullStatement

// SignedFullStatementWithPVD represents a signed full statement along with associated Persisted Validation Data (PVD).
type SignedFullStatementWithPVD struct {
	SignedFullStatement SignedFullStatement

	// PersistedValidationData must be set only for `Seconded` statement.
	// otherwise, it should be nil.
	PersistedValidationData *PersistedValidationData
}

type UncheckedSignedCompactStatement struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload EncodableCompactStatement `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// SignedStatement represents a signed compact statement, suitable to be sent to the chain.
type SignedStatement UncheckedSignedCompactStatement
