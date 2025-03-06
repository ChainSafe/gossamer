// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfielddistribution

import (
	"context"
	"fmt"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-distribution"))

type perRelayParentData struct {
	// Signing context for a particular relay parent.
	sessionIndex parachaintypes.SessionIndex // the required part of the signing context

	// Set of validators for a particular relay parent.
	validatorsSet []parachaintypes.ValidatorID

	// Set of validators for a particular relay parent for which we
	// received a valid `BitfieldGossipMessage`.
	// Also serves as the list of known messages for peers connecting
	// after bitfield gossips were already received.
	onePerValidator map[parachaintypes.ValidatorID]*validationprotocol.BitfieldDistributionMessage

	// Avoid duplicate message transmission to our peers.
	messageSentToPeer map[peer.ID]map[parachaintypes.ValidatorID]struct{}

	// Track messages that were already received by a peer to prevent flooding.
	messageReceivedFromPeer map[peer.ID]map[parachaintypes.ValidatorID]struct{}
}

type BitfieldDistribution struct {
	subSystemToOverseer chan<- any

	peerViews map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32 // ignore v1 peers
	}
	ourView        parachaintypes.View
	topologies     SessionBoundGridTopologyStorage // TODO: impl this part in the #4357
	perRelayParent map[common.Hash]*perRelayParentData

	mu sync.Mutex
}

func NewBitfieldDistribution(overseerChan chan<- any) *BitfieldDistribution {
	return &BitfieldDistribution{
		subSystemToOverseer: overseerChan,
		peerViews: make(map[peer.ID]struct {
			view            parachaintypes.View
			protocolVersion uint32
		}),
		ourView:        parachaintypes.View{},
		topologies:     nil, // TODO:
		perRelayParent: make(map[common.Hash]*perRelayParentData),
	}
}

func (b *BitfieldDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := b.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

// logic points:
// Before gossiping incoming bitfields, they must be checked to be signed by one of the validators of the validator set relevant to the current relay parent.
// Only accept bitfields relevant to our current view
// only distribute bitfields to other peers when relevant to their most recent view
// Accept and distribute only one bitfield per validator.

// processMessage processes messages sent to the BitfieldDistribution subsystem
func (b *BitfieldDistribution) processMessage(msg any) error {
	switch msg := msg.(type) {
	case validationprotocol.BitfieldDistributionMessage:
		err := b.ProcessBitfieldDistributionMessageSignal(msg)
		if err != nil {
			return fmt.Errorf("processing bitfield distribution message signal: %w", err)
		}
	case networkbridgeevents.PeerConnected:
		err := b.ProcessPeerConnectedSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer connected signal: %w", err)
		}
	case networkbridgeevents.PeerDisconnected:
		err := b.ProcessPeerDisconnectedSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer disconnected signal: %w", err)
		}
	case networkbridgeevents.NewGossipTopology:
		err := b.ProcessNewGossipTopologySignal(msg)
		if err != nil {
			return fmt.Errorf("processing new gossip topology signal: %w", err)
		}
	case networkbridgeevents.PeerViewChange:
		err := b.ProcessPeerViewChangeSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer view change signal: %w", err)
		}
	case networkbridgeevents.OurViewChange:
		err := b.ProcessOurViewChangeSignal(msg)
		if err != nil {
			return fmt.Errorf("processing our view change signal: %w", err)
		}
	case networkbridgeevents.UpdatedAuthorityIDs:
		err := b.ProcessUpdatedAuthorityIDsSignal(msg)
		if err != nil {
			return fmt.Errorf("processing updated authority IDs signal: %w", err)
		}
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := b.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		return b.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (b *BitfieldDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.BitfieldDistribution
}

func (b *BitfieldDistribution) ProcessPeerConnectedSignal(signal networkbridgeevents.PeerConnected) error {
	go func(pc networkbridgeevents.PeerConnected) {
		// only care about version 2 and 3
		if pc.ProtocolVersion == 2 || pc.ProtocolVersion == 3 {
			b.mu.Lock()
			b.peerViews[pc.PeerID] = struct {
				view            parachaintypes.View
				protocolVersion uint32 // ignore v1 peers
			}{
				view:            parachaintypes.View{}, // TODO: default view?
				protocolVersion: pc.ProtocolVersion,
			}
			b.mu.Unlock()
		}
	}(signal)

	return nil
}

func (b *BitfieldDistribution) ProcessPeerDisconnectedSignal(signal networkbridgeevents.PeerDisconnected) error {
	go func() {
		b.mu.Lock()
		delete(b.peerViews, signal.PeerID)
		b.mu.Unlock()
	}()

	return nil
}

func (b *BitfieldDistribution) ProcessNewGossipTopologySignal(signal networkbridgeevents.NewGossipTopology) error {
	//TODO implement me
	panic("implement me")
}

// TODO: improve or penalize the reputation of peers based on the messages that are received relative to the current view.
// this is where ReputationAggregator need to weight in
func (b *BitfieldDistribution) ProcessPeerViewChangeSignal(signal networkbridgeevents.PeerViewChange) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessOurViewChangeSignal(signal networkbridgeevents.OurViewChange) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessUpdatedAuthorityIDsSignal(signal networkbridgeevents.UpdatedAuthorityIDs) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessBitfieldDistributionMessageSignal(signal validationprotocol.BitfieldDistributionMessage) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (b *BitfieldDistribution) Stop() {
	logger.Infof("Stopping BitfieldDistribution subsystem")
}
