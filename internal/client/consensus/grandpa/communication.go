package grandpa

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	networkSync "github.com/ChainSafe/gossamer/internal/client/network/sync"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// How often to rebroadcast neighbor packets, in cases where no new packets are created.
const neighborRebroadcastPeriod = 2 * 60 * time.Second

// cost scalars for reporting peers.
var (
	pastRejection    = network.NewReputationChange(-50, "Grandpa: Past message")
	badSignature     = network.NewReputationChange(-100, "Grandpa: Bad signature")
	malformedCatchUp = network.NewReputationChange(-1000, "Grandpa: Malformed catch-up")
	malformedCommit  = network.NewReputationChange(-1000, "Grandpa: Malformed commit")
	futureMessage    = network.NewReputationChange(-500, "Grandpa: Future message")
	unknownVoter     = network.NewReputationChange(-150, "Grandpa: Unknown voter")

	invalidViewChange              = network.NewReputationChange(-500, "Grandpa:Invalid view change")
	duplicateNeighborMessage       = network.NewReputationChange(-500, "Grandpa: Duplicate neighbor message without grace period")
	perUndecodeableByte      int32 = -5
	perSignatureChecked      int32 = -25
	perBlockLoaded           int32 = -10
	invalidCatchUp                 = network.NewReputationChange(-5000, "Grandpa: Invalid catch-up")
	invalidCommit                  = network.NewReputationChange(-5000, "Grandpa: Invalid commit")
	outOfScopeMessage              = network.NewReputationChange(-500, "Grandpa: Out-of-scope message")
	catchUpRequestTimeout          = network.NewReputationChange(-200, "Grandpa: Catch-up request timeout")

	// cost of answering a catch up request
	catchUpReply            = network.NewReputationChange(-200, "Grandpa: Catch-up reply")
	honestOutOfScopeCatchUp = network.NewReputationChange(-200, "Grandpa: Out-of-scope catch-up")
)

// benefit scalars for reporting peers.
var (
	neighborMessage             = network.NewReputationChange(100, "Grandpa: Neighbor message")
	roundMessage                = network.NewReputationChange(100, "Grandpa: Round message")
	basicValidatedCatchUp       = network.NewReputationChange(200, "Grandpa: Catch-up message")
	basicValidatedCommit        = network.NewReputationChange(100, "Grandpa: Commit")
	perEquivocation       int32 = 10
)

// / A type that ties together our local authority id and a keystore where it is
// / available for signing.
// pub struct LocalIdKeystore((AuthorityId, KeystorePtr));
type localIDKeystore struct {
	primitives.AuthorityID
	keystore.KeyStore
}

// / Returns a reference to our local authority id.
func (lk *localIDKeystore) localID() primitives.AuthorityID {
	return lk.AuthorityID
}

// / Returns a reference to the keystore.
//
//		fn keystore(&self) -> KeystorePtr {
//			(self.0).1.clone()
//		}
//	}
// func (lk *localIDKeystore) KeyStore() *keystore.KeyStore {
// 	return lk.KeyStore
// }

// impl From<(AuthorityId, KeystorePtr)> for LocalIdKeystore {
// 	fn from(inner: (AuthorityId, KeystorePtr)) -> LocalIdKeystore {
// 		LocalIdKeystore(inner)
// 	}
// }

// / A handle to the network.
// /
// / Something that provides the capabilities needed for the `gossip_network::Network` trait.
type Network interface {
	gossip.Network
}

// / A handle to syncing-related services.
// /
// / Something that provides the ability to set a fork sync request for a particular block.
type Syncing[H, N any] interface {
	service.NetworkSyncForkRequest[H, N]
	service.NetworkBlock[H, N]
	networkSync.SyncEventStream
}

// / Create a unique topic for a round and set-id combo.
func roundTopic[H runtime.Hash, Hasher runtime.Hasher[H]](round Round, setID SetID) H {
	hasher := (*new(Hasher))
	return hasher.Hash([]byte(fmt.Sprintf("%d-%d", setID, round)))
}

