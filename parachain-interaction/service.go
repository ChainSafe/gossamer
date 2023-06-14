// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaininteraction

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Service struct {
	Network Network
}

func NewService(net Network, genesisHash common.Hash) (*Service, error) {
	// TODO: Where do I get forkID and version from from?
	forkID := ""
	var version uint32 = 1

	validationProtocolID := GeneratePeersetProtocolName(ValidationProtocol, forkID, genesisHash, version)

	// register validation protocol
	err := net.RegisterNotificationsProtocol(
		protocol.ID(validationProtocolID),
		network.ValidationMsgType,
		getValidationHandshake,
		decodeValidationHandshake,
		validateValidationHandshake,
		decodeValidationMessage,
		handleValidationMessage,
		nil,
		MaxValidationMessageSize,
	)
	if err != nil {
		// try with legacy protocol id
		err1 := net.RegisterNotificationsProtocol(
			protocol.ID(LEGACY_VALIDATION_PROTOCOL_V1),
			network.ValidationMsgType,
			getValidationHandshake,
			decodeValidationHandshake,
			validateValidationHandshake,
			decodeValidationMessage,
			handleValidationMessage,
			nil,
			MaxValidationMessageSize,
		)

		if err1 != nil {
			return nil, fmt.Errorf("registering validation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	collationProtocolID := GeneratePeersetProtocolName(CollationProtocol, forkID, genesisHash, version)

	// register collation protocol
	err = net.RegisterNotificationsProtocol(
		protocol.ID(collationProtocolID),
		network.CollationMsgType,
		getCollatorHandshake,
		decodeCollatorHandshake,
		validateCollatorHandshake,
		decodeCollationMessage,
		handleCollationMessage,
		nil,
		MaxCollationMessageSize,
	)
	if err != nil {
		// try with legacy protocol id
		err1 := net.RegisterNotificationsProtocol(
			protocol.ID(LEGACY_COLLATION_PROTOCOL_V1),
			network.CollationMsgType,
			getCollatorHandshake,
			decodeCollatorHandshake,
			validateCollatorHandshake,
			decodeCollationMessage,
			handleCollationMessage,
			nil,
			MaxCollationMessageSize,
		)

		if err1 != nil {
			return nil, fmt.Errorf("registering collation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	parachainService := &Service{
		Network: net,
	}

	go parachainService.run()

	return parachainService, nil
}

// Start starts the Handler
func (Service) Start() error {
	return nil
}

// Stop stops the Handler
func (Service) Stop() error {
	return nil
}

// main loop of parachain service
func (s Service) run() {

	// NOTE: this is a temporary test, just to show that we can send messages to peers
	//
	time.Sleep(time.Second * 15)
	// let's try sending a collation message  and validation message to a peer and see what happens
	collationMessage := CollationProtocolV1{}
	s.Network.GossipMessage(&collationMessage)

	validationMessage := ValidationProtocolV1{}
	s.Network.GossipMessage(&validationMessage)

}

// Network is the interface required by parachain service for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg network.NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID byte,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
}
