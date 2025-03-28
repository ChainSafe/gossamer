// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossip

import (
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/stretchr/testify/require"
)

type AllowAll struct{}

func (ao AllowAll) NewPeer(context ValidatorContext[hash.H256], who peerid.PeerID, role role.ObservedRole) {
}
func (ao AllowAll) PeerDisconnected(context ValidatorContext[hash.H256], who peerid.PeerID) {
}
func (ao AllowAll) Validate(context ValidatorContext[hash.H256], sender peerid.PeerID, data []byte) ValidationResult {
	return ValidationResultProcessAndKeep[hash.H256]{
		Hash: hash.NewH256(),
	}
}
func (ao AllowAll) MessageExpired() func(topic hash.H256, message []byte) bool {
	return func(topic hash.H256, message []byte) bool {
		return false
	}
}
func (ao AllowAll) MessageAllowed() func(who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte) bool {
	return func(who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte) bool {
		return true
	}
}

type AllowOne struct{}

func (ao AllowOne) NewPeer(context ValidatorContext[hash.H256], who peerid.PeerID, role role.ObservedRole) {
}
func (ao AllowOne) PeerDisconnected(context ValidatorContext[hash.H256], who peerid.PeerID) {
}
func (ao AllowOne) Validate(context ValidatorContext[hash.H256], sender peerid.PeerID, data []byte) ValidationResult {
	if data[0] == 1 {
		return ValidationResultProcessAndKeep[hash.H256]{
			Hash: hash.NewH256(),
		}
	}
	return ValidationResultDiscard{}
}
func (ao AllowOne) MessageExpired() func(topic hash.H256, message []byte) bool {
	return func(topic hash.H256, message []byte) bool {
		return message[0] != 1
	}
}
func (ao AllowOne) MessageAllowed() func(who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte) bool {
	return func(who peerid.PeerID, intent MessageIntent, topic hash.H256, data []byte) bool {
		return true
	}
}

var (
	_ Validator[hash.H256] = AllowAll{}
	_ Validator[hash.H256] = AllowOne{}
)

type PeerIDReputationChange struct {
	peerid.PeerID
	network.ReputationChange
}

type NoOpNetwork struct {
	peerReports    []PeerIDReputationChange
	peerReportsMtx sync.Mutex
}

func (*NoOpNetwork) SetAuthorizedPeers(peers map[peerid.PeerID]struct{}) {
	panic("unimpl")
}
func (*NoOpNetwork) SetAuthorizedOnly(reservedOnly bool) {
	panic("unimpl")
}
func (*NoOpNetwork) AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr) {
	panic("unimpl")
}
func (non *NoOpNetwork) ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange) {
	non.peerReportsMtx.Lock()
	defer non.peerReportsMtx.Unlock()
	non.peerReports = append(non.peerReports, PeerIDReputationChange{
		PeerID:           peerID,
		ReputationChange: costBenefit,
	})
}
func (*NoOpNetwork) DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName) {
	panic("unimpl")
}
func (*NoOpNetwork) AcceptUnreservedPeers() {
	panic("unimpl")
}
func (*NoOpNetwork) DenyUnreservedPeers() {
	panic("unimpl")
}
func (*NoOpNetwork) AddReservedPeer(peer config.MultiaddrPeerId) error {
	panic("unimpl")
}
func (*NoOpNetwork) RemoveReservedPeer(peerID peerid.PeerID) {
	panic("unimpl")
}
func (*NoOpNetwork) SetReservedPeers(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (*NoOpNetwork) AddPeersToReservedSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (*NoOpNetwork) RemovePeersFromReservedSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (*NoOpNetwork) AddToPeersSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error {
	panic("unimpl")
}
func (*NoOpNetwork) RemoveFromPeersSet(protocol network.ProtocolName, peers []peerid.PeerID) {
	panic("unimpl")
}
func (*NoOpNetwork) SyncNumConnected() uint {
	panic("unimpl")
}
func (*NoOpNetwork) EventStream(name string) chan event.Event {
	panic("unimpl")
}
func (*NoOpNetwork) AnnounceBlock(hash hash.H256, data []byte) {
	panic("unimpl")
}
func (*NoOpNetwork) NewBestBlockImported(hash hash.H256, number uint64) {
	panic("unimpl")
}
func (*NoOpNetwork) AddSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("unimpl")
}
func (*NoOpNetwork) RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName) {
	panic("unimpl")
}
func (*NoOpNetwork) PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole {
	panic("unimpl")
}
func (*NoOpNetwork) ReservedPeers() <-chan struct {
	Peers []peerid.PeerID
	Error error
} {
	panic("unimpl")
}

var (
	_ service.NetworkPeers                    = &NoOpNetwork{}
	_ service.NetworkEventStream              = &NoOpNetwork{}
	_ service.NetworkBlock[hash.H256, uint64] = &NoOpNetwork{}
)

type NoOpNotificationService struct{}

