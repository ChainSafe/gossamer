// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	availability_store "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	collatorprotocol "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	CollationProtocolVersion  = 1
	ValidationProtocolVersion = 1
)

type Service struct {
	Network  Network
	overseer *overseer.Overseer
}

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain"))

func NewService(net Network, forkID string, st *state.Service, ks keystore.Keystore) (*Service, error) {
	overseer := overseer.NewOverseer(st.Block)
	err := overseer.Start()
	if err != nil {
		return nil, fmt.Errorf("starting overseer: %w", err)
	}
	genesisHash := st.Block.GenesisHash()

	networkBridge := networkbridge.Register(overseer.SubsystemsToOverseer, net)
	overseer.RegisterSubsystem(networkBridge)

	availabilityStore, err := availability_store.Register(overseer.SubsystemsToOverseer, st)
	if err != nil {
		return nil, fmt.Errorf("registering availability store: %w", err)
	}
	availabilityStore.OverseerToSubSystem = overseer.RegisterSubsystem(availabilityStore)

	validationProtocolID := GeneratePeersetProtocolName(
		ValidationProtocolName, forkID, genesisHash, ValidationProtocolVersion)

	// register validation protocol
	err = validationprotocol.Register(net, protocol.ID(validationProtocolID))
	if err != nil {
		return nil, fmt.Errorf("registering validation protocol: %w", err)
	}

	collationProtocolID := GeneratePeersetProtocolName(
		CollationProtocolName, forkID, genesisHash, CollationProtocolVersion)

	// register collation protocol
	cpvs, err := collatorprotocol.Register(net, protocol.ID(collationProtocolID), overseer.SubsystemsToOverseer)
	if err != nil {
		return nil, err
	}
	cpvs.BlockState = st.Block
	cpvs.Keystore = ks
	cpvs.OverseerToSubSystem = overseer.RegisterSubsystem(cpvs)

	parachainService := &Service{
		Network:  net,
		overseer: overseer,
	}

	go parachainService.run(st.Block)

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
func (s Service) run(blockState *state.BlockState) {
	overseer := s.overseer

	candidateBacking := backing.New(overseer.SubsystemsToOverseer)
	candidateBacking.BlockState = blockState
	candidateBacking.OverseerToSubSystem = overseer.RegisterSubsystem(candidateBacking)

	// TODO: Add `Prospective Parachains` Subsystem. create an issue.

	// NOTE: this is a temporary test, just to show that we can send messages to peers
	//
	time.Sleep(time.Second * 15)
	// let's try sending a collation message  and validation message to a peer and see what happens
	collatorProtocolMessage := collatorprotocolmessages.NewCollatorProtocolMessage()
	// NOTE: This is just to test. We should not be sending declare messages, since we are not a collator, just a validator
	_ = collatorProtocolMessage.Set(collatorprotocolmessages.Declare{})
	collationMessage := collatorprotocolmessages.NewCollationProtocol()

	_ = collationMessage.Set(collatorProtocolMessage)
	s.Network.GossipMessage(&collationMessage)

	statementDistributionLargeStatement := validationprotocol.StatementDistribution{
		StatementDistributionMessage: validationprotocol.NewStatementDistributionMessage()}
	err := statementDistributionLargeStatement.Set(validationprotocol.LargePayload{
		RelayParent:   common.Hash{},
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{}},
		SignedBy:      5,
		Signature:     parachaintypes.ValidatorSignature{},
	})
	if err != nil {
		logger.Errorf("creating test statement message: %w\n", err)
	}

	validationMessage := validationprotocol.NewValidationProtocolVDT()
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
	GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
		maxResponseSize uint64) *network.RequestResponseProtocol
	ReportPeer(change peerset.ReputationChange, p peer.ID)
	DisconnectPeer(setID int, p peer.ID)
	GetNetworkEventsChannel() chan *network.NetworkEventInfo
	FreeNetworkEventsChannel(ch chan *network.NetworkEventInfo)
}
