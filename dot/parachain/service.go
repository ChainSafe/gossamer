// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocol "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
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
	Network Network
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
	err = collatorprotocol.Register(net, protocol.ID(collationProtocolID))
	if err != nil {
		return nil, err
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
	collatorProtocolMessage := collatorprotocol.NewCollatorProtocolMessage()
	// NOTE: This is just to test. We should not be sending declare messages, since we are not a collator, just a validator
	collatorProtocolMessage.Set(collatorprotocol.Declare{})
	collationMessage := collatorprotocol.NewCollationProtocol()
	collationMessage.Set(collatorProtocolMessage)

	s.Network.GossipMessage(&collationMessage)

	statementDistributionLargeStatement := StatementDistribution{NewStatementDistributionMessage()}
	err := statementDistributionLargeStatement.Set(LargePayload{
		RelayParent:   common.Hash{},
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{}},
		SignedBy:      5,
		Signature:     parachaintypes.ValidatorSignature{},
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
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}
