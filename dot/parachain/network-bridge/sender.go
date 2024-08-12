// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"fmt"

	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type NetworkBridgeSender struct {
	net                  Network
	SubsystemsToOverseer chan<- any
}

func Register(overseerChan chan<- any, net Network) *NetworkBridgeSender {
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

		for _, to := range msg.To {
			err = nbs.net.SendMessage(to, wireMessage)
			if err != nil {
				return fmt.Errorf("sending message: %w", err)
			}
		}
		// TODO: add ConnectTOResolvedValidators, SendRequests
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
