# Communication Module

## gossip.rs
- Implementation of polite GRANDPA
- `GossipMessage` enum exists here, with module level types for:
    - `VoteMessage`
    - `FullCommitMessage`
    - `VersionedNeighborPacket`
    - `CatchUpRequest`
    - `CatchUp`
- `Misbehavior` enum that affect peer reputation
- `Peers` type that keeps track of all peers
- Nice to have: Metrics on gossip

## mod.rs
- `NetworkBridge` type that bridges between underlying network service, and gossips GRANDPA messages
- some message validation for catch-up messages and compact commits

## periodic.rs
- `NeighborPacketWorker` type that forwards neighbour packets form finality-grandpa and forwards through the `NetworkBridge`

# Import Module
- `GrandpaBlockImport` type is found here.  Scans each imported block for a change in authority set and enacts the changes.
    - Contains attribute to the `Client`

# Environment module
- `Environment` is the environment GRANDPA runs in.  Has access to `NetworkBridge`, client `Backend`, justification senders, voting rules, etc.

# lib entrypoint
- `VoterWork` is found here which powers the GRANDPA voter in finality-grandpa.
    - Has access to the `Environment`
- Some work needs to be done to port over the client interfaces expected by `VoterWork` and `Environment`

# Tests module
- Broader integration type testing of the consensus grandpa client over all modules.

