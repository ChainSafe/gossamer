// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type UncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload scale.BitVec `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

type Bitfield struct {
	Hash                                common.Hash                         `scale:"1"`
	UncheckedSignedAvailabilityBitfield UncheckedSignedAvailabilityBitfield `scale:"2"`
}

// Index returns the VaryingDataType Index
func (b Bitfield) Index() uint {
	return 0
}

type BitfieldDistributionMessage scale.VaryingDataType

//	func (bdm BitfieldDistributionMessage) Index() uint {
//		return 0
//	}
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
	vdt := scale.MustNewVaryingDataType(Bitfield{})
	return BitfieldDistribution(vdt)
}

func (bd BitfieldDistribution) Index() uint {
	return 1
}
func (bd BitfieldDistribution) New() BitfieldDistribution {
	return NewBitfieldDistributionVDT()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (bd *BitfieldDistribution) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*bd)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*bd = BitfieldDistribution(vdt)
	return nil
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

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (ad *ApprovalDistribution) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*ad)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*ad = ApprovalDistribution(vdt)
	return nil
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

// Value returns the value from the underlying VaryingDataType
func (sd *StatementDistribution) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*sd)
	return vdt.Value()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (sd *StatementDistribution) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*sd)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*sd = StatementDistribution(vdt)
	return nil
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

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (vp *ValidationProtocol) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*vp)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*vp = ValidationProtocol(vdt)
	return nil
}
