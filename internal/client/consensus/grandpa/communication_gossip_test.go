// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"math"
	"slices"
	"testing"
	"time"

	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var _ gossip.Validator[hash.H256] = &gossipValidator[hash.H256, uint64, runtime.BlakeTwo256]{}

func createUint64(v uint64) *uint64 {
	return &v
}
func Test_view(t *testing.T) {
	t.Run("considerVote", func(t *testing.T) {
		view := view[uint64]{
			round:      Round(100),
			setID:      SetID(1),
			lastCommit: createUint64(1000),
			lastUpdate: nil,
		}

		require.Equal(t, view.considerVote(Round(98), SetID(1)), considerRejectPast)
		require.Equal(t, view.considerVote(Round(1), SetID(0)), considerRejectPast)
		require.Equal(t, view.considerVote(Round(1000), SetID(0)), considerRejectPast)

		require.Equal(t, view.considerVote(Round(99), SetID(1)), considerAccept)
		require.Equal(t, view.considerVote(Round(100), SetID(1)), considerAccept)
		require.Equal(t, view.considerVote(Round(101), SetID(1)), considerAccept)

		require.Equal(t, view.considerVote(Round(102), SetID(1)), considerRejectFuture)
		require.Equal(t, view.considerVote(Round(1), SetID(2)), considerRejectFuture)
		require.Equal(t, view.considerVote(Round(1000), SetID(2)), considerRejectFuture)
	})

	t.Run("considerGlobal", func(t *testing.T) {
		view := view[uint64]{
			round:      Round(100),
			setID:      SetID(2),
			lastCommit: createUint64(1000),
			lastUpdate: nil,
		}

		require.Equal(t, view.considerGlobal(SetID(3), 1), considerRejectFuture)
		require.Equal(t, view.considerGlobal(SetID(3), 1000), considerRejectFuture)
		require.Equal(t, view.considerGlobal(SetID(3), 10000), considerRejectFuture)

		require.Equal(t, view.considerGlobal(SetID(1), 1), considerRejectPast)
		require.Equal(t, view.considerGlobal(SetID(1), 1000), considerRejectPast)
		require.Equal(t, view.considerGlobal(SetID(1), 10000), considerRejectPast)

		require.Equal(t, view.considerGlobal(SetID(2), 1), considerRejectPast)
		require.Equal(t, view.considerGlobal(SetID(2), 1000), considerRejectPast)
		require.Equal(t, view.considerGlobal(SetID(2), 1001), considerAccept)
		require.Equal(t, view.considerGlobal(SetID(2), 10000), considerAccept)
	})
}

