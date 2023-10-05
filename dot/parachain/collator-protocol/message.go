// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

const legacyCollationProtocolV1 = "/polkadot/collation/1"

type CollationProtocolValues interface {
	CollatorProtocolMessage
}

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol struct {
	inner any
}

func setCollationProtocol[Value CollationProtocolValues](mvdt *CollationProtocol, value Value) {
	mvdt.inner = value
}

func (mvdt *CollationProtocol) SetValue(value any) (err error) {
	switch value := value.(type) {
	case CollatorProtocolMessage:
		setCollationProtocol(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationProtocol) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case CollatorProtocolMessage:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollationProtocol) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollationProtocol) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(CollatorProtocolMessage), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	return CollationProtocol{}
}

type CollatorProtocolMessageValues interface {
	Declare | AdvertiseCollation | CollationSeconded
}

// CollatorProtocolMessage represents Network messages used by the collator protocol subsystem
type CollatorProtocolMessage struct {
	inner any
}

func setCollatorProtocolMessage[Value CollatorProtocolMessageValues](mvdt *CollatorProtocolMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *CollatorProtocolMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Declare:
		setCollatorProtocolMessage(mvdt, value)
		return

	case AdvertiseCollation:
		setCollatorProtocolMessage(mvdt, value)
		return

	case CollationSeconded:
		setCollatorProtocolMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollatorProtocolMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Declare:
		return 0, mvdt.inner, nil

	case AdvertiseCollation:
		return 1, mvdt.inner, nil

	case CollationSeconded:
		return 4, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollatorProtocolMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollatorProtocolMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Declare), nil

	case 1:
		return *new(AdvertiseCollation), nil

	case 4:
		return *new(CollationSeconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewCollatorProtocolMessage() CollatorProtocolMessage {
	return CollatorProtocolMessage{}
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        parachaintypes.CollatorID        `scale:"1"`
	ParaID            uint32                           `scale:"2"`
	CollatorSignature parachaintypes.CollatorSignature `scale:"3"`
}

// AdvertiseCollation contains a relay parent hash and is used to advertise a collation to a validator.
// This will only advertise a collation if there exists one for the given relay parent and the given peer is
// set as validator for our para at the given relay parent.
// It can only be sent once the peer has declared that they are a collator with given ID
type AdvertiseCollation common.Hash

// CollationSeconded represents that a collation sent to a validator was seconded.
type CollationSeconded struct {
	RelayParent common.Hash                                 `scale:"1"`
	Statement   parachaintypes.UncheckedSignedFullStatement `scale:"2"`
}

// Index returns the index of varying data type
func (CollationSeconded) Index() uint {
	return 4
}

const MaxCollationMessageSize uint64 = 100 * 1024

// Type returns CollationMsgType
func (CollationProtocol) Type() network.MessageType {
	return network.CollationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (cp CollationProtocol) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (cp CollationProtocol) Encode() ([]byte, error) {
	enc, err := scale.Marshal(cp)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	collationMessage := CollationProtocol{}

	err := scale.Unmarshal(in, &collationMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &collationMessage, nil
}

type ProspectiveCandidate struct {
	CandidateHash      parachaintypes.CandidateHash
	ParentHeadDataHash common.Hash
}

type CollationStatus int

const (
	// We are waiting for a collation to be advertised to us.
	Waiting CollationStatus = iota
	// We are currently fetching a collation.
	Fetching
	// We are waiting that a collation is being validated.
	WaitingOnValidation
	// We have seconded a collation.
	Seconded
)

func (cpvs CollatorProtocolValidatorSide) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {
	if msg.Type() != network.CollationMsgType {
		return false, fmt.Errorf("unexpected message type, expected: %d, found:%d",
			network.CollationMsgType, msg.Type())
	}

	collatorProtocol, ok := msg.(*CollationProtocol)
	if !ok {
		return false, fmt.Errorf("failed to cast into collator protocol message, expected: *CollationProtocol, got: %T", msg)
	}

	collatorProtocolV, err := collatorProtocol.Value()
	if err != nil {
		return false, fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolV.(CollatorProtocolMessage)
	if !ok {
		return false, errors.New("expected value to be collator protocol message")
	}

	index, _, err := collatorProtocolMessage.IndexValue()
	if err != nil {
		return false, err
	}
	switch index {
	// TODO: Make sure that V1 and VStaging both types are covered
	// All the types covered currently are V1.
	case 0: // Declare
		// TODO: handle collator declaration https://github.com/ChainSafe/gossamer/issues/3513
	case 1: // AdvertiseCollation
		// TODO: handle collation advertisement https://github.com/ChainSafe/gossamer/issues/3514
	case 2: // CollationSeconded
		logger.Errorf("unexpected collation seconded message from peer %s, decreasing its reputation", sender)
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	}

	return false, nil
}

func getCollatorHandshake() (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func decodeCollatorHandshake(_ []byte) (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func validateCollatorHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

type collatorHandshake struct{}

// String formats a collatorHandshake as a string
func (*collatorHandshake) String() string {
	return "collatorHandshake"
}

// Encode encodes a collatorHandshake message using SCALE
func (*collatorHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a collatorHandshake
func (*collatorHandshake) Decode(_ []byte) error {
	return nil
}

// IsValid returns true
func (*collatorHandshake) IsValid() bool {
	return true
}
