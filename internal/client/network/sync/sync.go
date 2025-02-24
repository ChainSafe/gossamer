package sync

import (
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
)

// SyncEvent is a syncing related event that other protocols can subscribe to.
type SyncEvent interface {
	isSyncEvent()
}

// PeerConnected is the peer that the syncing implementation is tracking connected.
type SyncEventPeerConnected peerid.PeerID

// PeerDisconnected is the peer that the syncing implementation was tracking disconnected.
type SyncEventPeerDisconnected peerid.PeerID

func (SyncEventPeerConnected) isSyncEvent()    {}
func (SyncEventPeerDisconnected) isSyncEvent() {}

type SyncEventStream interface {
	// Subscribe to syncing related events.
	EventStream(name string) chan SyncEvent
}
