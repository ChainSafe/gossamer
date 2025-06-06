// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/network"
	gossip "github.com/ChainSafe/gossamer/internal/client/network-gossip"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	"github.com/ChainSafe/gossamer/internal/client/network/sync"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type Event interface {
	isEvent()
}
type EventWriteNotification struct {
	peerid.PeerID
	Notification []byte
}
type EventReport struct {
	peerid.PeerID
	network.ReputationChange
}

func (EventWriteNotification) isEvent() {}
func (EventReport) isEvent()            {}

type TestNetwork struct {
	sender chan Event
}

func (TestNetwork) SetAuthorizedPeers(peers map[peerid.PeerID]struct{})            { panic("unimpl") }
func (TestNetwork) SetAuthorizedOnly(reservedOnly bool)                            { panic("unimpl") }
func (TestNetwork) AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr) { panic("unimpl") }
func (tn *TestNetwork) ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange) {
	tn.sender <- EventReport{peerID, costBenefit}
}
func (TestNetwork) PeerReputation(peerID peerid.PeerID) int32                       { panic("unimpl") }
func (TestNetwork) DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName) { panic("unimpl") }
func (TestNetwork) AcceptUnreservedPeers()                                          { panic("unimpl") }
func (TestNetwork) DenyUnreservedPeers()                                            { panic("unimpl") }
func (TestNetwork) AddReservedPeer(peer config.MultiaddrPeerId) error               { panic("unimpl") }
func (TestNetwork) RemoveReservedPeer(peerID peerid.PeerID)                         { panic("unimpl") }
func (TestNetwork) SetReservedPeers(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) AddPeersToReservedSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) RemovePeersFromReservedSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (TestNetwork) AddToPeersSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (TestNetwork) RemoveFromPeersSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (TestNetwork) SyncNumConnected() uint { panic("unimpl") }
func (TestNetwork) PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole {
	var roles role.Roles
	err := scale.Unmarshal(handshake, &roles)
	if err != nil {
		return nil
	}
	role := roles.ObservedRole()
	return &role
}
func (TestNetwork) ReservedPeers() <-chan struct {
	Peers []peerid.PeerID
	Error error
} {
	panic("unimpl")
}

func (TestNetwork) EventStream(name string) chan event.Event {
	ch := make(chan event.Event)
	return ch
}
func (TestNetwork) AnnounceBlock(hash hash.H256, data []byte)                       { panic("unimpl") }
func (TestNetwork) NewBestBlockImported(hash hash.H256, number uint64)              { panic("unimpl") }
func (TestNetwork) AddSetReserved(who peerid.PeerID, protocol network.ProtocolName) { panic("unimpl") }
func (TestNetwork) RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("unimpl")
}

func (TestNetwork) SetSyncForkRequest(peers []peerid.PeerID, hash hash.H256, number uint64) {}

func (TestNetwork) BroadcastTopic(topic hash.H256, force bool)                   {}
func (TestNetwork) BroadcastMessage(topic hash.H256, message []byte, force bool) {}
func (tn *TestNetwork) SendMessage(who peerid.PeerID, message []byte) {
	tn.sender <- EventWriteNotification{who, message}
}
func (TestNetwork) SendTopic(who peerid.PeerID, topic hash.H256, force bool) {}

var (
	_ service.NetworkPeers                              = &TestNetwork{}
	_ service.NetworkEventStream                        = &TestNetwork{}
	_ service.NetworkBlock[hash.H256, uint64]           = &TestNetwork{}
	_ service.NetworkSyncForkRequest[hash.H256, uint64] = &TestNetwork{}
	_ gossip.ValidatorContext[hash.H256]                = &TestNetwork{}
	_ Network                                           = &TestNetwork{}
)

type TestSync struct{}

func (TestSync) EventStream(name string) chan sync.SyncEvent {
	ch := make(chan sync.SyncEvent)
	return ch
}

