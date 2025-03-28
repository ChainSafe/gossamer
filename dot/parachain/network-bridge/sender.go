// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type NetworkBridgeSender struct {
	net                  Network
	SubsystemsToOverseer chan<- any
}

func RegisterSender(overseerChan chan<- any, net Network) *NetworkBridgeSender {
	return &NetworkBridgeSender{
		net:                  net,
		SubsystemsToOverseer: overseerChan,
	}
}

func (nbs *NetworkBridgeSender) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := nbs.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (nbs *NetworkBridgeSender) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeSender
}

func (nbs *NetworkBridgeSender) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// nothing to do here
	return nil
}

func (nbs *NetworkBridgeSender) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (nbs *NetworkBridgeSender) Stop() {}

func (nbs *NetworkBridgeSender) processMessage(msg any) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case networkbridgemessages.SendCollationMessage:
		wireMessage := WireMessage{}
		err := wireMessage.SetValue(msg.CollationProtocolMessage)
		if err != nil {
			return fmt.Errorf("setting wire message: %w", err)
		}

		wireMessage.SetType(network.CollationMsgType)

		for _, to := range msg.To {
			err = nbs.net.SendMessage(to, wireMessage)
			if err != nil {
				return fmt.Errorf("sending message: %w", err)
			}
		}

	case networkbridgemessages.SendValidationMessage:
		wireMessage := WireMessage{}
		err := wireMessage.SetValue(msg.ValidationProtocolMessage)
		if err != nil {
			return fmt.Errorf("setting wire message: %w", err)
		}

		wireMessage.SetType(network.ValidationMsgType)

		for _, to := range msg.To {
			err = nbs.net.SendMessage(to, wireMessage)
			if err != nil {
				return fmt.Errorf("sending message: %w", err)
			}
		}
	case networkbridgemessages.SendRequests:
		nbs.sendRequests(msg.Requests, msg.IfDisconnected)
		// TODO: add ConnectTOResolvedValidators
	case networkbridgemessages.ConnectToValidators:
		// TODO
	case networkbridgemessages.ReportPeer:
		nbs.net.ReportPeer(msg.ReputationChange, msg.PeerID)
	case networkbridgemessages.DisconnectPeer:
		// We are using set ID 1 for validation protocol and 2 for collation protocol
		nbs.net.DisconnectPeer(int(msg.PeerSet)+1, msg.Peer)
	}

	return nil
}

const requestTimeout = 2 * time.Second

// PoV is probably the largest message and is currently set at 5MB, but will likely be increased to 10MB in the future.
// see: https://github.com/paritytech/polkadot-sdk/issues/5334
// Maybe message types should have a MaxSize() method instead of using the same value for all messages.
const maxResponseSize uint64 = 5 * 1024 * 1024

func (nbs *NetworkBridgeSender) sendRequests(
	requests []*networkbridgemessages.OutgoingRequest,
	ifDisconnected networkbridgemessages.IfDisconnectedBehavior, //nolint:unparam
) {
	for _, request := range requests {
		if request.IsCancelled() {
			close(request.Result)
			continue
		}

		result := nbs.sendRequest(request)

		if !request.IsCancelled() {
			request.Result <- result
			request.Cancel() // only called here to avoid resource leaks
		}
		close(request.Result)
	}
}

func (nbs *NetworkBridgeSender) sendRequest(
	request *networkbridgemessages.OutgoingRequest,
) networkbridgemessages.ReqRespResult {
	protoID := request.Payload.Protocol().String()
	protocol := nbs.net.GetRequestResponseProtocol(protoID, requestTimeout, maxResponseSize)
	response := request.Payload.Response()
	result := networkbridgemessages.ReqRespResult{}

	var peerID peer.ID
	switch request.Recipient.(type) {
	case parachaintypes.PeerID:
		peerID = peer.ID(request.Recipient.(parachaintypes.PeerID))
	case parachaintypes.AuthorityDiscoveryID:
		peerID = "unknown" // TODO: perform authority discovery (#4500)
	default:
		panic("unreachable")
	}

	err := protocol.Do(peerID, request.Payload, response)
	if err != nil {
		result.Error = err
	} else {
		result.Response = response
	}

	return result
}