// / Create a unique topic for global messages on a set ID.
func globalTopic[H runtime.Hash, Hasher runtime.Hasher[H]](setID SetID) H {
	hasher := (*new(Hasher))
	return hasher.Hash([]byte(fmt.Sprintf("%d-GLOBAL", setID)))
}

// / Bridge between the underlying network service, gossiping consensus messages and Grandpa
type networkBridge[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	service         Network
	sync            Syncing[H, N]
	gossipEngine    gossip.GossipEngine[H, N, Hasher]
	gossipEngineMtx sync.Mutex
	validator       *gossipValidator[H, N, Hasher]

	/// Sender side of the neighbor packet channel.
	///
	/// Packets sent into this channel are processed by the `NeighborPacketWorker` and passed on to
	/// the underlying `GossipEngine`.
	// neighbor_sender: periodic::NeighborPacketSender<B>,
	neighborSender neighbourPacketSender[N]

	/// `NeighborPacketWorker` processing packets sent through the `NeighborPacketSender`.
	// `NetworkBridge` is required to be cloneable, thus one needs to be able to clone its
	// children, thus one has to wrap `neighbor_packet_worker` with an `Arc` `Mutex`.
	// neighbor_packet_worker: Arc<Mutex<periodic::NeighborPacketWorker<B>>>,
	neighborPacketWorker    neighborPacketWorker[N]
	neighborPacketWorkerMtx sync.Mutex

	/// Receiver side of the peer report stream populated by the gossip validator, forwarded to the
	/// gossip engine.
	// `NetworkBridge` is required to be cloneable, thus one needs to be able to clone its
	// children, thus one has to wrap gossip_validator_report_stream with an `Arc` `Mutex`. Given
	// that it is just an `UnboundedReceiver`, one could also switch to a
	// multi-producer-*multi*-consumer channel implementation.
	// gossip_validator_report_stream: Arc<Mutex<TracingUnboundedReceiver<PeerReport>>>,
	gossipValidatorReportStream    chan peerReport
	gossipValidatorReportStreamMtx sync.Mutex

	// telemetry: Option<TelemetryHandle>,
	// TODO: telemetry
}

func newNetworkBridge[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	service Network,
	sync Syncing[H, N],
	notificationService service.NotificationService,
	config Config,
	setState *SharedVoterSetState[H, N],
	// TODO: metrics, telemetry
) *networkBridge[H, N, Hasher] {
	protocol := config.ProtocolName
	validator, reportStream := newGossipValidator[H, N, Hasher](config, setState)

	gossipEngine := gossip.NewGossipEngine[H, N, Hasher](service, sync, notificationService, protocol, validator)

	{
		// register all previous votes with the gossip service so that they're
		// available to peers potentially stuck on a previous round.
		setState.innerMtx.RLock()
		completed := setState.inner.completedRounds()
		setState.innerMtx.RUnlock()
		setID, voters := completed.setInfo()
		validator.noteSet(SetID(setID), voters, func(to []peerid.PeerID, msg neighborPacket[N]) {})

		for _, round := range completed.iter() {
			topic := roundTopic[H, Hasher](Round(round.Number), SetID(setID))

			// we need to note the round with the gossip validator otherwise
			// messages will be ignored.
			validator.noteRound(Round(round.Number), func(to []peerid.PeerID, msg neighborPacket[N]) {})

			for _, signed := range round.Votes {
				var gossipMessage gossipMessageVDT[H, N]
				gossipMessage.inner = gossipMessageVote[H, N](voteMessage[H, N]{
					Message: signed,
					Round:   Round(round.Number),
					SetID:   SetID(setID),
				})
				gossipEngine.RegisterGossipMessage(topic, scale.MustMarshal(gossipMessage))
			}

			logger.Tracef("Registered %d messages for topic %v (round: %d, setID: %d)", len(round.Votes), topic, round.Number, setID)
		}
	}

	neighborPacketWorker, neighborPacketSender := newNeighborPacketWorker[N](neighborRebroadcastPeriod)
	nb := networkBridge[H, N, Hasher]{
		service:                     service,
		sync:                        sync,
		gossipEngine:                gossipEngine,
		validator:                   validator,
		neighborSender:              neighborPacketSender,
		neighborPacketWorker:        neighborPacketWorker,
		gossipValidatorReportStream: reportStream,
	}

	go func() {
		for {
			err := nb.poll()
			if err != nil {
				panic(err)
			}
		}
	}()

	return &nb
}

