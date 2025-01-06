// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
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
type Seconded CommittedCandidateReceipt

// Valid represents a statement that a validator has deemed a candidate valid.
type Valid CandidateHash

// encodeSignData encodes the statement and signing context into a byte slice.
func encodeSignData(statement StatementVDT, signingContext SigningContext) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)

	compact, err := statement.CompactStatement()
	if err != nil {
		return nil, fmt.Errorf("getting compact statement: %w", err)
	}

	err = encoder.Encode(compact)
	if err != nil {
		return nil, fmt.Errorf("encoding compact statement: %w", err)
	}

	err = encoder.Encode(signingContext)
	if err != nil {
		return nil, fmt.Errorf("encoding signing context: %w", err)
	}

	return buffer.Bytes(), nil
}

// CompactStatement returns a compact representation of the statement.
func (s StatementVDT) CompactStatement() (any, error) {
	switch s := s.inner.(type) {
	case Valid:
		return CompactStatement[Valid]{Value: s}, nil
	case Seconded:
		hash, err := GetCandidateHash(CommittedCandidateReceipt(s))
		if err != nil {
			return nil, fmt.Errorf("getting candidate hash: %w", err)
		}
		return CompactStatement[SecondedCandidateHash]{Value: SecondedCandidateHash(hash)}, nil
	}
	return nil, fmt.Errorf("unsupported type")
}

func (s *StatementVDT) Sign(
	keystore keystore.Keystore,
	signingContext SigningContext,
	key ValidatorID,
) (*ValidatorSignature, error) {
	data, err := encodeSignData(*s, signingContext)
	if err != nil {
		return nil, fmt.Errorf("encoding data to sign: %w", err)
	}

	validatorPublicKey, err := sr25519.NewPublicKey(key[:])
	if err != nil {
		return nil, fmt.Errorf("getting public key: %w", err)
	}

	signatureBytes, err := keystore.GetKeypair(validatorPublicKey).Sign(data)
	if err != nil {
		return nil, fmt.Errorf("signing data: %w", err)
	}

	var signature Signature
	copy(signature[:], signatureBytes)
	valSign := ValidatorSignature(signature)
	return &valSign, nil
}

// VerifySignature verifies the validator signature for the statement.
func (s *StatementVDT) VerifySignature(
	validator ValidatorID,
	signingContext SigningContext,
	validatorSignature ValidatorSignature,
) (bool, error) {
	data, err := encodeSignData(*s, signingContext)
	if err != nil {
		return false, fmt.Errorf("encoding signed data: %w", err)
	}

	publicKey, err := sr25519.NewPublicKey(validator[:])
	if err != nil {
		return false, fmt.Errorf("getting public key: %w", err)
	}

	return publicKey.Verify(data, validatorSignature[:])
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

type SecondedCandidateHash CandidateHash

type CompactStatementValues interface {
	Valid | SecondedCandidateHash
}

// compactStatementInner is a helper struct that is used to encode/decode CompactStatement.
type compactStatementInner struct {
	inner any
}

func setCompactStatement[Value CompactStatementValues](mvdt *compactStatementInner, value Value) {
	mvdt.inner = value
}

func (mvdt *compactStatementInner) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Valid:
		setCompactStatement(mvdt, value)
		return
	case SecondedCandidateHash:
		setCompactStatement(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt compactStatementInner) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Valid:
		return 2, mvdt.inner, nil
	case SecondedCandidateHash:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt compactStatementInner) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt compactStatementInner) ValueAt(index uint) (value any, err error) {
	switch index {
	case 2:
		return Valid{}, nil
	case 1:
		return SecondedCandidateHash{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// CompactStatement is a compact representation of a statement that can be made about parachain candidates.
// this is the actual value that is signed.
type CompactStatement[T CompactStatementValues] struct {
	Value T
}

func (c CompactStatement[CompactStatementValues]) MarshalSCALE() ([]byte, error) {
	inner := compactStatementInner{}
	err := inner.SetValue(c.Value)
	if err != nil {
		return nil, fmt.Errorf("setting value: %w", err)
	}

	buffer := bytes.NewBuffer(backingStatementMagic[:])
	encoder := scale.NewEncoder(buffer)

	err = encoder.Encode(inner)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *CompactStatement[CompactStatementValues]) UnmarshalSCALE(reader io.Reader) error {
	decoder := scale.NewDecoder(reader)

	var magicBytes [4]byte
	err := decoder.Decode(&magicBytes)
	if err != nil {
		return err
	}

	if !bytes.Equal(magicBytes[:], backingStatementMagic[:]) {
		return fmt.Errorf("invalid magic bytes")
	}

	var inner compactStatementInner
	err = decoder.Decode(&inner)
	if err != nil {
		return fmt.Errorf("decoding compactStatementInner: %w", err)
	}

	value, err := inner.Value()
	if err != nil {
		return fmt.Errorf("getting value: %w", err)
	}

	c.Value = value.(CompactStatementValues)
	return nil
}
