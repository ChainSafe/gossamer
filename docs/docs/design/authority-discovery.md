---
layout: default
title: Authority Discovery Overview
permalink: /design/authority-discovery/
---

# Authority Discovery

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
