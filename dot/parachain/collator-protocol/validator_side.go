// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/exp/slices"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "collator-protocol"))

const (
	activityPoll            = 10 * time.Millisecond
	maxUnsharedDownloadTime = 100 * time.Millisecond
)

var (
	ErrUnexpectedMessageOnCollationProtocol = errors.New("unexpected message on collation protocol")
	ErrUnknownPeer                          = errors.New("unknown peer")
	ErrNotExpectedOnValidatorSide           = errors.New("message is not expected on the validator side of the protocol")
	ErrCollationNotInView                   = errors.New("collation is not in our view")
	ErrPeerIDNotFoundForCollator            = errors.New("peer id not found for collator")
	ErrProtocolMismatch                     = errors.New("an advertisement format doesn't match the relay parent")
	ErrSecondedLimitReached                 = errors.New("para reached a limit of seconded" +
		" candidates for this relay parent")
	ErrRelayParentUnknown     = errors.New("relay parent is unknown")
	ErrUndeclaredPara         = errors.New("peer has not declared its para id")
	ErrInvalidAssignment      = errors.New("we're assigned to a different para at the given relay parent")
	ErrInvalidAdvertisement   = errors.New("advertisement is invalid")
	ErrUndeclaredCollator     = errors.New("no prior declare message received for this collator")
	ErrOutOfView              = errors.New("collation relay parent is out of our view")
	ErrDuplicateAdvertisement = errors.New("advertisement is already known")
	ErrPeerLimitReached       = errors.New("limit for announcements per peer is reached")
	ErrNotAdvertised          = errors.New("collation was not previously advertised")

	ErrInvalidStringFormat = errors.New("invalid string format for fetched collation info")
)

func (cpvs CollatorProtocolValidatorSide) Run(
	ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
	inactivityTicker := time.NewTicker(activityPoll)

	for {
		select {
		// TODO: polkadot-rust changes reputation in batches, so we do the same?
		case msg, ok := <-cpvs.OverseerToSubSystem:
			if !ok {
				return
			}

			err := cpvs.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}

		case event := <-cpvs.networkEventInfoChan:
			cpvs.handleNetworkEvents(*event)
		case <-inactivityTicker.C:
			// TODO: disconnect inactive peers
			// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1301

		case unfetchedCollation := <-cpvs.unfetchedCollation:
			// TODO: If we can't get the collation from given collator within MAX_UNSHARED_DOWNLOAD_TIME,
			// we will start another one from the next collator.

			// check if this peer id has advertised this relay parent
			peerData := cpvs.peerData[unfetchedCollation.PendingCollation.PeerID]
			if peerData.HasAdvertised(unfetchedCollation.PendingCollation.RelayParent, nil) {
				// if so request collation from this peer id
				collation, err := cpvs.requestCollation(unfetchedCollation.PendingCollation.RelayParent,
					unfetchedCollation.PendingCollation.ParaID, unfetchedCollation.PendingCollation.PeerID)
				if err != nil {
					logger.Errorf("fetching collation: %w", err)
				}
				cpvs.fetchedCollations = append(cpvs.fetchedCollations, *collation)
			}

		case <-cpvs.ctx.Done():
			if err := cpvs.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
		}
	}
}

func (CollatorProtocolValidatorSide) Name() parachaintypes.SubSystemName {
	return parachaintypes.CollationProtocol
}

func (cpvs CollatorProtocolValidatorSide) handleNetworkEvents(event network.NetworkEventInfo) {
	switch event.Event {
	case network.Connected:
		_, ok := cpvs.peerData[event.PeerID]
		if !ok {
			cpvs.peerData[event.PeerID] = PeerData{
				state: PeerStateInfo{
					PeerState: Connected,
					Instant:   time.Now(),
				},
			}
		}
	case network.Disconnected:
		delete(cpvs.peerData, event.PeerID)
	}
}

