// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/saturating"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gammazero/deque"
)

// Maximum number of rounds we are behind a peer before issuing a
// catch up request.
const catchUpThreshold uint64 = 2

// The total round duration measured in periods of gossip duration:
// 2 gossip durations for prevote timer
// 2 gossip durations for precommit timer
// 1 gossip duration for precommits to spread
const roundDuration uint32 = 5

// The period, measured in rounds, since the latest round start, after which we will start propagating gossip messages
// to more nodes than just the lucky ones.
const propagationSome float32 = 1.5

// The period, measured in rounds, since the latest round start, after which we will start propagating gossip messages
// to all the nodes we are connected to.
const propagationAll float32 = 3.0

// Assuming a network of 3000 nodes, using a fanout of 4, after about 6 iterations of gossip a message has very likely
// reached all nodes on the network (`log4(3000)`).
const luckyPeers uint = 4

type report struct {
	peerid.PeerID
	network.ReputationChange
}

const rebroadcastAfter time.Duration = 60 * 5 * time.Second

// An outcome of examining a message.
type consider uint

const (
	// Accept the message.
	considerAccept consider = iota + 1
	// Message is too early. Reject.
	considerRejectPast
	// Message is from the future. Reject.
	considerRejectFuture
	// Message cannot be evaluated. Reject.
	considerRejectOutOfScope
)

// A view of protocol state.
type view[N runtime.Number] struct {
	// the current round we are at.
	round Round
	// the current voter set id.
	setID SetID
	// commit-finalized block height, if any.
	lastCommit *N
	// last time we heard from peer, used for spamming detection.
	lastUpdate *time.Time
}

func newView[N runtime.Number]() view[N] {
	return view[N]{
		round:      1,
		setID:      0,
		lastCommit: nil,
		lastUpdate: nil,
	}
}

// Consider a round and set ID combination under a current view.
func (v view[N]) considerVote(round Round, setID SetID) consider {
	// only from current set
	if setID < v.setID {
		return considerRejectPast
	} else if setID > v.setID {
		return considerRejectFuture
	}

	// only r-1 ... r+1
	if round > saturating.Add(v.round, 1) {
		return considerRejectFuture
	} else if round < saturating.Sub(v.round, 1) {
		return considerRejectPast
	}

	return considerAccept
}

// Consider a set-id global message. Rounds are not taken into account, but are implicitly because we gate on
// finalization of a further block than a previous commit.
func (v view[N]) considerGlobal(setID SetID, number N) consider {
	// only from current set
	if setID < v.setID {
		return considerRejectPast
	} else if setID > v.setID {
		return considerRejectFuture
	}

	// only commits which claim to prove a higher block number than the one we're aware of.
	if v.lastCommit == nil {
		return considerAccept
	}
	if *v.lastCommit < number {
		return considerAccept
	}
	return considerRejectPast
}

type numberRoundSetID[N runtime.Number] struct {
	Number N
	Round
	SetID
}

// A local view of protocol state. Similar to `View` but we additionally track the round and set id at which the last
// commit was observed, and the instant at which the current round started.
type localView[N runtime.Number] struct {
	round      Round
	setID      SetID
	lastCommit *numberRoundSetID[N]
	roundStart time.Time
}

func newLocalView[N runtime.Number](setID SetID, round Round) *localView[N] {
	return &localView[N]{
		round:      round,
		setID:      setID,
		lastCommit: nil,
		roundStart: time.Now(),
	}
}

// Converts the local view to a `view` discarding round and set id information about the last commit.
func (lv *localView[N]) view() view[N] {
	return view[N]{
		round:      lv.round,
		setID:      lv.setID,
		lastCommit: lv.lastCommitHeight(),
		lastUpdate: nil,
	}
}

// Update the set ID. implies a reset to round 1.
func (lv *localView[N]) updateSet(setID SetID) {
	if setID != lv.setID {
		lv.setID = setID
		lv.round = 1
		lv.roundStart = time.Now()
	}
}

// Updates the current round.
func (lv *localView[N]) updateRound(round Round) {
	lv.round = round
	lv.roundStart = time.Now()
}

// Returns the height of the block that the last observed commit finalizes.
func (lv *localView[N]) lastCommitHeight() *N {
	if lv.lastCommit != nil {
		return &lv.lastCommit.Number
	}
	return nil
}

const keepRecentRounds uint = 3

type roundSetID struct {
	Round
	SetID
}

type keepTopicsMapEntry struct {
	*Round
	SetID
}

// Tracks gossip topics that we are keeping messages for. We keep topics of:
//   - the last `keepRecentRounds` complete GRANDPA rounds,
//   - the topic for the current and next round,
//   - and a global topic for commit and catch-up messages.
type keepTopics[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	currentSet SetID
	rounds     deque.Deque[roundSetID]
	reverseMap map[H]keepTopicsMapEntry
}

func newKeepTopics[H runtime.Hash, Hasher runtime.Hasher[H]]() keepTopics[H, Hasher] {
	var dq deque.Deque[roundSetID]
	dq.SetBaseCap(int(keepRecentRounds) + 2)
	return keepTopics[H, Hasher]{
		currentSet: SetID(0),
		rounds:     dq,
		reverseMap: make(map[H]keepTopicsMapEntry),
	}
}

func (kt *keepTopics[H, Hasher]) push(round Round, setID SetID) {
	if setID > kt.currentSet {
		kt.currentSet = setID
	}

	// under normal operation the given round is already tracked (since we
	// track one round ahead). if we skip rounds (with a catch up) the given
	// round topic might not be tracked yet.
	if kt.rounds.Index(func(rsi roundSetID) bool {
		return rsi == roundSetID{
			Round: round,
			SetID: setID,
		}
	}) < 0 {
		kt.rounds.PushBack(roundSetID{
			Round: round,
			SetID: setID,
		})
	}

	// we also accept messages for the next round
	kt.rounds.PushBack(roundSetID{
		Round: saturating.Add(round, 1),
		SetID: setID,
	})

	// the 2 is for the current and next round.
	for kt.rounds.Len() > int(keepRecentRounds)+2 {
		kt.rounds.PopFront()
	}

	mapped := make(map[H]keepTopicsMapEntry)
	k := globalTopic[H, Hasher](kt.currentSet)
	mapped[k] = keepTopicsMapEntry{
		Round: nil,
		SetID: kt.currentSet,
	}

	for i := 0; i < kt.rounds.Len(); i++ {
		rsi := kt.rounds.At(i)
		k := roundTopic[H, Hasher](rsi.Round, rsi.SetID)
		mapped[k] = keepTopicsMapEntry{
			Round: &rsi.Round,
			SetID: rsi.SetID,
		}
	}

	kt.reverseMap = mapped
}

func (kt *keepTopics[H, Hasher]) topicInfo(topic H) *keepTopicsMapEntry {
	entry, ok := kt.reverseMap[topic]
	if !ok {
		return nil
	}
	return &entry
}

