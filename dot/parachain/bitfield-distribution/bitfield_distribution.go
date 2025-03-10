// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfielddistribution

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
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

// TODO: impl me
func NewPerRelayParentData(sessionIndex parachaintypes.SessionIndex) {

}

// messageFromValidatorNeededByPeer determines if that particular message signed by a
// validator is needed by the given peer.
func (p *perRelayParentData) messageFromValidatorNeededByPeer(peerID peer.ID, signedBy parachaintypes.ValidatorID) bool {
	_, sendToExist := p.messageSentToPeer[peerID][signedBy]
	_, receiveFromExist := p.messageReceivedFromPeer[peerID][signedBy]

	return sendToExist && receiveFromExist
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
	case networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]: // TODO: are we sure this type would be validationprotocol.ValidationProtocol?
		err := b.ProcessPeerMessageSignal(msg)
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
				view:            parachaintypes.View{}, // default view
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

func (b *BitfieldDistribution) ProcessPeerMessageSignal(signal networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessUpdatedAuthorityIDsSignal(signal networkbridgeevents.UpdatedAuthorityIDs) error {
	//TODO implement me
	panic("implement me")
}

// TODO: sending bitfield messages to provisioning subsystem and to other peers
// handle_bitfield_distribution()
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

// relayMessage distributes a given valid and signature checked bitfield message.
//
// Can be originated by another subsystem or received via network from another peer.
func relayMessage(jobData *perRelayParentData, topologyNeighbors *grid.GridNeighbours, peers map[peer.ID]struct {
	view            parachaintypes.View
	protocolVersion uint32 // ignore v1 peers
}, validatorID parachaintypes.ValidatorID, message parachaintypes.DistributeBitfield,
	requiredRouting grid.RequiredRouting, subsystemToOverSeerChan chan<- any) {
	relayParent := message.RelayParent

	// notify the overseer about a new and valid signed bitfield
	go func() {
		subsystemToOverSeerChan <- provisionermessages.ProvisionableDataBitfield{
			RelayParent: relayParent,
			Bitfield:    message.Bitfield,
		}
	}()

	// pass on the bitfield distribution to all interested peers
	// 1. get interested peers
	interestedPeers := make(map[peer.ID]uint32)

	for peerID, peerData := range peers {
		if peerData.view.Contains(message.RelayParent) {
			if jobData.messageFromValidatorNeededByPeer(peerID, validatorID) {
				needRouting := topologyNeighbors.ShouldRouteToPeer(requiredRouting, peerID)
				// TODO: need random routing?

				if needRouting {
					interestedPeers[peerID] = peerData.protocolVersion
				}
			}
		}
	}

	if len(interestedPeers) == 0 {
		logger.Infof("no peers are interested in gossip for relay parent")
		return
	}

	// 2. insert the message sent for this peerData into job_data
	for peerID := range interestedPeers {
		jobData.messageSentToPeer[peerID] = map[parachaintypes.ValidatorID]struct{}{validatorID: {}}
	}

	// 3. send NetworkBridgeTxMessage::SendValidationMessage to all v2 and v3 peers
	v2InterestedPeers := filterByPeerVersion(interestedPeers, 2)
	if len(v2InterestedPeers) != 0 {
		go func() {
			// TODO: add version support for validation protocol
			v := &validationprotocol.ValidationProtocol{}
			err := v.SetValue(message)
			if err != nil {
				logger.Errorf("processing relay message for v2 protocol: %s", err.Error())
				return
			}
			subsystemToOverSeerChan <- networkbridgemessages.SendValidationMessage{
				To:                        v2InterestedPeers,
				ValidationProtocolMessage: *v,
			}
		}()
	}

	v3InterestedPeers := filterByPeerVersion(interestedPeers, 3)
	if len(v3InterestedPeers) != 0 {
		go func() {
			// TODO: add version support for validation protocol
			v := &validationprotocol.ValidationProtocol{}
			err := v.SetValue(message)
			if err != nil {
				logger.Errorf("processing relay message for v3 protocol: %s", err.Error())
				return
			}
			subsystemToOverSeerChan <- networkbridgemessages.SendValidationMessage{
				To:                        v3InterestedPeers,
				ValidationProtocolMessage: *v,
			}
		}()
	}
}

func filterByPeerVersion(peers map[peer.ID]uint32, protocolVersion uint32) []peer.ID {
	re := make([]peer.ID, 0)

	for peerID, version := range peers {
		if version == protocolVersion {
			re = append(re, peerID)
		}
	}

	return re
}
