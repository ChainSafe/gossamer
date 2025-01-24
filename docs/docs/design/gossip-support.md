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