func Test_peers(t *testing.T) {
	t.Run("unknown_peer_cannot_be_updated", func(t *testing.T) {
		peers := newPeers[uint64](neighborRebroadcastPeriod)
		id := peerid.NewRandomPeerID()

		update := neighborPacket[uint64]{Round: Round(5), SetID: SetID(10), CommitFinalizedHeight: 50}

		view, err := peers.updatePeerState(id, update)
		require.NoError(t, err)
		require.Nil(t, view)

		// connect & disconnect.
		peers.newPeer(id, role.ObservedRoleAuthority)
		peers.peerDisconnected(id)

		view, err = peers.updatePeerState(id, update)
		require.NoError(t, err)
		require.Nil(t, view)
	})

	t.Run("update_peer_state", func(t *testing.T) {
		update1 := neighborPacket[uint64]{Round: Round(5), SetID: SetID(10), CommitFinalizedHeight: 50}
		update2 := neighborPacket[uint64]{Round: Round(6), SetID: SetID(10), CommitFinalizedHeight: 60}
		update3 := neighborPacket[uint64]{Round: Round(2), SetID: SetID(11), CommitFinalizedHeight: 61}
		update4 := neighborPacket[uint64]{Round: Round(3), SetID: SetID(11), CommitFinalizedHeight: 80}

		// Use shorter rebroadcast period to safely roll the clock back in the last test
		// and don't hit the system boot time on systems with unsigned time.
		const shortNeighborRebroadcastPeriod = 1 * time.Second
		p := newPeers[uint64](shortNeighborRebroadcastPeriod)
		id := peerid.NewRandomPeerID()

		p.newPeer(id, role.ObservedRoleAuthority)

		checkUpdate := func(peers *peers[uint64], update neighborPacket[uint64]) {
			view, err := peers.updatePeerState(id, update)
			require.NoError(t, err)
			require.NotNil(t, view)
			require.Equal(t, update.Round, view.round)
			require.Equal(t, update.SetID, view.setID)
			require.Equal(t, update.CommitFinalizedHeight, *view.lastCommit)
		}

		checkUpdate(&p, update1)
		checkUpdate(&p, update2)
		checkUpdate(&p, update3)
		checkUpdate(&p, update4)

		// Allow duplicate neighbor packets if enough time has passed.
		peerInfo := p.inner[id]
		view := peerInfo.view
		lastUpdate := time.Now().Add(-shortNeighborRebroadcastPeriod)
		view.lastUpdate = &lastUpdate
		peerInfo.view = view
		p.inner[id] = peerInfo

		checkUpdate(&p, update4)
	})

	t.Run("invalid_view_change", func(t *testing.T) {
		p := newPeers[uint64](neighborRebroadcastPeriod)

		id := peerid.NewRandomPeerID()
		p.newPeer(id, role.ObservedRoleAuthority)

		p.updatePeerState(id, neighborPacket[uint64]{Round: Round(10), SetID: SetID(10), CommitFinalizedHeight: 10})

		checkUpdate := func(peers *peers[uint64], update neighborPacket[uint64], misbehavior misbehavior) {
			_, err := peers.updatePeerState(id, update)
			require.Error(t, err)
			require.Equal(t, misbehavior, err)
		}

		// round moves backwards.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(9), SetID: SetID(10), CommitFinalizedHeight: 10},
			misbehaviorInvalidViewChange{})
		// set ID moves backwards.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(10), SetID: SetID(9), CommitFinalizedHeight: 10},
			misbehaviorInvalidViewChange{})
		// commit finalized height moves backwards.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(10), SetID: SetID(10), CommitFinalizedHeight: 9},
			misbehaviorInvalidViewChange{})
		// duplicate packet without grace period.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(10), SetID: SetID(10), CommitFinalizedHeight: 10},
			misbehaviorDuplicateNeighborMessage{})
		// commit finalized height moves backwards while round moves forward.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(11), SetID: SetID(10), CommitFinalizedHeight: 9},
			misbehaviorInvalidViewChange{})
		// commit finalized height moves backwards while set ID moves forward.
		checkUpdate(&p, neighborPacket[uint64]{Round: Round(10), SetID: SetID(11), CommitFinalizedHeight: 9},
			misbehaviorInvalidViewChange{})
	})
}

