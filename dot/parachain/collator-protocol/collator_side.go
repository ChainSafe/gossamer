package collatorprotocol

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var ErrNotExpectedOnCollatorSide = errors.New("message is not expected on the collator side of the protocol")

type CollatorProtocolCollatorSide struct {
	net         Network
	collatingOn parachaintypes.ParaID
}

func (cpcs CollatorProtocolCollatorSide) processMessage(msg any) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case collatorprotocolmessages.CollateOn:
		cpcs.collatingOn = parachaintypes.ParaID(msg)
	case collatorprotocolmessages.DistributeCollation:
		// TODO: handle distribute collation message
	case collatorprotocolmessages.ReportCollator:
		return fmt.Errorf("ReportCollator %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.NetworkBridgeUpdate:
		// TODO: handle network message https://github.com/ChainSafe/gossamer/issues/3515
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1457 //nolint
	case collatorprotocolmessages.Seconded:
		return fmt.Errorf("Seconded %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.Backed:
		return fmt.Errorf("Backed %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.Invalid:
		return fmt.Errorf("Invalid %w", ErrNotExpectedOnCollatorSide)
	case parachaintypes.ActiveLeavesUpdateSignal:
		cpcs.ProcessActiveLeavesUpdateSignal()
	case parachaintypes.BlockFinalizedSignal:
		cpcs.ProcessBlockFinalizedSignal()

	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil
}

func (cpcs CollatorProtocolCollatorSide) ProcessActiveLeavesUpdateSignal() {
	// TODO: handle active leaves update signal
}

func (cpcs CollatorProtocolCollatorSide) ProcessBlockFinalizedSignal() {
	// NOTE: nothing to do here
}

func (cpcs CollatorProtocolCollatorSide) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {

	// we don't propagate collation messages, so it will always be false
	propagate := false

	if msg.Type() != network.CollationMsgType {
		return propagate, fmt.Errorf("%w, expected: %d, found:%d", ErrUnexpectedMessageOnCollationProtocol,
			network.CollationMsgType, msg.Type())
	}

	collatorProtocol, ok := msg.(*CollationProtocol)
	if !ok {
		return propagate, fmt.Errorf(
			"failed to cast into collator protocol message, expected: *CollationProtocol, got: %T",
			msg)
	}

	collatorProtocolV, err := collatorProtocol.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolV.(CollatorProtocolMessage)
	if !ok {
		return propagate, errors.New("expected value to be collator protocol message")
	}

	collatorProtocolMessageV, err := collatorProtocolMessage.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol message value: %w", err)
	}

	switch collatorProtocolMessageV.Index() {
	// TODO: Create an issue to cover v2 types. #3534
	case 0: // Declare
		logger.Errorf("unexpected collation declare message from peer %s, decreasing its reputation", sender)
		cpcs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	case 1: // AdvertiseCollation
		logger.Errorf("unexpected collation advertise collation message from peer %s, decreasing its reputation", sender)
		cpcs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	case 2: // CollationSeconded
		// TODO: handle collation seconded message
	}

	return propagate, nil
}

func RegisterCollatorSide(net Network, protocolID protocol.ID, o overseer.OverseerI) (*CollatorProtocolCollatorSide, error) {
	// collationFetchingReqResProtocol := net.GetRequestResponseProtocol(
	// 	string(protocolID), collationFetchingRequestTimeout, collationFetchingMaxResponseSize)

	cpcs := CollatorProtocolCollatorSide{
		net: net,
	}

	// register collation protocol
	err := net.RegisterNotificationsProtocol(
		protocolID,
		network.CollationMsgType,
		getCollatorHandshake,
		decodeCollatorHandshake,
		validateCollatorHandshake,
		decodeCollationMessage,
		cpcs.handleCollationMessage,
		nil,
		MaxCollationMessageSize,
	)
	if err != nil {
		// try with legacy protocol id
		err1 := net.RegisterNotificationsProtocol(
			protocol.ID(legacyCollationProtocolV1),
			network.CollationMsgType,
			getCollatorHandshake,
			decodeCollatorHandshake,
			validateCollatorHandshake,
			decodeCollationMessage,
			cpcs.handleCollationMessage,
			nil,
			MaxCollationMessageSize,
		)

		if err1 != nil {
			return nil, fmt.Errorf("registering collation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	return &cpcs, nil
}
