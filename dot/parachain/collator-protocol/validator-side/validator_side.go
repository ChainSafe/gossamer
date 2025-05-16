package validatorside

import (
	"context"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	/// Maximum PoV size we support right now.
	maxPoVSize                       = 5 * 1024 * 1024
	collationFetchingRequestTimeout  = time.Millisecond * 1200
	collationFetchingMaxResponseSize = maxPoVSize + 10000 // 10MB

	activityPoll = 1 * time.Second
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-validator-side"))

// Network is the interface required by parachain service for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
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
		maxResponseSize uint64) network.RequestMaker
}

type validatorSideState struct {
	// TODO: implement collation request handler
	collationRequests chan any

	// when the request in the collationRequests is cancelled
	// we should dequeue the next request attempt in the corresponding
	// `collations_per_relay_parent`
	collationFetchTimeouts chan collationFetchTimeout
}

var _ overseer.Overseer

type CollatorProtocolValidatorSide struct {
	collationFetchingReqResProtocol network.RequestMaker
}

func New(net Network, protocolID protocol.ID, overseerChan chan<- any,
	blockState *state.BlockState, ks keystore.Keystore) *CollatorProtocolValidatorSide {
	collationFetchingReqResProtocol := net.GetRequestResponseProtocol(
		string(protocolID), collationFetchingRequestTimeout, collationFetchingMaxResponseSize)

	return &CollatorProtocolValidatorSide{
		collationFetchingReqResProtocol: collationFetchingReqResProtocol,
	}
}

func (*CollatorProtocolValidatorSide) Name() parachaintypes.SubSystemName {
	return parachaintypes.CollationProtocol
}

func (cpvs *CollatorProtocolValidatorSide) Run(
	ctx context.Context, overseerToSubSystem <-chan any) {
	inactivityTicker := time.NewTicker(activityPoll)

	subsystemState := &validatorSideState{}

	for {
		select {
		// TODO: implement reputation aggregator
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			if err := cpvs.processMessage(msg); err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}

		case <-inactivityTicker.C:
			// TODO: disconnect inactive peers, Issue #4256
		case _ = <-subsystemState.collationRequests:
			// TODO: implement collation request handler
			// handle_collation_fetch_respose
			// kick_off_seconding
			// dequeue_next_collation_and_fetch

		case _ = <-subsystemState.collationFetchTimeouts:
			// implement dequeue_next_collation_and_fetch

		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
		}
	}
}

func (cpvs *CollatorProtocolValidatorSide) processMessage(msg any) error {
	return nil
}

func (*CollatorProtocolValidatorSide) Stop() {
	// nothing to do
}

func (*CollatorProtocolValidatorSide) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// nothing to do
	return nil
}

func (*CollatorProtocolValidatorSide) ProcessBlockFinalizedSignal(signal parachaintypes.
	BlockFinalizedSignal) error {
	// nothing to do
	return nil
}
