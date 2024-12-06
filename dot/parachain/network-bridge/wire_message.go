// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
)

type WireMessage struct {
	inner       any
	messageType network.MessageType
}

type WireMessageValues interface {
	ViewUpdate | ProtocolMessage
}

func setMyVaryingDataType[Value WireMessageValues](mvdt *WireMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *WireMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ViewUpdate:
		setMyVaryingDataType(mvdt, value)
		return
	case ProtocolMessage:
		setMyVaryingDataType(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt WireMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ProtocolMessage:
		return 1, mvdt.inner, nil
	case ViewUpdate:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt WireMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt WireMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return ProtocolMessage{}, nil
	case 2:
		return ViewUpdate{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func (w *WireMessage) SetType(messageType network.MessageType) {
	// NOTE: We need a message type only to know where to send it
	w.messageType = messageType
}

func (w WireMessage) Type() network.MessageType {
	return w.messageType
}

func (w WireMessage) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := w.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (w WireMessage) Encode() ([]byte, error) {
	enc, err := scale.Marshal(w)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

type ViewUpdate parachaintypes.View

type ProtocolMessage struct {
	inner any
}

type ProtocolMessageValues interface {
	collatorprotocolmessages.CollationProtocol | validationprotocol.ValidationProtocol
}

func setProtocolMessageVaryingDataType[Value ProtocolMessageValues](pvdt *ProtocolMessage, value Value) {
	pvdt.inner = value
}

func (pvdt *ProtocolMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case collatorprotocolmessages.CollationProtocol:
		setProtocolMessageVaryingDataType(pvdt, value)
		return
	case validationprotocol.ValidationProtocol:
		setProtocolMessageVaryingDataType(pvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (pvdt ProtocolMessage) IndexValue() (index uint, value any, err error) {
	switch pvdt.inner.(type) {
	case collatorprotocolmessages.CollationProtocol:
		return 1, pvdt.inner, nil
	case validationprotocol.ValidationProtocol:
		return 2, pvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (pvdt ProtocolMessage) Value() (value any, err error) {
	_, value, err = pvdt.IndexValue()
	return
}

func (pvdt ProtocolMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return collatorprotocolmessages.CollationProtocol{}, nil
	case 2:
		return validationprotocol.ValidationProtocol{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}
