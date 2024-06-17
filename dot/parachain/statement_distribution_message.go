// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type StatementDistributionMessageValues interface {
	Statement | LargePayload
}

// StatementDistributionMessage represents network messages used by the statement distribution subsystem
type StatementDistributionMessage struct {
	inner any
}

func setStatementDistributionMessage[Value StatementDistributionMessageValues](
	mvdt *StatementDistributionMessage, value Value,
) {
	mvdt.inner = value
}

func (mvdt *StatementDistributionMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Statement:
		setStatementDistributionMessage(mvdt, value)
		return

	case LargePayload:
		setStatementDistributionMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt StatementDistributionMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Statement:
		return 0, mvdt.inner, nil

	case LargePayload:
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
		return *new(Statement), nil

	case 1:
		return *new(LargePayload), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewStatementDistributionMessage returns a new statement distribution message varying data type
func NewStatementDistributionMessage() StatementDistributionMessage {
	return StatementDistributionMessage{}
}

// Statement represents a signed full statement under a given relay-parent.
type Statement struct {
	Hash                         common.Hash                                 `scale:"1"`
	UncheckedSignedFullStatement parachaintypes.UncheckedSignedFullStatement `scale:"2"`
}

// LargePayload represents Seconded statement with large payload
// (e.g. containing a runtime upgrade).
//
// We only gossip the hash in that case, actual payloads can be fetched from sending node
// via request/response.
type LargePayload StatementMetadata

// StatementMetadata represents the data that makes a statement unique.
type StatementMetadata struct {
	// Relay parent this statement is relevant under.
	RelayParent common.Hash `scale:"1"`

	// Hash of the candidate that got validated.
	CandidateHash parachaintypes.CandidateHash `scale:"2"`

	// Validator that attested the validity.
	SignedBy parachaintypes.ValidatorIndex `scale:"3"`

	// Signature of seconding validator.
	Signature parachaintypes.ValidatorSignature `scale:"4"`
}
