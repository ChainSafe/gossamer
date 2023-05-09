// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// PersistedValidationData provides information about how to create the inputs for validation of a candidate.
// This information is derived from the chain state and will vary from para to para, although some
// fields may be the same for every para.
//
// Since this data is used to form inputs to the validation function, it needs to be persisted by the
// availability system to avoid dependence on availability of the relay-chain state.
//
// Furthermore, the validation data acts as a way to authorize the additional data the collator needs
// to pass to the validation function. For example, the validation function can check whether the incoming
// messages (e.g. downward messages) were actually sent by using the data provided in the validation data
// using so called MQC heads.
//
// Since the commitments of the validation function are checked by the relay-chain, secondary checkers
// can rely on the invariant that the relay-chain only includes para-blocks for which these checks have
// already been done. As such, there is no need for the validation data used to inform validators and
// collators about the checks the relay-chain will perform to be persisted by the availability system.
//
// The `PersistedValidationData` should be relatively lightweight primarily because it is constructed
// during inclusion for each candidate and therefore lies on the critical path of inclusion.
type PersistedValidationData struct {
	ParentHead             []byte
	RelayParentNumber      uint32
	RelayParentStorageRoot common.Hash
	MaxPovSize             uint32
}

// OccupiedCoreAssumption is an assumption being made about the state of an occupied core.
type OccupiedCoreAssumption scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (o *OccupiedCoreAssumption) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*o)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*o = OccupiedCoreAssumption(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (o *OccupiedCoreAssumption) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*o)
	return vdt.Value()
}

// ValidationCode is Parachain validation code.
type ValidationCode []byte

// CandidateCommitments are Commitments made in a `CandidateReceipt`. Many of these are outputs of validation.
type CandidateCommitments struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []UpwardMessage `scale:"1"`
	// Horizontal messages sent by the parachain.
	HorizontalMessages []OutboundHrmpMessage `scale:"2"`
	// New validation code.
	NewValidationCode *ValidationCode `scale:"3"`
	// The head-data produced as a result of execution.
	HeadData HeadData `scale:"4"`
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32 `scale:"5"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"6"`
}

// UpwardMessage is a message from a parachain to its Relay Chain.
type UpwardMessage []byte

// OutboundHrmpMessage is an HRMP message seen from the perspective of a sender.
type OutboundHrmpMessage struct {
	Recipient uint32 `scale:"1"`
	Data      []byte `scale:"2"`
}

// HeadData is Parachain head data included in the chain.
type HeadData []byte
