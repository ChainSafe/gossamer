package networkbridge

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func RegisterCollationProtocol(net Network, nbr NetworkBridgeReceiver, protocolID protocol.ID, overseerChan chan<- any) error {
	// register collation protocol
	err := net.RegisterNotificationsProtocol(
		protocolID,
		network.CollationMsgType,
		getCollatorHandshake,
		decodeCollatorHandshake,
		validateCollatorHandshake,
		decodeCollationMessage,
		nbr.handleCollationMessage,
		nil,
		collatorprotocolmessages.MaxCollationMessageSize,
	)
	if err != nil {
		return fmt.Errorf("registering collation protocol, new: %w", err)
	}

	return nil
}

func RegisterValidationProtocol(net Network, nbr NetworkBridgeReceiver, protocolID protocol.ID, overseerChan chan<- any) error {
	// register validation protocol
	err := net.RegisterNotificationsProtocol(
		protocolID,
		network.ValidationMsgType,
		getValidationHandshake,
		decodeValidationHandshake,
		validateValidationHandshake,
		decodeValidationMessage,
		nbr.handleValidationMessage,
		nil,
		MaxValidationMessageSize,
	)
	if err != nil {
		return fmt.Errorf("registering validation protocol, new: %w", err)
	}

	return nil
}
