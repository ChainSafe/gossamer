# Grid Topology

## Questions

### What subsystem is in charge for create topology? 

[Gossip Support Subsystem](https://github.com/paritytech/polkadot-sdk/blob/master/polkadot/node/network/gossip-support/src/lib.rs) is responsible to update the grid topology.

While [handling active leaves](https://github.com/paritytech/polkadot-sdk/blob/4059282fc7b6ec965cc22a9a0df5920a4f3a4101/polkadot/node/network/gossip-support/src/lib.rs#L210) it checks if the session changes, if so it triggers the function [`update_gossip_topology`](https://github.com/paritytech/polkadot-sdk/blob/4059282fc7b6ec965cc22a9a0df5920a4f3a4101/polkadot/node/network/gossip-support/src/lib.rs#L674). After generating the new topology, it is propagated with:

```rs
sender.send_message(NetworkBridgeRxMessage::NewGossipTopology {...})
```

### How is the flow that leads to the grid topology creation?

The [Gossip Support Subsystem](https://github.com/paritytech/polkadot-sdk/blob/master/polkadot/node/network/gossip-support/src/lib.rs), as any other subsystem, listens for [overseer active leaves updates](https://github.com/paritytech/polkadot-sdk/blob/4059282fc7b6ec965cc22a9a0df5920a4f3a4101/polkadot/node/network/gossip-support/src/lib.rs#L191) and that is the signal that might trigger a topology update.

For a new leaf activated, the subsystem will retrieve its session index (that's made through a runtime call), also the subsystem holds the `last_session_index` within its state, and it compares if the leaf session index is greater than its latest session index, if it is true then the function `update_gossip_topology` will be called.


### How is the new topology propagated?

The new topology is propagated by the function `update_gossip_topology` using the message `NetworkBridgeRxMessage::NewGossipTopology`. Actually what is propagated is not the instance of the topology but the information that is needed to produce the topology, which are:
- `Session Index`
- `Local Index` (our validator index)
- `Canonical Shuffling` (the validator indexes that were shuffled using [`fisher_yates_shuffle`](https://github.com/paritytech/polkadot-sdk/blob/4059282fc7b6ec965cc22a9a0df5920a4f3a4101/polkadot/node/network/gossip-support/src/lib.rs#L734))
- `shuffled_indices` (a mapping to find the shuffled validators by its index)

Now, this information is not propagated to all subsystems, [this event is being heard only by `Network Bridge Subsystem`](https://github.com/paritytech/polkadot-sdk/blob/cdf107de700388a52a17b2fb852c98420c78278e/polkadot/node/network/bridge/src/rx/mod.rs#L759). `Network Bridge Subsystem` will retrieve all the peer IDs for the new provided topology using the `Authority Discovery Service`, and once it get all it will create the `SessionGridTopology` instance and then dispatch it under the message `NetworkBridgeEvent::NewGossipTopology`

The propagation happens like in the following chain:

`Gossip Support Subsystem` sends `NetworkBridgeRxMessage::NewGossipTopology` to `Network Bridge Subsystem` that sends `NetworkBridgeEvent::NewGossipTopology` to all subsystems.

### What information is needed to produce the topology?

Before generating the topology we need first all the authorities that will take place in the new session. This information can be retrieved from the runtime. The next information is the randomness used to shuffle these validator indices which is retrieved from the runtime BABE Current Epoch Randomness. Besides that we also must have the current session index as well as our validator index a.k.a `local index`.

### Are there any existing RFCs?

No

## Design


* A validator producing a message sends it to its row-neighbors and its column-neighbors
* A validator receiving a message originating from one of its row-neighbors sends it to its column-neighbors
* A validator receiving a message originating from one of its column-neighbors sends it to its row-neighbors

### [Tracking Peers](https://github.com/paritytech/polkadot-sdk/blob/586ab7f65ed64e46088466f3a90d0ac79513a6b4/polkadot/node/network/protocol/src/grid_topology.rs#L56)

Under a given session we should track for all the peers few informations, such as:
- Peer IDs: For a single peer we should have its peer ID and in some cases we can have more than one peer ID
- Validator Index: the index of the validator in the **discovery keys**
- Authority Discovery ID: the authority discovery public key of the validator (used by network to discovery the peer ID of a given authority public key)

### [Session Grid Topology](https://github.com/paritytech/polkadot-sdk/blob/586ab7f65ed64e46088466f3a90d0ac79513a6b4/polkadot/node/network/protocol/src/grid_topology.rs#L69)

This struct represents the raw session grid topology and what are the validators that composes it.

#### Functionalities

- [Update Authority IDs](https://github.com/paritytech/polkadot-sdk/blob/645878a27115db52e5d63115699b4bbb89034067/polkadot/node/network/protocol/src/grid_topology.rs#L93)
    Given a peer id and a set of authority discovery id updates the topology info to include the peer id for that auth id.

- [Compute Grid Neighbors](https://github.com/paritytech/polkadot-sdk/blob/645878a27115db52e5d63115699b4bbb89034067/polkadot/node/network/protocol/src/grid_topology.rs#L115)
    Given the validator index, it produces the outgoing routing logic for a particular peer

- [Matrix Neighbors](https://github.com/paritytech/polkadot-sdk/blob/645878a27115db52e5d63115699b4bbb89034067/polkadot/node/network/protocol/src/grid_topology.rs#L155)
    Given the validator index and the length of the whole peers set, returns the row neighbors and column neighbors

#### Usage

The session grid topology is instantiated by `Network Bridge Subsystem` when it receives the `NetworkBridgeRxMessage::NewGossipTopology` event.

### [Grid Neighbors](https://github.com/paritytech/polkadot-sdk/blob/645878a27115db52e5d63115699b4bbb89034067/polkadot/node/network/protocol/src/grid_topology.rs#L186)

This struct extracts the actual row and column neighbors for a specific validator index from the `Session Grid Topology`.

#### Functionalities

- Should Routing To
    Given a peer id OR a validator index, check if we should route its message to our row OR column neighbor. That is done by checking if the received message came from a row neighbor or from a column neighbor.

- Peers Diff
    Given two topologies return difference in the set of peers




### Session Grid Topology Storage

Its important to highlight that each of the subsystems that use Session Grid Topology stores it differently. For example `Bitfield Distribution` only needs to hold the current and previous session grid topology, implemented under `SessionBoundGridTopologyStorage`, while `Approval Distribution` stores a map from session index to grid topology. I would recommend to place each storage method under the respective subsystem, given that each subsystem uses and stores the information in a specific way.