// topics to send to a neighbor based on their view.
func neighborTopics[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](view *view[N]) []H {
	s := view.setID
	var topics []H = []H{globalTopic[H, Hasher](s), roundTopic[H, Hasher](view.round, s)}

	if view.round != 0 {
		r := view.round - 1
		topics = append(topics, roundTopic[H, Hasher](r, s))
	}

	return topics
}

// This is the gossipMessage [scale.VaryingDataType] impl that gets encoded and sent over the network.
type gossipMessageVDT[H runtime.Hash, N runtime.Number] struct {
	inner gossipMessage
}

type gossipMessageVDTValues[H runtime.Hash, N runtime.Number] interface {
	gossipMessageVote[H, N] | gossipMessageCommit[H, N] | gossipMessageNeighbor[N] | gossipMessageCatchUpRequest |
		gossipMessageCatchUp[H, N]
	gossipMessage
}

func setGossipMessageVDT[
	H runtime.Hash, N runtime.Number, Value gossipMessageVDTValues[H, N],
](mvdt *gossipMessageVDT[H, N], value Value) {
	mvdt.inner = value
}

func (mvdt *gossipMessageVDT[H, N]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case gossipMessageVote[H, N]:
		setGossipMessageVDT[H, N](mvdt, value)
		return
	case gossipMessageCommit[H, N]:
		setGossipMessageVDT[H, N](mvdt, value)
		return
	case gossipMessageNeighbor[N]:
		setGossipMessageVDT[H, N](mvdt, value)
		return
	case gossipMessageCatchUpRequest:
		setGossipMessageVDT[H, N](mvdt, value)
		return
	case gossipMessageCatchUp[H, N]:
		setGossipMessageVDT[H, N](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt gossipMessageVDT[H, N]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case gossipMessageVote[H, N]:
		return 0, mvdt.inner, nil
	case gossipMessageCommit[H, N]:
		return 1, mvdt.inner, nil
	case gossipMessageNeighbor[N]:
		return 2, mvdt.inner, nil
	case gossipMessageCatchUpRequest:
		return 3, mvdt.inner, nil
	case gossipMessageCatchUp[H, N]:
		return 4, mvdt.inner, nil
	default:
		return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
	}
}

func (mvdt gossipMessageVDT[H, N]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt gossipMessageVDT[H, N]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(gossipMessageVote[H, N]), nil
	case 1:
		return *new(gossipMessageCommit[H, N]), nil
	case 2:
		return *new(gossipMessageNeighbor[N]), nil
	case 3:
		return *new(gossipMessageCatchUpRequest), nil
	case 4:
		return *new(gossipMessageCatchUp[H, N]), nil
	default:
		return nil, scale.ErrUnsupportedVaryingDataTypeValue
	}
}

// Grandpa gossip message type.
type gossipMessage interface {
	isGossipMessage()
}

// Grandpa message with round and set info.
type gossipMessageVote[H runtime.Hash, N runtime.Number] voteMessage[H, N]

// Grandpa commit message with round and set info.
type gossipMessageCommit[H runtime.Hash, N runtime.Number] fullCommitMessage[H, N]

// A neighbor packet. Not repropagated.
type gossipMessageNeighbor[N runtime.Number] versionedNeighborPacket[N]

func (vnp gossipMessageNeighbor[N]) MarshalSCALE() ([]byte, error) {
	return versionedNeighborPacket[N](vnp).MarshalSCALE()
}

func (gmn *gossipMessageNeighbor[N]) UnmarshalSCALE(reader io.Reader) error {
	vnp := versionedNeighborPacket[N]{}
	err := vnp.UnmarshalSCALE(reader)
	if err != nil {
		return err
	}
	*gmn = gossipMessageNeighbor[N](vnp)
	return nil
}

// Grandpa catch up request message with round and set info. Not repropagated.
type gossipMessageCatchUpRequest catchUpRequestMessage

// Grandpa catch up message with round and set info. Not repropagated.
type gossipMessageCatchUp[H runtime.Hash, N runtime.Number] fullCatchUpMessage[H, N]

func (gossipMessageVote[H, N]) isGossipMessage()     {}
func (gossipMessageCommit[H, N]) isGossipMessage()   {}
func (gossipMessageNeighbor[N]) isGossipMessage()    {}
func (gossipMessageCatchUpRequest) isGossipMessage() {}
func (gossipMessageCatchUp[H, N]) isGossipMessage()  {}

// Network level vote message with topic information.
type voteMessage[H runtime.Hash, N runtime.Number] struct {
	// The round this message is from.
	Round Round
	// The voter set ID this message is from.
	SetID SetID
	// The message itself.
	Message primitives.SignedMessage[H, N]
}

// Network level commit message with topic information.
type fullCommitMessage[H runtime.Hash, N runtime.Number] struct {
	// The round this message is from.
	Round Round
	// The voter set ID this message is from.
	SetID SetID
	// The compact commit message.
	Message primitives.CompactCommit[H, N]
}

// V1 neighbor packet. Neighbor packets are sent from nodes to their peers
// and are not repropagated. These contain information about the node's state.
type neighborPacket[N runtime.Number] struct {
	// The round the node is currently at.
	Round Round
	// The set ID the node is currently at.
	SetID SetID
	// The highest finalizing commit observed.
	CommitFinalizedHeight N
}

// A versioned neighbor packet.
type versionedNeighborPacket[N runtime.Number] struct {
	neighborPacket[N]
}

func (vnp versionedNeighborPacket[N]) MarshalSCALE() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	_, err := encoder.Write([]byte{1})
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(vnp.neighborPacket)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (vnp *versionedNeighborPacket[N]) UnmarshalSCALE(reader io.Reader) error {
	decoder := scale.NewDecoder(reader)
	var index byte
	if err := decoder.Decode(&index); err != nil {
		return err
	}
	switch index {
	case 1:
		return decoder.Decode(&vnp.neighborPacket)
	default:
		return fmt.Errorf("unsupported index")
	}
}

// A catch up request for a given round (or any further round) localized by set id.
type catchUpRequestMessage struct {
	// The round that we want to catch up to.
	Round
	// The voter set ID this message is from.
	SetID
}

// Network level catch up message with topic information.
type fullCatchUpMessage[H runtime.Hash, N runtime.Number] struct {
	// The voter set ID this message is from.
	SetID SetID
	// The compact commit message.
	Message primitives.CatchUp[H, N]
}

// Misbehavior that peers can perform.  `cost` gives a cost that can be used to perform cost/benefit analysis of a peer.
type misbehavior interface {
	cost() network.ReputationChange
	error
}

// invalid neighbor message, considering the last one.
type misbehaviorInvalidViewChange struct{}

// duplicate neighbor message.
type misbehaviorDuplicateNeighborMessage struct{}

// could not decode neighbor message. bytes-length of the packet.
type misbehaviorUndecodablePacket int32

// Bad catch up message (invalid signatures).
type misbehaviorBadCatchUpMessage struct {
	signaturesChecked int32
}

// Bad commit message
type misbehaviorBadCommitMessage struct {
	signaturesChecked   int32
	blocksLoaded        int32
	equivocationsCaught int32
}

