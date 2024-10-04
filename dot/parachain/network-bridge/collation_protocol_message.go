package networkbridge

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	wireMessage := WireMessage{}
	err := wireMessage.SetValue(collatorprotocolmessages.CollationProtocol{})
	if err != nil {
		return nil, fmt.Errorf("setting collation protocol message: %w", err)
	}
	err = scale.Unmarshal(in, &wireMessage)
	if err != nil {
		return nil, fmt.Errorf("decoding message: %w", err)
	}

	collationMessageV, err := wireMessage.Value()
	if err != nil {
		return nil, fmt.Errorf("getting collation protocol message value: %w", err)
	}
	collationMessage, ok := collationMessageV.(collatorprotocolmessages.CollationProtocol)
	if !ok {
		return nil, fmt.Errorf("casting to collation protocol message")
	}
	return &collationMessage, nil
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
