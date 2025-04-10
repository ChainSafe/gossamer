---
layout: default
title: Authority Discovery Overview
permalink: /design/authority-discovery/
---
# Authority Discovery Design

Authority Discovery in Substrate enables authorities to discover and directly connect to other authorities.  In the Substrate implementation it is comprised of two components, the [`Worker`] and the [`Service`].

## `Worker`

An authority discovery [`Worker`] can publish the local node's addresses as well as discover
those of other nodes via a [Kademlia DHT](https://pl-launchpad.io/curriculum/libp2p/dht/).

When constructing the worker via constructor, we can specify the [`Role`]. The [`Role`] enum has two values: `PublishAndDiscover` and `Discover`. 

### `Role::PublishAndDiscover`

If the worker is constructed with the `PublishAndDiscover` role, it will 
- Retrieve its external addresses (including peer id).
- Get the list of keys owned by the local node participating in the current authority set.
- Sign the addresses with the keys.
- Put addresses and signature as a record with the authority id as a key on the Kademlia DHT.

### `Role::Discover`

If the worker is constructed with either `PublishAndDiscover` or `Discover` role, it will:
- Retrieve the current and next set of authorities.
- Start DHT queries for the ids of the authorities.
- Validate the signatures of the retrieved key value pairs.
- Add the retrieved external addresses as priority nodes to the network peerset.
- Allow querying of the collected addresses via the accompanying [`Service`].

### `Worker` Dependencies

There are a number of dependencies that are supplied and stored as part of the `Worker::new` constructor.  The notable dependencies are described in further detail.

#### `AuthorityDiscovery` as `Client`

The [`AuthorityDiscovery`] trait contains only two methods: `authorities` and `best_hash`.

The `authorities` is the same method found in [`ProvideRuntimeApi`] trait, implying a runtime call is expected to be made to get the authorities at a given block hash.

The `best_hash` method is the same found in [`HeaderBackend`] trait, which we have already implemented in the Gossamer version of the base `Client`.

#### `NetworkProvider`

Within Authority Discover a trait named [`NetworkProvider`] is defined as:
```rust
/// NetworkProvider provides [`Worker`] with all necessary hooks into the
/// underlying Substrate networking. Using this trait abstraction instead of
/// `sc_network::NetworkService` directly is necessary to unit test [`Worker`].
pub trait NetworkProvider:
	NetworkDHTProvider + NetworkStateInfo + NetworkSigner + Send + Sync
{
}
```
I've copied all of `NetworkDHTProvider`, `NetworkStateInfo` and `NetworkSigner` traits below:
```rust
/// Provides access to the networking DHT.
pub trait NetworkDHTProvider {
	/// Start getting a value from the DHT.
	fn get_value(&self, key: &KademliaKey);

	/// Start putting a value in the DHT.
	fn put_value(&self, key: KademliaKey, value: Vec<u8>);

	/// Start putting the record to `peers`.
	///
	/// If `update_local_storage` is true the local storage is udpated as well.
	fn put_record_to(&self, record: Record, peers: HashSet<PeerId>, update_local_storage: bool);

	/// Store a record in the DHT memory store.
	fn store_record(
		&self,
		key: KademliaKey,
		value: Vec<u8>,
		publisher: Option<PeerId>,
		expires: Option<Instant>,
	);

	/// Register this node as a provider for `key` on the DHT.
	fn start_providing(&self, key: KademliaKey);

	/// Deregister this node as a provider for `key` on the DHT.
	fn stop_providing(&self, key: KademliaKey);

	/// Start getting the list of providers for `key` on the DHT.
	fn get_providers(&self, key: KademliaKey);
}
```
For `NetworkDHTProvider` functionality the current `go-libp2p-kad-dht` implementation supports everything except `store_record`.  This is a function that exists in the `rust-libp2p` library.  Essentially, it is a directed message to a peer to update the expiry for that key/value.  This is used in `Worker` to update authority records on other known peers.

```rust
/// Trait for providing information about the local network state
pub trait NetworkStateInfo {
	/// Returns the local external addresses.
	fn external_addresses(&self) -> Vec<Multiaddr>;

	/// Returns the listening addresses (without trailing `/p2p/` with our `PeerId`).
	fn listen_addresses(&self) -> Vec<Multiaddr>;

	/// Returns the local Peer ID.
	fn local_peer_id(&self) -> PeerId;
}
```
The `NetworkStateInfo` trait/interface can be be implemented by calling existing Gossamer code.

```rust
/// Signer with network identity
pub trait NetworkSigner {
	/// Signs the message with the `KeyPair` that defines the local [`PeerId`].
	fn sign_with_local_identity(&self, msg: Vec<u8>) -> Result<Signature, SigningError>;

	/// Verify signature using peer's public key.
	///
	/// `public_key` must be Protobuf-encoded ed25519 public key.
	///
	/// Returns `Err(())` if public cannot be parsed into a valid ed25519 public key.
	fn verify(
		&self,
		peer_id: sc_network_types::PeerId,
		public_key: &Vec<u8>,
		signature: &Vec<u8>,
		message: &Vec<u8>,
	) -> Result<bool, String>;
}
```
The `NetworkSigner` trait/interface can be implemented by calling existing Gossamer code.

#### `DhtEventStream`

`DhtEventStream` is essentially a stream of item type [`DhtEvent`] enum. The enum is as follows:
```rust
pub enum DhtEvent {
	/// The value was found.
	ValueFound(PeerRecord),

	/// The requested record has not been found in the DHT.
	ValueNotFound(Key),

	/// The record has been successfully inserted into the DHT.
	ValuePut(Key),

	/// An error has occurred while putting a record into the DHT.
	ValuePutFailed(Key),

	/// An error occured while registering as a content provider on the DHT.
	StartProvidingFailed(Key),

	/// The DHT received a put record request.
	PutRecordRequest(Key, Vec<u8>, Option<sc_network_types::PeerId>, Option<std::time::Instant>),

	/// The providers for [`Key`] were found.
	ProvidersFound(Key, Vec<PeerId>),

	/// The providers for [`Key`] were not found.
	ProvidersNotFound(Key),
}
```

In Gossamer we currently use `go-libp2p-kad-dht` package for DHT functionality.  `go-libp2p-kad-dht` does not currently emit any of these events.  The `Worker` only handles cases of `ValueFound`, `ValueNotFound`, `ValuePut`, `ValuePutFailed`, and `PutRecordRequest`.  

#### Implementing `NetworkDhtProvider`

I believe we will need to fork `go-libp2p-kad-dht` or look for viable alternatives DHT packages to support the `put_record_to` functionality. Without this functionality we will not be able to run the Substrate Authority Discovery protocol as expected by Substrate based nodes. Given the `put_record_to` method is updating existing records based on creation time to specific peers, we will need to expose the message sender in `IpfsDHT` and add the functionality to a wrapper type, or add the function to it and hope to merge back upstream.

#### Implementing `DhtEventStream`

The rust `libp2p-kad` crate emits events of enum type [`KademliaEvent`](https://github.com/libp2p/rust-libp2p/blob/master/protocols/kad/src/behaviour.rs#L2745). The ones that need to be implemented in the Go Kademlia DHT library are `KademliaEvent::OutboundQueryProgressed` and `KademliaEvent::InboundRequest`.

##### `KademliaEvent::InboundRequest`
`InboundRequest` has a `request` attribute which is of type [`InboundRequest`](https://github.com/libp2p/rust-libp2p/blob/master/protocols/kad/src/behaviour.rs#L2854).  If the request is of type [`PutRecord`](https://github.com/libp2p/rust-libp2p/blob/master/protocols/kad/src/behaviour.rs#L2878) we should be emitting an event that we can translate to `DhtEvent::PutRecordRequest`.  

In `go-libp2p-kad-dht` it is unclear if we are able to listen on events that are inbound (aka come from other nodes). We will need to investigate into the codebase to see if there are current events that can be listened on to achieve this functionality.  

##### `KademliaEvent::OutboundQueryProgressed`
For `OutboundQueryProgressed` there is a `result` attribute of type [`QueryResult`](https://github.com/libp2p/rust-libp2p/blob/master/protocols/kad/src/behaviour.rs#L2887).  If the result is of variant type `QueryResult::GetRecord` this signals that an outbound request has been made to the DHT.  `QueryResult::GetRecord` is of type `GetRecordResult` which is a result type `Result<GetRecordOk, GetRecordError>`. `GetRecordOK` is as follows:
```rust
/// The successful result of [`Behaviour::get_record`].
pub enum GetRecordOk {
    FoundRecord(PeerRecord),
    FinishedWithNoAdditionalRecord {
        /// If caching is enabled, these are the peers closest
        /// _to the record key_ (not the local node) that were queried but
        /// did not return the record, sorted by distance to the record key
        /// from closest to farthest. How many of these are tracked is configured
        /// by [`Config::set_caching`].
        ///
        /// Writing back the cache at these peers is a manual operation.
        /// ie. you may wish to use these candidates with [`Behaviour::put_record_to`]
        /// after selecting one of the returned records.
        cache_candidates: BTreeMap<kbucket::Distance, PeerId>,
    },
}
```
There be an emitted event of `DhtEvent::ValueFound` whenever `GetRecordOk::FoundRecord` is the result.  If there's an error `GetRecordError`, a `DhtEvent::ValueNotFound` event should be sent over the `DhtEventStream`.

If `OutboundQueryProgressed` result attribute is of type `QueryResult::PutRecord`, we should be emitting a `DhtEvent::ValuePut` event if it was successful, and a`DhtEvent::ValuePutFailed` event if it failed.

In `go-libp2p-kad-dht` both the `GetValue` and `PutValue` functions are synchronous calls that take in a supplied `context.Context` for cancellation.  We should be able to emit these events in a type that wraps `IpfsDHT` by spawning goroutines to emit the `DhtEvent` variants.  

## `Service`

The [`Service`] is an accompanying type to the `Worker` which communicates over a channel with item type [`ServicetoWorkerMessage`] enum.  The enum is as follows:
```rust
/// Message send from the [`Service`] to the [`Worker`].
pub(crate) enum ServicetoWorkerMsg {
	/// See [`Service::get_addresses_by_authority_id`].
	GetAddressesByAuthorityId(AuthorityId, oneshot::Sender<Option<HashSet<Multiaddr>>>),
	/// See [`Service::get_authority_ids_by_peer_id`].
	GetAuthorityIdsByPeerId(PeerId, oneshot::Sender<Option<HashSet<AuthorityId>>>),
}
```
The messages are sent to the worker when calling the public API of the `Service`.  The `Service` public API is essentially the following two methods:

```rust
pub async fn get_addresses_by_authority_id(
    &mut self,
    authority: AuthorityId,
) -> Option<HashSet<Multiaddr>>

pub async fn get_authority_ids_by_peer_id(
    &mut self,
    peer_id: PeerId,
) -> Option<HashSet<AuthorityId>> {
```

I'm not absolutely sure we need to seperate the `Worker` and `Service` functionality into two distinct types like they do in Substrate.  But these two functions are the public interface to the Authority Discovery package, and both should return a "one shot" channel that return a set of multiaddr, or authority ids.

## Questions

### Is the authority discovery implemented on Kademlia for a custom peer discovery?

No, it uses Kademlia capabilities however it adds a custom handlers for DHT Events, like: DhtEvent::ValueFound
- DhtEvent::ValueNotFound
- DhtEvent::ValuePut
- DhtEvent::ValuePutFailed
- DhtEvent::PutRecordRequest

These events are produced by the Discovery layer (Kademlia) and propagated to the `authority-discovery` that handles each of them. Specifically for `DhtEvent::PutRecordRequest` it does a validation step and then store the record. 

What happens is that the `authority-discovery` is the actual protocol implementation for discovery in substrate/polkadot given that it produces and store the records that are available in the DHT for other peers to query, validate and use.

### What are the functionalities of the authority discovery, beyond discovery new peers based on its authority ids?

- Publish the node external address, more specifically, create the authority discovery record, sign it and distributing through DHT allowing other nodes to query it.
- Handle DHT events, validating incoming records and caching other authorities address.

### What subsystems uses the authority discovery mechanism? Describe the usages.

- Availability Distribution
- Availability Recovery
- Gossip Support

### Check if we can currently extend the authority discovery protocol can to golang kademlia p2p library

Yes, we pretty much can implement the protocol. 
- We are able to encode the records using the same [proto schema used by substrate/polkadot](https://github.com/paritytech/polkadot-sdk/blob/master/substrate/client/authority-discovery/src/worker/schema/dht-v3.proto) - We can query for specific keys in DHT.
- What can write a custom validator to validate incoming records.
- golang-libp2p does not works in the same way rust-libp2p works by enabling us to receive those DHT events but we can achieve the same protocol implementation using what the library provide to us.

[`Worker`]:https://github.com/paritytech/polkadot-sdk/blob/4b054c60b1641612ef0a76dcc75eed5dd23a18cf/substrate/client/authority-discovery/src/worker.rs#L233
[`Service`]:https://github.com/paritytech/polkadot-sdk/blob/414a8fc2eda3bb72e30cefdba628cf6c361cd6e1/substrate/client/authority-discovery/src/service.rs#L34
[`Role`]:https://github.com/paritytech/polkadot-sdk/blob/4b054c60b1641612ef0a76dcc75eed5dd23a18cf/substrate/client/authority-discovery/src/worker.rs#L86
[`AuthorityDiscovery`]:https://github.com/paritytech/polkadot-sdk/blob/4b054c60b1641612ef0a76dcc75eed5dd23a18cf/substrate/client/authority-discovery/src/worker.rs#L205
[`ProvideRuntimeApi`]:https://github.com/paritytech/polkadot-sdk/blob/c5444f381fdba68aa9cb73b39cc63f34604da156/substrate/primitives/api/src/lib.rs#L750
[`HeaderBackend`]:https://github.com/paritytech/polkadot-sdk/blob/fdb4554e26ebdd4d729158501a3ddb3c6ebdfb6f/substrate/primitives/blockchain/src/backend.rs#L37
[`NetworkProvider`]:https://github.com/paritytech/polkadot-sdk/blob/4b054c60b1641612ef0a76dcc75eed5dd23a18cf/substrate/client/authority-discovery/src/worker.rs#L1029
[`DhtEvent`]:https://github.com/paritytech/polkadot-sdk/blob/4b054c60b1641612ef0a76dcc75eed5dd23a18cf/substrate/client/network/src/event.rs#L35
[`Service`]:https://github.com/paritytech/polkadot-sdk/blob/414a8fc2eda3bb72e30cefdba628cf6c361cd6e1/substrate/client/authority-discovery/src/service.rs#L34
[`ServicetoWorkerMessage`]:https://github.com/paritytech/polkadot-sdk/blob/80616f6d03661106326b621e9cc3ee1d2fa283ed/substrate/client/authority-discovery/src/lib.rs#L168