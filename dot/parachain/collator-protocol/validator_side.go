package collatorprotocol

import (
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
	ACTIVITY_POLL              = 10 * time.Millisecond
	CHECK_COLLATIONS_POLL      = 50 * time.Millisecond
	MAX_UNSHARED_DOWNLOAD_TIME = 100 * time.Millisecond
)

var (
	ErrUnknownOverseerMessage     = errors.New("unknown overseer message type")
	ErrNotExpectedOnValidatorSide = errors.New("message is not expected on the validator side of the protocol")
	ErrCollationNotInView         = errors.New("collation is not in our view")
)

func (cpvs CollatorProtocolValidatorSide) runCollatorProtocol() {
	// Collator-to-Validator networking is more difficult than Validator-to-Validator networking
	// because the set of possible collators for any given para is unbounded, unlike the validator
	// set. Validator-to-Validator networking protocols can easily be implemented as gossip
	// because the data can be bounded, and validators can authenticate each other by their PeerIds
	// for the purposes of instantiating and accepting connections.

	// Since, at least at the level of the para abstraction, the collator-set for any given para
	// is unbounded, validators need to make sure that they are receiving connections from capable
	// and honest collators and that their bandwidth and time are not being wasted by attackers.
	// Communicating across this trust-boundary is the most difficult part of this subsystem.

	// TODO:
	// - handle requests from other subsystems to fetch a collation on a specific ParaId and relay-parent.
	// These requests are made with the request response protocol CollationFetchingRequest request. While doing
	// this we should also check if we already have gather collations on that para id and relay parent.
	// So, we will need to store all the collations.
	// - request only one collation at a time per relay parent. This reduces the bandwidth requirements and
	// as we can second only one candidate per relay parent, the others are probably not required anyway.
	// If the request times out, we need to note the collator as being unreliable and reduce its priority
	// relative to other collators.
	// - Other subsystems will report the collator using say `CollatorProtocolMessage::ReportCollator`.
	// We apply a cost to the PeerId associated with the collator and potentially disconnect or blacklist it.
	// If the collation is seconded, we notify the collator and apply a benefit to the PeerId associated with the collator.

	inactivityTicker := time.NewTicker(ACTIVITY_POLL)
	checkCollationTicker := time.NewTicker(CHECK_COLLATIONS_POLL)
	// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1257
	for {
		select {

		// add a case to check if there is a message from overseer
		// polkadot-rust changes reputation in batches, so we do the same?
		case <-cpvs.overseerChan:

		case <-inactivityTicker.C:
			// disconnect inactive peers
			// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1301
		case <-checkCollationTicker.C:
			// process pending collation requests. So, we need to store pending collation requests.
			// this is a lot more detailed in polkadot. they are maintaining a list of collations per relay parent,
			// processing for responses of previously made requests, etc. obviously, all that is because there
			// because there are many such collation request to process.
			// I can't directly convert rust futures to go channels, right now. I might just do it in the stupid manner\
			// add complexity later.

			// process existing list of collation fetching requests and apply reputation changes based on that

			// If our request doesn't get processed in a fixed amount of time we should try getting collations from a different collator.

			// TODO: Look into why collations are per relay parent?
			for _, unfetchedCollation := range cpvs.unfetchedCollations {

				// TODO: If we can't get the collation from given collator within MAX_UNSHARED_DOWNLOAD_TIME,
				// we will start another one from the next collator.

				// check if this peer id has advertised this relay parent
				peerData := cpvs.peerData[unfetchedCollation.PendingCollation.PeerID]
				if peerData.HasAdvertisedRelayParent(unfetchedCollation.PendingCollation.RelayParent) {
					// if so request collation from this peer id
					collation, err := cpvs.requestCollation(unfetchedCollation.PendingCollation.RelayParent, unfetchedCollation.PendingCollation.ParaID, unfetchedCollation.PendingCollation.PeerID)
					if err != nil {
						logger.Errorf("fetching collation: %w", err)
						// TODO: What to do with this failed request?
					}
					cpvs.fetchedCollations = append(cpvs.fetchedCollations, *collation)
				}
			}
		}
	}
}

// requestCollation requests a collation from the network.
// This function will
// - check for duplicate requests
// - check if the requested collation is in our view
func (cpvs CollatorProtocolValidatorSide) requestCollation(relayParent common.Hash,
	paraID parachaintypes.ParaID, peerID peer.ID) (*parachaintypes.Collation, error) {
	if !slices.Contains[common.Hash](cpvs.ourView.heads, relayParent) {
		return nil, ErrCollationNotInView
	}

	// make collation fetching request
	collationFetchingRequest := CollationFetchingRequest{
		RelayParent: relayParent,
		ParaID:      paraID,
	}

	collationFetchingResponse := NewCollationFetchingResponse()
	// TODO: find out the approprate value of collationFetchingResponseTimeout
	// collationFetchingResponseTimeout will be part of collationFetchingReqResProtocol

	// collationFetchingResponseTimeout := 5 * time.Second
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
		return nil, fmt.Errorf("collation fetching response value is not of type CollationVDT")
	}
	collation := parachaintypes.Collation(collationVDT)

	return &collation, nil
}