// A message received that's from the future relative to our view.
type misbehaviorFutureMessage struct{}

// A message received that cannot be evaluated relative to our view. This happens before we have a view and have sent
// out neighbor packets.
type misbehaviorOutOfScopeMessage struct{}

func (misbehaviorInvalidViewChange) cost() network.ReputationChange {
	return invalidViewChange
}
func (misbehaviorDuplicateNeighborMessage) cost() network.ReputationChange {
	return duplicateNeighborMessage
}
func (mup misbehaviorUndecodablePacket) cost() network.ReputationChange {
	return network.NewReputationChange(
		saturating.Mul(int32(mup), perUndecodeableByte),
		"Grandpa: Bad packet",
	)
}
func (mbcm misbehaviorBadCatchUpMessage) cost() network.ReputationChange {
	return network.NewReputationChange(
		saturating.Mul(mbcm.signaturesChecked, perSignatureChecked),
		"Grandpa: Bad packet",
	)
}
func (mbcm misbehaviorBadCommitMessage) cost() network.ReputationChange {
	cost := saturating.Mul(mbcm.signaturesChecked, perSignatureChecked)
	cost = saturating.Add(cost, saturating.Mul(perBlockLoaded, mbcm.blocksLoaded))
	benefit := saturating.Mul(mbcm.equivocationsCaught, perEquivocation)

	return network.NewReputationChange(
		saturating.Add(benefit, cost),
		"Grandpa: Bad commit",
	)
}
func (misbehaviorFutureMessage) cost() network.ReputationChange {
	return futureMessage
}
func (misbehaviorOutOfScopeMessage) cost() network.ReputationChange {
	return outOfScopeMessage
}

func (m misbehaviorInvalidViewChange) Error() string        { return m.cost().Reason }
func (m misbehaviorDuplicateNeighborMessage) Error() string { return m.cost().Reason }
func (m misbehaviorUndecodablePacket) Error() string        { return m.cost().Reason }
func (m misbehaviorBadCatchUpMessage) Error() string        { return m.cost().Reason }
func (m misbehaviorBadCommitMessage) Error() string         { return m.cost().Reason }
func (m misbehaviorFutureMessage) Error() string            { return m.cost().Reason }
func (m misbehaviorOutOfScopeMessage) Error() string        { return m.cost().Reason }

type peerInfo[N runtime.Number] struct {
	view  view[N]
	roles role.ObservedRole
}

// The peers we're connected to in gossip.
type peers[N runtime.Number] struct {
	inner map[peerid.PeerID]peerInfo[N]
	// The randomly picked set of `luckyPeers` we'll gossip to in the first stage of round gossiping.
	firstStagePeers map[peerid.PeerID]struct{}
	// The randomly picked set of peers we'll gossip to in the second stage of gossiping if the first stage didn't
	// allow us to spread the voting data enough to conclude the round. This set should have size `sqrt(connected_peers)`.
	secondStagePeers map[peerid.PeerID]struct{}
	// The randomly picked set of `luckyPeers` light clients we'll gossip commit messages to.
	luckyLightPeers map[peerid.PeerID]struct{}
	// Neighbor packet rebroadcast period --- we reduce the reputation of peers sending duplicate packets too often.
	neighborRebroadcastPeriod time.Duration
}

func newPeers[N runtime.Number](neighborRebroadcastPeriod time.Duration) peers[N] {
	return peers[N]{
		inner:                     make(map[peerid.PeerID]peerInfo[N]),
		firstStagePeers:           make(map[peerid.PeerID]struct{}),
		secondStagePeers:          make(map[peerid.PeerID]struct{}),
		luckyLightPeers:           make(map[peerid.PeerID]struct{}),
		neighborRebroadcastPeriod: neighborRebroadcastPeriod,
	}
}

func (p *peers[N]) newPeer(who peerid.PeerID, observedRole role.ObservedRole) {
	switch observedRole {
	case role.ObservedRoleAuthority:
		if len(p.firstStagePeers) < int(luckyPeers) {
			p.firstStagePeers[who] = struct{}{}
		} else if len(p.secondStagePeers) < int(luckyPeers) {
			p.secondStagePeers[who] = struct{}{}
		}
	case role.ObservedRoleLight:
		if len(p.luckyLightPeers) < int(luckyPeers) {
			p.luckyLightPeers[who] = struct{}{}
		}
	default:
	}

	p.inner[who] = peerInfo[N]{view: newView[N](), roles: observedRole}
}

func (p *peers[N]) peerDisconnected(who peerid.PeerID) {
	delete(p.inner, who)
	// This does not happen often enough compared to round duration, so we don't reshuffle.
	delete(p.firstStagePeers, who)
	delete(p.secondStagePeers, who)
	delete(p.luckyLightPeers, who)
}

// returns a reference to the new view, if the peer is known.
func (p *peers[N]) updatePeerState(who peerid.PeerID, update neighborPacket[N]) (*view[N], misbehavior) {
	peer, ok := p.inner[who]
	if !ok {
		return nil, nil
	}

	invalidChange := peer.view.setID > update.SetID ||
		peer.view.round > update.Round && peer.view.setID == update.SetID ||
		(peer.view.lastCommit != nil && *peer.view.lastCommit > update.CommitFinalizedHeight)

	if invalidChange {
		return nil, misbehaviorInvalidViewChange{}
	}

	now := time.Now()
	var duplicatePacket bool = update.SetID == peer.view.setID &&
		update.Round == peer.view.round &&
		(peer.view.lastCommit != nil && *peer.view.lastCommit == update.CommitFinalizedHeight)

	if duplicatePacket {
		if lastUpdate := peer.view.lastUpdate; lastUpdate != nil {
			if now.Before(lastUpdate.Add(p.neighborRebroadcastPeriod / 2)) {
				return nil, misbehaviorDuplicateNeighborMessage{}
			}
		}
	}

	peer.view = view[N]{
		round:      update.Round,
		setID:      update.SetID,
		lastCommit: &update.CommitFinalizedHeight,
		lastUpdate: &now,
	}
	p.inner[who] = peer

	logger.Tracef("Peer %s updated view. Now at %v, %v", who, peer.view.round, peer.view.setID)

	return &peer.view, nil
}

func (p *peers[N]) updateCommitHeight(who peerid.PeerID, newHeight N) misbehavior {
	peer, ok := p.inner[who]
	if !ok {
		return nil
	}

	// this doesn't allow a peer to send us unlimited commits with the same height, because there is still a
	// misbehavior condition based on sending commits that are <= the best we are aware of.
	if lastCommit := peer.view.lastCommit; lastCommit != nil && *lastCommit > newHeight {
		return misbehaviorInvalidViewChange{}
	}

	peer.view.lastCommit = &newHeight
	return nil
}

func (p *peers[N]) peer(who peerid.PeerID) *peerInfo[N] {
	info, ok := p.inner[who]
	if !ok {
		return nil
	}
	return &info
}

