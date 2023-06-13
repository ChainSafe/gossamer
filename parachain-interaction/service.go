package parachaininteraction

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachains"))
var maxReads = 256
var maxResponseSize uint64 = 1024 * 1024 * 16 // 16mb

var legacyCollatorProtocolID = protocol.ID("/polkadot/collation/1")

// Notes:
/*
There are two types of peersets, validation and collation
*/

type Service struct {
	Network Network
}

func NewService(net Network, genesisHash common.Hash) (*Service, error) {
	// TODO: Where do I get forkID and version from from?
	forkID := ""
	var version uint32 = 1

	validationProtocolID := GeneratePeersetProtocolName(ValidationProtocol, forkID, genesisHash, version)
	// register validation protocol
	// TODO: It seems like handshake is None, but be sure of it.
	err := net.RegisterNotificationsProtocol(
		protocol.ID(validationProtocolID),
		network.ValidationMsgType,
		func() (network.Handshake, error) {
			return nil, nil
		},
		func(_ []byte) (network.Handshake, error) {
			return nil, nil
		},
		func(_ peer.ID, _ network.Handshake) error {
			return nil
		},
		decodeValidationMessage,
		handleValidationMessage,
		nil,
		MaxValidationMessageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("registering validation protocol: %w", err)
	}

	collationProtocolID := GeneratePeersetProtocolName(CollationProtocol, forkID, genesisHash, version)
	// register collation protocol
	// TODO: It seems like handshake is None, but be sure of it.
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
		return nil, fmt.Errorf("registering collation protocol: %w", err)
	}

	err = net.RegisterNotificationsProtocol(
		protocol.ID(legacyCollatorProtocolID),
		network.CollationMsgType1,
		getCollatorHandshake,
		decodeCollatorHandshake,
		validateCollatorHandshake,
		decodeCollationMessage,
		handleCollationMessage,
		nil,
		MaxCollationMessageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("registering collation protocol: %w", err)
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

	time.Sleep(time.Second * 15)
	// run collator protocol
	// let's try sending a collation message to a peer and see what happens
	collationMessage := CollationProtocolV1{}
	s.Network.GossipMessage(&collationMessage)

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
	GetRequestResponseProtocol(protocolID protocol.ID, requestTimeout time.Duration, maxResponseSize uint64) *network.RequestResponseProtocol
}

var (
	requestTimeout = time.Second * 20
)
