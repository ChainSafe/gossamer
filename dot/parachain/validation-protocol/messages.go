// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package validationprotocol

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const MaxValidationMessageSize uint64 = 100 * 1024

// Bitfield avalibility bitfield for given relay-parent hash
type Bitfield struct {
	Hash                                common.Hash                                        `scale:"1"`
	UncheckedSignedAvailabilityBitfield parachaintypes.UncheckedSignedAvailabilityBitfield `scale:"2"`
}

type BitfieldDistributionMessageValues interface {
	Bitfield
}

// BitfieldDistributionMessage Network messages used by bitfield distribution subsystem
type BitfieldDistributionMessage struct {
	inner any
}

func setBitfieldDistributionMessage[Value BitfieldDistributionMessageValues](
	mvdt *BitfieldDistributionMessage, value Value,
) {
	mvdt.inner = value
}

func (mvdt *BitfieldDistributionMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Bitfield:
		setBitfieldDistributionMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt BitfieldDistributionMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Bitfield:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt BitfieldDistributionMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt BitfieldDistributionMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Bitfield), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewBitfieldDistributionMessageVDT returns a new BitfieldDistributionMessage VaryingDataType
func NewBitfieldDistributionMessageVDT() BitfieldDistributionMessage {
	return BitfieldDistributionMessage{}
}

// BitfieldDistribution struct holding BitfieldDistributionMessage
type BitfieldDistribution struct {
	BitfieldDistributionMessage
}

// ApprovalDistribution struct holding ApprovalDistributionMessage
type ApprovalDistribution struct {
	ApprovalDistributionMessage
}

// StatementDistribution struct holding StatementDistributionMessage
type StatementDistribution struct {
	StatementDistributionMessage
}

type ValidationProtocolValues interface {
	BitfieldDistribution | StatementDistribution | ApprovalDistribution
}

// ValidationProtocol VaryingDataType for ValidationProtocol
type ValidationProtocol struct {
	inner any
}

func setValidationProtocol[Value ValidationProtocolValues](mvdt *ValidationProtocol, value Value) {
	mvdt.inner = value
}

func (mvdt *ValidationProtocol) SetValue(value any) (err error) {
	switch value := value.(type) {
	case BitfieldDistribution:
		setValidationProtocol(mvdt, value)
		return

	case StatementDistribution:
		setValidationProtocol(mvdt, value)
		return

	case ApprovalDistribution:
		setValidationProtocol(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ValidationProtocol) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case BitfieldDistribution:
		return 1, mvdt.inner, nil

	case StatementDistribution:
		return 3, mvdt.inner, nil

	case ApprovalDistribution:
		return 4, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ValidationProtocol) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ValidationProtocol) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return *new(BitfieldDistribution), nil

	case 3:
		return *new(StatementDistribution), nil

	case 4:
		return *new(ApprovalDistribution), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewValidationProtocolVDT constructor or ValidationProtocol VaryingDataType
func NewValidationProtocolVDT() ValidationProtocol {
	return ValidationProtocol{}
}

// Type returns ValidationMsgType
func (*ValidationProtocol) Type() network.MessageType {
	return network.ValidationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (vp *ValidationProtocol) Hash() (common.Hash, error) {
	encMsg, err := vp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (vp *ValidationProtocol) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*vp)
	if err != nil {
		return enc, err
	}
	return enc, nil
}
