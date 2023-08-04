// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type StatementDistributionMessageValues interface {
	SignedFullStatement | SecondedStatementWithLargePayload
}

// StatementDistributionMessage represents network messages used by the statement distribution subsystem
type StatementDistributionMessage struct {
	inner any
}

func setStatementDistributionMessage[Value StatementDistributionMessageValues](mvdt *StatementDistributionMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *StatementDistributionMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case SignedFullStatement:
		setStatementDistributionMessage(mvdt, value)
		return

	case SecondedStatementWithLargePayload:
		setStatementDistributionMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt StatementDistributionMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case SignedFullStatement:
		return 0, mvdt.inner, nil

	case SecondedStatementWithLargePayload:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt StatementDistributionMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt StatementDistributionMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(SignedFullStatement), nil

	case 1:
		return *new(SecondedStatementWithLargePayload), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewStatementDistributionMessage returns a new statement distribution message varying data type
func NewStatementDistributionMessage() StatementDistributionMessage {
	return StatementDistributionMessage{}
}

// SignedFullStatement represents a signed full statement under a given relay-parent.
type SignedFullStatement struct {
	Hash                         common.Hash                  `scale:"1"`
	UncheckedSignedFullStatement UncheckedSignedFullStatement `scale:"2"`
}

// SecondedStatementWithLargePayload represents Seconded statement with large payload
// (e.g. containing a runtime upgrade).
//
// We only gossip the hash in that case, actual payloads can be fetched from sending node
// via request/response.
type SecondedStatementWithLargePayload StatementMetadata

// UncheckedSignedFullStatement is a Variant of `SignedFullStatement` where the signature has not yet been verified.
type UncheckedSignedFullStatement struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload Statement `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex parachaintypes.ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// StatementMetadata represents the data that makes a statement unique.
type StatementMetadata struct {
	// Relay parent this statement is relevant under.
	RelayParent common.Hash `scale:"1"`

	// Hash of the candidate that got validated.
	CandidateHash CandidateHash `scale:"2"`

	// Validator that attested the validity.
	SignedBy parachaintypes.ValidatorIndex `scale:"3"`

	// Signature of seconding validator.
	Signature ValidatorSignature `scale:"4"`
}

// ValidatorSignature represents the signature with which parachain validators sign blocks.
type ValidatorSignature Signature

// Signature represents a cryptographic signature.
type Signature [64]byte
