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
	Signature parachaintypes.ValidatorSignature `scale:"3"`
}

// Bitfield avalibility bitfield for given relay-parent hash
type Bitfield struct {
	Hash                                common.Hash                         `scale:"1"`
	UncheckedSignedAvailabilityBitfield UncheckedSignedAvailabilityBitfield `scale:"2"`
}

// Index returns the VaryingDataType Index
func (Bitfield) Index() uint {
	return 0
}

// BitfieldDistributionMessage Network messages used by bitfield distribution subsystem
type BitfieldDistributionMessage scale.VaryingDataType

// NewBitfieldDistributionMessageVDT returns a new BitfieldDistributionMessage VaryingDataType
func NewBitfieldDistributionMessageVDT() BitfieldDistributionMessage {
	vdt := scale.MustNewVaryingDataType(Bitfield{})
	return BitfieldDistributionMessage(vdt)
}

// New creates new BitfieldDistributionMessage
func (BitfieldDistributionMessage) New() BitfieldDistributionMessage {
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

// BitfieldDistribution struct holding BitfieldDistributionMessage
type BitfieldDistribution struct {
	BitfieldDistributionMessage
}

// Index VaryingDataType index of Bitfield Distribution
func (BitfieldDistribution) Index() uint {
	return 1
}

// ApprovalDistribution struct holding ApprovalDistributionMessage
type ApprovalDistribution struct {
	ApprovalDistributionMessage
}

// Index VaryingDataType index of ApprovalDistribution
func (ApprovalDistribution) Index() uint {
	return 4
}

// StatementDistribution struct holding StatementDistributionMessage
type StatementDistribution struct {
	StatementDistributionMessage
}

// Index VaryingDataType index for StatementDistribution
func (StatementDistribution) Index() uint {
	return 3
}

// ValidationProtocol VaryingDataType for ValidationProtocol
type ValidationProtocol scale.VaryingDataType

// NewValidationProtocolVDT constructor or ValidationProtocol VaryingDataType
func NewValidationProtocolVDT() ValidationProtocol {
	vdt, err := scale.NewVaryingDataType(BitfieldDistribution{}, StatementDistribution{}, ApprovalDistribution{})
	if err != nil {
		panic(err)
	}
	return ValidationProtocol(vdt)
}

// New returns new ValidationProtocol VDT
func (ValidationProtocol) New() ValidationProtocol {
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
