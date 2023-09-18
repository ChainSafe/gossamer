// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	"time"

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
	Network  Network
	Overseer *overseer.Overseer
}

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain"))

func NewService(net Network, forkID string, genesisHash common.Hash) (*Service, error) {
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

	overseer := overseer.NewOverseer()
	parachainService := &Service{
		Network:  net,
		Overseer: overseer,
	}

	go parachainService.run()

	return parachainService, nil
}

// Start starts the Handler
func (s Service) Start() error {
	logger.Infof("starting overseer")
	errChan, err := s.Overseer.Start()
	if err != nil {
		return err
	}
	go func() {
		for errC := range errChan {
			// TODO: handle error
			fmt.Printf("overseer start error: %v\n", errC)
		}
	}()

	return nil
}

// Stop stops the Handler
func (s Service) Stop() error {
	logger.Infof("stopping overseer")
	return s.Overseer.Stop()
}

// main loop of parachain service
func (s Service) run() {

	// NOTE: this is a temporary test, just to show that we can send messages to peers
	//
	time.Sleep(time.Second * 15)
	// let's try sending a collation message  and validation message to a peer and see what happens
	collationMessage := CollationProtocolV1{}
	s.Network.GossipMessage(&collationMessage)

	statementDistributionLargeStatement := StatementDistribution{NewStatementDistributionMessage()}
	err := statementDistributionLargeStatement.Set(LargePayload{
		RelayParent:   common.Hash{},
		CandidateHash: CandidateHash{Value: common.Hash{}},
		SignedBy:      5,
		Signature:     ValidatorSignature{},
	})
	if err != nil {
		logger.Errorf("creating test statement message: %w\n", err)
	}

	validationMessage := NewValidationProtocolVDT()
	err = validationMessage.Set(statementDistributionLargeStatement)
	if err != nil {
		logger.Errorf("creating test validation message: %w\n", err)
	}
	s.Network.GossipMessage(&validationMessage)

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
