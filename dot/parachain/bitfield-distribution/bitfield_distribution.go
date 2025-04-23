// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfielddistribution

import (
	"context"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"

	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"

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
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-distribution"))

type perRelayParentData struct {
	// Signing context for a particular relay parent.
	signingContext parachaintypes.SigningContext

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

func newPerRelayParentData(signingContext parachaintypes.SigningContext, validatorSet []parachaintypes.ValidatorID,
) *perRelayParentData {
	return &perRelayParentData{
		signingContext:          signingContext,
		validatorsSet:           validatorSet,
		onePerValidator:         make(map[parachaintypes.ValidatorID]*validationprotocol.BitfieldDistributionMessage),
		messageSentToPeer:       make(map[peer.ID]map[parachaintypes.ValidatorID]struct{}),
		messageReceivedFromPeer: make(map[peer.ID]map[parachaintypes.ValidatorID]struct{}),
	}
}

// messageFromValidatorNeededByPeer determines if that particular message signed by a
// validator is needed by the given peer.
func (p *perRelayParentData) messageFromValidatorNeededByPeer(
	peerID peer.ID,
	signedBy parachaintypes.ValidatorID,
) bool {
	_, sendToExist := p.messageSentToPeer[peerID][signedBy]
	_, receiveFromExist := p.messageReceivedFromPeer[peerID][signedBy]

	return !sendToExist && !receiveFromExist
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// BitfieldDistribution is the parachain subsystem that is responsible for gossipping signed availability bitfields.
// The bitfields express which parachain block candidates the signing validator considers available.
type BitfieldDistribution struct {
	subSystemToOverseer chan<- any

	peerViews      map[peer.ID]*networkbridge.PeerDataViewWithVersion
	ourView        parachaintypes.View
	topologies     *grid.SessionGridTopologyStorage
	perRelayParent map[common.Hash]*perRelayParentData
	reputation     *util.ReputationAggregator

	blockState BlockState

	// TODO: Metrics

	mu sync.Mutex
}

func NewBitfieldDistribution(overseerChan chan<- any, blockState BlockState) *BitfieldDistribution {
	initTopologyEntry := &grid.SessionGridTopologyEntry{
		Topology:        grid.NewSessionGridTopology(make([]uint, 0), make([]grid.TopologyPeerInfo, 0)),
		LocalNeighbours: grid.NewEmptyGridNeighbours(),
	}
	t := &grid.SessionGridTopologyStorage{
		CurrentTopology: initTopologyEntry,
		PrevTopology:    initTopologyEntry,
	}

	return &BitfieldDistribution{
		subSystemToOverseer: overseerChan,
		peerViews:           make(map[peer.ID]*networkbridge.PeerDataViewWithVersion),
		ourView:             parachaintypes.View{},
		topologies:          t,
		perRelayParent:      make(map[common.Hash]*perRelayParentData),
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false // Always accumulate
		}),
		blockState: blockState,
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

// processMessage processes messages sent to the BitfieldDistribution subsystem
func (b *BitfieldDistribution) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.DistributeBitfield:
		err := b.processBitfieldDistributionMessage(msg)
		if err != nil {
			return fmt.Errorf("processing bitfield distribution message: %w", err)
		}
	case networkbridgeevents.PeerConnected:
		b.processPeerConnectedEvent(msg)
	case networkbridgeevents.PeerDisconnected:
		b.processPeerDisconnectedEvent(msg)
	case networkbridgeevents.NewGossipTopology:
		err := b.processNewGossipTopologyEvent(msg)
		if err != nil {
			return fmt.Errorf("processing new gossip topology event: %w", err)
		}
	case networkbridgeevents.PeerViewChange:
		b.processPeerViewChangeEvent(msg)
	case networkbridgeevents.OurViewChange:
		b.processOurViewChangeEvent(msg)
	case networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]:
		err := b.processIncomingPeerMessageEvent(msg)
		if err != nil {
			return fmt.Errorf("processing incoming peer message event: %w", err)
		}
	case networkbridgeevents.UpdatedAuthorityIDs:
		err := b.processUpdatedAuthorityIDsEvent(msg)
		if err != nil {
			return fmt.Errorf("processing updated authority IDs event: %w", err)
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

// processBitfieldDistributionMessage handles the signal incoming from bitfield signing subsystem
// should be unchecked as bitfield signing subsystem only send over the unchecked ones
func (b *BitfieldDistribution) processBitfieldDistributionMessage(msg parachaintypes.DistributeBitfield) error {
	logger.Tracef("process BitfieldDistributionMessage handler for relayParent: %s", msg.RelayParent)

	// prepare the relay message data
	jobData := b.perRelayParent[msg.RelayParent]
	if jobData == nil {
		logger.Debugf("not supposed to work on relay parent related data")
		return nil
	}

	sessionIdx := jobData.signingContext.SessionIndex

	if len(jobData.validatorsSet) == 0 {
		logger.Debugf("validator set is empty")
		return nil
	}
	validatorIdx := msg.Bitfield.ValidatorIndex
	if int(validatorIdx) >= len(jobData.validatorsSet) {
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

	checkedBitfield, err := msg.Bitfield.ToCheck(vpk)
	if err != nil {
		return fmt.Errorf("unable to verfy the signed bitfield message against the validator"+
			": %s, err :%s", validatorID, err)
	}

	// construct the relay message
	checkedBitfieldMessage := validationprotocol.CheckedBitfield{
		Hash:                              msg.RelayParent,
		CheckedSignedAvailabilityBitfield: *checkedBitfield,
	}

	relayMessage(jobData, topology, b.peerViews, validatorID, checkedBitfieldMessage, requiredRouting,
		b.subSystemToOverseer)

	return nil
}

func (b *BitfieldDistribution) processPeerConnectedEvent(event networkbridgeevents.PeerConnected) {
	logger.Tracef("peer connected: %s", event.PeerID)

	// only care about version 2 and 3
	// TODO: add protocol version support
	if event.ProtocolVersion == 2 || event.ProtocolVersion == 3 {
		b.mu.Lock()
		b.peerViews[event.PeerID] = &networkbridge.PeerDataViewWithVersion{
			View:            parachaintypes.View{}, // default view
			ProtocolVersion: event.ProtocolVersion,
		}
		b.mu.Unlock()
	}
}

func (b *BitfieldDistribution) processPeerDisconnectedEvent(event networkbridgeevents.PeerDisconnected) {
	logger.Tracef("peer disconnected: %s", event.PeerID)

	b.mu.Lock()
	delete(b.peerViews, event.PeerID)
	b.mu.Unlock()
}

func (b *BitfieldDistribution) processNewGossipTopologyEvent(event networkbridgeevents.NewGossipTopology) error {
	logger.Tracef("process NewGossipTopology event")

	sessionIdx := event.Session
	newTopology := event.Topology
	prevNeighbors := b.topologies.CurrentTopology.LocalNeighbours

	peers := make(map[peer.ID]struct{})
	for _, val := range newTopology.PeerIDs {
		peers[val] = struct{}{}
	}

	shuffledIndices := make([]uint, len(event.Topology.ShuffledIndices))
	for i, v := range event.Topology.ShuffledIndices {
		shuffledIndices[i] = uint(v)
	}

	canonicalShuffling := make([]grid.TopologyPeerInfo, len(event.Topology.CanonicalShuffling))
	for i, info := range event.Topology.CanonicalShuffling {
		t := grid.TopologyPeerInfo{
			Peers:          info.PeerID,
			ValidatorIndex: info.ValidatorIndex,
			DiscoveryID:    types.AuthorityID(info.DiscoveryID),
		}
		canonicalShuffling[i] = t
	}

	t := &grid.SessionGridTopology{
		Peers:              peers,
		ShuffledIndices:    shuffledIndices,
		CanonicalShuffling: canonicalShuffling,
	}
	err := b.topologies.UpdateCurrentTopology(sessionIdx, t, *event.LocalIndex)
	if err != nil {
		return err
	}

	newlyAdded := b.topologies.CurrentTopology.LocalNeighbours.PeersDiff(prevNeighbors)

	logger.Debugf("new gossip topology received: %s", newlyAdded)

	for _, id := range newlyAdded {
		peerView := b.peerViews[id]
		if peerView == nil {
			// For peers which are currently unknown, we'll send topology-related
			// messages to them when they connect and send their first view update.
			continue
		}
		oldView := peerView.View
		// in case we already knew that peer in the past
		// it might have had an existing view, we use to initialize
		// and minimise the delta on `PeerViewChange` to be sent
		peerView.View = parachaintypes.View{}

		b.handlePeerViewChange(id, oldView)
	}

	return nil
}

func (b *BitfieldDistribution) processPeerViewChangeEvent(event networkbridgeevents.PeerViewChange) {
	logger.Tracef("process PeerViewChange event")

	if b.peerViews[event.PeerID] != nil {
		b.handlePeerViewChange(event.PeerID, event.View)
	}
}

func (b *BitfieldDistribution) processOurViewChangeEvent(event networkbridgeevents.OurViewChange) {
	logger.Tracef("process OurViewChange event: %v", event)

	oldView := b.ourView
	b.ourView = event.View

	// in the new view but not in the old view
	for _, added := range b.ourView.Difference(oldView) {
		// newly added views will be handled in the active leaves update handler, ideally this should never happen
		if b.perRelayParent[added] == nil {
			logger.Errorf("our view contains %s, but not in active heads", added.String())
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// in the old view but not in the new view
	for _, removed := range oldView.Difference(b.ourView) {
		delete(b.perRelayParent, removed)
	}
}

func (b *BitfieldDistribution) processIncomingPeerMessageEvent(event networkbridgeevents.PeerMessage[validationprotocol.
	ValidationProtocol]) error {
	logger.Tracef("process incoming PeerMessage event: %v", event)

	v, err := event.Message.Value()
	if err != nil {
		return err
	}
	value, ok := v.(validationprotocol.BitfieldDistribution)
	if !ok {
		return fmt.Errorf("invalid value for peer message handler, supporsed to be " +
			"validationprotocol.BitfieldDistribution")
	}
	m, err := value.BitfieldDistributionMessage.Value()
	if err != nil {
		return err
	}
	uncheckedBitfield, ok := m.(validationprotocol.UncheckedBitfield)
	if !ok {
		return fmt.Errorf("invalid value for uncheckedBitfield")
	}
	relayParent := uncheckedBitfield.Hash
	bitfield := uncheckedBitfield.UncheckedSignedAvailabilityBitfield

	logger.Debugf("reveived bitfield gossip from peer")

	// not our concern
	if !b.ourView.Contains(relayParent) {
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Not interested in that parent hash",
		}, relayParent)
		return nil
	}

	jobData := b.perRelayParent[relayParent]
	if jobData == nil {
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Not interested in that parent hash",
		}, relayParent)
		return nil
	}

	validatorIdx := bitfield.ValidatorIndex
	validatorSet := jobData.validatorsSet

	if len(validatorSet) == 0 {
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMinor,
			Reason: "Missing peer session key",
		}, relayParent)
		return nil
	}

	if int(validatorIdx) >= len(jobData.validatorsSet) {
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMajor,
			Reason: "Bitfield validator index invalid",
		}, relayParent)
		logger.Debugf("could not find a validator for index %d", validatorIdx)
		return nil

	}
	validatorID := jobData.validatorsSet[validatorIdx]

	receivedSet := jobData.messageReceivedFromPeer[event.PeerID]
	if receivedSet == nil {
		receivedSet = make(map[parachaintypes.ValidatorID]struct{})
		receivedSet[validatorID] = struct{}{}
	} else {
		_, ok := receivedSet[validatorID]
		if ok {
			logger.Debugf("duplicated message in messageReceivedFromPeer")

			modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
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
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
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
		modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
			Type:   util.CostMajor,
			Reason: "Bitfield signature invalid",
		}, relayParent)
		return err
	}

	// prepare for the relay message call
	topology := b.topologies.GetTopologyOrFallback(jobData.signingContext.SessionIndex).LocalNeighbours
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

	relayMessage(jobData, topology, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	modifyReputation(b.reputation, b.subSystemToOverseer, event.PeerID, util.UnifiedReputationChange{
		Type:   util.BenefitMinorFirst,
		Reason: "Valid message with new information",
	}, relayParent)

	return nil
}

