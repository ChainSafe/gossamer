// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	CollationProtocolVersion  = 1
	ValidationProtocolVersion = 1
)

type Service struct {
	Network           Network
	chNotificationMsg chan network.NotificationsMessage
}

type Config struct {
	LogLvl log.Level
}

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain"))

func NewService(config *Config, net Network, genesisHash common.Hash) (*Service, error) {
	logger.Patch(log.SetLevel(config.LogLvl))

	channel := make(chan network.NotificationsMessage)
	parachainService := &Service{
		Network:           net,
		chNotificationMsg: channel,
	}

	// TODO: Use actual fork id from chain spec #3373
	forkID := ""

	validationProtocolID := GeneratePeersetProtocolName(
		ValidationProtocolName, forkID, genesisHash, ValidationProtocolVersion)

	// register validation protocol
	err := net.RegisterNotificationsProtocol(
		protocol.ID(validationProtocolID),
		network.ValidationMsgType,
		getValidationHandshake,
		decodeValidationHandshake,
		validateValidationHandshake,
		decodeValidationMessage,
		parachainService.handleValidationMessage,
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
			parachainService.handleValidationMessage,
			nil,
			MaxValidationMessageSize,
		)

		if err1 != nil {
			return nil, fmt.Errorf("registering validation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	collationProtocolID := GeneratePeersetProtocolName(
		CollationProtocolName, forkID, genesisHash, CollationProtocolVersion)

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
	// TODO: add done channel to signal stopping this process
	for {
		select {
		case msg := <-s.chNotificationMsg:
			switch msg.Type() {
			case network.ValidationMsgType:
				validationMessage := msg.(*WireMessage)
				logger.Debugf("RMSG: %v", validationMessage)
				s.handleMessage(validationMessage)
			}
		}
	}
}

func (s Service) handleMessage(msg *WireMessage) {
	// todo: determine message type and do action on that message
	// for ViewUpdate message, relay message to all peers is peer set
	// TODO: how do we build the validators and collation peer sets?
	logger.Debugf("In handle message %v\n", msg.Type())
}

// Network is the interface required by parachain service for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg network.NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID network.MessageType,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
}