// / Note the beginning of a new round to the `GossipValidator`.
func (nb *networkBridge[H, N, Hasher]) noteRound(round Round, setID SetID, voters *grandpa.VoterSet[primitives.AuthorityID]) {
	authorities := make([]primitives.AuthorityID, voters.Len())
	for i, ivi := range voters.Voters() {
		authorities[i] = ivi.ID
	}
	nb.validator.noteSet(
		setID,
		authorities,
		func(to []peerid.PeerID, msg neighborPacket[N]) {
			select {
			case nb.neighborSender <- peerIDsNeighborPacket[N]{PeerIDs: to, NeighborPacket: msg}:
			default:
				panic("wtf?")
			}
		},
	)

	nb.validator.noteRound(
		round,
		func(to []peerid.PeerID, msg neighborPacket[N]) {
			select {
			case nb.neighborSender <- peerIDsNeighborPacket[N]{PeerIDs: to, NeighborPacket: msg}:
			default:
				panic("wtf?")
			}
		},
	)
}

// / Get a stream of signature-checked round messages from the network as well as a sink for
// / round messages to the network all within the current set.
func (nb *networkBridge[H, N, Hasher]) roundCommunication(
	keystore *localIDKeystore,
	round Round,
	setID SetID,
	voters *grandpa.VoterSet[primitives.AuthorityID],
	hasVoted hasVoted[H, N],
) (chan primitives.SignedMessage[H, N], outgoingMessages[H, N, Hasher]) {
	nb.noteRound(round, setID, voters)

	var ks *localIDKeystore
	if keystore != nil {
		id := ks.localID()
		if voters.Contains(id) {
			ks = keystore
		}
	}

	topic := roundTopic[H, Hasher](round, setID)

	nb.gossipEngineMtx.Lock()
	messages := nb.gossipEngine.MessagesFor(topic)
	nb.gossipEngineMtx.Unlock()

	incoming := make(chan primitives.SignedMessage[H, N])
	go func() {
		defer close(incoming)
		for notification := range messages {
			var decoded gossipMessageVDT[H, N]
			err := scale.Unmarshal(notification.Message, &decoded)
			if err != nil {
				logger.Debugf("Skipping malformed message %v: %v", notification, err)
				continue
			}
			message, err := decoded.Value()
			if err != nil {
				logger.Debugf("Skipping malformed message %v: %v", notification, err)
				continue
			}
			switch message := message.(type) {
			case gossipMessageVote[H, N]:
				// check signature.
				if !voters.Contains(message.Message.ID) {
					logger.Debugf("Skipping message from unknown voter %v", message.Message.ID)
					continue
				}
				signedMessage := message.Message
				incoming <- signedMessage
			default:
				logger.Debugf("Skipping unknown message type")
				continue
			}
		}
	}()

	sender := make(chan primitives.SignedMessage[H, N])
	outgoing := outgoingMessages[H, N, Hasher]{
		keystore: ks,
		round:    round,
		setID:    setID,
		network:  &nb.gossipEngine,
		sender:   sender,
		hasVoted: hasVoted,
	}

	// Combine incoming votes from external GRANDPA nodes with outgoing
	// votes from our own GRANDPA voter to have a single
	// vote-import-pipeline.
	combinedIncoming := make(chan primitives.SignedMessage[H, N])
	go func() {
		defer close(combinedIncoming)
		var incomingOk, senderOk bool
		var msg primitives.SignedMessage[H, N]
		for {
			select {
			case msg, incomingOk = <-incoming:
				if !incomingOk {
					continue
				}
				combinedIncoming <- msg
			case msg, senderOk = <-sender:
				if !senderOk {
					continue
				}
				combinedIncoming <- msg
			}
			if !incomingOk && !senderOk {
				break
			}
		}
	}()

	// (incoming, outgoing)
	return combinedIncoming, outgoing
}

