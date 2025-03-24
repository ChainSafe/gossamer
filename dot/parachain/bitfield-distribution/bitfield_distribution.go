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
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-distribution"))

type perRelayParentData struct {
	// Signing context for a particular relay parent.
	// the required part of the signing context
	sessionIndex parachaintypes.SessionIndex

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

func newPerRelayParentData(sessionIndex parachaintypes.SessionIndex, validatorSet []parachaintypes.ValidatorID) *perRelayParentData {
	return &perRelayParentData{
		sessionIndex:            sessionIndex,
		validatorsSet:           validatorSet,
		onePerValidator:         make(map[parachaintypes.ValidatorID]*validationprotocol.BitfieldDistributionMessage),
		messageSentToPeer:       make(map[peer.ID]map[parachaintypes.ValidatorID]struct{}),
		messageReceivedFromPeer: make(map[peer.ID]map[parachaintypes.ValidatorID]struct{}),
	}
}

// messageFromValidatorNeededByPeer determines if that particular message signed by a
// validator is needed by the given peer.
func (p *perRelayParentData) messageFromValidatorNeededByPeer(peerID peer.ID, signedBy parachaintypes.ValidatorID) bool {
	_, sendToExist := p.messageSentToPeer[peerID][signedBy]
	_, receiveFromExist := p.messageReceivedFromPeer[peerID][signedBy]

	return !sendToExist && !receiveFromExist
}

type BitfieldDistribution struct {
	subSystemToOverseer chan<- any

	peerViews map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}
	ourView        parachaintypes.View
	topologies     grid.SessionGridTopologyStorage
	perRelayParent map[common.Hash]*perRelayParentData
	reputation     *util.ReputationAggregator

	// TODO: Metrics

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
		topologies:     grid.SessionGridTopologyStorage{}, // TODO: update_topology the topologies in NewGossipTopology signal
		perRelayParent: make(map[common.Hash]*perRelayParentData),
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false // Always accumulate
		}),
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
	case networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]:
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

func (b *BitfieldDistribution) ProcessBitfieldDistributionMessageSignal(signal validationprotocol.BitfieldDistributionMessage) error {
	value, err := signal.Value()
	if err != nil {
		return err
	}

	// the BitfieldDistributionMessage incoming should be unchecked as bitfield signing subsystem only send over the
	// unchecked ones
	bitfieldDistributionMess, ok := value.(validationprotocol.UncheckedBitfield)
	if !ok {
		return fmt.Errorf("unexpected message type: %T", value)
	}

	// prepare the relay message data
	jobData := b.perRelayParent[bitfieldDistributionMess.Hash]
	if jobData == nil {
		logger.Debugf("not supposed to work on relay parent related data")
		return nil
	}

	sessionIdx := jobData.sessionIndex

	if len(jobData.validatorsSet) == 0 {
		logger.Debugf("validator set is empty")
		return nil
	}
	validatorIdx := bitfieldDistributionMess.UncheckedSignedAvailabilityBitfield.ValidatorIndex
	if uint32(validatorIdx) >= uint32(len(jobData.validatorsSet)) {
		logger.Debugf("could not find a validator for index %d", validatorIdx)
		return nil
	}
	validatorID := jobData.validatorsSet[validatorIdx]

	topology := b.topologies.GetTopologyOrFallback(sessionIdx).LocalNeighbours

	requiredRouting := topology.RequiredRoutingByIndex(validatorIdx, true)

	// check the unchecked bitfield message against the validator
	vpk, err := sr25519.NewPublicKey(validatorID[:])
	if err != nil {
		return err
	}
	checkedBitfield, err := bitfieldDistributionMess.UncheckedSignedAvailabilityBitfield.ToCheck(vpk)
	if err != nil {
		return fmt.Errorf("unable to verfy the signed bitfield message against the validator"+
			": %s, err :%s", validatorID, err)
	}

	// construct the relay message
	checkedBitfieldMessage := validationprotocol.CheckedBitfield{
		Hash:                              bitfieldDistributionMess.Hash,
		CheckedSignedAvailabilityBitfield: *checkedBitfield,
	}

	go relayMessage(jobData, topology, b.peerViews, validatorID, checkedBitfieldMessage, requiredRouting, b.subSystemToOverseer)

	return nil
}