func (b *BitfieldDistribution) processUpdatedAuthorityIDsEvent(event networkbridgeevents.UpdatedAuthorityIDs) error {
	logger.Tracef("process UpdatedAuthorityIDs event: %v", event)

	ids := make(map[types.AuthorityID]struct{})
	for _, id := range event.AuthorityDiscoveryIDs {
		ids[types.AuthorityID(id)] = struct{}{}
	}
	ok, err := b.topologies.CurrentTopology.UpdateAuthoritiesIDs(event.PeerID, ids)
	if err != nil {
		logger.Errorf("error while updating authority IDs : %s", err.Error())
		return err
	}
	if !ok {
		logger.Warnf("could not update authority IDs : %v", event.AuthorityDiscoveryIDs)
		return nil
	}

	return nil
}

func (b *BitfieldDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	activatedLeaf := signal.Activated
	if activatedLeaf == nil {
		return nil
	}

	relayParent := activatedLeaf.Hash

	logger.Tracef("Process ActiveLeavesUpdateSignal activatedLeaf: %v", relayParent)

	validatorSet, signingContext, err := b.queryBasics(relayParent)
	if err != nil {
		logger.Warnf("could not query basic info for relayParent: %s, error: %v", relayParent.String(), err)
		return err
	}

	b.mu.Lock()
	b.perRelayParent[relayParent] = newPerRelayParentData(*signingContext, validatorSet)
	b.mu.Unlock()

	return nil
}