func (cpvs *CollatorProtocolValidatorSide) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) {
	// I might need to separate the collator protocol into two parts, one that deals with the
	// network and other that deals with other subsystems.
	// Make everythin less messing.
	// https://github.com/paritytech/polkadot-sdk/blob/1b5f4243d159fbb7cf7067241aca8a37f3dbf7ed/polkadot/node/network/bridge/src/rx/mod.rs#L798

	// this active leaves are handled in bridge in rust, read the code of bridge properly

	// TODO update cpvs.activeLeaves by adding new active leaves and removing deactivated ones

	// TODO: get the value for majorSyncing for syncing package
	// majorSyncing means you are 5 blocks behind the tip of the chain and thus more aggressively
	// download blocks etc to reach the tip of the chain faster.
	var majorSyncing bool

	liveHeads := cpvs.liveHeads[:]
	liveHeads = append(liveHeads, parachaintypes.ActivatedLeaf{
		Hash:   signal.Activated.Hash,
		Number: signal.Activated.Number,
	})

	newLiveHeads := []parachaintypes.ActivatedLeaf{}

	for _, head := range liveHeads {
		if slices.Contains(signal.Deactivated, head.Hash) {
			newLiveHeads = append(newLiveHeads, head)
		}
	}

	sort.Sort(SortableActivatedLeaves(newLiveHeads))
	// TODO: do I need to store these live heads or just pass them to update view?
	cpvs.liveHeads = newLiveHeads

	if !majorSyncing {
		// update our view
		err := cpvs.updateOurView(liveHeads)
		if err != nil {
			logger.Errorf("updating our view: %w", err)
		}
	}
}

func (cpvs *CollatorProtocolValidatorSide) updateOurView(liveHeads []parachaintypes.ActivatedLeaf) error {
	headHashes := []common.Hash{}
	for _, head := range cpvs.liveHeads {
		headHashes = append(headHashes, head.Hash)
	}
	newView := View{
		heads:           headHashes,
		finalizedNumber: cpvs.finalizedNumber,
	}

	if cpvs.localView == nil {
		*cpvs.localView = newView
		return nil
	}

	if cpvs.localView.checkHeadsEqual(newView) {
		// nothing to update
		return nil
	}

	*cpvs.localView = newView

	// TODO: send ViewUpdate to all the collation peers and validation peers (v1, v2, v3)
	// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/bridge/src/rx/mod.rs#L969-L1013

	// TODO: Create our view and send collation events to all subsystems about our view change
	// Just create the network bridge and do both of these tasks as part of those. That's the only way it makes sense.

	err := cpvs.handleOurViewChange(newView)
	if err != nil {
		return fmt.Errorf("handling our view change: %w", err)
	}
	return nil
}