func (p *peers[N]) reshuffle() {
	// we want to randomly select peers into three sets according to the following logic:
	//   - first set: luckyPeers random peers where at least luckyPeers/2 are authorities (unless we're not connected
	//     to that many authorities)
	//   - second set: max(luckyPeers, sqrt(peers)) peers where at least luckyPeers are authorities.
	//   - third set: luckyPeers random light client peers
	type peer struct {
		peerid.PeerID
		peerInfo[N]
	}
	peers := make([]peer, len(p.inner))
	i := 0
	for peerID, peerInfo := range p.inner {
		peers[i] = peer{peerID, peerInfo}
		i++
	}
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	shuffledPeers := peers

	var shuffledAuthorities []peerid.PeerID
	for _, peer := range shuffledPeers {
		if peer.roles == role.ObservedRoleAuthority {
			shuffledAuthorities = append(shuffledAuthorities, peer.PeerID)
		}
	}

	firstStagePeers := make(map[peerid.PeerID]struct{})
	secondStagePeers := make(map[peerid.PeerID]struct{})

	// we start by allocating authorities to the first stage set and when the minimum of `luckyPeers / 2` is filled we
	// start allocating to the second stage set.
	halfLucky := luckyPeers / 2
	oneAndAHalfLucky := luckyPeers + halfLucky
	for nAuthoritiesAdded, peerID := range shuffledAuthorities {
		if uint(nAuthoritiesAdded) < halfLucky {
			firstStagePeers[peerID] = struct{}{}
		} else if uint(nAuthoritiesAdded) < oneAndAHalfLucky {
			secondStagePeers[peerID] = struct{}{}
		} else {
			break
		}
	}

	// fill up first and second sets with remaining peers (either full or authorities) prioritizing filling the first
	// set over the second.
	nSecondStatePeers := uint(math.Max(float64(luckyPeers), math.Sqrt(float64(len(shuffledPeers)))))
	for _, peer := range shuffledPeers {
		if peer.peerInfo.roles.IsLight() {
			continue
		}

		if uint(len(firstStagePeers)) < luckyPeers {
			firstStagePeers[peer.PeerID] = struct{}{}
			delete(secondStagePeers, peer.PeerID)
		} else if uint(len(secondStagePeers)) < nSecondStatePeers {
			if _, ok := firstStagePeers[peer.PeerID]; !ok {
				secondStagePeers[peer.PeerID] = struct{}{}
			}
		} else {
			break
		}
	}

	// pick `luckyPeers` random light peers
	luckyLightPeers := make(map[peerid.PeerID]struct{})
	for _, peer := range shuffledPeers {
		if peer.roles.IsLight() {
			luckyLightPeers[peer.PeerID] = struct{}{}
		}
		if uint(len(luckyLightPeers)) >= luckyPeers {
			break
		}
	}

	p.firstStagePeers = firstStagePeers
	p.secondStagePeers = secondStagePeers
	p.luckyLightPeers = luckyLightPeers
}

type action interface {
	isAction()
}

// repropagate under given topic, to the given peers, applying cost/benefit to originator.
type actionKeep[H runtime.Hash] struct {
	Hash H
	network.ReputationChange
}

// discard and process.
type actionProcessAndDiscard[H runtime.Hash] struct {
	Hash H
	network.ReputationChange
}

// discard, applying cost/benefit to originator.
type actionDiscard[H runtime.Hash] struct {
	network.ReputationChange
}

func (actionKeep[H]) isAction()              {}
func (actionProcessAndDiscard[H]) isAction() {}
func (actionDiscard[H]) isAction()           {}

// State of catch up request handling.
type pendingCatchUp interface {
	isPendingCatchUp()
}

// No pending catch up requests.
type pendingCatchUpNone struct{}

// Pending catch up request which has not been answered yet.
type pendingCatchUpRequesting struct {
	who     peerid.PeerID
	request catchUpRequestMessage
	instant time.Time
}

// Pending catch up request that was answered and is being processed.
type pendingCatchUpProcessing struct {
	instant time.Time
}

func (pendingCatchUpNone) isPendingCatchUp()       {}
func (pendingCatchUpRequesting) isPendingCatchUp() {}
func (pendingCatchUpProcessing) isPendingCatchUp() {}

// Configuration for the round catch-up mechanism.
type catchUpConfig[N runtime.Number] interface {
	requestAllowed(peer peerInfo[N]) bool
}

// Catch requests are enabled, our node will issue them whenever it sees a neighbor packet for a round further than
// `catchUpThreshold`. If `onlyFromAuthorities` is set, the node will only send catch-up requests to other authorities
// it is connected to. This is useful if the GRANDPA observer protocol is live on the network, in which case full nodes
// (non-authorities) don't have the necessary round data to answer catch-up requests.
type catchUpConfigEnabled[N runtime.Number] struct {
	onlyFromAuthorities bool
}

// Catch-up requests are disabled, our node will never issue them. This is useful for the GRANDPA observer mode, where
// we are only interested in commit messages and don't need to follow the full round protocol.
type catchUpConfigDisabled[N runtime.Number] struct{} //nolint: unused

func (e catchUpConfigEnabled[N]) requestAllowed(peer peerInfo[N]) bool { //nolint: unused
	switch peer.roles {
	case role.ObservedRoleAuthority:
		return true
	case role.ObservedRoleLight:
		return false
	case role.ObservedRoleFull:
		return !e.onlyFromAuthorities
	default:
		panic("unreachable")
	}
}
func (catchUpConfigDisabled[N]) requestAllowed(peer peerInfo[N]) bool { //nolint: unused
	return false
}

type inner[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	localView       *localView[N]
	peers           peers[N]
	liveTopics      keepTopics[H, Hasher]
	authorities     []primitives.AuthorityID
	config          Config
	nextRebroadcast time.Time
	pendingCatchUp  pendingCatchUp
	catchUpConfig   catchUpConfig[N]
}

func newInner[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](config Config) inner[H, N, Hasher] {
	var catchUpConfig catchUpConfig[N]
	if config.ObserverEnabled {
		panic("we have not implemented the observer")
	} else {
		// if the observer protocol isn't enabled and we're not a light client, then any full node should be able to
		// answer catch-up requests.
		catchUpConfig = catchUpConfigEnabled[N]{false}
	}

	return inner[H, N, Hasher]{
		localView:       nil,
		peers:           newPeers[N](neighborRebroadcastPeriod),
		liveTopics:      newKeepTopics[H, Hasher](),
		nextRebroadcast: time.Now().Add(rebroadcastAfter),
		authorities:     make([]primitives.AuthorityID, 0),
		pendingCatchUp:  pendingCatchUpNone{},
		catchUpConfig:   catchUpConfig,
		config:          config,
	}
}

// Note a round in the current set has started. Does nothing if the last call to the function was with the same round.
func (i *inner[H, N, Hasher]) noteRound(round Round) *peerIDsNeighborPacket[N] {
	if i.localView.round == round {
		// Do not send neighbor packets out if `round` has not changed, such behavior is punishable.
		return nil
	}

	setID := i.localView.setID

	logger.Debugf("Voter %s noting beginning of round (%d, %d) to network.", i.config.name(), round, setID)

	i.localView.updateRound(round)

	i.liveTopics.push(round, setID)
	i.peers.reshuffle()

	return i.multicastNeighborPacket()
}

