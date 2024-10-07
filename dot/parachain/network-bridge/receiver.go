// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	events "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "network-bridge"))

var (
	ErrFinalizedNumber                       = errors.New("finalized number is greater than or equal to the block number")
	ErrInvalidStringFormat                   = errors.New("invalid string format for fetched collation info")
	ErrUnexpectedMessageOnCollationProtocol  = errors.New("unexpected message on collation protocol")
	ErrUnexpectedMessageOnValidationProtocol = errors.New("unexpected message on validation protocol")
)

type NetworkBridgeReceiver struct {
	net Network

	BlockState *state.BlockState
	Keystore   keystore.Keystore

	localView *View

	// heads are sorted in descending order by block number
	liveHeads []parachaintypes.ActivatedLeaf

	finalizedNumber uint32

	SubsystemsToOverseer chan<- any

	networkEventInfoChan chan *network.NetworkEventInfo

	authorityDiscoveryService AuthorityDiscoveryService
}

type CollationStatus int

const (
	// We are waiting for a collation to be advertised to us.
	Waiting CollationStatus = iota
	// We are currently fetching a collation.
	Fetching
	// We are waiting that a collation is being validated.
	WaitingOnValidation
	// We have seconded a collation.
	Seconded
)

type CollationEvent struct {
	CollatorId       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type PendingCollation struct {
	RelayParent          common.Hash
	ParaID               parachaintypes.ParaID
	PeerID               peer.ID
	CommitmentHash       *common.Hash
	ProspectiveCandidate *ProspectiveCandidate
}

type ProspectiveCandidate struct {
	CandidateHash      parachaintypes.CandidateHash
	ParentHeadDataHash common.Hash
}

func RegisterReceiver(overseerChan chan<- any, net Network,
	collationProtocolID protocol.ID, validationProtocolID protocol.ID) (*NetworkBridgeReceiver, error) {
	nbr := &NetworkBridgeReceiver{
		net:                  net,
		SubsystemsToOverseer: overseerChan,
		networkEventInfoChan: net.GetNetworkEventsChannel(),
	}

	err := RegisterCollationProtocol(net, *nbr, collationProtocolID, overseerChan)
	if err != nil {
		return nil, fmt.Errorf("registering collation protocol: %w", err)
	}

	err = RegisterValidationProtocol(net, *nbr, validationProtocolID, overseerChan)
	if err != nil {
		return nil, fmt.Errorf("registering validation protocol: %w", err)
	}
	return nbr, nil
}

func (nbr *NetworkBridgeReceiver) Run(ctx context.Context, overseerToSubSystem <-chan any) {

	for {
		select {
		case msg := <-overseerToSubSystem:
			err := nbr.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case event := <-nbr.networkEventInfoChan:
			nbr.handleNetworkEvents(*event)
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (nbr *NetworkBridgeReceiver) handleNetworkEvents(event network.NetworkEventInfo) {
	switch event.Event {
	case network.Connected:
		nbr.SubsystemsToOverseer <- events.PeerConnected{
			PeerID:         event.PeerID,
			OverservedRole: event.Role,
			// TODO: Add protocol versions when we have them
		}
	case network.Disconnected:
		nbr.SubsystemsToOverseer <- events.PeerDisconnected{
			PeerID: event.PeerID,
		}
	}
}

func (nbr *NetworkBridgeReceiver) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeReceiver
}

func (nbr *NetworkBridgeReceiver) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) error {

	// TODO: #4207 get the value for majorSyncing for syncing package
	// majorSyncing means you are 5 blocks behind the tip of the chain and thus more aggressively
	// download blocks etc to reach the tip of the chain faster.
	var majorSyncing bool

	nbr.liveHeads = append(nbr.liveHeads, parachaintypes.ActivatedLeaf{
		Hash:   signal.Activated.Hash,
		Number: signal.Activated.Number,
	})

	newLiveHeads := []parachaintypes.ActivatedLeaf{}

	for _, head := range nbr.liveHeads {
		if slices.Contains(signal.Deactivated, head.Hash) {
			newLiveHeads = append(newLiveHeads, head)
		}
	}

	sort.Sort(SortableActivatedLeaves(newLiveHeads))
	nbr.liveHeads = newLiveHeads

	if !majorSyncing {
		// update our view
		err := nbr.updateOurView()
		if err != nil {
			return fmt.Errorf("updating our view: %w", err)
		}
	}
	return nil
}

type SortableActivatedLeaves []parachaintypes.ActivatedLeaf

func (s SortableActivatedLeaves) Len() int {
	return len(s)
}

func (s SortableActivatedLeaves) Less(i, j int) bool {
	return s[i].Number > s[j].Number
}

func (s SortableActivatedLeaves) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (nbr *NetworkBridgeReceiver) updateOurView() error { //nolint
	headHashes := []common.Hash{}
	for _, head := range nbr.liveHeads {
		headHashes = append(headHashes, head.Hash)
	}
	newView := View{
		Heads:           headHashes,
		FinalizedNumber: nbr.finalizedNumber,
	}

	// If this is the first view update since becoming active, but our view is empty,
	// there is no need to send anything.
	if nbr.localView == nil {
		*nbr.localView = newView
		return nil
	}

	// we only want to send a view update if the heads have changed.
	// A change in finalized block is not enough to trigger a view update.
	if nbr.localView.checkHeadsEqual(newView) {
		// nothing to update
		return nil
	}

	*nbr.localView = newView

	// TODO: #4156 send ViewUpdate to all the collation peers and validation peers (v1, v2, v3)
	// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/bridge/src/rx/mod.rs#L969-L1013

	// TODO #4156 Create our view and send collation events to all subsystems about our view change
	// Just create the network bridge and do both of these tasks as part of those. That's the only way it makes sense.

	return nil
}

func (nbr *NetworkBridgeReceiver) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Handle ViewUpdate message. ViewUpdate happens on both protocols. #4156 #4155

	// we don't propagate collation messages, so it will always be false
	propagate := false

	if msg.Type() != network.CollationMsgType {
		return propagate, fmt.Errorf("%w, expected: %d, found:%d", ErrUnexpectedMessageOnCollationProtocol,
			network.CollationMsgType, msg.Type())
	}

	collatorProtocol, ok := msg.(*collatorprotocolmessages.CollationProtocol)
	if !ok {
		return propagate, fmt.Errorf(
			"failed to cast into collator protocol message, expected: *CollationProtocol, got: %T",
			msg)
	}

	nbr.SubsystemsToOverseer <- events.PeerMessage[collatorprotocolmessages.CollationProtocol]{
		PeerID:  sender,
		Message: *collatorProtocol,
	}

	return propagate, nil
}

func (nbr *NetworkBridgeReceiver) handleValidationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {
	// we don't propagate collation messages, so it will always be false
	propagate := false

	if msg.Type() != network.ValidationMsgType {
		return propagate, fmt.Errorf("%w, expected: %d, found:%d", ErrUnexpectedMessageOnValidationProtocol,
			network.ValidationMsgType, msg.Type())
	}

	validationProtocol, ok := msg.(*validationprotocol.ValidationProtocol)
	if !ok {
		return propagate, fmt.Errorf(
			"failed to cast into collator protocol message, expected: *CollationProtocol, got: %T",
			msg)
	}

	nbr.SubsystemsToOverseer <- events.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  sender,
		Message: *validationProtocol,
	}

	return propagate, nil
}