type UnfetchedCollation struct {
	CollatorID       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type PendingCollation struct {
	RelayParent    common.Hash
	ParaID         parachaintypes.ParaID
	PeerID         peer.ID
	CommitmentHash *common.Hash
}

type PeerData struct {
	view  View
	state PeerStateInfo
}

func (peerData PeerData) HasAdvertisedRelayParent(relayParent common.Hash) bool {
	if peerData.state.PeerState == Connected {
		return false
	}

	for _, head := range peerData.view.heads {
		if head == relayParent {
			return true
		}
	}

	return false
}

func (peerData PeerData) InsertAdvertisement() error {

	// TODO: Make this method real
	return nil
}

type PeerStateInfo struct {
	PeerState PeerState
	// instant at which peer got connected
	Instant            time.Time
	CollatingPeerState CollatingPeerState
}

type CollatingPeerState struct {
	CollatorID     parachaintypes.CollatorID
	ParaID         parachaintypes.ParaID
	advertisements []common.Hash
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
	finalizedNumber uint32
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
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

type CollationEvent struct {
	CollatorId       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

//	struct PerRelayParent {
//		prospective_parachains_mode: ProspectiveParachainsMode,
//		assignment: GroupAssignments,
//		collations: Collations,
//	}
type ProspectiveParachainsMode struct {
	// if disabled, there are no prospective parachains. Runtime API does not have support for `async_backing_params`
	isEnabled bool

	// these values would be present only if `isEnabled` is true

	// The maximum number of para blocks between the para head in a relay parent and a new candidate.
	// Restricts nodes from building arbitrary long chains and spamming other validators.
	maxCandidateDepth uint

	// How many ancestors of a relay parent are allowed to build candidates on top of.
	allowedAncestryLen uint
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

type CollatorProtocolValidatorSide struct {
	net Network

	collationFetchingReqResProtocol *network.RequestResponseProtocol

	// TODO: Check if there are better ways to store this.
	// Check if this should be stored in storage instead
	collations map[uint32]map[common.Hash]parachaintypes.Collation
	// TODO: This will almost certainly change, array is not efficient to search into.
	// See if this should be stored in storage.
	adverisements []Advertisement

	pendingCollationFetchingRequests []CollationFetchingRequest

	unfetchedCollations []UnfetchedCollation

	fetchedCollations []parachaintypes.Collation
	// track all active collators and their data
	peerData map[peer.ID]PeerData

	ourView View

	overseerChan chan interface{}

	// Keep track of all pending candidate collations
	pendingCandidates map[common.Hash]CollationEvent

	// Parachains we're currently assigned to. With async backing enabled
	// this includes assignments from the implicit view.
	currentAssignments map[parachaintypes.ParaID]uint

	// state tracked per relay parent
	perRelayParent map[common.Hash]PerRelayParent // map[replay parent]PerRelayParent

	// TODO: In rust this is a map, let's see if we can get away with a map
	// blocked_advertisements: HashMap<(ParaId, Hash), Vec<BlockedAdvertisement>>,
	BlockedAdvertisements []BlockedAdvertisement
	// TODO: Check if the same peerset
	// peerset

	// 	/// All state relevant for the validator side of the protocol lives here.
	// #[derive(Default)]
	// struct State {
	// 	/// Our own view.
	// 	view: OurView,

	// 	/// Active paras based on our view. We only accept collators from these paras.
	// 	active_paras: ActiveParas,

	// 	/// Track all active collators and their data.
	// 	peer_data: HashMap<PeerId, PeerData>,

	// 	/// The collations we have requested by relay parent and para id.
	// 	///
	// 	/// For each relay parent and para id we may be connected to a number
	// 	/// of collators each of those may have advertised a different collation.
	// 	/// So we group such cases here.
	// 	requested_collations: HashMap<PendingCollation, PerRequest>,

	// 	/// Metrics.
	// 	metrics: Metrics,

	// 	/// Span per relay parent.
	// 	span_per_relay_parent: HashMap<Hash, PerLeafSpan>,

	// 	/// Keep track of all fetch collation requests
	// 	collation_fetches: FuturesUnordered<BoxFuture<'static, PendingCollationFetch>>,

	// 	/// When a timer in this `FuturesUnordered` triggers, we should dequeue the next request
	// 	/// attempt in the corresponding `collations_per_relay_parent`.
	// 	///
	// 	/// A triggering timer means that the fetching took too long for our taste and we should give
	// 	/// another collator the chance to be faster (dequeue next fetch request as well).
	// 	collation_fetch_timeouts: FuturesUnordered<BoxFuture<'static, (CollatorId, Hash)>>,

	// 	/// Information about the collations per relay parent.
	// 	collations_per_relay_parent: HashMap<Hash, CollationsPerRelayParent>,

	// 	/// Keep track of all pending candidate collations
	// 	pending_candidates: HashMap<Hash, CollationEvent>,

	// 	/// Aggregated reputation change
	// 	reputation: ReputationAggregator,
	// }

}

type SecondedOverseerMsg struct {
	Parent common.Hash
	Stmt   parachaintypes.Statement
}

type InvalidOverseeMsg struct {
	Parent           common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
}

func (cpvs CollatorProtocolValidatorSide) getPeerIDFromCollatorID(collatorID parachaintypes.CollatorID) (peer.ID, bool) {
	for peerID, peerData := range cpvs.peerData {
		if peerData.state.PeerState == Collating && peerData.state.CollatingPeerState.CollatorID == collatorID {
			return peerID, true
		}
	}

	return "", false
}

func (cpvs CollatorProtocolValidatorSide) processMessage(msgType MessageType, msg interface{}) error {
	// run this function as a goroutine, ideally

	switch msgType {
	case CollateOnMsg, DistributeCollationMsg:
		return fmt.Errorf("%s %w", msgType, ErrNotExpectedOnValidatorSide)
	case ReportCollatorMsg:
		// TODO: report collator https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1168
	case NetworkBridgeUpdateMsg:
		// TODO: handle network message https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1171
	case SecondedMsg:
		// TODO: https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1180

		secondedOverseerMsg, ok := msg.(SecondedOverseerMsg)
		if !ok {
			// error
		}

		collationEvent, ok := cpvs.pendingCandidates[secondedOverseerMsg.Parent]
		if !ok {
			// error
		}

		// note good collation, which is basically modify reputation
		peerID, ok := cpvs.getPeerIDFromCollatorID(collationEvent.CollatorId)
		if !ok {
			// error
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BenefitNotifyGoodValue,
			Reason: peerset.BenefitNotifyGoodReason,
		}, peerID)

		// notify collation seconded, send a CollationSeconded message on the network on collation protocol

		vdt_parent := NewCollationProtocol()
		vdt_child := NewCollatorProtocolMessage()

		err := vdt_child.Set(CollationSeconded{
			RelayParent: secondedOverseerMsg.Parent,
			Statement: parachaintypes.UncheckedSignedFullStatement{
				Payload: secondedOverseerMsg.Stmt,
				// TODO: How do I add validator index and validator signature here?
			},
		})
		if err != nil {
			return fmt.Errorf("setting collation seconded: %w", err)
		}

		err = vdt_parent.Set(vdt_child)
		if err != nil {
			return fmt.Errorf("setting collation seconded: %w", err)
		}

		// this is deliberate.
		// this is the second time we are modifying the reputation of the peer, in this block.
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BenefitNotifyGoodValue,
			Reason: peerset.BenefitNotifyGoodReason,
		}, peerID)

		// TODO: Change collation status to Seconded
		// if let Some(collations) = state.collations_per_relay_parent.get_mut(&parent) {
		// 	collations.status = CollationStatus::Seconded;
		// }

	case InvalidMsg:
		// TODO: corresponding rust code is a little confusing. Understand it better and confirm
		// that you are doing the right thing.
		// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1211
		invalidOverseerMsg, ok := msg.(InvalidOverseeMsg)
		if !ok {
			// error
		}

		collationEvent, ok := cpvs.pendingCandidates[invalidOverseerMsg.Parent]
		if !ok {
			return nil
		}

		if *collationEvent.PendingCollation.CommitmentHash == (invalidOverseerMsg.CandidateReceipt.CommitmentsHash) {
			delete(cpvs.pendingCandidates, invalidOverseerMsg.Parent)
		} else {
			logger.Error("reported invalid candidate for unknown `pending_candidate`")
			return nil
		}

		peerID, ok := cpvs.getPeerIDFromCollatorID(collationEvent.CollatorId)
		if !ok {
			// error
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)

		// TODO: collatipn was invalid, let's penalise it

		// 1. polkadot-rust keeps track of pending candidates, but I will have to spend more time
		// to understand what it does and why
		// 2. report collator
		// 3. dequeue collation and fetch next
	default:
		return ErrUnknownOverseerMessage
	}

	return nil
}

type MessageType int

const (
	CollateOnMsg MessageType = iota
	DistributeCollationMsg
	ReportCollatorMsg
	NetworkBridgeUpdateMsg // TODO: not quite sure if we would need this or something similar to this
	SecondedMsg
	InvalidMsg
)

func (mt MessageType) String() string {
	switch mt {
	case CollateOnMsg:
		return "CollateOn"
	case DistributeCollationMsg:
		return "DistributeCollation"
	case ReportCollatorMsg:
		return "ReportCollator"
	case SecondedMsg:
		return "Seconded"
	case InvalidMsg:
		return "Invalid"
	default:
		panic(ErrUnknownOverseerMessage)
	}
}

type Advertisement struct {
	ParaID      uint32
	Collator    parachaintypes.CollatorID
	RelayParent common.Hash
}