// Note that a voter set with given ID has started. Does nothing if the last call to the function was with the same
// SetID.
func (i *inner[H, N, Hasher]) noteSet(setID SetID, authorities []primitives.AuthorityID) *peerIDsNeighborPacket[N] {
	if i.localView == nil {
		i.localView = newLocalView[N](setID, 1)
	} else if i.localView.setID == setID {
		a := make(map[primitives.AuthorityID]struct{})
		b := make(map[primitives.AuthorityID]struct{})

		for _, auth := range i.authorities {
			a[auth] = struct{}{}
		}
		for _, auth := range authorities {
			b[auth] = struct{}{}
		}

		diffAuthorities := !maps.Equal(a, b)

		if diffAuthorities {
			logger.Debugf(
				"Gossip validator noted set %s twice with differnet authorities. "+
					"Was the authority set hard forked?", setID)

			i.authorities = authorities
		}

		// Do not send neighbor packets out if the `setID` has not changed. Such behavior is punishable.
		return nil
	}

	i.localView.updateSet(setID)
	i.liveTopics.push(1, setID)
	i.authorities = authorities

	return i.multicastNeighborPacket()
}

// Note that we've imported a commit finalizing a given block. Does nothing if the last call to the function was with
// the same or higher `finalized` number. `setID` & `round` are the ones the commit message is from.
func (i *inner[H, N, Hasher]) noteCommitFinalized(round Round, setID SetID, finalized N) *peerIDsNeighborPacket[N] {
	if i.localView == nil {
		return nil
	}
	lch := i.localView.lastCommitHeight()
	if lch == nil || (*lch < finalized) {
		i.localView.lastCommit = &numberRoundSetID[N]{
			Number: finalized,
			Round:  round,
			SetID:  setID,
		}
	} else {
		return nil
	}

	return i.multicastNeighborPacket()
}

func (i *inner[H, N, Hasher]) considerVote(round Round, setID SetID) consider {
	if i.localView == nil {
		return considerRejectOutOfScope
	}
	v := i.localView.view()
	return v.considerVote(round, setID)
}

func (i *inner[H, N, Hasher]) considerGlobal(setID SetID, number N) consider {
	if i.localView == nil {
		return considerRejectOutOfScope
	}
	v := i.localView.view()
	return v.considerGlobal(setID, number)
}

func (i *inner[H, N, Hasher]) costPastRejection(_ peerid.PeerID, _ Round, _ SetID) network.ReputationChange {
	// hardcoded for now.
	return pastRejection
}

func (i *inner[H, N, Hasher]) validateRoundMessage(who peerid.PeerID, full voteMessage[H, N]) action {
	switch i.considerVote(full.Round, full.SetID) {
	case considerRejectFuture:
		return actionDiscard[H]{ReputationChange: misbehaviorFutureMessage{}.cost()}
	case considerRejectOutOfScope:
		return actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	case considerRejectPast:
		return actionDiscard[H]{ReputationChange: i.costPastRejection(who, full.Round, full.SetID)}
	case considerAccept:
	default:
		panic("unreachable")
	}

	// ensure authority is part of the set.
	if !slices.Contains(i.authorities, full.Message.ID) {
		logger.Debugf("Message from unknown voter: %s", full.Message.ID)
		// TODO: telemetry
		return actionDiscard[H]{unknownVoter}
	}

	if !primitives.CheckMessageSignature(
		full.Message.Message,
		full.Message.ID,
		full.Message.Signature,
		primitives.RoundNumber(full.Round),
		primitives.SetID(full.SetID),
	) {
		logger.Debugf("Bad message signature %s", full.Message.ID)
		// TODO: Telemetry
		return actionDiscard[H]{badSignature}
	}

	topic := roundTopic[H, Hasher](full.Round, full.SetID)
	return actionKeep[H]{Hash: topic, ReputationChange: roundMessage}
}

func (i *inner[H, N, Hasher]) validateCommitMessage(who peerid.PeerID, full fullCommitMessage[H, N]) action {
	misbehavior := i.peers.updateCommitHeight(who, full.Message.TargetNumber)
	if misbehavior != nil {
		return actionDiscard[H]{ReputationChange: misbehavior.cost()}
	}

	switch i.considerGlobal(full.SetID, full.Message.TargetNumber) {
	case considerRejectFuture:
		return actionDiscard[H]{ReputationChange: misbehaviorFutureMessage{}.cost()}
	case considerRejectPast:
		return actionDiscard[H]{ReputationChange: i.costPastRejection(who, full.Round, full.SetID)}
	case considerRejectOutOfScope:
		return actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	case considerAccept:
	default:
		panic("unreachable")
	}

	if len(full.Message.Precommits) != len(full.Message.AuthData) || len(full.Message.Precommits) == 0 {
		logger.Debugf("Malformed compact commit")
		// TODO: telemetry
		return actionDiscard[H]{ReputationChange: malformedCommit}
	}

	// always discard commits initially and rebroadcast after doing full checking.
	topic := globalTopic[H, Hasher](full.SetID)
	return actionProcessAndDiscard[H]{Hash: topic, ReputationChange: basicValidatedCommit}
}

func (i *inner[H, N, Hasher]) validateCatchUpMessage(who peerid.PeerID, full fullCatchUpMessage[H, N]) action {
	switch pending := i.pendingCatchUp.(type) {
	case pendingCatchUpRequesting:
		peer := pending.who
		if peer != who {
			return actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
		}

		if pending.request.SetID != full.SetID {
			return actionDiscard[H]{ReputationChange: malformedCatchUp}
		}

		if uint64(pending.request.Round) > full.Message.RoundNumber {
			return actionDiscard[H]{ReputationChange: malformedCatchUp}
		}

		if len(full.Message.Prevotes) == 0 || len(full.Message.Precommits) == 0 {
			return actionDiscard[H]{ReputationChange: malformedCatchUp}
		}

		// move request to pending processing state, we won't push out any catch up requests until we import this one
		// (either with a success or failure).
		i.pendingCatchUp = pendingCatchUpProcessing{instant: pending.instant}
		i.noteCatchUpMessageProcessed()

		// always discard catch up messages, they're point-to-point
		topic := globalTopic[H, Hasher](full.SetID)
		return actionProcessAndDiscard[H]{Hash: topic, ReputationChange: basicValidatedCatchUp}
	default:
		return actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	}
}

func (i *inner[H, N, Hasher]) noteCatchUpMessageProcessed() {
	switch i.pendingCatchUp.(type) {
	case pendingCatchUpProcessing:
		i.pendingCatchUp = pendingCatchUpNone{}
	default:
		logger.Debugf("Noted processed catch up message when state was: %v", i.pendingCatchUp)
	}
}

