// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	events "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "network-bridge"))

var (
	ErrFinalizedNumber                       = errors.New("finalized number is greater than or equal to the block number")
	ErrInvalidStringFormat                   = errors.New("invalid string format for fetched collation info")
	ErrUnexpectedMessageOnCollationProtocol  = errors.New("unexpected message on collation protocol")
	ErrUnexpectedMessageOnValidationProtocol = errors.New("unexpected message on collation protocol")
)

type NetworkBridgeReceiver struct {
	net Network

	BlockState *state.BlockState
	Keystore   keystore.Keystore

	localView *View

	// Parachains we're currently assigned to. With async backing enabled
	// this includes assignments from the implicit view.
	currentAssignments map[parachaintypes.ParaID]uint

	/// All active leaves observed by us, including both that do and do not
	/// support prospective parachains. This mapping works as a replacement for
	/// [`polkadot_node_network_protocol::View`] and can be dropped once the transition
	/// to asynchronous backing is done.
	activeLeaves map[common.Hash]parachaintypes.ProspectiveParachainsMode

	// state tracked per relay parent
	perRelayParent map[common.Hash]PerRelayParent // map[relay parent]PerRelayParent

	// Collations that we have successfully requested from peers and waiting
	// on validation.
	fetchedCandidates map[string]CollationEvent
	// heads are sorted in descending order by block number
	liveHeads []parachaintypes.ActivatedLeaf

	finalizedNumber uint32

	OverseerToSubSystem  <-chan any
	SubsystemsToOverseer chan<- any

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

// Identifier of a fetched collation
type fetchedCollationInfo struct {
	// Candidate's relay parent
	relayParent   common.Hash
	paraID        parachaintypes.ParaID
	candidateHash parachaintypes.CandidateHash
	// Id of the collator the collation was fetched from
	collatorID parachaintypes.CollatorID
}

func (f fetchedCollationInfo) String() string {
	return fmt.Sprintf("relay parent: %s, para id: %d, candidate hash: %s, collator id: %+v",
		f.relayParent.String(), f.paraID, f.candidateHash.Value.String(), f.collatorID)
}

func fetchedCandidateFromString(str string) (fetchedCollationInfo, error) {
	splits := strings.Split(str, ",")
	if len(splits) != 4 {
		return fetchedCollationInfo{}, fmt.Errorf("%w: %s", ErrInvalidStringFormat, str)
	}

	relayParent, err := common.HexToHash(strings.TrimSpace(splits[0]))
	if err != nil {
		return fetchedCollationInfo{}, fmt.Errorf("getting relay parent: %w", err)
	}

	paraID, err := strconv.ParseUint(strings.TrimSpace(splits[1]), 10, 64)
	if err != nil {
		return fetchedCollationInfo{}, fmt.Errorf("getting para id: %w", err)
	}

	candidateHashBytes, err := common.HexToBytes(strings.TrimSpace(splits[2]))
	if err != nil {
		return fetchedCollationInfo{}, fmt.Errorf("getting candidate hash bytes: %w", err)
	}

	candidateHash := parachaintypes.CandidateHash{
		Value: common.NewHash(candidateHashBytes),
	}

	var collatorID parachaintypes.CollatorID
	collatorIDBytes, err := common.HexToBytes(strings.TrimSpace(splits[3]))
	if err != nil {
		return fetchedCollationInfo{}, fmt.Errorf("getting collator id bytes: %w", err)
	}
	copy(collatorID[:], collatorIDBytes)

	return fetchedCollationInfo{
		relayParent:   relayParent,
		paraID:        parachaintypes.ParaID(paraID),
		candidateHash: candidateHash,
		collatorID:    collatorID,
	}, nil
}

type PerRelayParent struct {
	prospectiveParachainMode parachaintypes.ProspectiveParachainsMode
	assignment               *parachaintypes.ParaID
	collations               Collations
}

type Collations struct {
	// What is the current status in regards to a collation for this relay parent?
	status CollationStatus
	// how many collations have been seconded
	secondedCount uint
	// Collation that were advertised to us, but we did not yet fetch.
	waitingQueue []UnfetchedCollation // : VecDeque<(PendingCollation, CollatorId)>,
}

type UnfetchedCollation struct {
	CollatorID       parachaintypes.CollatorID
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

// IsSecondedLimitReached check the limit of seconded candidates for a given para has been reached.
func (collations Collations) IsSecondedLimitReached(relayParentMode parachaintypes.ProspectiveParachainsMode) bool {
	var secondedLimit uint
	if relayParentMode.IsEnabled {
		secondedLimit = relayParentMode.MaxCandidateDepth + 1
	} else {
		secondedLimit = 1
	}

	return collations.secondedCount >= secondedLimit
}

func RegisterReceiver(overseerChan chan<- any, net Network, collationProtocolID protocol.ID, validationProtocolID protocol.ID) (*NetworkBridgeReceiver, error) {
	nbr := &NetworkBridgeReceiver{
		net:                  net,
		SubsystemsToOverseer: overseerChan,
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

func (nbr *NetworkBridgeReceiver) Run(ctx context.Context, OverseerToSubSystem chan any,
	SubSystemToOverseer chan any) {
	for msg := range nbr.OverseerToSubSystem {
		err := nbr.processMessage(msg)
		if err != nil {
			logger.Errorf("processing overseer message: %w", err)
		}
	}
}

func (nbr *NetworkBridgeReceiver) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeReceiver
}

func (nbr *NetworkBridgeReceiver) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) error {

	// TODO update cpvs.activeLeaves by adding new active leaves and removing deactivated ones

	// TODO: get the value for majorSyncing for syncing package
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
	// TODO: do I need to store these live heads or just pass them to update view?
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

func (nbr *NetworkBridgeReceiver) updateOurView() error {
	headHashes := []common.Hash{}
	for _, head := range nbr.liveHeads {
		headHashes = append(headHashes, head.Hash)
	}
	newView := View{
		heads:           headHashes,
		finalizedNumber: nbr.finalizedNumber,
	}

	if nbr.localView == nil {
		*nbr.localView = newView
		return nil
	}

	if nbr.localView.checkHeadsEqual(newView) {
		// nothing to update
		return nil
	}

	*nbr.localView = newView

	// TODO: send ViewUpdate to all the collation peers and validation peers (v1, v2, v3)
	// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/bridge/src/rx/mod.rs#L969-L1013

	// TODO: Create our view and send collation events to all subsystems about our view change
	// Just create the network bridge and do both of these tasks as part of those. That's the only way it makes sense.

	err := nbr.handleOurViewChange(newView)
	if err != nil {
		return fmt.Errorf("handling our view change: %w", err)
	}
	return nil
}

func (nbr *NetworkBridgeReceiver) handleOurViewChange(view View) error {
	// 1. Find out removed leaves (hashes) and newly added leaves
	// 2. Go over each new leaves,
	// - check if perspective parachain mode is enabled
	// - assign incoming
	// - insert active leaves and per relay parent
	activeLeaves := nbr.activeLeaves

	removed := []common.Hash{}
	for activeLeaf := range activeLeaves {
		if !slices.Contains(view.heads, activeLeaf) {
			removed = append(removed, activeLeaf)
		}
	}

	newlyAdded := []common.Hash{}
	for _, head := range view.heads {
		if _, ok := activeLeaves[head]; !ok {
			newlyAdded = append(newlyAdded, head)
		}
	}

	// handled newly added leaves
	for _, leaf := range newlyAdded {
		mode := prospectiveParachainMode()

		perRelayParent := &PerRelayParent{
			prospectiveParachainMode: mode,
		}

		err := nbr.assignIncoming(leaf, perRelayParent)
		if err != nil {
			return fmt.Errorf("assigning incoming: %w", err)
		}
		nbr.activeLeaves[leaf] = mode
		nbr.perRelayParent[leaf] = *perRelayParent

		//nolint:staticcheck
		if mode.IsEnabled {
			// TODO: Add it when we have async backing
			// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1303 //nolint
		}
	}

	// handle removed leaves
	for _, leaf := range removed {
		delete(nbr.activeLeaves, leaf)

		mode := prospectiveParachainMode()
		pruned := []common.Hash{}
		if mode.IsEnabled {
			// TODO: Do this when we have async backing
			// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1340 //nolint
		} else {
			pruned = append(pruned, leaf)
		}

		for _, prunedLeaf := range pruned {
			perRelayParent, ok := nbr.perRelayParent[prunedLeaf]
			if ok {
				nbr.removeOutgoing(perRelayParent)
				delete(nbr.perRelayParent, prunedLeaf)
			}

			for fetchedCandidateStr := range nbr.fetchedCandidates {
				fetchedCollation, err := fetchedCandidateFromString(fetchedCandidateStr)
				if err != nil {
					// this should never really happen
					return fmt.Errorf("getting fetched collation from string: %w", err)
				}

				if fetchedCollation.relayParent == prunedLeaf {
					delete(nbr.fetchedCandidates, fetchedCandidateStr)
				}
			}
		}

		// TODO
		// Remove blocked advertisements that left the view. cpvs.BlockedAdvertisements
		// Re-trigger previously failed requests again. requestUnBlockedCollations
		// prune old advertisements
		// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1361-L1396

	}

	return nil
}

func (nbr *NetworkBridgeReceiver) removeOutgoing(perRelayParent PerRelayParent) {
	if perRelayParent.assignment != nil {
		entry := nbr.currentAssignments[*perRelayParent.assignment]
		entry--
		if entry == 0 {
			logger.Infof("unassigned from parachain with ID %d", *perRelayParent.assignment)
			delete(nbr.currentAssignments, *perRelayParent.assignment)
			return
		}

		nbr.currentAssignments[*perRelayParent.assignment] = entry
	}
}

// signingKeyAndIndex finds the first key we can sign with from the given set of validators,
// if any, and returns it along with the validator index.
func signingKeyAndIndex(validators []parachaintypes.ValidatorID, ks keystore.Keystore,
) (*parachaintypes.ValidatorID, parachaintypes.ValidatorIndex) {
	for i, validator := range validators {
		publicKey, _ := sr25519.NewPublicKey(validator[:])
		keypair := ks.GetKeypair(publicKey)

		if keypair != nil {
			return &validator, parachaintypes.ValidatorIndex(i)
		}
	}

	return nil, 0
}

func (nbr *NetworkBridgeReceiver) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {

	// TODO: this notification has to be a WireMessage. Check if it is a WireMessage
	// TODO: Handle ViewUpdate message. ViewUpdate happens on both protocols.

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
		PeerID:   sender,
		Messaage: *collatorProtocol,
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
		PeerID:   sender,
		Messaage: *validationProtocol,
	}

	return propagate, nil
}

func (nbr *NetworkBridgeReceiver) assignIncoming(relayParent common.Hash, perRelayParent *PerRelayParent,
) error {
	// TODO: get this instance using relay parent
	instance, err := nbr.BlockState.GetRuntime(relayParent)
	if err != nil {
		return fmt.Errorf("getting runtime instance: %w", err)
	}

	validators, err := instance.ParachainHostValidators()
	if err != nil {
		return fmt.Errorf("getting validators: %w", err)
	}

	validatorGroups, err := instance.ParachainHostValidatorGroups()
	if err != nil {
		return fmt.Errorf("getting validator groups: %w", err)
	}

	availabilityCores, err := instance.ParachainHostAvailabilityCores()
	if err != nil {
		return fmt.Errorf("getting availability cores: %w", err)
	}

	validator, validatorIndex := signingKeyAndIndex(validators, nbr.Keystore)
	if validator == nil {
		// return with an error?
		return nil
	}

	groupIndex, ok := findValidatorGroup(validatorIndex, *validatorGroups)
	if !ok {
		logger.Trace("not a validator")
		return nil
	}

	coreIndexNow := validatorGroups.GroupRotationInfo.CoreForGroup(groupIndex, uint8(len(availabilityCores)))
	coreNow, err := availabilityCores[coreIndexNow.Index].Value()
	if err != nil {
		return fmt.Errorf("getting core now: %w", err)
	}

	var paraNow parachaintypes.ParaID
	var paraNowSet bool
	switch c := coreNow.(type) /*coreNow.Index()*/ {
	case parachaintypes.OccupiedCore:
		paraNow = parachaintypes.ParaID(c.CandidateDescriptor.ParaID)
		paraNowSet = true
	case parachaintypes.ScheduledCore:
		paraNow = c.ParaID
		paraNowSet = true
	case parachaintypes.Free:
		// Nothing to do in case of free

	}

	if !paraNowSet {
		entry := nbr.currentAssignments[paraNow]
		entry++
		nbr.currentAssignments[paraNow] = entry
		if entry == 1 {
			logger.Infof("got assigned to parachain with ID %d", paraNow)
		}
	} else {
		perRelayParent.assignment = &paraNow
	}

	return nil
}

func findValidatorGroup(validatorIndex parachaintypes.ValidatorIndex, validatorGroups parachaintypes.ValidatorGroups,
) (parachaintypes.GroupIndex, bool) {
	for groupIndex, validatorGroup := range validatorGroups.Validators {
		for _, i := range validatorGroup {
			if i == validatorIndex {
				return parachaintypes.GroupIndex(groupIndex), true
			}
		}
	}

	return 0, false
}

func prospectiveParachainMode() parachaintypes.ProspectiveParachainsMode {
	// TODO: complete this method by calling the runtime function
	// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/subsystem-util/src/runtime/mod.rs#L496 //nolint
	// NOTE: We will return false until we have support for async backing
	return parachaintypes.ProspectiveParachainsMode{
		IsEnabled: false,
	}
}

func (nbr *NetworkBridgeReceiver) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	if nbr.finalizedNumber >= signal.BlockNumber {
		return ErrFinalizedNumber
	}
	nbr.finalizedNumber = signal.BlockNumber
	return nil
}

func (nbr *NetworkBridgeReceiver) Stop() {}

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

func getTopologyPeers(authorityDiscoveryService AuthorityDiscoveryService, neighbours []events.CanonicalShuffling) []events.TopologyPeerInfo {
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
