package sync

import libp2p "github.com/libp2p/go-libp2p/core"

// / Syncing-related events that other protocols can subscribe to.
type SyncEvents interface {
	PeerConnected | PeerDisconnected
}

// / Syncing-related event that other protocols can subscribe to.
type SyncEvent any

// / Peer that the syncing implementation is tracking connected.
type PeerConnected libp2p.PeerID

// / Peer that the syncing implementation was tracking disconnected.
type PeerDisconnected libp2p.PeerID

type SyncEventStream interface {
	// Subscribe to syncing-related events.
	// fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = SyncEvent> + Send>>;
	EventStream(name string) chan SyncEvent
}