func (i *inner[H, N, Hasher]) handleCatchUpRequest(
	who peerid.PeerID, request catchUpRequestMessage, setState *SharedVoterSetState[H, N]) (gossipMessage, action) {
	if i.localView == nil {
		return nil, actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	}

	if request.SetID != i.localView.setID {
		// NOTE: When we're close to a set change there is potentially a race where the peer sent us the request before
		// it observed that we had transitioned to a new set. In this case we charge a lower cost.
		if saturating.Add(request.SetID, 1) == i.localView.setID &&
			saturating.Sub(uint64(i.localView.round), catchUpThreshold) == 0 {
			return nil, actionDiscard[H]{ReputationChange: honestOutOfScopeCatchUp}
		}

		return nil, actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	}

	peer := i.peers.peer(who)
	if peer == nil {
		return nil, actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	} else if peer.view.round >= request.Round {
		return nil, actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	}

	setState.innerMtx.RLock()
	lastCompletedRound := setState.inner.lastCompletedRound()
	setState.innerMtx.RUnlock()
	if Round(lastCompletedRound.Number) < request.Round {
		return nil, actionDiscard[H]{ReputationChange: misbehaviorOutOfScopeMessage{}.cost()}
	}

	logger.Tracef("Replying to catch-up request for round %d from %s with round %d",
		request.Round, who, lastCompletedRound.Number)

	prevotes := make([]grandpa.SignedPrevote[H, N, primitives.AuthoritySignature, primitives.AuthorityID], 0)
	precommits := make([]grandpa.SignedPrecommit[H, N, primitives.AuthoritySignature, primitives.AuthorityID], 0)

	// NOTE: the set of votes stored in `LastCompletedRound` is a minimal set of votes, i.e. at most one equivocation
	// is stored per voter. The code below assumes this invariant is maintained when creating the catch up reply since
	// peers won't accept catch-up messages that have too many equivocations (we exceed the fault-tolerance bound).
	for _, vote := range lastCompletedRound.Votes {
		switch message := vote.Message.(type) {
		case grandpa.Prevote[H, N]:
			prevotes = append(prevotes, grandpa.SignedPrevote[
				H, N, primitives.AuthoritySignature, primitives.AuthorityID,
			]{
				Prevote:   message,
				Signature: vote.Signature,
				ID:        vote.ID,
			})
		case grandpa.Precommit[H, N]:
			precommits = append(precommits, grandpa.SignedPrecommit[
				H, N, primitives.AuthoritySignature, primitives.AuthorityID,
			]{
				Precommit: message,
				Signature: vote.Signature,
				ID:        vote.ID,
			})
		default:
		}
	}

	baseHash, baseNumber := lastCompletedRound.Base.Hash, lastCompletedRound.Base.Number

	catchUp := primitives.CatchUp[H, N]{
		RoundNumber: uint64(lastCompletedRound.Number),
		Prevotes:    prevotes,
		Precommits:  precommits,
		BaseHash:    baseHash,
		BaseNumber:  baseNumber,
	}

	fullCatchUp := gossipMessageCatchUp[H, N](
		fullCatchUpMessage[H, N]{
			SetID:   request.SetID,
			Message: catchUp,
		},
	)

	return fullCatchUp, actionDiscard[H]{ReputationChange: catchUpReply}
}

func (i *inner[H, N, Hasher]) tryCatchUp(who peerid.PeerID) (gossipMessage, *report) {
	var catchUp gossipMessage
	var report *report

	// if the peer is on the same set and ahead of us by a margin bigger than `catchUpThreshold` then we should ask it
	// for a catch up message. we only send catch-up requests to authorities, observers won't be able to reply since
	// they don't follow the full GRANDPA protocol and therefore might not have the vote data available.
	if peer := i.peers.peer(who); peer != nil && i.localView != nil {
		if i.catchUpConfig.requestAllowed(*peer) &&
			peer.view.setID == i.localView.setID &&
			saturating.Sub(uint64(peer.view.round), catchUpThreshold) > uint64(i.localView.round) {

			// send catch up request if allowed
			round := peer.view.round - 1 // peer.view.round is > 0
			request := catchUpRequestMessage{SetID: peer.view.setID, Round: round}
			catchUpAllowed, catchUpReport := i.noteCatchUpRequest(who, request)

			if catchUpAllowed {
				logger.Debugf("Sending catch-up request for round %d to %s", round, who)
				catchUp = gossipMessageCatchUpRequest(request)
			}

			report = catchUpReport
		}
	}

	return catchUp, report
}

func (i *inner[H, N, Hasher]) importNeighborMessage(
	who peerid.PeerID, update neighborPacket[N],
) ([]H, action, gossipMessage, *report) {
	var costBenefit network.ReputationChange
	var topics []H

	view, misbehavior := i.peers.updatePeerState(who, update)
	if misbehavior != nil {
		costBenefit = misbehavior.cost()
		topics = nil
	} else {
		costBenefit = neighborMessage
		topics = neighborTopics[H, N, Hasher](view)
	}

	var catchUp gossipMessage
	var report *report
	if misbehavior == nil {
		catchUp, report = i.tryCatchUp(who)
	}

	// always discard neighbor messages, it's only valid for one hop.
	action := actionDiscard[H]{ReputationChange: costBenefit}

	return topics, action, catchUp, report
}

func (i *inner[H, N, Hasher]) multicastNeighborPacket() *peerIDsNeighborPacket[N] {
	if i.localView != nil {
		var commitFinalizedHeight N = 0
		if h := i.localView.lastCommitHeight(); h != nil {
			commitFinalizedHeight = *h
		}
		packet := neighborPacket[N]{
			Round:                 i.localView.round,
			SetID:                 i.localView.setID,
			CommitFinalizedHeight: commitFinalizedHeight,
		}

		peers := make([]peerid.PeerID, 0)
		for peerID := range i.peers.inner {
			peers = append(peers, peerID)
		}
		return &peerIDsNeighborPacket[N]{
			PeerIDs:        peers,
			NeighborPacket: packet,
		}
	}
	return nil
}

func (i *inner[H, N, Hasher]) noteCatchUpRequest(
	who peerid.PeerID, catchUpRequest catchUpRequestMessage,
) (bool, *report) {
	const (
		catchUpRequestTimeoutDuration = 45 * time.Second
		catchUpProcessTimeoutDuration = 30 * time.Second
	)
	var rep *report
	switch pending := i.pendingCatchUp.(type) {
	case pendingCatchUpRequesting:
		if time.Since(pending.instant) <= catchUpRequestTimeoutDuration {
			return false, nil
		}
		// report peer for timeout
		rep = &report{PeerID: pending.who, ReputationChange: catchUpRequestTimeout}
	case pendingCatchUpProcessing:
		if time.Since(pending.instant) < catchUpProcessTimeoutDuration {
			return false, nil
		}
	default:
	}

	i.pendingCatchUp = pendingCatchUpRequesting{
		who:     who,
		request: catchUpRequest,
		instant: time.Now(),
	}

	return true, rep
}