// / Set up the global communication streams.
func (nb *networkBridge[H, N, Hasher]) globalCommunication(
	setID SetID,
	voters *grandpa.VoterSet[primitives.AuthorityID],
	isVoter bool,
) (chan communicationIn[H, N], commitsOut[H, N, Hasher]) {
	authorities := make([]primitives.AuthorityID, voters.Len())
	for i, ivi := range voters.Voters() {
		authorities[i] = ivi.ID
	}
	nb.validator.noteSet(setID, authorities, func(to []peerid.PeerID, msg neighborPacket[N]) {
		select {
		case nb.neighborSender <- peerIDsNeighborPacket[N]{PeerIDs: to, NeighborPacket: msg}:
		default:
			panic("wtf?")
		}
	})

	topic := globalTopic[H, Hasher](setID)
	incoming := incomingGlobal(&nb.gossipEngine, topic, voters, nb.validator, nb.neighborSender)

	outgoing := newCommitsOut(&nb.gossipEngine, setID, isVoter, nb.validator, nb.neighborSender)

	// 	let outgoing = outgoing.with(|out| {
	// 		let voter::CommunicationOut::Commit(round, commit) = out;
	// 		future::ok((round, commit))
	// 	});

	return incoming, outgoing
}

// / Notifies the sync service to try and sync the given block from the given
// / peers.
// /
// / If the given vector of peers is empty then the underlying implementation
// / should make a best effort to fetch the block from any peers it is
// / connected to (NOTE: this assumption will change in the future #3629).
func (nb *networkBridge[H, N, Hasher]) SetSyncForkRequest(
	peers []peerid.PeerID,
	hash H,
	number N,
) {
	nb.sync.SetSyncForkRequest(peers, hash, number)
}

// impl<B: BlockT, N: Network<B>, S: Syncing<B>> Future for NetworkBridge<B, N, S> {
// 	type Output = Result<(), Error>;

// fn poll(self: Pin<&mut Self>, cx: &mut Context) -> Poll<Self::Output> {
func (nb *networkBridge[H, N, Hasher]) poll() error {
	neighborPacketStream := nb.neighborPacketWorker.Stream()
	for {
		select {
		case message, ok := <-neighborPacketStream:
			if !ok {
				return fmt.Errorf("Neighbor packet worker stream closed.")
			}
			var gossipMessage gossipMessageVDT[H, N]
			gossipMessage.inner = message.GossipMessage
			nb.gossipEngineMtx.Lock()
			nb.gossipEngine.SendMessage(message.PeerIDs, scale.MustMarshal(gossipMessage))
			nb.gossipEngineMtx.Unlock()
		case report, ok := <-nb.gossipValidatorReportStream:
			if !ok {
				return fmt.Errorf("Gossip validator report stream closed.")
			}
			nb.gossipEngineMtx.Lock()
			nb.gossipEngine.Report(report.who, report.costBenefit)
			nb.gossipEngineMtx.Unlock()
		}
	}
}

