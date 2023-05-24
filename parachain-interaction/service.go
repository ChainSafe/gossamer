package parachaininteraction

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const MaxValidationMessageSize uint64 = 100 * 1024

type Service struct {
	Network Network
}

func NewService(net Network, genesisHash common.Hash) *Service {

	// TODO: Change this and give different message type for each protocol
	msgType := byte(10)
	// TODO: Where do I get forkID and version from from?
	forkID := ""
	var version uint32 = 0

	validationProtocolID := GeneratePeersetProtocolName(ValidationProtocol, forkID, genesisHash, version)
	// register validation protocol
	// TODO: It seems like handshake is None, but be sure of it.
	net.RegisterNotificationsProtocol(
		protocol.ID(validationProtocolID),
		msgType,
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
	return &Service{
		Network: net,
	}
}

// Start starts the Handler
func (Service) Start() error {
	return nil
}

// Stop stops the Handler
func (Service) Stop() error {
	return nil
}

// Network is the interface required by GRANDPA for the network
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

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	// TODO: add things
	return nil, nil
}

func handleValidationMessage(peerID peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	return false, nil
}
