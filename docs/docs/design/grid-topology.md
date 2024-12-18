# Grid Topology

* A validator producing a message sends it to its row-neighbors and its column-neighbors
* A validator receiving a message originating from one of its row-neighbors sends it to its column-neighbors
* A validator receiving a message originating from one of its column-neighbors sends it to its row-neighbors

### [Tracking Peers](https://github.com/paritytech/polkadot-sdk/blob/586ab7f65ed64e46088466f3a90d0ac79513a6b4/polkadot/node/network/protocol/src/grid_topology.rs#L56)

Under a given session we should track for all the peers few informations, such as:
- Peer IDs: For a single peer we should have its peer ID and in some cases we can have more than one peer ID
- Validator Index: the index of the validator in the **discovery keys**
- Authority Discovery ID: the authority discovery public key of the validator (used by network to discovery the peer ID of a given authority public key)

### [Session Grid Topology](https://github.com/paritytech/polkadot-sdk/blob/586ab7f65ed64e46088466f3a90d0ac79513a6b4/polkadot/node/network/protocol/src/grid_topology.rs#L69)