func (NoOpNotificationService) OpenSubstream(peer peerid.PeerID) <-chan error {
	panic("unimpl")
}
func (NoOpNotificationService) CloseSubstream(peer peerid.PeerID) <-chan error {
	panic("unimpl")
}
func (NoOpNotificationService) SendSyncNotification(peer peerid.PeerID, notification []byte) {
	panic("unimpl")
}
func (NoOpNotificationService) SendAsyncNotification(peer peerid.PeerID, notification []byte) <-chan error {
	panic("unimpl")
}
func (NoOpNotificationService) SetHandshake(handshake []byte) <-chan error {
	panic("unimpl")
}
func (NoOpNotificationService) TrySetHandshake(handshake []byte) error {
	panic("unimpl")
}
func (NoOpNotificationService) NextEvent() <-chan service.NotificationEvent {
	panic("unimpl")
}
func (NoOpNotificationService) Protocol() network.ProtocolName {
	panic("unimpl")
}
func (NoOpNotificationService) MessageSink(peer peerid.PeerID) service.MessageSink {
	panic("unimpl")
}

var _ service.NotificationService = NoOpNotificationService{}

func pushMessage(
	consensus *consensusGossip[hash.H256, runtime.BlakeTwo256], topic hash.H256, h hash.H256, message []byte,
) {
	consensus.knownMessages.Add(h, nil)
	consensus.messages = append(consensus.messages, messageEntry[hash.H256]{
		messageHash: h,
		topic:       topic,
		message:     message,
		sender:      nil,
	})
}

func TestConsensusGossip(t *testing.T) {
	t.Run("collects_garbage", func(t *testing.T) {
		prevHash := hash.NewRandomH256()
		bestHash := hash.NewRandomH256()
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")
		m1Hash := hash.NewRandomH256()
		m2Hash := hash.NewRandomH256()
		m1 := []byte{1, 2, 3}
		m2 := []byte{4, 5, 6}

		pushMessage(consensus, prevHash, m1Hash, m1)
		pushMessage(consensus, bestHash, m2Hash, m2)
		consensus.knownMessages.Add(m1Hash, nil)
		consensus.knownMessages.Add(m2Hash, nil)

		consensus.CollectGarbage()
		require.Equal(t, 2, len(consensus.messages))
		require.Equal(t, 2, consensus.knownMessages.Len())

		consensus.validator = AllowOne{}

		// m2 is expired
		consensus.CollectGarbage()
		require.Equal(t, 1, len(consensus.messages))
		// known messages are only pruned based on size.
		require.Equal(t, 2, consensus.knownMessages.Len())
		_, ok := consensus.knownMessages.Get(m2Hash)
		require.True(t, ok)
	})

	t.Run("message_stream_include_those_sent_before_asking", func(t *testing.T) {
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")

		// Register message.
		message := []byte{4, 5, 6}
		topic := runtime.BlakeTwo256{}.Hash([]byte{1, 2, 3})
		consensus.RegisterMessage(topic, message)

		require.Equal(t, TopicNotification{
			Message: message,
			Sender:  nil,
		}, consensus.MessagesFor(topic)[0])
	})

	t.Run("can_keep_multiple_messages_per_topic", func(t *testing.T) {
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")

		topic := hash.NewRandomH256()
		msgA := []byte{1, 2, 3}
		msgB := []byte{4, 5, 6}

		consensus.RegisterMessage(topic, msgA)
		consensus.RegisterMessage(topic, msgB)

		require.Equal(t, 2, len(consensus.messages))
	})

	t.Run("peer_is_removed_on_disconnect", func(t *testing.T) {
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")

		notifcationService := NoOpNotificationService{}

		peerID := peerid.NewRandomPeerID()
		consensus.NewPeer(notifcationService, peerID, role.ObservedRoleFull)
		_, ok := consensus.peers[peerID]
		require.True(t, ok)

		consensus.PeerDisconnected(notifcationService, peerID)
		_, ok = consensus.peers[peerID]
		require.False(t, ok)
	})

	t.Run("on_incoming_ignores_discarded_messages", func(t *testing.T) {
		notifcationService := NoOpNotificationService{}
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")
		toForward := consensus.OnIncoming(nil, notifcationService, peerid.NewRandomPeerID(), [][]byte{{1, 2, 3}})

		require.Empty(t, toForward)
	})

	t.Run("on_incoming_ignores_unregistered_peer", func(t *testing.T) {
		network := NoOpNetwork{}
		notifcationService := NoOpNotificationService{}
		remote := peerid.NewRandomPeerID()

		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")
		toForward := consensus.OnIncoming(&network, notifcationService, remote, [][]byte{{1, 2, 3}})

		require.Empty(t, toForward)
	})

	// Two peers can send us the same gossip message. We should not report the second peer
	// sending the gossip message as long as its the first time the peer send us this message.
	t.Run("do_not_report_peer_for_first_time_duplicate_gossip_message", func(t *testing.T) {
		consensus := newConsensusGossip[hash.H256, runtime.BlakeTwo256](AllowAll{}, "/foo")

		network := NoOpNetwork{}
		notifcationService := NoOpNotificationService{}

		peerID := peerid.NewRandomPeerID()
		consensus.NewPeer(notifcationService, peerID, role.ObservedRoleFull)
		require.Contains(t, consensus.peers, peerID)

		peerID2 := peerid.NewRandomPeerID()
		consensus.NewPeer(notifcationService, peerID2, role.ObservedRoleFull)
		require.Contains(t, consensus.peers, peerID2)

		message := [][]byte{{1, 2, 3}}
		consensus.OnIncoming(&network, notifcationService, peerID, message)
		consensus.OnIncoming(&network, notifcationService, peerID2, message)

		require.Equal(t, []PeerIDReputationChange{{peerID, gossipSuccess}}, network.peerReports)
	})
}
