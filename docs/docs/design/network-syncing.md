---
layout: default
title: Network and Syncing Interface Implementation
permalink: /design/network-syncing-interfaces/
---
# `Network` and `Syncing` Implementation Design

The [`Network`] and [`Syncing`] interfaces were introduced in [PR #4572](https://github.com/ChainSafe/gossamer/pull/4572) as part of the `gossip` package and represent the network and syncing interfaces that are dependencies for the `GossipEngine` type.  What follows is an assessment on each method found in both `Network` and `Syncing` interfaces and how we propose to implement them using components already found in the Gossamer codebase and/or introduce new components.

## [`Network`]

The `Network` interface is as follows:
```go
// Network is the abstraction over a network.
type Network interface {
	service.NetworkPeers
	service.NetworkEventStream
	AddSetReserved(who peerid.PeerID, protocol network.ProtocolName)
	RemoveSetReserved(who peerid.PeerID, protocol network.ProtocolName)
}
```
It embeds two interface `NetworkPeers` and `NetworkEventStream`.
### [`NetworkPeers`]
```go
// NetworkPeers provides low-level API for manipulating network peers.
type NetworkPeers interface {
	// Set authorized peers.
	//
	// Need a better solution to manage authorized peers, but now just use reserved peers for
	// prototyping.
	SetAuthorizedPeers(peers map[peerid.PeerID]struct{})
	// Set authorized_only flag.
	//
	// Need a better solution to decide authorized_only, but now just use reservedOnly flag for prototyping.
	SetAuthorizedOnly(reservedOnly bool)
	// Adds an address known to a node.
	AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr)
	// Report a given peer as either beneficial (+) or costly (-) according to the given scalar.
	ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange)
	// Disconnect from a node as soon as possible.
	//
	// This triggers the same effects as if the connection had closed itself spontaneously.
	//
	// See also ["NetworkPeers::remove_from_peers_set"], which has the same effect but also prevents the local node
	// from re-establishing an outgoing substream to this peer until it is added again.
	DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName)
	// Connect to unreserved peers and allow unreserved peers to connect for syncing purposes.
	AcceptUnreservedPeers()
	// Disconnect from unreserved peers and deny new unreserved peers to connect for syncing purposes.
	DenyUnreservedPeers()
	// Adds a PeerID and its MultiAddr as reserved for a sync protocol (default peer set).
	//
	// Returns an error if the given string is not a valid multiaddress or contains an invalid peer ID (which includes
	// the local peer ID).
	AddReservedPeer(peer config.MultiaddrPeerId) error
	// Removes a [peerid.PeerID] from the list of reserved peers for a sync protocol (default peer set).
	RemoveReservedPeer(peerID peerid.PeerID)
	// Sets the reserved set of a protocol to the given set of peers.
	//
	// Each Multiaddr must end with a "/p2p/" component containing the peer id. It can also
	// consist of only "/p2p/<peerid>".
	//
	// The node will start establishing/accepting connections and substreams to/from peers in this set, if it doesn't
	// have any substream open with them yet.
	//
	// Note however, if a call to this function results in less peers on the reserved set, they will not necessarily
	// get disconnected (depending on available free slots in the peer set). If you want to also disconnect those
	// removed peers, you will have to call RemoveFromPeersSet on those in addition to updating the reserved set. You
	// can omit this step if the peer set is in reserved only mode.
	//
	// Returns an error if one of the given addresses is invalid or contains an invalid peer ID (which includes the
	// local peer id).
	SetReservedPeers(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error
	// Add peers to a peer set.
	//
	// Each Multiaddr must end with a "/p2p/" component containing the peer id. It can also consist of only
	// "/p2p/<peerid>".
	//
	// Returns an error if one of the given addresses is invalid or contains an invalid peer id (which includes the
	// local peer id).
	AddPeersToReservedSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error
	// Remove peers from a peer set.
	RemovePeersFromReservedSet(protocol network.ProtocolName, peers []peerid.PeerID)
	// Add a peer to a set of peers.
	//
	// If the set has slots available, it will try to open a substream with this peer.
	//
	// Each Multiaddr must end with a "/p2p/" component containing the peer id. It can also consist of only
	// "/p2p/<peerid>".
	//
	// Returns an error if one of the given addresses is invalid or contains an invalid peer id (which includes the
	// local peer id).
	AddToPeersSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error
	// Remove peers from a peer set.
	//
	// If we currently have an open substream with this peer, it will soon be closed.
	RemoveFromPeersSet(protocol network.ProtocolName, peers []peerid.PeerID)
	// Returns the number of peers in the sync peer set we're connected to.
	SyncNumConnected() uint

	// Attempt to get peer role.
	//
	// Right now the peer role is decoded from the received handshake for all protocols ("/block-announces/1" has other
	// information as well). If the handshake cannot be decoded into a role, the role queried from peer storeand if the
	// role is not stored there either, nil is returned and the peer should be discarded.
	PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole

	// Get the list of reserved peers.
	//
	// Returns an error if the network worker is no longer running.
	ReservedPeers() <-chan struct {
		Peers []peerid.PeerID
		Error error
	}
}
```

The `SetAuthorizedPeers` and `SetAuthorizedOnly` methods are only called in offchain workers in substrate, so we can safely de-prioritize implementing this.  Currently in substrate they just utilize reserved peer functionality to implement this.

The `AddKnownAddress` method is only used by [polkadot](https://github.com/paritytech/polkadot-sdk/blob/21fbd6b59d37fd18a00d0ec4b6f72dc376d63010/polkadot/node/network/bridge/src/network.rs#L268) and [cumulus](https://github.com/paritytech/polkadot-sdk/blob/3ff1b1db36260cbc47297ab753e2dcec1f5999fd/cumulus/client/bootnodes/src/discovery.rs#L369).  This can also be deprioritized given that GRANDPA doesn't use this method.  The `add_known_address` method is implemented by sending a [`ServiceToWorkerMsg::AddKnownAddress`](https://github.com/paritytech/polkadot-sdk/blob/3ff1b1db36260cbc47297ab753e2dcec1f5999fd/substrate/client/network/src/service.rs#L1324) message to the background [`sc_network::service::NetworkWorker`](https://github.com/paritytech/polkadot-sdk/blob/3ff1b1db36260cbc47297ab753e2dcec1f5999fd/substrate/client/network/src/service.rs#L1347) which eventually calls `add_known_address` on `DiscoveryBehaviour` ([link](https://github.com/paritytech/polkadot-sdk/blob/3ff1b1db36260cbc47297ab753e2dcec1f5999fd/substrate/client/network/src/discovery.rs#L362)). This can be replicated in Gossamer by calling `Connect` on `libp2phost.Host` which we can also do on our private [`discovery`](https://github.com/ChainSafe/gossamer/blob/e0a30efc9af4da112dd85102089cadedbc5f197e/dot/network/discovery.go#L37) type within the `network` package.

`ReportPeer` is already implemented by our existing `peerset.Handler`. We just need to create some sort of translation type to convert between the new `PeerID` and `ReputationChange` and call [`Handler.ReportPeer`](https://github.com/ChainSafe/gossamer/blob/3729f32087e3fe4ebd0be35306a98bae97a0b022/dot/peerset/handler.go#L79).

`DisconnectPeer` is already implemented by `PeerState` via [`disconnect`](https://github.com/ChainSafe/gossamer/blob/dbe6858a7363d61247019ebf81dde4ada90242c8/dot/peerset/peerstate.go#L345). Currently this only exposed by polkadot through [`polkadot_network_bridge::network::Network`](https://github.com/paritytech/polkadot-sdk/blob/21fbd6b59d37fd18a00d0ec4b6f72dc376d63010/polkadot/node/network/bridge/src/network.rs#L159) trait and through a [`NetworkServiceHandle`](https://github.com/paritytech/polkadot-sdk/blob/12d9052459ade7fc7588807bf0775d7c7d135e82/substrate/client/network/sync/src/service/network.rs#L84) exclusively used by the syncing engine.  This can be de-prioritized given that GRANDPA doesn't actually need to disconnect peers.

`AcceptUnreservedPeers` and `DenyUnreservedPeers` doesn't look to be called anywhere in substrate.  We can safely omit this and even remove it from the interface.

`AddReservedPeers` and `RemoveReservedPeer` isn't called by GRANDPA so there is no immediate requirement to implement this.  However we already have this functionality within the `host` private type in `network` ([link](https://github.com/ChainSafe/gossamer/blob/2eff00475ac1234ac1701f596808f93b3bf0bd46/dot/network/host.go#L408)).

`SetReservedPeers`, `AddPeersToReservedSet`, `RemovePeersFromReservedSet`, `AddToPeersSet`, and `SyncNumConnected` are all methods to modify the reserved and regular set of peers specific to a protocol.  This will definitely be needed for the Parachains initiative.  `AddPeersToReservedSet` and `RemovedPeersFromReservedSet` are called by `AddSetReserved` and `RemoveSetReserved` which are just helper functions to add one or remove one peer.  These two helper methods are called in the GRANDPA integration.  Our current method peerset `Handler` doesn't expose adding reserved peers based on protocol.  We will need to expose this adding/removing peers per protocol in the `Handler` and utilize it for GRANDPA.  

`SyncNumConnected` doesn't appear to be used in `polkadot-sdk`.  We can safely remove from interface.

`PeerRole` is used to check handshakes for the role, but if unable to decode the role from the handshake it checks the underlying peerstore.  Storing of the role per peer is currently not supported by our peerstore implementation.  I think we can implement it without checking the peerstore for now and only rely on decoding the supplied handshake.  

`ReservedPeers` is currently only used by system calls via RPC.  I think we can safely remove from this interface, since our current RPC packages do not use this interface.




### [`NetworkEventStream`]
```go
// NetworkEventStream provides access to network-level event stream.
type NetworkEventStream interface {
	// Returns a stream containing the events that happen on the network. If this method is called multiple times, the
	// events are duplicated. The stream never ends (unless the network worker gets shut down). The name passed is
	// used to identify the channel in the Prometheus metrics.
	EventStream(name string) chan event.Event
}
```

`NetworkEventStream` is currently used in `cumulus`, `polkadot` and in `authority-discovery` found in `substrate`.  We will need to implement this at some point for our Authority Discovery epic.  Given we don't need it for GRANDPA it makes sense to leave it as unimplemented.

## [`Syncing`]

The [`Syncing`] interface is as follows:

```go
// Syncing is the abstraction over the syncing subsystem.
type Syncing[H, N any] interface {
	sync.SyncEventStream
	service.NetworkBlock[H, N]
}
```

### [`SyncEventStream`]

```go
type SyncEventStream interface {
	// Subscribe to syncing related events.
	EventStream(name string) chan SyncEvent
}
```
[`SyncEventStream`] contains a single `EventStream` method with returns a channel of type [`SyncEvent`] where the only two types that are returned are `SyncEventPeerConnected` or `SyncEventPeerDisconncted`.  This method is required by `GossipEngine` which is utilized by the GRANDPA as a dependency.  The `GossipEngine` will add/remove from the GRANDPA protocol specific peerset based on these received `SyncEvent` types.  

After investigation into the stream handlers, it looks like we do not currently support detection of `EOF` or `ErrClosedPipe` when reading from the stream, which we need to capture and propagate back to the networking stack to remove the peer from the peerset, and the [`peerViewSet`](https://github.com/ChainSafe/gossamer/blob/55446ab7d85c1c1094a645712a8d3c8d0ea9a514/dot/sync/peer_view.go#L19). We will need to create a [`SyncEvent`] channel and emit these events on successful stream connection and disconnection.

### [`NetworkBlock`]

[`NetworkBlock`] contains two methods, `AnnounceBlock` and `NewBestBlockImported`. 

```go
// NetworkBlock provides ability to announce blocks to the network.
type NetworkBlock[BlockHash any, BlockNumber any] interface {
	// Make sure an important block is propagated to peers.
	//
	// In chain based consensus, we often need to make sure non-best forks are at least temporarily synced. This
	// function forces such an announcement.
	AnnounceBlock(hash BlockHash, data []byte)

	// Inform the network service about new best imported block.
	NewBestBlockImported(hash BlockHash, number BlockNumber)
}

```

`AnnounceBlock` method is used by `GossipEngine` to announce blocks before GRANDPA specific messages are sent.  We already have functionality to create instances `network.BlockAnnounceMessage` using [`core.createBlockAnnounce`](https://github.com/ChainSafe/gossamer/blob/205b72beca190262c12bd13ccd38d12bd026d06f/dot/core/service.go#L230) that can be sent over the `Network` via `Network.GossipMessage`.  We will need to replicate/make public the `BlockAnnounceMessage` constructor that also has the `Network` to send block announces.

`NewBestBlockImported` is currently not used in the Gossamer codebase and can remain unimplemented.



[`Network`]:https://github.com/ChainSafe/gossamer/blob/3249bd27f02a2c28a6aa1881cca922cce02afea1/internal/client/network-gossip/network_gossip.go#L14
[`Syncing`]:https://github.com/ChainSafe/gossamer/blob/3249bd27f02a2c28a6aa1881cca922cce02afea1/internal/client/network-gossip/network_gossip.go#L22
[`NetworkPeers`]:https://github.com/ChainSafe/gossamer/blob/f32d39c27d464d524955d1bf21c5e1d4dcff3491/internal/client/network/service/service.go#L26
[`NetworkEventStream`]:https://github.com/ChainSafe/gossamer/blob/f32d39c27d464d524955d1bf21c5e1d4dcff3491/internal/client/network/service/service.go#L118
[`SyncEventStream`]:https://github.com/ChainSafe/gossamer/blob/3249bd27f02a2c28a6aa1881cca922cce02afea1/internal/client/network/sync/sync.go#L24
[`SyncEvent`]:https://github.com/ChainSafe/gossamer/blob/3249bd27f02a2c28a6aa1881cca922cce02afea1/internal/client/network/sync/sync.go#L11
[`NetworkBlock`]:https://github.com/ChainSafe/gossamer/blob/f32d39c27d464d524955d1bf21c5e1d4dcff3491/internal/client/network/service/service.go#L126