func (TestSync) AnnounceBlock(hash hash.H256, data []byte)                               { panic("unimpl") }
func (TestSync) NewBestBlockImported(hash hash.H256, number uint64)                      { panic("unimpl") }
func (TestSync) AddSetReserved(who peerid.PeerID, protocol network.ProtocolName)         { panic("unimpl") }
func (TestSync) RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName)      { panic("unimpl") }
func (TestSync) SetSyncForkRequest(peers []peerid.PeerID, hash hash.H256, number uint64) {}

var _ Syncing[hash.H256, uint64] = &TestSync{}

type TestNotificationService struct {
	sender chan Event
	rx     chan service.NotificationEvent
}

func (TestNotificationService) OpenSubstream(peer peerid.PeerID) <-chan error  { panic("unimpl") }
func (TestNotificationService) CloseSubstream(peer peerid.PeerID) <-chan error { panic("unimpl") }
func (tns *TestNotificationService) SendSyncNotification(peer peerid.PeerID, notification []byte) {
	tns.sender <- EventWriteNotification{peer, notification}
}
func (TestNotificationService) SendAsyncNotification(peer peerid.PeerID, notification []byte) <-chan error {
	panic("unimpl")
}
func (TestNotificationService) SetHandshake(handshake []byte) <-chan error { panic("unimpl") }
func (TestNotificationService) TrySetHandshake(handshake []byte) error     { panic("unimpl") }
func (tns *TestNotificationService) NextEvent() <-chan service.NotificationEvent {
	return tns.rx
}
func (TestNotificationService) Protocol() network.ProtocolName                     { panic("unimpl") }
func (TestNotificationService) MessageSink(peer peerid.PeerID) service.MessageSink { panic("unimpl") }

type Tester struct {
	*networkBridge[hash.H256, uint64, runtime.BlakeTwo256]
	*gossipValidator[hash.H256, uint64, runtime.BlakeTwo256]
	events         chan Event
	notificationTx chan service.NotificationEvent
}

func (t *Tester) TriggerGossipValidatorReputationChange(p peerid.PeerID) {
	t.gossipValidator.Validate(NoopContext{}, p, []byte{1, 2, 3})
}

// some random config (not really needed)
func getConfig() Config {
	return Config{
		GossipDuration:                10 * time.Millisecond,
		JustificationGenerationPeriod: 256,
		KeyStore:                      nil,
		Name:                          nil,
		LocalRole:                     role.RoleAuthority,
		ProtocolName:                  "grandpa_protocol_name",
	}
}

// dummy voter set state
func sharedVoterSetState(t *testing.T) *SharedVoterSetState[hash.H256, uint64] {
	state := grandpa.NewRoundState[hash.H256, uint64](
		grandpa.HashNumber[hash.H256, uint64]{Hash: "", Number: 0})
	base := state.PrevoteGHOST
	require.NotNil(t, base)

	auths := []pgrandpa.AuthorityIDWeight{{
		AuthorityID: pgrandpa.AuthorityID(bytes.Repeat([]byte{1}, 32)), AuthorityWeight: 1},
	}
	voters, err := NewGenesisAuthoritySet[hash.H256, uint64](auths)
	require.NoError(t, err)

	setState := newVoterSetStateLive(0, *voters, *base)

	return NewSharedVoterSetState(setState)
}

type NoopContext struct{}

func (NoopContext) BroadcastTopic(topic hash.H256, force bool)                   {}
func (NoopContext) BroadcastMessage(topic hash.H256, message []byte, force bool) {}
func (NoopContext) SendMessage(who peerid.PeerID, message []byte)                {}
func (NoopContext) SendTopic(who peerid.PeerID, topic hash.H256, force bool)     {}

// needs to run in a tokio runtime.
func makeTestNetwork(t *testing.T) (*Tester, *TestNetwork) {
	events := make(chan Event, 100)
	notificationEvents := make(chan service.NotificationEvent)

	notificationService := TestNotificationService{sender: events, rx: notificationEvents}
	net := TestNetwork{sender: events}

	bridge := newNetworkBridge[hash.H256, uint64, runtime.BlakeTwo256](
		&net,
		TestSync{},
		&notificationService,
		getConfig(),
		sharedVoterSetState(t),
	)

	return &Tester{
		networkBridge:   bridge,
		gossipValidator: bridge.validator,
		events:          events,
		notificationTx:  notificationEvents,
	}, &net
}