// fn incoming_global<B: BlockT>(
func incomingGlobal[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	gossipEngine *gossip.GossipEngine[H, N, Hasher],
	topic H,
	voters *grandpa.VoterSet[primitives.AuthorityID],
	validator *gossipValidator[H, N, Hasher],
	neighborSender neighbourPacketSender[N],
	// TODO: telemetry
) chan communicationIn[H, N] {
	var processCommit = func(
		msg fullCommitMessage[H, N],
		notification *gossip.TopicNotification,
		gossipEngine *gossip.GossipEngine[H, N, Hasher],
		gossipValidator *gossipValidator[H, N, Hasher],
		voters *grandpa.VoterSet[primitives.AuthorityID],
	) communicationIn[H, N] {

		cost := checkCompactCommit[H, N, Hasher](
			msg.Message,
			voters,
			msg.Round,
			msg.SetID,
		)
		if cost != nil {
			if notification.Sender != nil {
				gossipEngine.Report(*notification.Sender, *cost)
			}
			return nil
		}

		round := msg.Round
		setID := msg.SetID
		commit := msg.Message
		finalizedNumber := commit.TargetNumber
		cb := func(outcome grandpa.CommitProcessingOutcome) {
			switch outcome.(type) {
			case grandpa.CommitProcessingOutcomeGood:
				// if it checks out, gossip it. not accounting for
				// any discrepancy between the actual ghost and the claimed
				// finalized number.
				gossipValidator.noteCommitFinalized(
					round,
					setID,
					finalizedNumber,
					func(to []peerid.PeerID, msg neighborPacket[N]) {
						neighborSender <- peerIDsNeighborPacket[N]{PeerIDs: to, NeighborPacket: msg}
					},
				)
				gossipEngine.GossipMessage(topic, notification.Message, false)
			case grandpa.CommitProcessingOutcomeBad:
				// report peer and do not gossip.
				if notification.Sender != nil {
					gossipEngine.Report(*notification.Sender, invalidCommit)
				}
			default:
				panic("unreachable")
			}
		}

		return grandpa.CommunicationInCommit[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
			Number:        uint64(round),
			CompactCommit: grandpa.CompactCommit[H, N, primitives.AuthoritySignature, primitives.AuthorityID](commit),
			Callback:      cb,
		}
	}

	var processCatchUp = func(
		msg fullCatchUpMessage[H, N],
		notification *gossip.TopicNotification,
		gossipEngine *gossip.GossipEngine[H, N, Hasher],
		gossipValidator *gossipValidator[H, N, Hasher],
		voters *grandpa.VoterSet[primitives.AuthorityID],
	) communicationIn[H, N] {
		cost := checkCatchUp[H, N, Hasher](msg.Message, voters, msg.SetID)
		if cost != nil {
			if notification.Sender != nil {
				gossipEngine.Report(*notification.Sender, *cost)
			}
			return nil
		}

		cb := func(outcome grandpa.CatchUpProcessingOutcome) {
			switch outcome.(type) {
			case grandpa.CatchUpProcessingOutcomeBad:
				// report peer
				if notification.Sender != nil {
					gossipEngine.Report(*notification.Sender, invalidCatchUp)
					notification.Sender = nil
				}
			default:
			}
			gossipValidator.noteCatchUpMessageProcessed()
		}

		return grandpa.CommunicationInCatchUp[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
			CatchUp:  grandpa.CatchUp[H, N, primitives.AuthoritySignature, primitives.AuthorityID](msg.Message),
			Callback: cb,
		}
	}

	notifications := gossipEngine.MessagesFor(topic)
	in := make(chan communicationIn[H, N], 100_000)
	go func() {
		defer close(in)
		for notification := range notifications {
			notif := notification
			var decoded gossipMessageVDT[H, N]
			err := scale.Unmarshal(notif.Message, &decoded)
			if err != nil {
				logger.Tracef("Skipping malformed message %v: %v", notif, err)
				continue
			}
			message, err := decoded.Value()
			if err != nil {
				logger.Debugf("Skipping malformed message %v: %v", notif, err)
				continue
			}
			switch message := message.(type) {
			case gossipMessageCommit[H, N]:
				in <- processCommit(fullCommitMessage[H, N](message), &notif, gossipEngine, validator, voters)
			case gossipMessageCatchUp[H, N]:
				in <- processCatchUp(fullCatchUpMessage[H, N](message), &notif, gossipEngine, validator, voters)
			default:
				logger.Debugf("Skipping unknown message type")
				continue
			}
		}
	}()

	return in
}

// / Type-safe wrapper around a round number.
// pub struct Round(pub RoundNumber);
type Round uint64

// / Type-safe wrapper around a set ID.
// pub struct SetID(pub SetIdNumber);
type SetID uint64

