# GRANDPA client implementation

## Context

The GRANDPA standalone package which is a port of the Parity `finality-grandpa` crate was completed in October 2023 ([PR link](https://github.com/ChainSafe/gossamer/pull/3235)). It was intended to be learning exercise to get a better understanding of the reference GRANDPA implementation, and to see if we could translate the async rust code along with generic support in Go. All of the voter logic is encapsulated by this package, where a long lived `Voter` can be instantiated which will process and participate in concurrent finality voting rounds.

Work on the client integration was started shortly after.  There have been a number of PRs merged to `feat/grandpa` branch that aim to replicate a lot of the client integration of the standalone GRANDPA package.  The client code is in `sc_consensus_grandpa` crate, which reference primitives found in `sp_consensus_grandpa`.  Some of the notable PRs which have primarily been authored by Jimmy are as follows:

### client(consensus/grandpa): Implement ChangeTree and AuthoritySet logic ([#3283](https://github.com/ChainSafe/gossamer/pull/3283))

Introduces `ChangeTree` which is a structure built to track pending changes across forks.  Introduces `AuthoritySet` which trackers the current authorities, as well associated any standard and/or forced authority set changes. 

### client(consensus/grandpa): implement grandpaDb and environment types ([#3383](https://github.com/ChainSafe/gossamer/pull/3383))

Implements the auxiliary data portion of the client GRANDPA integration.  Functionality like persisting the `voterSetState`, current authority set, best block justification to disk is implemented in this PR.

### client(consensus/grandpa): implement grandpa justification logic ([#3454](https://github.com/ChainSafe/gossamer/pull/3454))

Implements verification of GRANDPA justifications, and remaining auxiliary data persistance.

### client(consensus/grandpa): implement finality proof logic ([#3589](https://github.com/ChainSafe/gossamer/pull/3589))

Implements creation and verification of GRANDPA finality proofs.


## What Remains?

The following high level types, `BlockImport` trait implementation, and integration tests still need to be completed.

### `Environment`

[`Environment`] is a type that utilizes already introduced dependencies (`VoterSet`, `SharedAuthoritySet`, etc.) to provide functionality to the [`VoterWork`] type.  It is a self contained type that encapsulates all dependencies of the [`VoterWork`] type.

### `VoterWork` functionality

`sc_consensus_grandpa` main future is [`VoterWork`] type.  [`VoterWork`] uses `communication::NetworkBridge`, the grandpa `Voter` from the standalone GRANDPA package, and the [`Environment`] type as chief dependencies to process GRANDPA voter message and progress the standalone GRANDPA voter. Given the rust source utilizes async rust code, further exploration and prototyping needs to be done to implement it in a way that aligns with Go best practices with regards to concurrency.

### `NetworkBridge`
 [`NetworkBridge`] is the bridge between the underlying network service, gossiping consensus messages and GRANDPA. It contains an implementation of `Network` trait and an implementation of `Syncing` trait though these are just stored to instantiate [`GossipEngine`] which is a dependency.  There is also a [`GossipValidator`](https://github.com/paritytech/polkadot-sdk/blob/d2fd53645654d3b8e12cbf735b67b93078d70113/substrate/client/consensus/grandpa/src/communication/gossip.rs#L1486) type used as a dependency. 
 
 I propose replicating [`NetworkBridge`] given it's a good abstraction and a defined dependency of `VoterWork`.   We will also need to replicate [`GossipEngine`] given it's a depency of [`NetworkBridge`]. 
 
### `GossipEngine`

[`GossipEngine`] is the main long running parallel process that provides generalized gossip functinonality in substrate.  It will also be required for BEEFY implementation. [`GossipEngine`] requires an implementation of the `Network` trait as well as the `Syncing` trait.

#### `Network` trait

The top level [`Network`] trait along with it's inherited traits are as follows:
```rust
// from sc_consensus_grandpa::communication

/// A handle to the network.
///
/// Something that provides the capabilities needed for the `gossip_network::Network` trait.
pub trait Network<Block: BlockT>: GossipNetwork<Block> + Clone + Send + 'static {}
```
```rust
// from sc_network_gossip

/// Abstraction over a network.
pub trait Network<B: BlockT>: NetworkPeers + NetworkEventStream {
	fn add_set_reserved(&self, who: PeerId, protocol: ProtocolName) {
		let addr = Multiaddr::empty().with(Protocol::P2p(*who.as_ref()));
		let result = self.add_peers_to_reserved_set(protocol, iter::once(addr).collect());
		if let Err(err) = result {
			log::error!(target: "gossip", "add_set_reserved failed: {}", err);
		}
	}
	fn remove_set_reserved(&self, who: PeerId, protocol: ProtocolName) {
		let result = self.remove_peers_from_reserved_set(protocol, iter::once(who).collect());
		if let Err(err) = result {
			log::error!(target: "gossip", "remove_set_reserved failed: {}", err);
		}
	}
}

// from sc_network_service

/// Provides low-level API for manipulating network peers.
#[async_trait::async_trait]
pub trait NetworkPeers {
	/// Set authorized peers.
	///
	/// Need a better solution to manage authorized peers, but now just use reserved peers for
	/// prototyping.
	fn set_authorized_peers(&self, peers: HashSet<PeerId>);

	/// Set authorized_only flag.
	///
	/// Need a better solution to decide authorized_only, but now just use reserved_only flag for
	/// prototyping.
	fn set_authorized_only(&self, reserved_only: bool);

	/// Adds an address known to a node.
	fn add_known_address(&self, peer_id: PeerId, addr: Multiaddr);

	/// Report a given peer as either beneficial (+) or costly (-) according to the
	/// given scalar.
	fn report_peer(&self, peer_id: PeerId, cost_benefit: ReputationChange);

	/// Get peer reputation.
	fn peer_reputation(&self, peer_id: &PeerId) -> i32;

	/// Disconnect from a node as soon as possible.
	///
	/// This triggers the same effects as if the connection had closed itself spontaneously.
	fn disconnect_peer(&self, peer_id: PeerId, protocol: ProtocolName);

	/// Connect to unreserved peers and allow unreserved peers to connect for syncing purposes.
	fn accept_unreserved_peers(&self);

	/// Disconnect from unreserved peers and deny new unreserved peers to connect for syncing
	/// purposes.
	fn deny_unreserved_peers(&self);

	/// Adds a `PeerId` and its `Multiaddr` as reserved for a sync protocol (default peer set).
	///
	/// Returns an `Err` if the given string is not a valid multiaddress
	/// or contains an invalid peer ID (which includes the local peer ID).
	fn add_reserved_peer(&self, peer: MultiaddrWithPeerId) -> Result<(), String>;

	/// Removes a `PeerId` from the list of reserved peers for a sync protocol (default peer set).
	fn remove_reserved_peer(&self, peer_id: PeerId);

	/// Sets the reserved set of a protocol to the given set of peers.
	///
	/// Each `Multiaddr` must end with a `/p2p/` component containing the `PeerId`. It can also
	/// consist of only `/p2p/<peerid>`.
	///
	/// The node will start establishing/accepting connections and substreams to/from peers in this
	/// set, if it doesn't have any substream open with them yet.
	///
	/// Note however, if a call to this function results in less peers on the reserved set, they
	/// will not necessarily get disconnected (depending on available free slots in the peer set).
	/// If you want to also disconnect those removed peers, you will have to call
	/// `remove_from_peers_set` on those in addition to updating the reserved set. You can omit
	/// this step if the peer set is in reserved only mode.
	///
	/// Returns an `Err` if one of the given addresses is invalid or contains an
	/// invalid peer ID (which includes the local peer ID), or if `protocol` does not
	/// refer to a known protocol.
	fn set_reserved_peers(
		&self,
		protocol: ProtocolName,
		peers: HashSet<Multiaddr>,
	) -> Result<(), String>;

	/// Add peers to a peer set.
	///
	/// Each `Multiaddr` must end with a `/p2p/` component containing the `PeerId`. It can also
	/// consist of only `/p2p/<peerid>`.
	///
	/// Returns an `Err` if one of the given addresses is invalid or contains an
	/// invalid peer ID (which includes the local peer ID), or if `protocol` does not
	/// refer to a know protocol.
	fn add_peers_to_reserved_set(
		&self,
		protocol: ProtocolName,
		peers: HashSet<Multiaddr>,
	) -> Result<(), String>;

	/// Remove peers from a peer set.
	///
	/// Returns `Err` if `protocol` does not refer to a known protocol.
	fn remove_peers_from_reserved_set(
		&self,
		protocol: ProtocolName,
		peers: Vec<PeerId>,
	) -> Result<(), String>;

	/// Returns the number of peers in the sync peer set we're connected to.
	fn sync_num_connected(&self) -> usize;

	/// Attempt to get peer role.
	///
	/// Right now the peer role is decoded from the received handshake for all protocols
	/// (`/block-announces/1` has other information as well). If the handshake cannot be
	/// decoded into a role, the role queried from `PeerStore` and if the role is not stored
	/// there either, `None` is returned and the peer should be discarded.
	fn peer_role(&self, peer_id: PeerId, handshake: Vec<u8>) -> Option<ObservedRole>;

	/// Get the list of reserved peers.
	///
	/// Returns an error if the `NetworkWorker` is no longer running.
	async fn reserved_peers(&self) -> Result<Vec<PeerId>, ()>;
}

/// Provides access to network-level event stream.
pub trait NetworkEventStream {
	/// Returns a stream containing the events that happen on the network.
	///
	/// If this method is called multiple times, the events are duplicated.
	///
	/// The stream never ends (unless the `NetworkWorker` gets shut down).
	///
	/// The name passed is used to identify the channel in the Prometheus metrics. Note that the
	/// parameter is a `&'static str`, and not a `String`, in order to avoid accidentally having
	/// an unbounded set of Prometheus metrics, which would be quite bad in terms of memory
	fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = Event> + Send>>;
}
```
[`GossipEngine`] uses `add_set_reserved` and `remove_set_reserved`. Looks like Gossamer already support adding and removing reserved peers.  In substrate from what I gather, there are authorized peers, and there are reserved peers.  Reserved peers are peers that are in the authorized pool, that are assigned to a set for a given `ProtocolName`.  We should be able to add/remove from the reserved set.  Removing from the authorized set is done via peer reputation and I assume will be removed from any reserved sets if disconnected. `peer_role()` and `ObservedRole` are used qutie a bit in [`GossipEngine`].  `report_peer` is also used in [`GossipEngine`].  Doesn't seem like `NetworkEventStream` is actually used in GRANDPA client integration.

##### `Syncing` trait

The top level [`Syncing`] trait along with it's inherited traits are as follows:
```rust
// from sc_consensus_grandpa

/// A handle to syncing-related services.
///
/// Something that provides the ability to set a fork sync request for a particular block.
pub trait Syncing<Block: BlockT>:
	NetworkSyncForkRequest<Block::Hash, NumberFor<Block>>
	+ NetworkBlock<Block::Hash, NumberFor<Block>>
	+ SyncEventStream
	+ Clone
	+ Send
	+ 'static
{}
```
```rust
// from sc_network

/// Provides an ability to set a fork sync request for a particular block.
pub trait NetworkSyncForkRequest<BlockHash, BlockNumber> {
	/// Notifies the sync service to try and sync the given block from the given
	/// peers.
	///
	/// If the given vector of peers is empty then the underlying implementation
	/// should make a best effort to fetch the block from any peers it is
	/// connected to (NOTE: this assumption will change in the future #3629).
	fn set_sync_fork_request(&self, peers: Vec<PeerId>, hash: BlockHash, number: BlockNumber);
}

/// Provides ability to announce blocks to the network.
pub trait NetworkBlock<BlockHash, BlockNumber> {
	/// Make sure an important block is propagated to peers.
	///
	/// In chain-based consensus, we often need to make sure non-best forks are
	/// at least temporarily synced. This function forces such an announcement.
	fn announce_block(&self, hash: BlockHash, data: Option<Vec<u8>>);

	/// Inform the network service about new best imported block.
	fn new_best_block_imported(&self, hash: BlockHash, number: BlockNumber);
}

pub trait SyncEventStream: Send + Sync {
	/// Subscribe to syncing-related events.
	fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = SyncEvent> + Send>>;
}
```

`set_sync_fork_request` is called by [`GossipEngine`].  `announce_block` is also called through `GossipEngine.announce()` via `OutgoingMessages` sink type in `communication` crate.  Doesn't look like `new_best_block_imported` is actually used. `event_stream` is used in [`GossipEngine`] constructor.

### Implementation of `BlockImport`

There is a [`BlockImport`](https://github.com/paritytech/polkadot-sdk/blob/030cb4a71b0b390626a586bfe7117b7c66b4700c/substrate/client/consensus/common/src/block_import.rs#L308) trait that `sc_consensus_grandpa` implements.  In substrate, the `BlockImport` trait is called through a pipelining type called [`BasicQueue`].  An instance of [`BasicQueue`] creates a pipeline of calls to a sequence of implementations of `BlockImport`.  [`BlockImportParams`](https://github.com/paritytech/polkadot-sdk/blob/030cb4a71b0b390626a586bfe7117b7c66b4700c/substrate/client/consensus/common/src/block_import.rs#L170) are provided to each implementation as it progresses through the pipeline for `BlockImport::import_block` function, and is expected to return an `ImportResult` which an enum where the [`ImportResult::Imported`](https://github.com/paritytech/polkadot-sdk/blob/030cb4a71b0b390626a586bfe7117b7c66b4700c/substrate/client/consensus/common/src/block_import.rs#L34) variant is of type [`ImportedAux`](https://github.com/paritytech/polkadot-sdk/blob/030cb4a71b0b390626a586bfe7117b7c66b4700c/substrate/client/consensus/common/src/block_import.rs#L47), which contains data associated with the imported block.  Given that this is the process in substrate, both BEEFY and GRANDPA client implementations currently implement this, and the final implementation in the pipeline is `Client`, I think it makes sense to replicate this functionality given we already already begun replicating the `Client` type.

### Integration tests

There are tests found in `sc_consensus_grandpa` ([link](https://github.com/paritytech/polkadot-sdk/blob/feac7a521092c599d47df3e49084e6bff732c7db/substrate/client/consensus/grandpa/src/tests.rs#L19)) that create a mock network to test general functionality.  There are both startup, shutdown tests, as well as testing finalization of blocks and persistence of voter state.  I think it would be prudent to replicate the same tests minus the ones that utilize the `ObvserverWork`.   


[`VoterWork`]:(https://github.com/paritytech/polkadot-sdk/blob/d2fd53645654d3b8e12cbf735b67b93078d70113/substrate/client/consensus/grandpa/src/lib.rs#L867)
[`NetworkBridge`]:(https://github.com/paritytech/polkadot-sdk/blob/d2fd53645654d3b8e12cbf735b67b93078d70113/substrate/client/consensus/grandpa/src/communication/mod.rs#L212)
[`Environment`]:(https://github.com/paritytech/polkadot-sdk/blob/dc4047c7d05f593757328394d1accf0eb382d709/substrate/client/consensus/grandpa/src/environment.rs#L425)
[`GossipEngine`]:(https://github.com/paritytech/polkadot-sdk/blob/7c9e34b576ad91aba2575ddaaddca0ad1aecae83/substrate/client/network-gossip/src/bridge.rs#L48)
[`Network`]:(https://github.com/paritytech/polkadot-sdk/blob/d2fd53645654d3b8e12cbf735b67b93078d70113/substrate/client/consensus/grandpa/src/communication/mod.rs#L167)
[`Syncing`]:(https://github.com/paritytech/polkadot-sdk/blob/d2fd53645654d3b8e12cbf735b67b93078d70113/substrate/client/consensus/grandpa/src/communication/mod.rs#L179)
[`BasicQueue`]:(https://github.com/paritytech/polkadot-sdk/blob/12d9052459ade7fc7588807bf0775d7c7d135e82/substrate/client/consensus/common/src/import_queue/basic_queue.rs#L44)