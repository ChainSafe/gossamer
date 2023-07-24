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
	Network Network
}

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain"))

func NewService(net Network, genesisHash common.Hash) (*Service, error) {
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
	//time.Sleep(time.Second * 15)
	//// let's try sending a collation message  and validation message to a peer and see what happens
	//collationMessage := CollationProtocolV1{}
	//s.Network.GossipMessage(&collationMessage)

	//time.Sleep(time.Second * 10)
	//logger.Infof("creating bitfield distribution message")
	//bitfieldDistribution := NewBitfieldDistributionVDT()
	//err := bitfieldDistribution.Set(Bitfield{
	//	Hash: common.Hash{},
	//	UncheckedSignedAvailabilityBitfield: UncheckedSignedAvailabilityBitfield{
	//		Payload: scale.NewBitVec([]bool{true, true, true, true, true, true, true, true, true, true, true,
	//			true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true,
	//			true, true, true, true}),
	//		ValidatorIndex: 0,
	//		Signature:      ValidatorSignature{},
	//	},
	//})
	//if err != nil {
	//	logger.Errorf("creating test bitfield distribution: %w\n", err)
	//}

	//vpBitfieldDistribution := NewValidationProtocolVDT()
	//vpBitfieldDistribution.Set(bitfieldDistribution)
	//vpBitfieldDistributionVal, err := vpBitfieldDistribution.Value()
	//require.NoError(t, err)
	//
	//statementDistributionLargeStatement := NewStatementDistributionVDT()
	//err := statementDistributionLargeStatement.Set(SecondedStatementWithLargePayload{
	//	RelayParent:   common.Hash{},
	//	CandidateHash: CandidateHash{Value: common.Hash{}},
	//	SignedBy:      5,
	//	Signature:     ValidatorSignature{},
	//})
	//if err != nil {
	//	logger.Errorf("creating test statement message: %w\n", err)
	//}

	//validationMessage := NewValidationProtocolVDT()
	//err = validationMessage.Set(bitfieldDistribution)
	//if err != nil {
	//	logger.Errorf("creating test validation message: %w\n", err)
	//}
	//
	//h, err := validationMessage.Hash()
	//fmt.Printf("h %v e %v\n", h, err)
	//s.Network.GossipMessage(&validationMessage)
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