// The initial logic for filtering round messages follows the given state
// transitions:
//
//   - State 1: allowed to luckyPeers random peers (where at least luckyPeers/2 are authorities)
//   - State 2: allowed to max(luckyPeers, sqrt(random peers)) (where at least luckyPeers are authorities)
//   - State 3: allowed to all peers
//
// Transitions will be triggered on repropagation attempts by the underlying gossip layer.
func (i *inner[H, N, Hasher]) roundMessageAllowed(who peerid.PeerID) bool {
	rd := i.config.GossipDuration * time.Duration(roundDuration)
	if i.localView == nil {
		return false
	}
	roundElapsed := time.Since(i.localView.roundStart)
	if roundElapsed < time.Duration(float32(rd)*propagationSome) {
		_, ok := i.peers.firstStagePeers[who]
		return ok
	} else if roundElapsed < time.Duration(float32(rd)*propagationAll) {
		_, ok := i.peers.firstStagePeers[who]
		if ok {
			return ok
		}
		_, ok = i.peers.secondStagePeers[who]
		return ok
	} else {
		info := i.peers.peer(who)
		if info == nil {
			return false
		}
		return !info.roles.IsLight()
	}
}

// The initial logic for filtering global messages follows the given state transitions:
//
// - State 1: allowed to max(luckyPeers, sqrt(peers)) (where at least luckyPeers are authorities)
// - State 2: allowed to all peers
//
// We are more lenient with global messages since there should be a lot less global messages than round messages
// (just commits), and we want these to propagate to non-authorities fast enough so that they can observe finality.
//
// Transitions will be triggered on repropagation attempts by the underlying gossip layer, which should happen
// every 30 seconds.
func (i *inner[H, N, Hasher]) globalMessageAllowed(who peerid.PeerID) bool {
	rd := i.config.GossipDuration * time.Duration(roundDuration)
	if i.localView == nil {
		return false
	}
	roundElapsed := time.Since(i.localView.roundStart)
	if roundElapsed < time.Duration(float32(rd)*propagationAll) {
		_, ok := i.peers.firstStagePeers[who]
		if ok {
			return ok
		}
		_, ok = i.peers.secondStagePeers[who]
		if ok {
			return ok
		}
		_, ok = i.peers.luckyLightPeers[who]
		if ok {
			return ok
		}
		return false
	}
	return true
}

// A validator for GRANDPA gossip messages.
type gossipValidator[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	inner[H, N, Hasher]
	innerMtx     sync.RWMutex
	setState     *SharedVoterSetState[H, N]
	reportSender chan peerReport
	// TODO: metrics
	// TODO: telemetry
}

// Create a new gossip-validator. The current set is initialized to 0.
func newGossipValidator[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](
	config Config,
	setState *SharedVoterSetState[H, N],
	// TODO: metrics, telemetry
) (*gossipValidator[H, N, Hasher], chan peerReport) {
	ch := make(chan peerReport, 100_000)
	val := gossipValidator[H, N, Hasher]{
		inner:        newInner[H, N, Hasher](config),
		setState:     setState,
		reportSender: ch,
	}
	return &val, ch
}

// Note a round in the current set has started.
func (gv *gossipValidator[H, N, Hasher]) noteRound(
	round Round, sendNeighbor func(to []peerid.PeerID, msg neighborPacket[N]),
) {
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	msg := gv.inner.noteRound(round)
	if msg != nil {
		sendNeighbor(msg.PeerIDs, msg.NeighborPacket)
	}
}

// Note that a voter set with given ID has started. Updates the current set to given value and initializes the round
// to 0.
func (gv *gossipValidator[H, N, Hasher]) noteSet(
	setID SetID, authorities []primitives.AuthorityID, sendNeighbor func(to []peerid.PeerID, msg neighborPacket[N]),
) {
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	msg := gv.inner.noteSet(setID, authorities)
	if msg != nil {
		sendNeighbor(msg.PeerIDs, msg.NeighborPacket)
	}
}

// Note that we've imported a commit finalizing a given block. `setID` & `round` are the ones the commit message is
// from and not necessarily the latest set ID & round started.
func (gv *gossipValidator[H, N, Hasher]) noteCommitFinalized(
	round Round, setID SetID, finalized N, sendNeighbor func(to []peerid.PeerID, msg neighborPacket[N]),
) {
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	msg := gv.inner.noteCommitFinalized(round, setID, finalized)

	if msg != nil {
		sendNeighbor(msg.PeerIDs, msg.NeighborPacket)
	}
}

// Note that we've processed a catch up message.
func (gv *gossipValidator[H, N, Hasher]) noteCatchUpMessageProcessed() {
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	gv.inner.noteCatchUpMessageProcessed()
}

func (gv *gossipValidator[H, N, Hasher]) report(who peerid.PeerID, costBenefit network.ReputationChange) {
	gv.reportSender <- peerReport{who, costBenefit}
}

func (gv *gossipValidator[H, N, Hasher]) doValidate(who peerid.PeerID, data []byte) (action, []H, gossipMessage) {
	broadcastTopics := make([]H, 0)
	var peerReply gossipMessage

	// message name for Prometheus metric recording.
	var messageName string

	var a action
	var gossipMessage gossipMessageVDT[H, N]
	err := scale.Unmarshal(data, &gossipMessage)
	if err != nil {
		logger.Debugf("Error decoding message: %v", err)
		len := int32(math.Min(float64(math.MaxInt32), float64(len(data))))
		a = actionDiscard[H]{ReputationChange: misbehaviorUndecodablePacket(len).cost()}
	}
	val, err := gossipMessage.Value()
	if err != nil {
		logger.Debugf("Error decoding message: %v", err)
		len := int32(math.Min(float64(math.MaxInt32), float64(len(data))))
		a = actionDiscard[H]{ReputationChange: misbehaviorUndecodablePacket(len).cost()}
	}

	switch val := val.(type) {
	case gossipMessageVote[H, N]:
		messageName = "vote"
		gv.innerMtx.Lock()
		defer gv.innerMtx.Unlock()
		a = gv.inner.validateRoundMessage(who, voteMessage[H, N](val))
	case gossipMessageCommit[H, N]:
		messageName = "commit"
		gv.innerMtx.Lock()
		defer gv.innerMtx.Unlock()
		a = gv.inner.validateCommitMessage(who, fullCommitMessage[H, N](val))
	case gossipMessageNeighbor[N]:
		messageName = "neighbor"
		gv.innerMtx.Lock()
		defer gv.innerMtx.Unlock()
		topics, act, catchUp, report := gv.inner.importNeighborMessage(who, val.neighborPacket)

		if report != nil {
			gv.report(report.PeerID, report.ReputationChange)
		}

		broadcastTopics = topics
		peerReply = catchUp
		a = act
	case gossipMessageCatchUp[H, N]:
		messageName = "catch_up"
		gv.innerMtx.Lock()
		defer gv.innerMtx.Unlock()
		a = gv.inner.validateCatchUpMessage(who, fullCatchUpMessage[H, N](val))
	case gossipMessageCatchUpRequest:
		messageName = "catch_up_request"
		gv.innerMtx.Lock()
		defer gv.innerMtx.Unlock()
		reply, act := gv.inner.handleCatchUpRequest(who, catchUpRequestMessage(val), gv.setState)
		peerReply = reply
		a = act
	default:
		panic("unreachable")
	}

	// TODO: metrics
	_ = messageName

	return a, broadcastTopics, peerReply
}