// / A sink for outgoing messages to the network. Any messages that are sent will
// / be replaced, as appropriate, according to the given `HasVoted`.
// / NOTE: The votes are stored unsigned, which means that the signatures need to
// / be "stable", i.e. we should end up with the exact same signed message if we
// / use the same raw message and key to sign. This is currently true for
// / `ed25519` and `BLS` signatures (which we might use in the future), care must
// / be taken when switching to different key types.
type outgoingMessages[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	round    Round
	setID    SetID
	keystore *localIDKeystore
	sender   chan primitives.SignedMessage[H, N]
	network  *gossip.GossipEngine[H, N, Hasher]
	hasVoted hasVoted[H, N]
	// TODO: telemetry
}

// impl<B: BlockT> Unpin for OutgoingMessages<B> {}

// impl<Block: BlockT> Sink<Message<Block::Header>> for OutgoingMessages<Block> {
// 	type Error = Error;

// 	fn poll_ready(mut self: Pin<&mut Self>, cx: &mut Context) -> Poll<Result<(), Self::Error>> {
// 		Sink::poll_ready(Pin::new(&mut self.sender), cx).map(|elem| {
// 			elem.map_err(|e| {
// 				Error::Network(format!("Failed to poll_ready channel sender: {:?}", e))
// 			})
// 		})
// 	}

// fn start_send(
//
//	mut self: Pin<&mut Self>,
//	mut msg: Message<Block::Header>,
//
// ) -> Result<(), Self::Error> {
func (om *outgoingMessages[H, N, Hasher]) preSend(msg primitives.Message[H, N]) (primitives.Message[H, N], error) {
	// if we've voted on this round previously under the same key, send that vote instead
	switch msg.(type) {
	case grandpa.PrimaryPropose[H, N]:
		if propose := om.hasVoted.Propose(); propose != nil {
			msg = *propose
		}
	case grandpa.Prevote[H, N]:
		if prevote := om.hasVoted.Prevote(); prevote != nil {
			msg = *prevote
		}
	case grandpa.Precommit[H, N]:
		if precommit := om.hasVoted.Precommit(); precommit != nil {
			msg = *precommit
		}
	default:
		panic("unreachable")
	}

	// when locals exist, sign messages on import
	if om.keystore != nil {
		targetHash := msg.Target().Hash
		signed := primitives.SignMessage(
			om.keystore.KeyStore,
			msg,
			om.keystore.AuthorityID,
			primitives.RoundNumber(om.round),
			primitives.SetID(om.setID),
		)
		if signed == nil {
			return nil, fmt.Errorf("Failed to sign GRANDPA vote for round %d targeting %v", om.round, targetHash)
		}

		message := gossipMessageVote[H, N]{
			Message: primitives.SignedMessage[H, N](*signed),
			Round:   om.round,
			SetID:   om.setID,
		}

		logger.Debugf("Announcing block %v to peers which we voted on in round %d in set %d", targetHash, om.round, om.setID)

		// TODO: telemetry

		om.network.Announce(targetHash, nil)

		// propagate the message to peers
		topic := roundTopic[H, Hasher](om.round, om.setID)

		var gossipMessage gossipMessageVDT[H, N]
		gossipMessage.inner = message
		om.network.GossipMessage(topic, scale.MustMarshal(gossipMessage), false)

		// forward the message to the inner sender.
		// return self.sender.start_send(signed).map_err(|e| {
		// 	Error::Network(format!("Failed to start_send on channel sender: {:?}", e))
		// })
	}

	return msg, nil
}

// 	fn poll_flush(self: Pin<&mut Self>, _cx: &mut Context) -> Poll<Result<(), Self::Error>> {
// 		Poll::Ready(Ok(()))
// 	}

// 	fn poll_close(mut self: Pin<&mut Self>, cx: &mut Context) -> Poll<Result<(), Self::Error>> {
// 		Sink::poll_close(Pin::new(&mut self.sender), cx).map(|elem| {
// 			elem.map_err(|e| {
// 				Error::Network(format!("Failed to poll_close channel sender: {:?}", e))
// 			})
// 		})
// 	}
// }

