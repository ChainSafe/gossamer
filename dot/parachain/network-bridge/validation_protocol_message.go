package networkbridge

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxValidationMessageSize uint64 = 100 * 1024

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	wireMessage := WireMessage{}
	err := wireMessage.SetValue(validationprotocol.ValidationProtocol{})
	if err != nil {
		return nil, fmt.Errorf("setting validation protocol message: %w", err)
	}

	err = scale.Unmarshal(in, &wireMessage)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %w", err)
	}

	validationMessageV, err := wireMessage.Value()
	if err != nil {
		return nil, fmt.Errorf("getting validation protocol message value: %w", err)
	}
	validationMessage, ok := validationMessageV.(validationprotocol.ValidationProtocol)
	if !ok {
		return nil, fmt.Errorf("casting to validation protocol message")
	}

	return &validationMessage, nil
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