func Test_gossipValidator(t *testing.T) {
	// some random config (not really needed)
	var config = func() Config {
		return Config{
			GossipDuration:                10 * time.Millisecond,
			JustificationGenerationPeriod: 256,
			KeyStore:                      nil,
			Name:                          nil,
			LocalRole:                     role.RoleAuthority,
			ObserverEnabled:               false,
			ProtocolName:                  "grandpa_protocol_name",
		}
	}

	// dummy voter set state
	var voterSetState = func() *SharedVoterSetState[hash.H256, uint64] {
		baseHash := hash.H256("")
		vaseNumber := uint64(0)

		voters, err := NewGenesisAuthoritySet[hash.H256, uint64](primitives.AuthorityList{
			{
				AuthorityID:     primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
				AuthorityWeight: 1,
			},
		})
		require.NoError(t, err)

		setState := newVoterSetStateLive[hash.H256, uint64](0, *voters, grandpa.HashNumber[hash.H256, uint64]{
			Hash:   baseHash,
			Number: vaseNumber,
		})
		sharedState := NewSharedVoterSetState[hash.H256](setState)
		return sharedState
	}

	t.Run("messages_not_expired_immediately", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		setID := SetID(1)

		val.noteSet(setID, nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		for roundNum := uint64(1); roundNum < 10; roundNum++ {
			val.noteRound(Round(roundNum), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})
		}

		isExpired := val.MessageExpired()
		lastKeptRound := uint64(10) - uint64(keepRecentRounds) - 1

		// messages from old rounds are expired.
		for roundNum := uint64(1); roundNum < lastKeptRound; roundNum++ {
			topic := roundTopic[hash.H256, runtime.BlakeTwo256](Round(roundNum), SetID(1))
			require.True(t, isExpired(topic, []byte{1, 2, 3}))
		}

		// messages from not-too-old rounds are not expired.
		for roundNum := lastKeptRound; roundNum < 10; roundNum++ {
			topic := roundTopic[hash.H256, runtime.BlakeTwo256](Round(roundNum), SetID(1))
			require.False(t, isExpired(topic, []byte{1, 2, 3}))
		}
	})

	t.Run("message_from_unknown_authority_discarded", func(t *testing.T) {
		require.NotEqual(t, unknownVoter, badSignature)

		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())
		setID := SetID(1)
		auth := primitives.AuthorityID(bytes.Repeat([]byte{1}, 32))
		peer := peerid.NewRandomPeerID()

		val.noteSet(setID, []primitives.AuthorityID{auth}, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})
		val.noteRound(Round(1), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		val.innerMtx.RLock()
		defer val.innerMtx.RUnlock()
		unknown := val.inner.validateRoundMessage(
			peer,
			voteMessage[hash.H256, uint64]{
				Round: Round(1),
				SetID: setID,
				Message: primitives.SignedMessage[hash.H256, uint64]{
					Message: grandpa.Prevote[hash.H256, uint64]{
						TargetHash:   hash.H256(""),
						TargetNumber: 10,
					},
					Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
					ID:        primitives.AuthorityID(bytes.Repeat([]byte{2}, 32)),
				},
			},
		)

		badSig := val.inner.validateRoundMessage(
			peer,
			voteMessage[hash.H256, uint64]{
				Round: Round(1),
				SetID: setID,
				Message: primitives.SignedMessage[hash.H256, uint64]{
					Message: grandpa.Prevote[hash.H256, uint64]{
						TargetHash:   hash.H256(""),
						TargetNumber: 10,
					},
					Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
					ID:        auth,
				},
			},
		)

		require.Equal(t, unknown, actionDiscard[hash.H256]{unknownVoter})
		require.Equal(t, badSig, actionDiscard[hash.H256]{badSignature})
	})

	t.Run("unsolicited_catch_up_messages_discarded", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		setID := SetID(1)
		auth := primitives.AuthorityID(bytes.Repeat([]byte{1}, 32))
		peer := peerid.NewRandomPeerID()

		val.noteSet(setID, []primitives.AuthorityID{auth}, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})
		val.noteRound(Round(1), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		validateCatchUp := func() action {
			val.innerMtx.Lock()
			defer val.innerMtx.Unlock()
			return val.inner.validateCatchUpMessage(
				peer,
				fullCatchUpMessage[hash.H256, uint64]{
					SetID: setID,
					Message: primitives.CatchUp[hash.H256, uint64]{
						RoundNumber: 10,
						Prevotes:    nil,
						Precommits:  nil,
						BaseHash:    hash.H256(""),
						BaseNumber:  0,
					},
				},
			)
		}

		// the catch up is discarded because we have no pending request
		require.Equal(t, validateCatchUp(), actionDiscard[hash.H256]{outOfScopeMessage})

		noted, _ := val.inner.noteCatchUpRequest(
			peer,
			catchUpRequestMessage{
				SetID: setID,
				Round: Round(10),
			},
		)
		require.True(t, noted)

		// catch up is allowed because we have requested it, but it's rejected
		// because it's malformed (empty prevotes and precommits)
		require.Equal(t, validateCatchUp(), actionDiscard[hash.H256]{malformedCatchUp})
	})

	t.Run("unanswerable_catch_up_requests_discarded", func(t *testing.T) {
		// create voter set state with round 2 completed
		var setState *SharedVoterSetState[hash.H256, uint64]
		{
			vss := voterSetState()
			vss.innerMtx.RLock()
			defer vss.innerMtx.RUnlock()
			completedRounds := vss.inner.completedRounds()

			completedRounds.push(completedRound[hash.H256, uint64]{
				Number: 2,
				State:  grandpa.NewRoundState(grandpa.HashNumber[hash.H256, uint64]{}),
			})

			currentRounds := currentRounds[hash.H256, uint64]{}
			currentRounds[3] = hasVotedNo[hash.H256, uint64]{}

			live := voterSetStateLive[hash.H256, uint64]{completedRounds, currentRounds}
			setState = NewSharedVoterSetState(live)
		}

		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), setState)

		setID := SetID(1)
		auth := primitives.AuthorityID(bytes.Repeat([]byte{1}, 32))
		peer := peerid.NewRandomPeerID()

		val.noteSet(setID, []primitives.AuthorityID{auth}, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})
		val.noteRound(Round(3), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add the peer making the request to the validator, otherwise it is discarded
		val.innerMtx.Lock()
		defer val.innerMtx.Unlock()
		val.inner.peers.newPeer(peer, role.ObservedRoleAuthority)

		message, action := val.inner.handleCatchUpRequest(
			peer,
			catchUpRequestMessage{SetID: setID, Round: Round(10)},
			setState,
		)

		// we're at round 3, a catch up request for round 10 is out of scope
		require.Nil(t, message)
		require.Equal(t, action, actionDiscard[hash.H256]{outOfScopeMessage})

		message, action = val.inner.handleCatchUpRequest(
			peer,
			catchUpRequestMessage{SetID: setID, Round: Round(2)},
			setState,
		)

		// a catch up request for round 2 should be answered successfully
		require.NotNil(t, message)
		switch message := message.(type) {
		case gossipMessageCatchUp[hash.H256, uint64]:
			require.Equal(t, message.SetID, setID)
			require.Equal(t, message.Message.RoundNumber, uint64(2))
			require.Equal(t, action, actionDiscard[hash.H256]{catchUpReply})
		default:
			t.Fatalf("expected catch up message: %T", message)
		}
	})

	t.Run("detects_honest_out_of_scope_catch_requests", func(t *testing.T) {
		setState := voterSetState()
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), setState)

		// the validator starts at set id 2
		val.noteSet(SetID(2), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add the peer making the request to the validator, otherwise it is discarded
		peer := peerid.NewRandomPeerID()
		val.innerMtx.Lock()
		val.inner.peers.newPeer(peer, role.ObservedRoleAuthority)
		val.innerMtx.Unlock()

		sendRequest := func(setID uint64, round uint64) (gossipMessage, action) {
			val.innerMtx.Lock()
			defer val.innerMtx.Unlock()
			return val.inner.handleCatchUpRequest(
				peer,
				catchUpRequestMessage{SetID: SetID(setID), Round: Round(round)},
				setState,
			)
		}

		assertRes := func(res gossipMessage, a action, honest bool) {
			require.Nil(t, res)
			if honest {
				require.Equal(t, actionDiscard[hash.H256]{honestOutOfScopeCatchUp}, a)
			} else {
				require.Equal(t, actionDiscard[hash.H256]{outOfScopeMessage}, a)
			}
		}

		// the validator is at set id 2 and round 0. requests for set id 1
		// should not be answered but they should be considered an honest
		// mistake
		msg, a := sendRequest(1, 1)
		assertRes(msg, a, true)

		msg, a = sendRequest(1, 10)
		assertRes(msg, a, true)

		// requests for set id 0 should be considered out of scope
		msg, a = sendRequest(0, 1)
		assertRes(msg, a, false)

		msg, a = sendRequest(0, 10)
		assertRes(msg, a, false)

		// after the validator progresses further than catchUpThreshold in set
		// id 2, any request for set id 1 should no longer be considered an
		// honest mistake.
		val.noteRound(Round(3), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		msg, a = sendRequest(1, 1)
		assertRes(msg, a, false)

		msg, a = sendRequest(1, 2)
		assertRes(msg, a, false)
	})

	t.Run("issues_catch_up_request_on_neighbor_packet_import", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		// the validator starts at set id 1.
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add the peer making the request to the validator,
		// otherwise it is discarded.
		peer := peerid.NewRandomPeerID()
		val.innerMtx.Lock()
		val.inner.peers.newPeer(peer, role.ObservedRoleAuthority)
		val.innerMtx.Unlock()

		importNeighborMessage := func(setID uint64, round uint64) gossipMessage {
			_, _, catchUpRequest, _ := val.inner.importNeighborMessage(
				peer,
				neighborPacket[uint64]{Round: Round(round), SetID: SetID(setID), CommitFinalizedHeight: 42},
			)
			return catchUpRequest
		}

		// importing a neighbor message from a peer in the same set in a later
		// round should lead to a catch up request for the previous round.
		switch msg := importNeighborMessage(1, 42).(type) {
		case gossipMessageCatchUpRequest:
			require.Equal(t, SetID(1), msg.SetID)
			require.Equal(t, Round(41), msg.Round)
		default:
			t.Fatalf("expected catch up message: %T", msg)
		}

		// we note that we're at round 41.
		val.noteRound(Round(41), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// if we import a neighbor message within catchUpThreshold then we
		// won't request a catch up.
		switch msg := importNeighborMessage(1, 42).(type) {
		case nil:
		default:
			t.Fatalf("expected no catch up message: %T", msg)
		}

		// or if the peer is on a lower round.
		switch msg := importNeighborMessage(1, 40).(type) {
		case nil:
		default:
			t.Fatalf("expected no catch up message: %T", msg)
		}

		// we also don't request a catch up if the peer is in a different set.
		switch msg := importNeighborMessage(2, 42).(type) {
		case nil:
		default:
			t.Fatalf("expected no catch up message: %T", msg)
		}
	})

	t.Run("sends_catch_up_requests_to_non_authorities", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add the peer making the requests to the validator, otherwise it is
		// discarded.
		peerFull := peerid.NewRandomPeerID()
		val.innerMtx.Lock()
		defer val.innerMtx.Unlock()
		val.inner.peers.newPeer(peerFull, role.ObservedRoleFull)

		_, _, catchUpRequest, _ := val.inner.importNeighborMessage(
			peerFull,
			neighborPacket[uint64]{Round: Round(42), SetID: SetID(1), CommitFinalizedHeight: 50},
		)

		// importing a neighbor message from a peer in the same set in a later
		// round should lead to a catch up request, the node is not an
		// authority, but since the observer protocol is disabled we should
		// issue a catch-up request to it anyway.
		request, ok := catchUpRequest.(gossipMessageCatchUpRequest)
		require.True(t, ok)
		require.Equal(t, request.SetID, SetID(1))
		require.Equal(t, request.Round, Round(41))
	})

	t.Run("doesnt_expire_next_round_messages", func(t *testing.T) {
		// NOTE: this is a regression test
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		// the validator starts at set id 1.
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// we are at round 10
		val.noteRound(Round(9), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})
		val.noteRound(Round(10), func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		isExpired := val.MessageExpired()

		// we accept messages from rounds 9, 10 and 11
		// therefore neither of those should be considered expired
		for _, round := range []uint64{9, 10, 11} {
			require.False(t, isExpired(roundTopic[hash.H256, runtime.BlakeTwo256](Round(round), SetID(1)), []byte{}))
		}
	})

	t.Run("progressively_gossips_to_more_peers_as_round_duration_increases", func(t *testing.T) {
		config := config()
		config.GossipDuration = 300 * time.Second // Set to high value to prevent test race
		roundDuration := config.GossipDuration * time.Duration(roundDuration)

		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config, voterSetState())

		// the validator start at set id 0
		val.noteSet(SetID(0), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add 60 peers, 30 authorities and 30 full nodes
		authorities := make([]peerid.PeerID, 30)
		for i := 0; i < 30; i++ {
			authorities[i] = peerid.NewRandomPeerID()
		}

		fullNodes := make([]peerid.PeerID, 30)
		for i := 0; i < 30; i++ {
			fullNodes[i] = peerid.NewRandomPeerID()
		}

		for i := 0; i < 30; i++ {
			val.innerMtx.Lock()
			val.inner.peers.newPeer(authorities[i], role.ObservedRoleAuthority)
			val.inner.peers.newPeer(fullNodes[i], role.ObservedRoleFull)
			val.innerMtx.Unlock()
		}

		test := func(roundsElapsed float64, peers []peerid.PeerID) func() uint {
			// rewind n round durations
			val.innerMtx.Lock()

			val.inner.localView.roundStart = time.Now().Add(
				-time.Duration(float64(roundDuration.Milliseconds())*roundsElapsed) * time.Millisecond,
			)
			val.inner.peers.reshuffle()
			val.innerMtx.Unlock()

			messageAllowed := val.MessageAllowed()

			return func() uint {
				allowed := uint(0)
				for _, peer := range peers {
					if messageAllowed(
						peer,
						gossip.MessageIntentBroadcast,
						roundTopic[hash.H256, runtime.BlakeTwo256](Round(1), SetID(0)),
						nil,
					) {
						allowed++
					}
				}
				return allowed
			}
		}

		trial := func(test func() uint) uint {
			results := make([]uint, 1000)
			for i := 0; i < 1000; i++ {
				results[i] = test()
			}
			n := uint(len(results))
			sum := uint(0)
			for _, r := range results {
				sum += r
			}
			return sum / n
		}

		allPeers := slices.Concat(authorities, fullNodes)

		// on the first attempt we will only gossip to 4 peers, either
		// authorities or full nodes, but we'll guarantee that half of those
		// are authorities
		require.True(t, trial(test(1.0, authorities)) >= luckyPeers/2)
		require.Equal(t, trial(test(1.0, allPeers)), luckyPeers)

		// after more than 1.5 round durations have elapsed we should gossip to
		// `sqrt(peers)` we're connected to, but we guarantee that at least 4 of
		// those peers are authorities (plus the `luckyPeers` from the previous
		// stage)
		require.True(t, trial(test(float64(propagationSome)*1.1, authorities)) >= luckyPeers)
		require.Equal(t, trial(test(2.0, allPeers)), luckyPeers+uint(math.Sqrt(float64(len(allPeers)))))

		// after 3 rounds durations we should gossip to all peers we are
		// connected to
		require.Equal(t, trial(test(float64(propagationAll)*1.1, allPeers)), uint(len(allPeers)))
	})

	t.Run("never_gossips_round_messages_to_light_clients", func(t *testing.T) {
		config := config()
		roundDuration := config.GossipDuration * time.Duration(roundDuration)
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config, voterSetState())

		// the validator starts at set id 0
		val.noteSet(SetID(0), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// add a new light client as peer
		lightPeer := peerid.NewRandomPeerID()

		val.innerMtx.Lock()
		val.inner.peers.newPeer(lightPeer, role.ObservedRoleLight)
		val.innerMtx.Unlock()

		messageAllowed := val.MessageAllowed()
		require.False(t, messageAllowed(
			lightPeer,
			gossip.MessageIntentBroadcast,
			roundTopic[hash.H256, runtime.BlakeTwo256](Round(1), SetID(0)),
			nil,
		))

		// we reverse the round start time so that the elapsed time is higher
		// (which should lead to more peers getting the message)
		val.innerMtx.Lock()
		val.inner.localView.roundStart = time.Now().Add(-roundDuration * 10)
		val.innerMtx.Unlock()

		// even after the round has been going for 10 round durations we will never
		// gossip to light clients
		messageAllowed = val.MessageAllowed()
		require.False(t, messageAllowed(
			lightPeer,
			gossip.MessageIntentBroadcast,
			roundTopic[hash.H256, runtime.BlakeTwo256](Round(1), SetID(0)),
			nil,
		))

		// update the peer state and local state wrt commits
		val.innerMtx.Lock()
		val.inner.peers.updatePeerState(
			lightPeer,
			neighborPacket[uint64]{Round: Round(1), SetID: SetID(0), CommitFinalizedHeight: 1},
		)
		val.innerMtx.Unlock()

		val.noteCommitFinalized(Round(1), SetID(0), 2, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		var commit []byte
		{
			cc := primitives.CompactCommit[hash.H256, uint64]{
				TargetHash:   hash.NewRandomH256(),
				TargetNumber: 2,
				Precommits:   nil,
				AuthData:     nil,
			}

			var gossipMessage gossipMessageVDT[hash.H256, uint64]
			gossipMessage.inner = gossipMessageCommit[hash.H256, uint64](
				fullCommitMessage[hash.H256, uint64]{
					Round:   Round(2),
					SetID:   SetID(0),
					Message: cc,
				},
			)
			commit = scale.MustMarshal(gossipMessage)
		}

		// global messages are gossiped to light clients though
		messageAllowed = val.MessageAllowed()
		require.True(t, messageAllowed(
			lightPeer,
			gossip.MessageIntentBroadcast,
			globalTopic[hash.H256, runtime.BlakeTwo256](0),
			commit,
		))
	})

	t.Run("only_gossip_commits_to_peers_on_same_set", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		// the validator starts at set id 1
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// 	add a new peer at set id 1
		peer1 := peerid.NewRandomPeerID()

		// 	val.inner.write().peers.new_peer(peer1, ObservedRole::Authority);
		val.innerMtx.Lock()
		val.inner.peers.newPeer(peer1, role.ObservedRoleAuthority)
		val.innerMtx.Unlock()

		val.innerMtx.Lock()
		val.inner.peers.updatePeerState(
			peer1,
			neighborPacket[uint64]{Round: Round(1), SetID: SetID(1), CommitFinalizedHeight: 1},
		)
		val.innerMtx.Unlock()

		// peer2 will default to set id 0
		peer2 := peerid.NewRandomPeerID()
		val.innerMtx.Lock()
		val.inner.peers.newPeer(peer2, role.ObservedRoleAuthority)
		val.innerMtx.Unlock()

		// create a commit for round 1 of set id 1
		// targeting a block at height 2
		var commit []byte
		{
			cc := primitives.CompactCommit[hash.H256, uint64]{
				TargetHash:   hash.NewRandomH256(),
				TargetNumber: 2,
				Precommits:   nil,
				AuthData:     nil,
			}

			var gossipMessage gossipMessageVDT[hash.H256, uint64]
			gossipMessage.inner = gossipMessageCommit[hash.H256, uint64](
				fullCommitMessage[hash.H256, uint64]{
					Round:   Round(1),
					SetID:   SetID(1),
					Message: cc,
				},
			)
			commit = scale.MustMarshal(gossipMessage)
		}

		// note the commit in the validator
		val.noteCommitFinalized(Round(1), SetID(1), 2, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		messageAllowed := val.MessageAllowed()

		// the commit should be allowed to peer 1
		require.True(t, messageAllowed(
			peer1,
			gossip.MessageIntentBroadcast,
			globalTopic[hash.H256, runtime.BlakeTwo256](1),
			commit,
		))

		// but disallowed to peer 2 since the peer is on set id 0
		// the commit should be allowed to peer 1
		require.False(t, messageAllowed(
			peer2,
			gossip.MessageIntentBroadcast,
			globalTopic[hash.H256, runtime.BlakeTwo256](1),
			commit,
		))
	})

	t.Run("expire_commits_from_older_rounds", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		var commit = func(round uint64, setID uint64, targetNumber uint64) []byte {
			commit := primitives.CompactCommit[hash.H256, uint64]{
				TargetHash:   hash.NewRandomH256(),
				TargetNumber: targetNumber,
				Precommits:   nil,
				AuthData:     nil,
			}
			var gossipMessage gossipMessageVDT[hash.H256, uint64]
			gossipMessage.inner = gossipMessageCommit[hash.H256, uint64](
				fullCommitMessage[hash.H256, uint64]{
					Round:   Round(round),
					SetID:   SetID(setID),
					Message: commit,
				},
			)
			return scale.MustMarshal(gossipMessage)
		}

		// note the beginning of a new set with id 1
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// note a commit for round 1 in the validator
		// finalizing a block at height 2
		val.noteCommitFinalized(Round(1), SetID(1), 2, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		messageExpired := val.MessageExpired()

		// a commit message for round 1 that finalizes the same height as we
		// have observed previously should not be expired
		require.False(t, messageExpired(globalTopic[hash.H256, runtime.BlakeTwo256](1), commit(1, 1, 2)))

		// it should be expired if it is for a lower block
		require.True(t, messageExpired(globalTopic[hash.H256, runtime.BlakeTwo256](1), commit(1, 1, 1)))

		// or the same block height but from the previous round
		require.True(t, messageExpired(globalTopic[hash.H256, runtime.BlakeTwo256](1), commit(0, 1, 2)))
	})

	t.Run("allow_noting_different_authorities_for_same_set", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		a1 := []primitives.AuthorityID{primitives.AuthorityID(bytes.Repeat([]byte{0}, 32))}
		val.noteSet(SetID(1), a1, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		// 	assert_eq!(val.inner().read().authorities, a1);
		require.Equal(t, val.inner.authorities, a1)

		a2 := []primitives.AuthorityID{
			primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
			primitives.AuthorityID(bytes.Repeat([]byte{2}, 32)),
		}
		val.noteSet(SetID(1), a2, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		require.Equal(t, val.inner.authorities, a2)
	})

	t.Run("sends_neighbor_packets_to_all_peers_when_starting_a_new_round", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		// initialize the validator to a stable set id
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		authorityPeer := peerid.NewRandomPeerID()
		fullPeer := peerid.NewRandomPeerID()
		lightPeer := peerid.NewRandomPeerID()

		val.innerMtx.Lock()
		val.inner.peers.newPeer(authorityPeer, role.ObservedRoleAuthority)
		val.inner.peers.newPeer(fullPeer, role.ObservedRoleFull)
		val.inner.peers.newPeer(lightPeer, role.ObservedRoleLight)
		val.innerMtx.Unlock()

		val.noteRound(Round(2), func(to []peerid.PeerID, msg neighborPacket[uint64]) {
			require.Equal(t, len(to), 3)
			require.Contains(t, to, authorityPeer)
			require.Contains(t, to, fullPeer)
			require.Contains(t, to, lightPeer)
			require.Equal(t, neighborPacket[uint64]{
				Round: Round(2),
				SetID: SetID(1),
			}, msg)
		})
	})

	t.Run("sends_neighbor_packets_to_all_peers_when_starting_a_new_set", func(t *testing.T) {
		val, _ := newGossipValidator[hash.H256, uint64, runtime.BlakeTwo256](config(), voterSetState())

		// initialize the validator to a stable set id
		val.noteSet(SetID(1), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {})

		authorityPeer := peerid.NewRandomPeerID()
		fullPeer := peerid.NewRandomPeerID()
		lightPeer := peerid.NewRandomPeerID()

		val.innerMtx.Lock()
		val.inner.peers.newPeer(authorityPeer, role.ObservedRoleAuthority)
		val.inner.peers.newPeer(fullPeer, role.ObservedRoleFull)
		val.inner.peers.newPeer(lightPeer, role.ObservedRoleLight)
		val.innerMtx.Unlock()

		val.noteSet(SetID(2), nil, func(to []peerid.PeerID, msg neighborPacket[uint64]) {
			require.Equal(t, len(to), 3)
			require.Contains(t, to, authorityPeer)
			require.Contains(t, to, fullPeer)
			require.Contains(t, to, lightPeer)
			require.Equal(t, neighborPacket[uint64]{
				Round: Round(1),
				SetID: SetID(2),
			}, msg)
		})
	})
}

func Test_gossipMessageVDT(t *testing.T) {
	var neighbor = gossipMessageNeighbor[uint64](
		versionedNeighborPacket[uint64]{
			neighborPacket: neighborPacket[uint64]{
				Round:                 Round(100),
				SetID:                 SetID(1),
				CommitFinalizedHeight: 1000,
			},
		},
	)
	var gossipMesssage gossipMessageVDT[hash.H256, uint64]
	gossipMesssage.inner = neighbor

	bytes := scale.MustMarshal(gossipMesssage)
	require.NotEmpty(t, bytes)

	var dst gossipMessageVDT[hash.H256, uint64]
	require.NoError(t, scale.Unmarshal(bytes, &dst))
	val, err := dst.Value()
	require.NoError(t, err)

	require.Equal(t, neighbor, val)
}
