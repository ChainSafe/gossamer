// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxValidationMessageSize uint64 = 100 * 1024

// UncheckedSignedAvailabilityBitfield a signed bitfield with signature not yet checked
type UncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload scale.BitVec `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex parachaintypes.ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// Bitfield avalibility bitfield for given relay-parent hash
type Bitfield struct {
	Hash                                common.Hash                         `scale:"1"`
	UncheckedSignedAvailabilityBitfield UncheckedSignedAvailabilityBitfield `scale:"2"`
}

type BitfieldDistributionMessageValues interface {
	Bitfield
}

// BitfieldDistributionMessage Network messages used by bitfield distribution subsystem
type BitfieldDistributionMessage struct {
	inner any
}

func setBitfieldDistributionMessage[Value BitfieldDistributionMessageValues](mvdt *BitfieldDistributionMessage, value Value) {
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

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	validationMessage := ValidationProtocol{}

	err := scale.Unmarshal(in, &validationMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &validationMessage, nil
}

func handleValidationMessage(_ peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a validation message", msg)
	return false, nil
}

func getValidationHandshake() (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func decodeValidationHandshake(_ []byte) (network.Handshake, error) {
	return &validationHandshake{}, nil
}

func validateValidationHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

type validationHandshake struct{}

// String formats a validationHandshake as a string
func (*validationHandshake) String() string {
	return "validationHandshake"
}

// Encode encodes a validationHandshake message using SCALE
func (*validationHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a validationHandshake
func (*validationHandshake) Decode(_ []byte) error {
	return nil
}

// IsValid returns true
func (*validationHandshake) IsValid() bool {
	return true
}