func makeIDs(keys []ed25519.Keyring) []grandpa.IDWeight[pgrandpa.AuthorityID] {
	var ids []grandpa.IDWeight[pgrandpa.AuthorityID]
	for _, key := range keys {
		ids = append(ids, grandpa.IDWeight[pgrandpa.AuthorityID]{
			ID:     key.Public(),
			Weight: 1,
		})
	}
	return ids
}

func Test_networkBridge(t *testing.T) {
	t.Run("good_commit_leads_to_relay", func(t *testing.T) {
		private := []ed25519.Keyring{ed25519.Alice, ed25519.Bob, ed25519.Charlie}
		public := makeIDs(private)
		voterSet := grandpa.NewVoterSet(public)
		require.NotNil(t, voterSet)

		round := Round(1)
		setID := SetID(1)

		var commit pgrandpa.CompactCommit[hash.H256, uint64]
		{
			targetHash := hash.H256(bytes.Repeat([]byte{1}, 32))
			targetNumber := uint64(500)

			precommit := grandpa.Precommit[hash.H256, uint64]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
			}
			payload := pgrandpa.NewLocalizedPayload(
				pgrandpa.RoundNumber(round),
				pgrandpa.SetID(setID),
				precommit,
			)

			precommits := make([]grandpa.Precommit[hash.H256, uint64], 0)
			authData := make(grandpa.MultiAuthData[pgrandpa.AuthoritySignature, pgrandpa.AuthorityID], 0)

			for i, key := range private {
				precommits = append(precommits, precommit)
				signature := key.Sign(payload)
				authData = append(authData, grandpa.SignatureID[pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]{
					Signature: signature,
					ID:        public[i].ID,
				})
			}

			commit = pgrandpa.CompactCommit[hash.H256, uint64]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
				Precommits:   precommits,
				AuthData:     authData,
			}
		}

		var commitVDT gossipMessageVDT[hash.H256, uint64]
		commitVDT.inner = gossipMessageCommit[hash.H256, uint64](fullCommitMessage[hash.H256, uint64]{
			Round:   round,
			SetID:   setID,
			Message: commit,
		})
		encodedCommit := scale.MustMarshal(commitVDT)

		id := peerid.NewRandomPeerID()
		globalTopic := globalTopic[hash.H256, runtime.BlakeTwo256](setID)

		tester, network := makeTestNetwork(t)
		_ = network

		// register a peer.
		tester.gossipValidator.NewPeer(NoopContext{}, id, role.ObservedRoleFull)

		// start round, dispatch commit, and wait for broadcast.
		commitsIn, commitsOut := tester.networkBridge.globalCommunication(setID, *voterSet, false)
		_ = commitsOut

		{
			action, _, _ := tester.gossipValidator.doValidate(id, encodedCommit)
			switch action := action.(type) {
			case actionProcessAndDiscard[hash.H256]:
				require.Equal(t, globalTopic, action.Hash)
			default:
				t.Fatalf("wrong expected outcome from initial commit validation")
			}
		}

		commitToSend := encodedCommit
		networkBridge := tester.networkBridge
		_ = networkBridge

		// `networkBridge` will be operational as soon as it's created and it's waiting for events from the network.
		// Send it events that inform that a notification stream was opened and that a notification was received.
		// Since each protocol has its own notification stream, events need not be filtered.
		senderID := id

		var sendMessage = func() {
			tester.notificationTx <- service.NotificationEventNotificationStreamOpened{
				Peer:               senderID,
				Direction:          service.DirectionInbound,
				NegotiatedFallback: nil,
				Handshake:          scale.MustMarshal(role.RolesFull),
			}

			tester.notificationTx <- service.NotificationEventNotificationReceived{
				Peer:         senderID,
				Notification: commitToSend,
			}

			// Add a random peer which will be the recipient of this message
			receiverID := peerid.NewRandomPeerID()
			tester.notificationTx <- service.NotificationEventNotificationStreamOpened{
				Peer:               receiverID,
				Direction:          service.DirectionInbound,
				NegotiatedFallback: nil,
				Handshake:          scale.MustMarshal(role.RolesFull),
			}

			// Announce its local set being on the current set id through a neighbor packet, otherwise it won't be
			// eligible to receive the commit
			{
				update := versionedNeighborPacket[uint64]{
					neighborPacket: neighborPacket[uint64]{
						Round:                 round,
						SetID:                 setID,
						CommitFinalizedHeight: 1,
					},
				}
				msg := gossipMessageNeighbor[uint64](update)
				gossipMsg := gossipMessageVDT[hash.H256, uint64]{inner: msg}
				tester.notificationTx <- service.NotificationEventNotificationReceived{
					Peer:         receiverID,
					Notification: scale.MustMarshal(gossipMsg),
				}
			}
		}
		// when the commit comes in, we'll tell the callback it was good.
		var handleCommit = func() {
			for item := range commitsIn {
				switch item := item.(type) {
				case grandpa.CommunicationInCommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]:
					item.Callback(grandpa.CommitProcessingOutcomeGood{})
				default:
					panic("commit expected")
				}
			}
		}

		go sendMessage()
		go handleCommit()

		timer := time.NewTimer(10 * time.Second)
		var verified bool
	loop:
		for {
			select {
			case event, ok := <-tester.events:
				if !ok {
					break loop
				}
				switch event := event.(type) {
				case EventWriteNotification:
					if bytes.Equal(encodedCommit, event.Notification) {
						verified = true
						break loop
					}
				default:
				}
			case <-timer.C:
				break loop
			}
		}

		require.True(t, verified)
	})

	t.Run("bad_commit_leads_to_report", func(t *testing.T) {
		private := []ed25519.Keyring{ed25519.Alice, ed25519.Bob, ed25519.Charlie}
		public := makeIDs(private)
		voterSet := grandpa.NewVoterSet(public)

		round := Round(1)
		setID := SetID(1)

		var commit pgrandpa.CompactCommit[hash.H256, uint64]
		{
			targetHash := hash.H256(bytes.Repeat([]byte{1}, 32))
			targetNumber := uint64(500)

			precommit := grandpa.Precommit[hash.H256, uint64]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
			}
			payload := pgrandpa.NewLocalizedPayload(
				pgrandpa.RoundNumber(round),
				pgrandpa.SetID(setID),
				precommit,
			)

			precommits := make([]grandpa.Precommit[hash.H256, uint64], 0)
			authData := make(grandpa.MultiAuthData[pgrandpa.AuthoritySignature, pgrandpa.AuthorityID], 0)

			for i, key := range private {
				precommits = append(precommits, precommit)
				signature := key.Sign(payload)
				authData = append(authData, grandpa.SignatureID[pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]{
					Signature: signature,
					ID:        public[i].ID,
				})
			}

			commit = pgrandpa.CompactCommit[hash.H256, uint64]{
				TargetHash:   targetHash,
				TargetNumber: targetNumber,
				Precommits:   precommits,
				AuthData:     authData,
			}
		}

		var commitVDT gossipMessageVDT[hash.H256, uint64]
		commitVDT.inner = gossipMessageCommit[hash.H256, uint64](fullCommitMessage[hash.H256, uint64]{
			Round:   round,
			SetID:   setID,
			Message: commit,
		})
		encodedCommit := scale.MustMarshal(commitVDT)

		id := peerid.NewRandomPeerID()
		globalTopic := globalTopic[hash.H256, runtime.BlakeTwo256](setID)

		tester, _ := makeTestNetwork(t)

		// register a peer.
		tester.gossipValidator.NewPeer(NoopContext{}, id, role.ObservedRoleFull)

		// start round, dispatch commit, and wait for broadcast.
		commitsIn, _ := tester.networkBridge.globalCommunication(setID, *voterSet, false)

		{
			action, _, _ := tester.gossipValidator.doValidate(id, encodedCommit)
			switch action := action.(type) {
			case actionProcessAndDiscard[hash.H256]:
				require.Equal(t, globalTopic, action.Hash)
			default:
				t.Fatalf("wrong expected outcome from initial commit validation")
			}
		}

		commitToSend := encodedCommit

		// `NetworkBridge` will be operational as soon as it's created and it's waiting for events from the network.
		// Send it events that inform that a notification stream was opened and that a notification was received.
		//
		// Since each protocol has its own notification stream, events need not be filtered.
		senderID := id

		var sendMessage = func() {
			tester.notificationTx <- service.NotificationEventNotificationStreamOpened{
				Peer:               senderID,
				Direction:          service.DirectionInbound,
				NegotiatedFallback: nil,
				Handshake:          scale.MustMarshal(role.RolesFull),
			}

			tester.notificationTx <- service.NotificationEventNotificationReceived{
				Peer:         senderID,
				Notification: commitToSend,
			}
		}

		// when the commit comes in, we'll tell the callback it was bad.
		var handleCommit = func() {
			for item := range commitsIn {
				switch item := item.(type) {
				case grandpa.CommunicationInCommit[hash.H256, uint64, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]:
					item.Callback(grandpa.CommitProcessingOutcomeBad{})
				default:
					panic("commit expected")
				}
			}
		}

		go sendMessage()
		go handleCommit()

		// once the message is sent and commit is "handled" we should have a report event coming from the network.
		timer := time.NewTimer(10 * time.Second)
		var verified bool
	loop:
		for {
			select {
			case event, ok := <-tester.events:
				if !ok {
					break loop
				}
				switch event := event.(type) {
				case EventReport:
					if event.PeerID == id && event.ReputationChange == invalidCommit {
						verified = true
						break loop
					}
				default:
				}
			case <-timer.C:
				break loop
			}
		}

		require.True(t, verified)
	})

	t.Run("peer_with_higher_view_leads_to_catch_up_request", func(t *testing.T) {
		id := peerid.NewRandomPeerID()

		tester, network := makeTestNetwork(t)

		// register a peer with authority role.
		tester.gossipValidator.NewPeer(NoopContext{}, id, role.ObservedRoleAuthority)

		// send neighbor message at round 10 and height 50
		neighbor := gossipMessageNeighbor[uint64](versionedNeighborPacket[uint64]{
			neighborPacket: neighborPacket[uint64]{
				Round:                 Round(10),
				SetID:                 SetID(0),
				CommitFinalizedHeight: 50,
			},
		})
		gossipMessage := gossipMessageVDT[hash.H256, uint64]{inner: neighbor}
		result := tester.gossipValidator.Validate(network, id, scale.MustMarshal(gossipMessage))

		// neighbor packets are always discard
		switch result.(type) {
		case gossip.ValidationResultDiscard:
		default:
			t.Fatalf("wrong expected outcome from neighbor validation")
		}

		// a catch up request should be sent to the peer for round - 1
		timer := time.NewTimer(5 * time.Second).C
		var verified bool
	loop:
		for {
			select {
			case event, ok := <-tester.events:
				if !ok {
					break loop
				}
				switch event := event.(type) {
				case EventWriteNotification:
					require.Equal(t, id, event.PeerID)
					expectedMessage := gossipMessageVDT[hash.H256, uint64]{
						inner: gossipMessageCatchUpRequest(catchUpRequestMessage{
							SetID: SetID(0),
							Round: Round(9),
						}),
					}
					expected := scale.MustMarshal(expectedMessage)
					require.Equal(t, expected, event.Notification)
					verified = true
					break loop
				default:
					t.Logf("received unexpected event: %T", event)
				}
			case <-timer:
				break loop
			}
		}

		require.True(t, verified)
	})
}
