// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type AvailabilityBitfield scale.BitVec

type Bitfield struct {
	Hash                                common.Hash
	UncheckedSignedAvailabilityBitfield AvailabilityBitfield
}

// Index returns the VaryingDataType Index
func (b Bitfield) Index() uint {
	return 0
}

type BitfieldDistributionMessage scale.VaryingDataType

func (bdm BitfieldDistributionMessage) Index() uint {
	return 0
}
func NewBitfieldDistributionMessageVDT() BitfieldDistributionMessage {
	vdt := scale.MustNewVaryingDataType(Bitfield{})
	return BitfieldDistributionMessage(vdt)
}

func (bdm BitfieldDistributionMessage) New() BitfieldDistributionMessage {
	return NewBitfieldDistributionMessageVDT()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (bdm *BitfieldDistributionMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*bdm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*bdm = BitfieldDistributionMessage(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (bdm *BitfieldDistributionMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*bdm)
	return vdt.Value()
}

type BitfieldDistribution BitfieldDistributionMessage

func NewBitfieldDistributionVDT() BitfieldDistribution {
	vdt := scale.MustNewVaryingDataType(BitfieldDistributionMessage{})
	return BitfieldDistribution(vdt)
}

func (bd BitfieldDistribution) Index() uint {
	return 1
}
func (bd BitfieldDistribution) New() BitfieldDistribution {
	return NewBitfieldDistributionVDT()
}

type ApprovalDistribution ApprovalDistributionMessage

// NewApprovalDistributionMessageVDT ruturns a new ApprovalDistributionMessage VaryingDataType
func NewApprovalDistributionVDT() ApprovalDistribution {
	vdt, err := scale.NewVaryingDataType(Assignments{}, Approvals{})
	if err != nil {
		panic(err)
	}
	return ApprovalDistribution(vdt)
}

// New returns new ApprovalDistributionMessage VDT
func (ad ApprovalDistribution) New() ApprovalDistribution {
	return NewApprovalDistributionVDT()
}

func (ad ApprovalDistribution) Index() uint {
	return 4
}

type StatementDistribution StatementDistributionMessage

func NewStatementDistributionVDT() StatementDistribution {
	vdt, err := scale.NewVaryingDataType(SignedFullStatement{}, SecondedStatementWithLargePayload{})
	if err != nil {
		panic(err)
	}
	return StatementDistribution(vdt)
}

func (sd StatementDistribution) New() StatementDistribution {
	return NewStatementDistributionVDT()
}

func (sd StatementDistribution) Index() uint {
	return 3
}

type ValidationProtocol scale.VaryingDataType

func NewValidationProtocolVDT() ValidationProtocol {
	vdt, err := scale.NewVaryingDataType(BitfieldDistribution{}, StatementDistribution{}, ApprovalDistribution{})
	if err != nil {
		panic(err)
	}
	return ValidationProtocol(vdt)
}

// New returns new ApprovalDistributionMessage VDT
func (vp ValidationProtocol) New() ValidationProtocol {
	return NewValidationProtocolVDT()
}

// Value returns the value from the underlying VaryingDataType
func (vp *ValidationProtocol) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*vp)
	return vdt.Value()
}
