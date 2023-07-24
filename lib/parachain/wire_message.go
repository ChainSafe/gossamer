// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type WireMessage scale.VaryingDataType

type ProtocolMessage struct{}

// Index returns the VaryingDataType Index
func (ProtocolMessage) Index() uint {
	return 1
}

type ViewUpdate struct {
	View View
}

type View struct {
	// A bounded amount of chain heads.
	// Invariant: Sorted
	heads []common.Hash

	// The highest known finalized block number.
	FinalizedNumber parachaintypes.BlockNumber
}

// Index returns the VaryingDataType Index
func (ViewUpdate) Index() uint {
	return 2
}
func NewWireMessageVDT() WireMessage {
	vdt, err := scale.NewVaryingDataType(ProtocolMessage{}, ViewUpdate{})
	if err != nil {
		panic(err)
	}
	return WireMessage(vdt)
}

// New returns new WireMessage VDT
func (WireMessage) New() WireMessage {
	return NewWireMessageVDT()
}

// Value returns the value from the underlying VaryingDataType
func (wm *WireMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*wm)
	return vdt.Value()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (wm *WireMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*wm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*wm = WireMessage(vdt)
	return nil
}

// Type returns ValidationMsgType
func (WireMessage) Type() network.MessageType {
	return network.ValidationMsgType
}

// Hash returns the hash of the WireMessage
func (wm *WireMessage) Hash() (common.Hash, error) {
	encMsg, err := wm.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (wm *WireMessage) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*wm)
	if err != nil {
		return enc, err
	}
	return enc, nil
}