func (gv *gossipValidator[H, N, Hasher]) NewPeer(
	context gossip.ValidatorContext[H], who peerid.PeerID, roles role.ObservedRole,
) {
	gv.innerMtx.Lock()
	gv.inner.peers.newPeer(who, roles)
	gv.innerMtx.Unlock()

	var packet *neighborPacket[N]
	if gv.inner.localView != nil {
		var height N
		if h := gv.inner.localView.lastCommitHeight(); h != nil {
			height = *h
		}
		packet = &neighborPacket[N]{
			Round:                 gv.inner.localView.round,
			SetID:                 gv.inner.localView.setID,
			CommitFinalizedHeight: height,
		}
	}

	if packet != nil {
		var gossipMessage gossipMessageVDT[H, N]
		setGossipMessageVDT(&gossipMessage, gossipMessageNeighbor[N](
			versionedNeighborPacket[N]{neighborPacket: *packet}))

		packetData := scale.MustMarshal(gossipMessage)
		context.SendMessage(who, packetData)
	}
}

func (gv *gossipValidator[H, N, Hasher]) PeerDisconnected(_ gossip.ValidatorContext[H], who peerid.PeerID) {
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	gv.inner.peers.peerDisconnected(who)
}

func (gv *gossipValidator[H, N, Hasher]) Validate(
	context gossip.ValidatorContext[H],
	who peerid.PeerID,
	data []byte,
) gossip.ValidationResult {
	action, broadcastTopics, peerReply := gv.doValidate(who, data)

	// not with lock held!
	if peerReply != nil {
		var gossipMessage gossipMessageVDT[H, N]
		gossipMessage.inner = peerReply
		context.SendMessage(who, scale.MustMarshal(gossipMessage))
	}

	for _, topic := range broadcastTopics {
		context.SendTopic(who, topic, false)
	}

	switch action := action.(type) {
	case actionKeep[H]:
		gv.report(who, action.ReputationChange)
		context.BroadcastMessage(action.Hash, data, false)
		return gossip.ValidationResultProcessAndKeep[H]{Hash: action.Hash}
	case actionProcessAndDiscard[H]:
		gv.report(who, action.ReputationChange)
		return gossip.ValidationResultProcessAndDiscard[H]{Hash: action.Hash}
	case actionDiscard[H]:
		gv.report(who, action.ReputationChange)
		return gossip.ValidationResultDiscard{}
	default:
		panic("unreachable")
	}
}

func (gv *gossipValidator[H, N, Hasher]) MessageAllowed() func(
	who peerid.PeerID,
	intent gossip.MessageIntent,
	topic H,
	data []byte,
) bool {
	var doRebroadcast bool
	gv.innerMtx.Lock()
	defer gv.innerMtx.Unlock()
	now := time.Now()
	if now.After(gv.inner.nextRebroadcast) || now.Equal(gv.inner.nextRebroadcast) {
		gv.inner.nextRebroadcast = now.Add(rebroadcastAfter)
		doRebroadcast = true
	} else {
		doRebroadcast = false
	}

	return func(who peerid.PeerID, intent gossip.MessageIntent, topic H, data []byte) bool {
		gv.innerMtx.RLock()
		defer gv.innerMtx.RUnlock()

		if intent == gossip.MessageIntentPeriodicRebroadcast {
			return doRebroadcast
		}

		peer := gv.inner.peers.peer(who)
		if peer == nil {
			return false
		}

		// if the topic is not something we're keeping at the moment,do not send.
		entry := gv.inner.liveTopics.topicInfo(topic)
		if entry == nil {
			return false
		}
		maybeRound, setID := entry.Round, entry.SetID

		if intent == gossip.MessageIntentBroadcast {
			if maybeRound != nil {
				// early return if the global message isn't allowed at this stage.
				if !gv.inner.roundMessageAllowed(who) {
					return false
				}
			} else if !gv.inner.globalMessageAllowed(who) {
				// early return if the global message isn't allowed at this stage.
				return false
			}
		}

		// if the topic is not something the peer accepts, discard.
		if maybeRound != nil {
			return peer.view.considerVote(*maybeRound, setID) == considerAccept
		}

		// global message.
		if gv.inner.localView == nil {
			// cannot evaluate until we have a local view.
			return false
		}
		var gossipMessage gossipMessageVDT[H, N]
		err := scale.Unmarshal(data, &gossipMessage)
		if err != nil {
			logger.Debugf("Error decoding message: %v", err)
			return false
		}
		val, err := gossipMessage.Value()
		if err != nil {
			logger.Debugf("no value returned for gossipMessage")
			return false
		}
		switch val := val.(type) {
		case gossipMessageCommit[H, N]:
			lhs := peer.view.considerGlobal(setID, val.Message.TargetNumber) == considerAccept
			var rhs bool
			height := gv.inner.localView.lastCommitHeight()
			if height != nil {
				rhs = val.Message.TargetNumber == *height
			}
			return lhs && rhs
		case gossipMessageNeighbor[N]:
			return false
		case gossipMessageCatchUpRequest:
			return false
		case gossipMessageCatchUp[H, N]:
			return false
		case gossipMessageVote[H, N]:
			// should not be the case.
			return false
		default:
			panic("unreachable")
		}

	}
}

func (gv *gossipValidator[H, N, Hasher]) MessageExpired() func(topic H, data []byte) bool {
	return func(topic H, data []byte) bool {
		gv.innerMtx.RLock()
		defer gv.innerMtx.RUnlock()
		// if the topic is not one of the ones that we are keeping at the moment,
		// it is expired.
		info := gv.inner.liveTopics.topicInfo(topic)
		if info == nil {
			return true
		}
		// round messages don't require further checking.
		if info.Round != nil {
			return false
		}

		if gv.inner.localView == nil {
			// no local view means we can't evaluate or hold any topic.
			return true
		}

		// global messages -- only keep the best commit.
		var gossipMessage gossipMessageVDT[H, N]
		err := scale.Unmarshal(data, &gossipMessage)
		if err != nil {
			return false
		}
		val, err := gossipMessage.Value()
		if err != nil {
			return false
		}
		full, ok := val.(gossipMessageCommit[H, N])
		if !ok {
			return true
		}
		// we expire any commit message that doesn't target the same block
		// as our best commit or isn't from the same round and set id
		if gv.localView.lastCommit != nil {
			return !(full.Message.TargetNumber == gv.localView.lastCommit.Number &&
				full.Round == gv.localView.lastCommit.Round &&
				full.SetID == gv.localView.lastCommit.SetID)
		}
		return true
	}
}

// Report specifying a reputation change for a given peer.
type peerReport struct {
	who         peerid.PeerID
	costBenefit network.ReputationChange
}
