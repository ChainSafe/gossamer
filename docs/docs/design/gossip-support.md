---
layout: default
title: Gossip Support
permalink: /design/gossip-support/
---

# Gossip Support

Gossip support is a short subsystem that cares about handling active leaves and network bridge update events.

## Functionalities

- **Check connectivity**: The subsystem contains a timer that triggers every 600 seconds (10 minutes) a connectivity check that is a simple verification on the amount of authorities connected against the actual set of authorities, if the percentage of connected authorities is bellow 90% the node will issue WARN logs.

- **Handle Active Leaves**: 
  - Given the current values in the state, **last failure** and **last connection request**, the subsystem should force new requests or re-resolve authorities
  - Determine if the current session index has changed, if so determine relevant validators and issue connection request. The subsystem should change the **last sesison index** value only if it is possible to retrieve the new session info.
  - If we notice that a new session is starting we should update our authority ids cache and send a `NetworkBridgeRxMessage::UpdatedAuthorityIds` message with informations about the new set of authorities.

- **Handle Network Bridge Update**:
  - In the above point, when handling active leaves, we issue connection requests and whenever a new connection is stablished or a disconnection happens for some reason. The subsytems listen on any network bridge update about `NetworkBridgeEvent::PeerConnected` and `NetworkBridgeEvent::PeerDisconnected`, other updates are not important.

## State

The subsystem state is composed of:

- Keystore
- Last Session Index
- **Last Failure**: last time we could not resolve a third of the authorities
- **Last Connection Request**: is possible that validators change their peer IDs in during a session. If that happens we will reconnect to them in the best case after a session, so we need to try more often to resolve peers and reconnect to them. We cannot detect changes faster them Authority Discovery DHT queries.
- **Failure Start**: first time the node cannot reach the connectivity threshold
- **Resolved Authorities**: A map of resolved authority IDs to a set of multiaddr, basically we were able to find the addresses for authorities, not mean we are actually connected to them.
- **Connected Authorities**: A map of actual authorities ID's connected to their PeerId
- **Connected Peers**: A map of PeerId to authority IDs for fast lookup
- **Authority Discovery**: authority discovery service, allows us to resolve authorities addresses.

