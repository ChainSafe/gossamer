// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// WarpProofRequest is a struct for p2p warp proof request
type WarpProofRequest struct {
	Begin common.Hash
}

// Decode decodes the message into a WarpProofRequest
func (wsr *WarpProofRequest) Decode(in []byte) error {
	reader := bytes.NewReader(in)
	sd := scale.NewDecoder(reader)
	err := sd.Decode(&wsr)
	if err != nil {
		return err
	}

	return nil
}

// Encode encodes the warp sync request
func (wsr *WarpProofRequest) Encode() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(wsr)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// String returns the string representation of a WarpProofRequest
func (wsr *WarpProofRequest) String() string {
	if wsr == nil {
		return "WarpProofRequest=nil"
	}

	return fmt.Sprintf("WarpProofRequest begin=%v", wsr.Begin)
}

var _ P2PMessage = (*WarpProofRequest)(nil)