func (nbr *NetworkBridgeReceiver) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	if nbr.finalizedNumber >= signal.BlockNumber {
		return ErrFinalizedNumber
	}
	nbr.finalizedNumber = signal.BlockNumber
	return nil
}

func (nbr *NetworkBridgeReceiver) Stop() {
	nbr.net.FreeNetworkEventsChannel(nbr.networkEventInfoChan)
}

func (nbr *NetworkBridgeReceiver) processMessage(msg any) error { //nolint
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case networkbridgemessages.NewGossipTopology:
		peerTopologies := getTopologyPeers(nbr.authorityDiscoveryService, msg.CanonicalShuffling)

		newGossipTopology := events.NewGossipTopology{
			Session: msg.Session,
			Topotogy: events.SessionGridTopology{
				ShuffledIndices:    msg.ShuffledIndices,
				CanonicalShuffling: peerTopologies,
			},
			LocalIndex: msg.LocalIndex,
		}

		nbr.SubsystemsToOverseer <- newGossipTopology
	case networkbridgemessages.UpdateAuthorityIDs:
		// TODO: Make sure that this does not cause a cycle of same events repeating.

		// NOTE: This comes from the gossip support subsystem.
		nbr.SubsystemsToOverseer <- events.UpdatedAuthorityIDs{
			PeerID:                msg.PeerID,
			AuthorityDiscoveryIDs: msg.AuthorityDiscoveryIDs,
		}
	}

	return nil
}

func getTopologyPeers(authorityDiscoveryService AuthorityDiscoveryService,
	neighbours []events.CanonicalShuffling) []events.TopologyPeerInfo {

	peers := make([]events.TopologyPeerInfo, len(neighbours))

	for _, neighbour := range neighbours {
		peerID := authorityDiscoveryService.GetPeerIDByAuthorityID(neighbour.AuthorityDiscoveryID)
		peers = append(peers, events.TopologyPeerInfo{
			PeerID:         []peer.ID{peerID},
			ValidatorIndex: neighbour.ValidatorIndex,
			DiscoveryID:    neighbour.AuthorityDiscoveryID,
		})
	}

	return peers
}

type AuthorityDiscoveryService interface {
	GetPeerIDByAuthorityID(authorityID parachaintypes.AuthorityDiscoveryID) peer.ID
}