// checks a compact commit. returns the cost associated with processing it if
// the commit was bad.
func checkCompactCommit[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	msg primitives.CompactCommit[H, N],
	voters *grandpa.VoterSet[primitives.AuthorityID],
	round Round,
	setID SetID,
	// TODO: telemetry
) *network.ReputationChange {
	// 4f + 1 = equivocations from f voters.
	f := voters.TotalWeight() - voters.Threshold()
	fullThreshold := f + voters.TotalWeight()

	// check total weight is not out of range.
	var totalWeight grandpa.VoterWeight
	for _, auth := range msg.AuthData {
		voter := voters.Get(auth.ID)
		if voter != nil {
			totalWeight += voter.Weight()
			if totalWeight > fullThreshold {
				return &malformedCommit
			}
		} else {
			logger.Debugf("Skipping commit containing unknown voter %v", auth)
			return &malformedCommit
		}
	}

	if totalWeight < voters.Threshold() {
		return &malformedCommit
	}

	// check signatures on all contained precommits.
	for i, precommit := range msg.Precommits {
		if !primitives.CheckMessageSignature(
			precommit,
			msg.AuthData[i].ID,
			msg.AuthData[i].Signature,
			primitives.RoundNumber(round),
			primitives.SetID(setID),
		) {
			logger.Debugf("Bad commit message signature %v", msg.AuthData[i].ID)
			// TODO: telemetry
			cost := misbehaviorBadCommitMessage{
				signaturesChecked:   int32(i),
				blocksLoaded:        0,
				equivocationsCaught: 0,
			}.cost()
			return &cost
		}
	}
	return nil
}

// checks a catch up. returns the cost associated with processing it if
// the catch up was bad.
func checkCatchUp[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	msg primitives.CatchUp[H, N],
	voters *grandpa.VoterSet[primitives.AuthorityID],
	setID SetID,
	// TODO: telemetry
) *network.ReputationChange {
	// 4f + 1 = equivocations from f voters.
	f := voters.TotalWeight() - voters.Threshold()
	fullThreshold := f + voters.TotalWeight()

	// check total weight is not out of range for a set of votes.
	var checkWeight = func(
		voters *grandpa.VoterSet[primitives.AuthorityID],
		votes []primitives.AuthorityID,
		fullThreshold grandpa.VoterWeight,
	) *network.ReputationChange {
		var totalWeight grandpa.VoterWeight
		for _, id := range votes {
			voter := voters.Get(id)
			if voter != nil {
				totalWeight += voter.Weight()
				if totalWeight > fullThreshold {
					return &malformedCatchUp
				}
			} else {
				logger.Debugf("Skipping catch up message containing unknown voter %v", id)
				return &malformedCatchUp
			}
		}

		if totalWeight < voters.Threshold() {
			return &malformedCatchUp
		}

		return nil
	}

	prevotes := make([]primitives.AuthorityID, len(msg.Prevotes))
	for i, vote := range msg.Prevotes {
		prevotes[i] = vote.ID
	}
	rep := checkWeight(voters, prevotes, fullThreshold)
	if rep != nil {
		return rep
	}

	precommits := make([]primitives.AuthorityID, len(msg.Precommits))
	for i, vote := range msg.Precommits {
		precommits[i] = vote.ID
	}
	rep = checkWeight(voters, precommits, fullThreshold)
	if rep != nil {
		return rep
	}

	type messageSignatureID struct {
		Message primitives.Message[H, N]
		grandpa.SignatureID[primitives.AuthoritySignature, primitives.AuthorityID]
	}
	var checkSignatures = func(
		messages []messageSignatureID,
		round Round,
		setID SetID,
		signaturesChecked uint,
	) (uint, *network.ReputationChange) {
		for _, msg := range messages {
			signaturesChecked++
			if !primitives.CheckMessageSignature(
				msg.Message,
				msg.SignatureID.ID,
				msg.SignatureID.Signature,
				primitives.RoundNumber(round),
				primitives.SetID(setID),
			) {
				logger.Debugf("Bad catch up message signature %v", msg.SignatureID.ID)
				// TODO: telemetry
				cost := misbehaviorBadCatchUpMessage{
					signaturesChecked: int32(signaturesChecked),
				}.cost()
				return signaturesChecked, &cost
			}
		}

		return signaturesChecked, nil
	}

	prevotesMessages := make([]messageSignatureID, len(msg.Prevotes))
	for i, vote := range msg.Prevotes {
		prevotesMessages[i] = messageSignatureID{
			Message: primitives.Message[H, N](vote.Prevote),
			SignatureID: grandpa.SignatureID[primitives.AuthoritySignature, primitives.AuthorityID]{
				Signature: vote.Signature,
				ID:        vote.ID,
			},
		}
	}
	signaturedChecked, rep := checkSignatures(prevotesMessages, Round(msg.RoundNumber), setID, 0)
	if rep != nil {
		return rep
	}

	// check signatures on all contained precommits.
	precommitsMessages := make([]messageSignatureID, len(msg.Precommits))
	for i, vote := range msg.Precommits {
		precommitsMessages[i] = messageSignatureID{
			Message: primitives.Message[H, N](vote.Precommit),
			SignatureID: grandpa.SignatureID[primitives.AuthoritySignature, primitives.AuthorityID]{
				Signature: vote.Signature,
				ID:        vote.ID,
			},
		}
	}
	_, rep = checkSignatures(precommitsMessages, Round(msg.RoundNumber), setID, signaturedChecked)
	if rep != nil {
		return rep
	}

	return nil
}

