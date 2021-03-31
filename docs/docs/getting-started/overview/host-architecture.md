---
layout: default
title: Host Architecture
permalink: /host-architecture/
---

## Nodes

Gossamer includes a base node implementation called the **host node** that implements a shared base protocol for all blockchain protocols within the Polkadot ecosystem. The **host node** is used as the foundation for all **official nodes** within Gossamer and all **custom nodes** built with Gossamer.

### Host Node

The **host node** is the base node implementation. As the base node implementation, the **host node** is not complete without a configuration file, genesis file, compiled runtime, and runtime imports.

### Official Nodes

The **gssmr node** is an official node implementation for the Gossamer Testnet - a configuration file, genesis file, compiled runtime, and runtime imports used with the **host node**.

The **kusama node** is an official node implementation for the Kusama Network - a configuration file, genesis file, compiled runtime, and runtime imports used with the **host node**.

The **polkadot node** is an official node implementation for the Polkadot Network - a configuration file, genesis file, compiled runtime, and runtime imports used with the **host node**.

<!-- ### Custom Services

See [Custom Services](/advanced/custom-servives) for more information about building custom node implementations. -->

## Node Services

The **node services** are the main components of the **host node**:

- **[Core Service](#core-service)**
- **[Network Service](#network-service)**
- **[RPC Service](#rpc-service)**
- **[State Service](#state-service)**

Each **node service** adheres to a common interface:

```go
type Service interface {
	Start() error
	Stop() error
}
```

- All goroutines within **node services** should start inside `Start`
- All **node services**  can be terminated without consequences by calling `Stop`
- All **node services** whose `Start` method has not been called can be discarded without consequences

### Core Service

The **core service** is responsible for block production and finalization (consensus) and processing messages received from the **network service**; it initializes <a target="_blank" rel="noopener noreferrer" href="https://research.web3.foundation/en/latest/polkadot/BABE/Babe/">BABE</a> sessions and <a target="_blank" rel="noopener noreferrer" href="https://github.com/w3f/consensus/blob/master/pdf/grandpa.pdf">GRANDPA</a> rounds and validates blocks and transactions before committing them to the **state service**. 

- only the **core service** writes to block state
- only the **core service** writes to storage state

### Network Service

The **network service** is responsible for coordinating network host and peer interactions. It manages peer connections, receives and parses messages from connected peers and handles each message based on its type. If the message is a non-status message and we have confirmed the status of the connected peer, the message is sent to the **core service** to be processed.

- the **network service** only reads from block state
- only the **network service** writes to network state

#### Host Submodule

The **host submodule** is a wrapper for the libp2p host. This is used to abstract away the details of libp2p and to provide a simple reusable interface for the network host.

```go
type host struct {
	ctx        context.Context
	h          libp2phost.Host
	dht        *kaddht.IpfsDHT
	bootnodes  []peer.AddrInfo
	protocolID protocol.ID
}
```


### RPC Service

The **rpc service** is an implementation of the <a target="_blank" rel="noopener noreferrer" href="https://github.com/w3f/PSPs/blob/master/PSPs/drafts/psp-6.md">JSON-RPC PSP</a>.

### State Service

The **state service** is the source of truth for all chain and node state.


## Block production 

<img src="/assets/img/block_production.png" alt="block production" />

A block is broken down into two sections, **the header** & **the body**.

The first step is to get information about the parent block, for new blocks, this would be the head of the chain.

The **parent hash** and **state root** is added to the block header _(point 1 & 2)_

We then need to process the **extrinsics** _(point 3)_, extrinsics is used to describe any additional information to include in the block that isn't explicitly required to produce a block, such as **signed transactions** from accounts, or additional information added by the block author, like a **timestamp**.

Once processed, we get whats called an **extrinsic root** _(point 4)_, this is used to verify the extrinsics when publishing later on.

Finally, once all the contents of the block are in place, we then create the **digest**_(point 5)_, this is used to verify the blocks contents.

Information regarding the authoring of the block is stored in the **Babe header**, this allows verification of the block producer, the block, and the authority of the producer.

Finally, the last item of the digest, much like transactions, is a signature known as a **Seal**, this is a **signature of the header** to allow immediate verification of the integrity of a block.