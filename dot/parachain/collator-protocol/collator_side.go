// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// we don't propagate collation messages, so it will always be false
const propagate = false

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
		// TODO: handle distribute collation message #3824
	case collatorprotocolmessages.ReportCollator:
		return fmt.Errorf("report collator %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.NetworkBridgeUpdate:
		// TODO: handle network message #3824
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1457 //nolint
	case collatorprotocolmessages.Seconded:
		return fmt.Errorf("seconded %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.Backed:
		return fmt.Errorf("backed %w", ErrNotExpectedOnCollatorSide)
	case collatorprotocolmessages.Invalid:
		return fmt.Errorf("invalid %w", ErrNotExpectedOnCollatorSide)
	case parachaintypes.ActiveLeavesUpdateSignal:
		cpcs.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		cpcs.ProcessBlockFinalizedSignal(msg)

	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil
}

func (cpcs CollatorProtocolCollatorSide) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) {
	// TODO: handle active leaves update signal #3824
}

func (cpcs CollatorProtocolCollatorSide) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) {
	// NOTE: nothing to do here
}

func (cpcs CollatorProtocolCollatorSide) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {

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

	collatorProtocolVal, err := collatorProtocol.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolVal.(CollatorProtocolMessage)
	if !ok {
		return propagate, errors.New("expected value to be collator protocol message")
	}

	collatorProtocolMessageV, err := collatorProtocolMessage.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol message value: %w", err)
	}

	switch collatorProtocolMessageV.(type) {
	case Declare:
		logger.Errorf("unexpected collation declare message from peer %s, decreasing its reputation", sender)
		cpcs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	case AdvertiseCollation:
		logger.Errorf("unexpected collation advertise collation message from peer %s, decreasing its reputation", sender)
		cpcs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	case CollationSeconded:
		// TODO: handle collation seconded message #3824
	}

	return propagate, nil
}

func RegisterCollatorSide(net Network, protocolID protocol.ID) (*CollatorProtocolCollatorSide, error) {
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
		return nil, fmt.Errorf("registering collation protocol, new: %w", err)
	}

	// TODO: support legacy protocol as well, legacyCollationProtocolV1
	return &cpcs, nil
}