// / An output sink for commit messages.
type commitsOut[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	network         *gossip.GossipEngine[H, N, Hasher]
	setID           SetID
	isVoter         bool
	gossipValidator *gossipValidator[H, N, Hasher]
	neighborSender  neighbourPacketSender[N]
	// TODO: telemetry
}

// / Create a new commit output stream.
func newCommitsOut[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	network *gossip.GossipEngine[H, N, Hasher],
	setID SetID,
	isVoter bool,
	gossipValidator *gossipValidator[H, N, Hasher],
	neighborSender neighbourPacketSender[N],
	// TODO: telemetry
) commitsOut[H, N, Hasher] {
	return commitsOut[H, N, Hasher]{
		network:         network,
		setID:           setID,
		isVoter:         isVoter,
		gossipValidator: gossipValidator,
		neighborSender:  neighborSender,
	}
}

// impl<Block: BlockT> Sink<(RoundNumber, Commit<Block::Header>)> for CommitsOut<Block> {
// 	type Error = Error;

// 	fn poll_ready(self: Pin<&mut Self>, _: &mut Context) -> Poll<Result<(), Self::Error>> {
// 		Poll::Ready(Ok(()))
// 	}

func (co *commitsOut[H, N, Hasher]) preSend(
	round Round,
	commit primitives.Commit[H, N],
) error {
	if !co.isVoter {
		return nil
	}

	precommits := make([]grandpa.Precommit[H, N], len(commit.Precommits))
	authData := make(grandpa.MultiAuthData[primitives.AuthoritySignature, primitives.AuthorityID], len(commit.Precommits))
	for i, signed := range commit.Precommits {
		precommits[i] = signed.Precommit
		authData[i] = grandpa.SignatureID[primitives.AuthoritySignature, primitives.AuthorityID]{
			Signature: signed.Signature,
			ID:        signed.ID,
		}
	}

	compactCommit := primitives.CompactCommit[H, N]{
		TargetHash:   commit.TargetHash,
		TargetNumber: commit.TargetNumber,
		Precommits:   precommits,
		AuthData:     authData,
	}

	messageCommit := gossipMessageCommit[H, N](fullCommitMessage[H, N]{
		Round:   round,
		SetID:   co.setID,
		Message: compactCommit,
	})
	var message gossipMessageVDT[H, N]
	message.inner = messageCommit

	topic := globalTopic[H, Hasher](co.setID)

	// the gossip validator needs to be made aware of the best commit-height we know of
	// before gossiping
	co.gossipValidator.noteCommitFinalized(
		round,
		co.setID,
		commit.TargetNumber,
		func(to []peerid.PeerID, msg neighborPacket[N]) {
			co.neighborSender <- peerIDsNeighborPacket[N]{PeerIDs: to, NeighborPacket: msg}
		},
	)
	co.network.GossipMessage(topic, scale.MustMarshal(message), false)

	return nil
}