func (b *BitfieldDistribution) ProcessPeerConnectedSignal(signal networkbridgeevents.PeerConnected) error {
	go func(pc networkbridgeevents.PeerConnected) {
		// only care about version 2 and 3
		// TODO: add protocol version support
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
	valueIdx, value, err := signal.Message.IndexValue()
	if err != nil {
		return err
	}
	// index 1 is BitfieldDistribution
	if valueIdx != 1 {
		return fmt.Errorf("invalid value index for peer message handler, supporsed to be 1: %d", valueIdx)
	}
	// TODO: add protocol version support
	m, err := value.(validationprotocol.BitfieldDistribution).BitfieldDistributionMessage.Value()
	if err != nil {
		return err
	}
	uncheckedBitfield, ok := m.(validationprotocol.UncheckedBitfield)
	if !ok {
		return fmt.Errorf("invalid value index for peer message handler, supporsed to be 1: %d", valueIdx)
	}
	relayParent := uncheckedBitfield.Hash
	bitfield := uncheckedBitfield.UncheckedSignedAvailabilityBitfield

	logger.Debugf("reveived bitfield gossip from peer")

	// not our concern
	if !b.ourView.Contains(relayParent) {
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Not interested in that parent hash",
		}, relayParent)
		return nil
	}

	jobData := b.perRelayParent[relayParent]
	if jobData == nil {
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Not interested in that parent hash",
		}, relayParent)
		return nil
	}

	validatorIdx := bitfield.ValidatorIndex
	validatorSet := jobData.validatorsSet

	if len(validatorSet) == 0 {
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Missing peer session key",
		}, relayParent)
		return nil
	}

	if uint32(validatorIdx) >= uint32(len(jobData.validatorsSet)) {
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMajor,
			Reason: "Bitfield validator index invalid",
		}, relayParent)
		logger.Debugf("could not find a validator for index %d", validatorIdx)
		return nil

	}
	validatorID := jobData.validatorsSet[validatorIdx]

	receivedSet := jobData.messageReceivedFromPeer[signal.PeerID]
	if receivedSet == nil {
		receivedSet = make(map[parachaintypes.ValidatorID]struct{})
		receivedSet[validatorID] = struct{}{}
	} else {
		_, ok := receivedSet[validatorID]
		if ok {
			logger.Debugf("duplicated message in messageReceivedFromPeer")
			go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
				Type:   util.CostMinorRepeated,
				Reason: "Peer sent the same message multiple times",
			}, relayParent)
			return nil
		}
	}

	// compare the bitfield from subsystem state against the incoming signal
	m, err = jobData.onePerValidator[validatorID].Value()
	if err != nil {
		return err
	}
	storedBitfield, ok := m.(validationprotocol.UncheckedBitfield)
	if !ok {
		return fmt.Errorf("unable to casting the stored BitfieldDistributionMessage in onePerValidator "+
			"for validatorID %d", validatorID)
	}
	if storedBitfield.UncheckedSignedAvailabilityBitfield.IsEqual(bitfield) {
		// already received a message for validator
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.BenefitMinor,
			Reason: "Valid message",
		}, relayParent)
		return nil
	}

	// verify the bitfield data against the validator
	vpk, err := sr25519.NewPublicKey(validatorID[:])
	if err != nil {
		return err
	}
	checkedBitfield, err := bitfield.ToCheck(vpk)
	if err != nil {
		go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMajor,
			Reason: "Bitfield signature invalid",
		}, relayParent)
		return err
	}

	// prepare for the relay message call
	topology := b.topologies.GetTopologyOrFallback(jobData.sessionIndex).LocalNeighbours
	requiredRouting := topology.RequiredRoutingByIndex(validatorIdx, false)
	message := validationprotocol.CheckedBitfield{
		Hash:                              relayParent,
		CheckedSignedAvailabilityBitfield: *checkedBitfield,
	}

	// update the subsystem state
	vdm := validationprotocol.NewBitfieldDistributionMessageVDT()
	err = vdm.SetValue(message)
	if err != nil {
		return err
	}
	jobData.onePerValidator[validatorID] = &vdm

	go relayMessage(jobData, topology, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	go modifyReputation(b.reputation, b.subSystemToOverseer, signal.PeerID, util.UnifiedReputationChange{
		Type:   util.BenefitMinorFirst,
		Reason: "Valid message with new information",
	}, relayParent)

	return nil
}

