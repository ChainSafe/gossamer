// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
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
	ErrUnknownOverseerMessage               = errors.New("unknown overseer message type")
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
)

func (cpvs CollatorProtocolValidatorSide) Run(
	ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error {
	inactivityTicker := time.NewTicker(activityPoll)

	for {
		select {
		// TODO: polkadot-rust changes reputation in batches, so we do the same?
		case msg, ok := <-cpvs.OverseerToSubSystem:
			if !ok {
				return nil
			}

			err := cpvs.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
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

		}
	}
}

func (CollatorProtocolValidatorSide) Name() parachaintypes.SubSystemName {
	return parachaintypes.CollationProtocol
}

// requestCollation requests a collation from the network.
// This function will
// - check for duplicate requests
// - check if the requested collation is in our view
func (cpvs CollatorProtocolValidatorSide) requestCollation(relayParent common.Hash,
	paraID parachaintypes.ParaID, peerID peer.ID) (*parachaintypes.Collation, error) {

	// TODO: Make sure that the request can be done in MAX_UNSHARED_DOWNLOAD_TIME timeout
	if !slices.Contains[[]common.Hash](cpvs.ourView.heads, relayParent) {
		return nil, ErrCollationNotInView
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
	relayParentMode ProspectiveParachainsMode,
	implicitView ImplicitView,
	activeLeaves map[common.Hash]ProspectiveParachainsMode,
	paraID parachaintypes.ParaID,
) bool {
	if !relayParentMode.isEnabled {
		_, ok := activeLeaves[relayParent]
		return ok
	}

	for hash, mode := range activeLeaves {
		knownAllowedRelayParent := implicitView.KnownAllowedRelayParentsUnder(hash, paraID)
		if mode.isEnabled && knownAllowedRelayParent.String() == relayParent.String() {
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
	relayParentMode ProspectiveParachainsMode,
	candidateHash *parachaintypes.CandidateHash,
	implicitView ImplicitView,
	activeLeaves map[common.Hash]ProspectiveParachainsMode,
) (isAdvertisementInvalid bool, err error) {
	switch peerData.state.PeerState {
	case Connected:
		return false, ErrUndeclaredCollator
	case Collating:
		if !IsRelayParentInImplicitView(onRelayParent, relayParentMode, implicitView,
			activeLeaves, peerData.state.CollatingPeerState.ParaID) {
			return false, ErrOutOfView
		}

		if relayParentMode.isEnabled {
			// relayParentMode.maxCandidateDepth
			candidates, ok := peerData.state.CollatingPeerState.advertisements[onRelayParent]
			if ok && slices.Contains[[]parachaintypes.CandidateHash](candidates, *candidateHash) {
				return false, ErrDuplicateAdvertisement
			}

			if len(candidates) > int(relayParentMode.maxCandidateDepth) {
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

type View struct {
	// a bounded amount of chain heads
	heads []common.Hash
	// the highest known finalized number
	finalizedNumber uint32 //nolint
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
}

type CollationEvent struct {
	CollatorId       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type CollatorProtocolValidatorSide struct {
	net Network

	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any

	unfetchedCollation chan UnfetchedCollation

	collationFetchingReqResProtocol *network.RequestResponseProtocol

	fetchedCollations []parachaintypes.Collation
	// track all active collators and their data
	peerData map[peer.ID]PeerData

	ourView View

	// Parachains we're currently assigned to. With async backing enabled
	// this includes assignments from the implicit view.
	currentAssignments map[parachaintypes.ParaID]uint

	// state tracked per relay parent
	perRelayParent map[common.Hash]PerRelayParent // map[replay parent]PerRelayParent

	// Advertisements that were accepted as valid by collator protocol but rejected by backing.
	//
	// It's only legal to fetch collations that are either built on top of the root
	// of some fragment tree or have a parent node which represents backed candidate.
	// Otherwise, a validator will keep such advertisement in the memory and re-trigger
	// requests to backing on new backed candidates and activations.
	BlockedAdvertisements map[string][]BlockedAdvertisement

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
	activeLeaves map[common.Hash]ProspectiveParachainsMode

	fetchedCandidates map[string]CollationEvent
}

// Identifier of a fetched collation
type fetchedCollation struct {
	// Candidate's relay parent
	relayParent   common.Hash
	paraID        parachaintypes.ParaID
	candidateHash parachaintypes.CandidateHash
	// Id of the collator the collation was fetched from
	collatorID parachaintypes.CollatorID
}

func (f fetchedCollation) String() string {
	return fmt.Sprintf("relay parent: %s, para id: %d, candidate hash: %s, collator id: %+v",
		f.relayParent.String(), f.paraID, f.candidateHash.Value.String(), f.collatorID)
}

// Prospective parachains mode of a relay parent. Defined by
// the Runtime API version.
//
// Needed for the period of transition to asynchronous backing.
type ProspectiveParachainsMode struct {
	// if disabled, there are no prospective parachains. Runtime API does not have support for `async_backing_params`
	isEnabled bool

	// these values would be present only if `isEnabled` is true

	// The maximum number of para blocks between the para head in a relay parent and a new candidate.
	// Restricts nodes from building arbitrary long chains and spamming other validators.
	maxCandidateDepth uint
}

type PerRelayParent struct {
	prospectiveParachainMode ProspectiveParachainsMode
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
func (collations Collations) IsSecondedLimitReached(relayParentMode ProspectiveParachainsMode) bool {
	var secondedLimit uint
	if relayParentMode.isEnabled {
		secondedLimit = relayParentMode.maxCandidateDepth + 1
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

type CollateOn parachaintypes.ParaID

type DistributeCollation struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	PoV              parachaintypes.PoV
}

type ReportCollator parachaintypes.CollatorID

type NetworkBridgeUpdate struct {
	// TODO: not quite sure if we would need this or something similar to this
}

// SecondedOverseerMsg represents that the candidate we recommended to be seconded was validated successfully.
type SecondedOverseerMsg struct {
	Parent common.Hash
	Stmt   parachaintypes.StatementVDT
}

type Backed struct {
	ParaID parachaintypes.ParaID
	// Hash of the para head generated by candidate
	ParaHead common.Hash
}

func (b Backed) String() string {
	return fmt.Sprintf("para id: %d, para head: %s", b.ParaID, b.ParaHead.String())
}

// InvalidOverseerMsg represents an invalid candidata.
// We recommended a particular candidate to be seconded, but it was invalid; penalize the collator.
type InvalidOverseerMsg struct {
	Parent           common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
}

func (cpvs CollatorProtocolValidatorSide) processMessage(msg any) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case CollateOn:
		return fmt.Errorf("CollateOn %w", ErrNotExpectedOnValidatorSide)
	case DistributeCollation:
		return fmt.Errorf("DistributeCollation %w", ErrNotExpectedOnValidatorSide)
	case ReportCollator:
		peerID, ok := cpvs.getPeerIDFromCollatorID(parachaintypes.CollatorID(msg))
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)
	case NetworkBridgeUpdate:
		// TODO: handle network message https://github.com/ChainSafe/gossamer/issues/3515
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1457 //nolint
	case SecondedOverseerMsg:
		statementV, err := msg.Stmt.Value()
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

		candidateHashV, err := candidateReceipt.ToPlain().Hash()
		if err != nil {
			return fmt.Errorf("getting candidate hash from receipt: %w", err)
		}
		fetchedCollation := fetchedCollation{
			relayParent:   receipt.Descriptor.RelayParent,
			paraID:        parachaintypes.ParaID(receipt.Descriptor.ParaID),
			candidateHash: parachaintypes.CandidateHash{Value: candidateHashV},
			collatorID:    receipt.Descriptor.Collator,
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
				Statement: parachaintypes.UncheckedSignedFullStatement{
					Payload: msg.Stmt,
					// TODO:
					// ValidatorIndex: ,
					// Signature: ,
				},
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
	case Backed:
		// TODO: handle backed message https://github.com/ChainSafe/gossamer/issues/3517
		backed := msg
		_, ok := cpvs.BlockedAdvertisements[backed.String()]
		if ok {
			delete(cpvs.BlockedAdvertisements, backed.String())
			cpvs.requestUnblockedCollations(backed)
		}
	case InvalidOverseerMsg:
		invalidOverseerMsg := msg

		candidateHashV, err := msg.CandidateReceipt.Hash()
		if err != nil {
			return fmt.Errorf("getting candidate hash from receipt: %w", err)
		}
		fetchedCollation := fetchedCollation{
			relayParent:   msg.CandidateReceipt.Descriptor.RelayParent,
			paraID:        parachaintypes.ParaID(msg.CandidateReceipt.Descriptor.ParaID),
			candidateHash: parachaintypes.CandidateHash{Value: candidateHashV},
			collatorID:    msg.CandidateReceipt.Descriptor.Collator,
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
	default:
		return ErrUnknownOverseerMessage
	}

	return nil
}

// requestUnblockedCollations Checks whether any of the advertisements are unblocked and attempts to fetch them.
func (cpvs CollatorProtocolValidatorSide) requestUnblockedCollations(backed Backed) {

	for _, blockedAdvertisements := range cpvs.BlockedAdvertisements {
		for _, blockedAdvertisement := range blockedAdvertisements {
			isSecondingAllowed := cpvs.canSecond(
				backed.ParaID, blockedAdvertisement.candidateRelayParent, blockedAdvertisement.candidateHash, backed.ParaHead)

			if isSecondingAllowed {
				cpvs.enqueueCollation(
					blockedAdvertisement.candidateRelayParent,
					backed.ParaID,
					blockedAdvertisement.peerID,
					blockedAdvertisement.collatorID,
				)
			}
		}

	}
}