func (cpvs *CollatorProtocolValidatorSide) handleOurViewChange(view View) error {
	// 1. Find out removed leaves (hashes) and newly added leaves
	// 2. Go over each new leaves,
	// - check if perspective parachain mode is enabled
	// - assign incoming
	// - insert active leaves and per relay parent
	activeLeaves := cpvs.activeLeaves

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
		mode := prospectiveParachainMode(leaf)

		perRelayParent := &PerRelayParent{
			prospectiveParachainMode: mode,
		}

		err := cpvs.assignIncoming(leaf, perRelayParent)
		if err != nil {
			return fmt.Errorf("assigning incoming: %w", err)
		}
		cpvs.activeLeaves[leaf] = mode
		cpvs.perRelayParent[leaf] = *perRelayParent

		//nolint:staticcheck
		if mode.IsEnabled {
			// TODO: Add it when we have async backing
			// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1303 //nolint
		}
	}

	// handle removed leaves
	for _, leaf := range removed {
		delete(cpvs.activeLeaves, leaf)

		mode := prospectiveParachainMode(leaf)
		pruned := []common.Hash{}
		if mode.IsEnabled {
			// TODO: Do this when we have async backing
			// https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1340 //nolint
		} else {
			pruned = append(pruned, leaf)
		}

		for _, prunedLeaf := range pruned {
			perRelayParent, ok := cpvs.perRelayParent[prunedLeaf]
			if ok {
				cpvs.removeOutgoing(perRelayParent)
				delete(cpvs.perRelayParent, prunedLeaf)
			}

			for fetchedCandidateStr := range cpvs.fetchedCandidates {
				fetchedCollation, err := fetchedCandidateFromString(fetchedCandidateStr)
				if err != nil {
					// this should never really happen
					return fmt.Errorf("getting fetched collation from string: %w", err)
				}

				if fetchedCollation.relayParent == prunedLeaf {
					delete(cpvs.fetchedCandidates, fetchedCandidateStr)
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

func (cpvs *CollatorProtocolValidatorSide) removeOutgoing(perRelayParent PerRelayParent) {
	if perRelayParent.assignment != nil {
		entry := cpvs.currentAssignments[*perRelayParent.assignment]
		entry--
		if entry == 0 {
			logger.Infof("unassigned from parachain with ID %d", *perRelayParent.assignment)
			delete(cpvs.currentAssignments, *perRelayParent.assignment)
			return
		}

		cpvs.currentAssignments[*perRelayParent.assignment] = entry
	}
}

func (cpvs *CollatorProtocolValidatorSide) assignIncoming(relayParent common.Hash, perRelayParent *PerRelayParent,
) error {
	// TODO: get this instance using relay parent
	instance, err := cpvs.BlockState.GetRuntime(relayParent)
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

	validator, validatorIndex := signingKeyAndIndex(validators, cpvs.Keystore)
	if validator == nil {
		// return with an error?
		return nil
	}

	groupIndex, ok := findValidatorGroup(validatorIndex, *validatorGroups)
	if !ok {
		logger.Trace("not a validator")
		return nil
	}

	coreIndexNow := validatorGroups.GroupRotationInfo.CoreForGroup(groupIndex, uint8(len(availabilityCores.Types)))
	coreNow, err := availabilityCores.Types[coreIndexNow.Index].Value()
	if err != nil {
		return fmt.Errorf("getting core now: %w", err)
	}

	var paraNow *parachaintypes.ParaID

	switch c := coreNow.(type) /*coreNow.Index()*/ {
	case parachaintypes.OccupiedCore:
		*paraNow = parachaintypes.ParaID(c.CandidateDescriptor.ParaID)
	case parachaintypes.ScheduledCore:
		*paraNow = c.ParaID
	case parachaintypes.Free:
		// Nothing to do in case of free

	}

	if paraNow != nil {
		entry := cpvs.currentAssignments[*paraNow]
		entry++
		cpvs.currentAssignments[*paraNow] = entry
		if entry == 1 {
			logger.Infof("got assigned to parachain with ID %d", *paraNow)
		}
	}

	perRelayParent.assignment = paraNow
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

func prospectiveParachainMode(relayParent common.Hash) parachaintypes.ProspectiveParachainsMode {
	// NOTE: We will return false until we have support for async backing
	return parachaintypes.ProspectiveParachainsMode{
		IsEnabled: false,
	}
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

func (cpvs *CollatorProtocolValidatorSide) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) {
	if cpvs.finalizedNumber >= signal.BlockNumber {
		// error
		return
	}
	cpvs.finalizedNumber = signal.BlockNumber
}

func (cpvs CollatorProtocolValidatorSide) Stop() {
	cpvs.cancel()
	cpvs.net.FreeNetworkEventsChannel(cpvs.networkEventInfoChan)
}

// requestCollation requests a collation from the network.
// This function will
// - check for duplicate requests
// - check if the requested collation is in our view
func (cpvs CollatorProtocolValidatorSide) requestCollation(relayParent common.Hash,
	paraID parachaintypes.ParaID, peerID peer.ID) (*parachaintypes.Collation, error) {

	// TODO: Make sure that the request can be done in MAX_UNSHARED_DOWNLOAD_TIME timeout
	_, ok := cpvs.perRelayParent[relayParent]
	if !ok {
		return nil, ErrOutOfView
	}

	// make collation fetching request
	collationFetchingRequest := CollationFetchingRequest{
		RelayParent: relayParent,
		ParaID:      paraID,
	}

	collationFetchingResponse := NewCollationFetchingResponse()
	err := cpvs.collationFetchingReqResProtocol.Do(peerID, collationFetchingRequest, &collationFetchingResponse)
	if err != nil {
		return nil, fmt.Errorf("collation fetching request failed: %w", err)
	}

	v, err := collationFetchingResponse.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value of collation fetching response: %w", err)
	}
	collationVDT, ok := v.(CollationVDT)
	if !ok {
		return nil, fmt.Errorf("collation fetching response value expected: CollationVDT, got: %T", v)
	}
	collation := parachaintypes.Collation(collationVDT)

	return &collation, nil
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

type PeerData struct {
	view  View
	state PeerStateInfo
}

func (peerData *PeerData) HasAdvertised(
	relayParent common.Hash,
	mayBeCandidateHash *parachaintypes.CandidateHash) bool {
	if peerData.state.PeerState == Connected {
		return false
	}

	candidates, ok := peerData.state.CollatingPeerState.advertisements[relayParent]
	if mayBeCandidateHash == nil {
		return ok
	}

	return slices.Contains(candidates, *mayBeCandidateHash)
}

func (peerData *PeerData) SetCollating(collatorID parachaintypes.CollatorID, paraID parachaintypes.ParaID) {
	peerData.state = PeerStateInfo{
		PeerState: Collating,
		CollatingPeerState: CollatingPeerState{
			CollatorID: collatorID,
			ParaID:     paraID,
		},
	}
}

func IsRelayParentInImplicitView(
	relayParent common.Hash,
	relayParentMode parachaintypes.ProspectiveParachainsMode,
	implicitView ImplicitView,
	activeLeaves map[common.Hash]parachaintypes.ProspectiveParachainsMode,
	paraID parachaintypes.ParaID,
) bool {
	if !relayParentMode.IsEnabled {
		_, ok := activeLeaves[relayParent]
		return ok
	}

	for hash, mode := range activeLeaves {
		knownAllowedRelayParent := implicitView.KnownAllowedRelayParentsUnder(hash, paraID)
		if mode.IsEnabled && knownAllowedRelayParent.String() == relayParent.String() {
			return true
		}
	}

	return false
}

// Note an advertisement by the collator. Returns `true` if the advertisement was imported
// successfully. Fails if the advertisement is duplicate, out of view, or the peer has not
// declared itself a collator.
func (peerData *PeerData) InsertAdvertisement(
	onRelayParent common.Hash,
	relayParentMode parachaintypes.ProspectiveParachainsMode,
	candidateHash *parachaintypes.CandidateHash,
	implicitView ImplicitView,
	activeLeaves map[common.Hash]parachaintypes.ProspectiveParachainsMode,
) (isAdvertisementInvalid bool, err error) {
	switch peerData.state.PeerState {
	case Connected:
		return false, ErrUndeclaredCollator
	case Collating:
		if !IsRelayParentInImplicitView(onRelayParent, relayParentMode, implicitView,
			activeLeaves, peerData.state.CollatingPeerState.ParaID) {
			return false, ErrOutOfView
		}

		if relayParentMode.IsEnabled {
			// relayParentMode.maxCandidateDepth
			candidates, ok := peerData.state.CollatingPeerState.advertisements[onRelayParent]
			if ok && slices.Contains[[]parachaintypes.CandidateHash](candidates, *candidateHash) {
				return false, ErrDuplicateAdvertisement
			}

			if len(candidates) > int(relayParentMode.MaxCandidateDepth) {
				return false, ErrPeerLimitReached
			}
			candidates = append(candidates, *candidateHash)
			peerData.state.CollatingPeerState.advertisements[onRelayParent] = candidates
		} else {
			_, ok := peerData.state.CollatingPeerState.advertisements[onRelayParent]
			if ok {
				return false, ErrDuplicateAdvertisement
			}
			peerData.state.CollatingPeerState.advertisements[onRelayParent] = []parachaintypes.CandidateHash{*candidateHash}
		}

		peerData.state.CollatingPeerState.lastActive = time.Now()
	}
	return true, nil
}

type PeerStateInfo struct {
	PeerState PeerState
	// instant at which peer got connected
	Instant            time.Time
	CollatingPeerState CollatingPeerState
}

type CollatingPeerState struct {
	CollatorID parachaintypes.CollatorID
	ParaID     parachaintypes.ParaID
	// collations advertised by peer per relay parent
	advertisements map[common.Hash][]parachaintypes.CandidateHash
	lastActive     time.Time
}

type PeerState uint

const (
	Connected PeerState = iota
	Collating
)

// The maximum amount of heads a peer is allowed to have in their view at any time.
// We use the same limit to compute the view sent to peers locally.
const MaxViewHeads uint8 = 5

// A succinct representation of a peer's view. This consists of a bounded amount of chain heads
// and the highest known finalized block number.
//
// Up to `N` (5?) chain heads.
type View struct {
	// a bounded amount of chain heads
	heads []common.Hash
	// the highest known finalized number
	finalizedNumber uint32
}

type SortableHeads []common.Hash

func (s SortableHeads) Len() int {
	return len(s)
}

func (s SortableHeads) Less(i, j int) bool {
	return s[i].String() > s[j].String()
}

func (s SortableHeads) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// checkHeadsEqual checks if the heads of the view are equal to the heads of the other view.
func (v View) checkHeadsEqual(other View) bool {
	if len(v.heads) != len(other.heads) {
		return false
	}

	localHeads := v.heads
	sort.Sort(SortableHeads(localHeads))
	otherHeads := other.heads
	sort.Sort(SortableHeads(otherHeads))

	return reflect.DeepEqual(localHeads, otherHeads)
}

func ConstructView(liveHeads map[common.Hash]struct{}, finalizedNumber uint32) View {
	heads := make([]common.Hash, 0, len(liveHeads))
	for head := range liveHeads {
		heads = append(heads, head)
	}

	if len(heads) >= 5 {
		heads = heads[:5]
	}

	return View{
		heads:           heads,
		finalizedNumber: finalizedNumber,
	}
}

// Network is the interface required by parachain service for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg network.NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID network.MessageType,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
	GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
		maxResponseSize uint64) *network.RequestResponseProtocol
	ReportPeer(change peerset.ReputationChange, p peer.ID)
	GetNetworkEventsChannel() chan *network.NetworkEventInfo
	FreeNetworkEventsChannel(ch chan *network.NetworkEventInfo)
}

type CollationEvent struct {
	CollatorId       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type CollatorProtocolValidatorSide struct {
	ctx    context.Context
	cancel context.CancelFunc

	BlockState *state.BlockState
	net        Network
	Keystore   keystore.Keystore

	SubSystemToOverseer  chan<- any
	OverseerToSubSystem  <-chan any
	networkEventInfoChan chan *network.NetworkEventInfo

	unfetchedCollation chan UnfetchedCollation

	collationFetchingReqResProtocol *network.RequestResponseProtocol

	fetchedCollations []parachaintypes.Collation
	// track all active collators and their data
	peerData map[peer.ID]PeerData

	// TODO: Tech Debt
	// In polkadot-sdk (rust) code, following fields are common between validation protocol and collator protocol.
	// They are kept in network bridge. Network bridge has common logic for both validation and collator protocol.
	// I have kept it here for ease, since we don't have network bridge. Make a decision on this. Create a network
	// bridge if that seems appropriate.
	// And move these fields and some common logic there.
	localView *View
	// validationPeers []peer.ID
	// collationPeers []peer.ID

	// Parachains we're currently assigned to. With async backing enabled
	// this includes assignments from the implicit view.
	currentAssignments map[parachaintypes.ParaID]uint

	// state tracked per relay parent
	perRelayParent map[common.Hash]PerRelayParent // map[relay parent]PerRelayParent

	// Advertisements that were accepted as valid by collator protocol but rejected by backing.
	//
	// It's only legal to fetch collations that are either built on top of the root
	// of some fragment tree or have a parent node which represents backed candidate.
	// Otherwise, a validator will keep such advertisement in the memory and re-trigger
	// requests to backing on new backed candidates and activations.
	BlockedAdvertisements map[string][]blockedAdvertisement

	// Leaves that do support asynchronous backing along with
	// implicit ancestry. Leaves from the implicit view are present in
	// `active_leaves`, the opposite doesn't hold true.
	//
	// Relay-chain blocks which don't support prospective parachains are
	// never included in the fragment trees of active leaves which do. In
	// particular, this means that if a given relay parent belongs to implicit
	// ancestry of some active leaf, then it does support prospective parachains.
	implicitView ImplicitView

	/// All active leaves observed by us, including both that do and do not
	/// support prospective parachains. This mapping works as a replacement for
	/// [`polkadot_node_network_protocol::View`] and can be dropped once the transition
	/// to asynchronous backing is done.
	activeLeaves map[common.Hash]parachaintypes.ProspectiveParachainsMode

	// Collations that we have successfully requested from peers and waiting
	// on validation.
	fetchedCandidates map[string]CollationEvent

	// heads are sorted in descending order by block number
	liveHeads []parachaintypes.ActivatedLeaf

	finalizedNumber uint32
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

func (cpvs CollatorProtocolValidatorSide) getPeerIDFromCollatorID(collatorID parachaintypes.CollatorID,
) (peer.ID, bool) {
	for peerID, peerData := range cpvs.peerData {
		if peerData.state.CollatingPeerState.CollatorID == collatorID {
			return peerID, true
		}
	}

	return "", false
}

func (cpvs CollatorProtocolValidatorSide) processMessage(msg any) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case collatorprotocolmessages.CollateOn:
		return fmt.Errorf("CollateOn %w", ErrNotExpectedOnValidatorSide)
	case collatorprotocolmessages.DistributeCollation:
		return fmt.Errorf("DistributeCollation %w", ErrNotExpectedOnValidatorSide)
	case collatorprotocolmessages.ReportCollator:
		peerID, ok := cpvs.getPeerIDFromCollatorID(parachaintypes.CollatorID(msg))
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)
	case collatorprotocolmessages.NetworkBridgeUpdate:
		// TODO: handle network message https://github.com/ChainSafe/gossamer/issues/3515
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1457 //nolint
	case collatorprotocolmessages.Seconded:
		statementV, err := msg.Stmt.Payload.Value()
		if err != nil {
			return fmt.Errorf("getting value of statement: %w", err)
		}
		if statementV.Index() != 1 {
			return fmt.Errorf("expected a seconded statement")
		}

		receipt, ok := statementV.(parachaintypes.Seconded)
		if !ok {
			return fmt.Errorf("statement value expected: Seconded, got: %T", statementV)
		}

		candidateReceipt := parachaintypes.CommittedCandidateReceipt(receipt)

		fetchedCollation, err := newFetchedCollationInfo(candidateReceipt.ToPlain())
		if err != nil {
			return fmt.Errorf("getting fetched collation info: %w", err)
		}

		// remove the candidate from the list of fetched candidates
		collationEvent, ok := cpvs.fetchedCandidates[fetchedCollation.String()]
		if !ok {
			logger.Error("collation has been seconded, but the relay parent is deactivated")
			return nil
		}

		delete(cpvs.fetchedCandidates, fetchedCollation.String())

		// notify good collation
		peerID, ok := cpvs.getPeerIDFromCollatorID(collationEvent.CollatorId)
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BenefitNotifyGoodValue,
			Reason: peerset.BenefitNotifyGoodReason,
		}, peerID)

		// notify candidate seconded
		_, ok = cpvs.peerData[peerID]
		if ok {
			collatorProtocolMessage := NewCollatorProtocolMessage()
			err = collatorProtocolMessage.Set(CollationSeconded{
				RelayParent: msg.Parent,
				Statement:   parachaintypes.UncheckedSignedFullStatement(msg.Stmt),
			})
			if err != nil {
				return fmt.Errorf("setting collation seconded: %w", err)
			}
			collationMessage := NewCollationProtocol()

			err = collationMessage.Set(collatorProtocolMessage)
			if err != nil {
				return fmt.Errorf("setting collation message: %w", err)
			}

			err = cpvs.net.SendMessage(peerID, &collationMessage)
			if err != nil {
				return fmt.Errorf("sending collation message: %w", err)
			}

			perRelayParent, ok := cpvs.perRelayParent[msg.Parent]
			if ok {
				perRelayParent.collations.status = Seconded
				perRelayParent.collations.secondedCount++
				cpvs.perRelayParent[msg.Parent] = perRelayParent
			}

			// TODO: Few more things for async backing, but we don't have async backing yet
			// https://github.com/paritytech/polkadot-sdk/blob/7035034710ecb9c6a786284e5f771364c520598d/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1531-L1532
		}
	case collatorprotocolmessages.Backed:
		backed := msg
		_, ok := cpvs.BlockedAdvertisements[backed.String()]
		if ok {
			delete(cpvs.BlockedAdvertisements, backed.String())

			err := cpvs.requestUnblockedCollations(backed)
			if err != nil {
				return fmt.Errorf("requesting unblocked collations: %w", err)
			}
		}
	case collatorprotocolmessages.Invalid:
		invalidOverseerMsg := msg

		fetchedCollation, err := newFetchedCollationInfo(msg.CandidateReceipt)
		if err != nil {
			return fmt.Errorf("getting fetched collation info: %w", err)
		}

		collationEvent, ok := cpvs.fetchedCandidates[fetchedCollation.String()]
		if !ok {
			return nil
		}

		if *collationEvent.PendingCollation.CommitmentHash == (invalidOverseerMsg.CandidateReceipt.CommitmentsHash) {
			delete(cpvs.fetchedCandidates, fetchedCollation.String())
		} else {
			logger.Error("reported invalid candidate for unknown `pending_candidate`")
			return nil
		}

		peerID, ok := cpvs.getPeerIDFromCollatorID(collationEvent.CollatorId)
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)

	case parachaintypes.ActiveLeavesUpdateSignal:
		cpvs.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		cpvs.ProcessBlockFinalizedSignal(msg)

	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil
}

