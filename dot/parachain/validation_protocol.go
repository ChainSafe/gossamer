// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxValidationMessageSize uint64 = 100 * 1024

type ValidationProtocolV1 struct {
	//	TODO: Implement this struct https://github.com/ChainSafe/gossamer/issues/3318
}

// Type returns ValidationMsgType
func (*ValidationProtocolV1) Type() network.MessageType {
	return network.ValidationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (vp *ValidationProtocolV1) Hash() (common.Hash, error) {
	encMsg, err := vp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (vp *ValidationProtocolV1) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*vp)
	if err != nil {
		return enc, err
	}
	return enc, nil
}

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	validationMessage := ValidationProtocolV1{}

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
	return &validationHandshake{}, nil
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