func (b *BitfieldDistribution) ProcessUpdatedAuthorityIDsSignal(signal networkbridgeevents.UpdatedAuthorityIDs) error {
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
	protocolVersion uint32 // ignore v1 peers after we have validation protocol version support
}, validatorID parachaintypes.ValidatorID, message validationprotocol.CheckedBitfield,
	requiredRouting grid.RequiredRouting, subsystemToOverSeerChan chan<- any) {
	relayParent := message.Hash

	// notify the overseer about a new and valid signed bitfield
	go func() {
		subsystemToOverSeerChan <- provisionermessages.ProvisionableDataBitfield{
			RelayParent: relayParent,
			Bitfield:    message.CheckedSignedAvailabilityBitfield,
		}
	}()

	// pass on the bitfield distribution to all interested peers
	// 1. get interested peers
	interestedPeers := make(map[peer.ID]uint32)
	for peerID, peerData := range peers {
		if peerData.view.Contains(relayParent) {
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

	// 2. insert the message sent for this peerData into jobData
	for peerID := range interestedPeers {
		jobData.messageSentToPeer[peerID] = map[parachaintypes.ValidatorID]struct{}{validatorID: {}}
	}

	// 3. send NetworkBridgeTxMessage::SendValidationMessage to all v2 and v3 peers
	v2InterestedPeers := filterByPeerVersion(interestedPeers, 2)
	if len(v2InterestedPeers) != 0 {
		go func() {
			// TODO: add version support for validation protocol
			bdm := &validationprotocol.BitfieldDistributionMessage{}
			err := bdm.SetValue(message)
			if err != nil {
				logger.Errorf("processing relay message for v2 protocol when setting the value for "+
					"BitfieldDistributionMessage: %s", err.Error())
				return
			}
			v := &validationprotocol.ValidationProtocol{}
			err = v.SetValue(validationprotocol.BitfieldDistribution{BitfieldDistributionMessage: *bdm})
			if err != nil {
				logger.Errorf("processing relay message for v2 protocol when setting the value for "+
					"ValidationProtocol: %s", err.Error())
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
			bdm := &validationprotocol.BitfieldDistributionMessage{}
			err := bdm.SetValue(message)
			if err != nil {
				logger.Errorf("processing relay message for v3 protocol when setting the value for "+
					"BitfieldDistributionMessage: %s", err.Error())
				return
			}

			v := &validationprotocol.ValidationProtocol{}
			err = v.SetValue(validationprotocol.BitfieldDistribution{BitfieldDistributionMessage: *bdm})
			if err != nil {
				logger.Errorf("processing relay message for v3 protocol when setting the value for "+
					"ValidationProtocol: %s", err.Error())
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

func modifyReputation(reputation *util.ReputationAggregator, sender chan<- any, peer peer.ID, rep util.UnifiedReputationChange, relayParent common.Hash) {
	logger.Infof("reputation modified for peer %s on relay parent %s", peer.String(), relayParent.String())

	reputation.Modify(sender, peer, rep)
}