// requestUnblockedCollations Checks whether any of the advertisements are unblocked and attempts to fetch them.
func (cpvs CollatorProtocolValidatorSide) requestUnblockedCollations(backed collatorprotocolmessages.Backed) error {
	for _, blockedAdvertisements := range cpvs.BlockedAdvertisements {
		newBlockedAdvertisements := []blockedAdvertisement{}

		for _, blockedAdvertisement := range blockedAdvertisements {
			isSecondingAllowed, err := cpvs.canSecond(
				backed.ParaID, blockedAdvertisement.candidateRelayParent, blockedAdvertisement.candidateHash, backed.ParaHead)
			if err != nil {
				return fmt.Errorf("checking if seconding is allowed: %w", err)
			}

			if !isSecondingAllowed {
				newBlockedAdvertisements = append(newBlockedAdvertisements, blockedAdvertisement)
				continue
			}

			perRelayParent, ok := cpvs.perRelayParent[blockedAdvertisement.candidateRelayParent]
			if !ok {
				return ErrRelayParentUnknown
			}

			err = cpvs.enqueueCollation(
				perRelayParent.collations,
				blockedAdvertisement.candidateRelayParent,
				backed.ParaID,
				blockedAdvertisement.peerID,
				blockedAdvertisement.collatorID,
				nil, // nil for now until we have prospective parachain
			)
			if err != nil {
				return fmt.Errorf("enqueueing collation: %w", err)
			}
		}

		if len(newBlockedAdvertisements) == 0 {
			return nil
		}
		cpvs.BlockedAdvertisements[backed.String()] = newBlockedAdvertisements

	}

	return nil
}

func newFetchedCollationInfo(candidateReceipt parachaintypes.CandidateReceipt) (*fetchedCollationInfo, error) {
	candidateHash, err := candidateReceipt.Hash()
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash: %w", err)
	}
	return &fetchedCollationInfo{
		paraID:      parachaintypes.ParaID(candidateReceipt.Descriptor.ParaID),
		relayParent: candidateReceipt.Descriptor.RelayParent,
		collatorID:  candidateReceipt.Descriptor.Collator,
		candidateHash: parachaintypes.CandidateHash{
			Value: candidateHash,
		},
	}, nil
}