func (b *BitfieldDistribution) ProcessBlockFinalizedSignal(_ parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (b *BitfieldDistribution) Stop() {
	logger.Tracef("Stopping BitfieldDistribution subsystem")
}

// relayMessage distributes a given valid and signature checked bitfield message.
//
// Can be originated by another subsystem or received via network from another peer.
func relayMessage(
	jobData *perRelayParentData,
	topologyNeighbors *grid.GridNeighbours,
	peers map[peer.ID]*networkbridge.PeerDataViewWithVersion,
	validatorID parachaintypes.ValidatorID,
	message validationprotocol.CheckedBitfield,
	requiredRouting grid.RequiredRouting,
	subsystemToOverSeerChan chan<- any,
) {
	relayParent := message.Hash

	// notify the overseer about a new and valid signed bitfield
	subsystemToOverSeerChan <- provisionermessages.ProvisionableDataBitfield{
		RelayParent: relayParent,
		Bitfield:    message.CheckedSignedAvailabilityBitfield,
	}

	// pass on the bitfield distribution to all interested peers
	// 1. get interested peers
	interestedPeers := make(map[peer.ID]uint32)
	for peerID, peerData := range peers {
		if peerData != nil && peerData.View.Contains(relayParent) {
			if jobData.messageFromValidatorNeededByPeer(peerID, validatorID) {
				needRouting := topologyNeighbors.ShouldRouteToPeer(requiredRouting, peerID)
				if needRouting {
					interestedPeers[peerID] = peerData.ProtocolVersion
				}
			}
		}
	}

	if len(interestedPeers) == 0 {
		logger.Tracef("no peers are interested in gossip for relay parent")
		return
	}

	// 2. insert the message sent for this peerData into jobData
	for peerID := range interestedPeers {
		jobData.messageSentToPeer[peerID] = map[parachaintypes.ValidatorID]struct{}{validatorID: {}}
	}

	// 3. send NetworkBridgeTxMessage::SendValidationMessage to all v2 and v3 peers
	v2InterestedPeers := filterByPeerVersion(interestedPeers, 2)
	if len(v2InterestedPeers) != 0 {
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
	}

	v3InterestedPeers := filterByPeerVersion(interestedPeers, 3)
	if len(v3InterestedPeers) != 0 {
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
	}
}

// handlePeerViewChange sends the difference between two views which were not sent to that particular peer
func (b *BitfieldDistribution) handlePeerViewChange(
	origin peer.ID,
	view parachaintypes.View,
) {
	peerData := b.peerViews[origin]
	if peerData == nil {
		logger.Warnf("attempted to update peer view for unknown peer: %s", origin)
		return
	}

	added := peerData.View.ReplaceDifference(view)

	topology := b.topologies.CurrentTopology.LocalNeighbours
	isGossipPeer := topology.ShouldRouteToPeer(grid.RequiredRoutingGridXY, origin)

	if !isGossipPeer {
		logger.Tracef("peer view change is ignored")
		return
	}

	for _, hash := range added {
		jobData := b.perRelayParent[hash]
		if jobData != nil {
			for validatorId, message := range jobData.onePerValidator {
				if jobData.messageFromValidatorNeededByPeer(origin, validatorId) {
					v, err := message.Value()
					if err != nil {
						logger.Errorf("failed to extract the value from BitfieldDistributionMessage: %s",
							err.Error())
						return
					}
					bitfield, ok := v.(validationprotocol.CheckedBitfield)
					if !ok {
						logger.Error("attempted to cast the BitfieldDistributionMessage to CheckedBitfield " +
							"but got UncheckedBitfield")
						return
					}
					b.sendTrackedGossipMessage(origin, validatorId, bitfield)
				}
			}
		}
	}
}

// sendTrackedGossipMessage sends a gossip message and tracks it in the per relay parent data
func (b *BitfieldDistribution) sendTrackedGossipMessage(
	dest peer.ID,
	validatorId parachaintypes.ValidatorID,
	message validationprotocol.CheckedBitfield,
) {
	jobData := b.perRelayParent[message.Hash]
	if jobData == nil {
		return
	}

	logger.Tracef("sending gossip message")

	version := b.peerViews[dest]
	if version == nil {
		return
	}

	if jobData.messageSentToPeer[dest] == nil {
		jobData.messageSentToPeer[dest] = map[parachaintypes.ValidatorID]struct{}{
			validatorId: {},
		}
	}

	// TODO: add version support for validation protocol
	bdm := &validationprotocol.BitfieldDistributionMessage{}
	err := bdm.SetValue(message)
	if err != nil {
		logger.Errorf("processing BitfieldDistributionMessage: %s", err.Error())
		return
	}
	v := &validationprotocol.ValidationProtocol{}
	err = v.SetValue(validationprotocol.BitfieldDistribution{BitfieldDistributionMessage: *bdm})
	if err != nil {
		logger.Errorf("processing BitfieldDistribution: %s", err.Error())
		return
	}

	b.subSystemToOverseer <- networkbridgemessages.SendValidationMessage{
		To:                        []peer.ID{dest},
		ValidationProtocolMessage: *v,
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

func modifyReputation(reputation *util.ReputationAggregator, sender chan<- any, peer peer.ID,
	rep util.UnifiedReputationChange, relayParent common.Hash) {
	logger.Tracef("reputation modified for peer %s on relay parent %s", peer.String(), relayParent.String())

	reputation.Modify(sender, peer, rep)
}

// queryBasics queries our validator set and signing context for a particular relay parent
func (b *BitfieldDistribution) queryBasics(
	relayParent common.Hash,
) ([]parachaintypes.ValidatorID, *parachaintypes.SigningContext, error) {
	rt, err := b.blockState.GetRuntime(relayParent)
	if err != nil {
		return nil, nil, err
	}

	// query validators
	validatorSet, err := rt.ParachainHostValidators()
	if err != nil {
		return nil, nil, err
	}

	// query signing context
	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, nil, err
	}

	return validatorSet, &parachaintypes.SigningContext{
		SessionIndex: sessionIndex,
		ParentHash:   relayParent,
	}, nil
}
