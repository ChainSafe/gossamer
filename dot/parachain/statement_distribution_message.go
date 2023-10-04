// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// StatementDistributionMessage represents network messages used by the statement distribution subsystem
type StatementDistributionMessage scale.VaryingDataType

// NewStatementDistributionMessage returns a new statement distribution message varying data type
func NewStatementDistributionMessage() StatementDistributionMessage {
	vdt := scale.MustNewVaryingDataType(Statement{}, LargePayload{})
	return StatementDistributionMessage(vdt)
}

// New will enable scale to create new instance when needed
func (StatementDistributionMessage) New() StatementDistributionMessage {
	return NewStatementDistributionMessage()
}

// Set will set a value using the underlying  varying data type
func (sdm *StatementDistributionMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*sdm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*sdm = StatementDistributionMessage(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (sdm *StatementDistributionMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*sdm)
	return vdt.Value()
}

// Statement represents a signed full statement under a given relay-parent.
type Statement struct {
	Hash                         common.Hash                                 `scale:"1"`
	UncheckedSignedFullStatement parachaintypes.UncheckedSignedFullStatement `scale:"2"`
}

// Index returns the index of varying data type
func (Statement) Index() uint {
	return 0
}

// LargePayload represents Seconded statement with large payload
// (e.g. containing a runtime upgrade).
//
// We only gossip the hash in that case, actual payloads can be fetched from sending node
// via request/response.
type LargePayload StatementMetadata

// Index returns the index of varying data type
func (LargePayload) Index() uint {
	return 1
}

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
