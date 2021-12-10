# Gossamer `network` Package

This package implements the [peer-to-peer networking capabilities](https://crates.parity.io/sc_network/index.html)
provided by the [Substrate](https://docs.substrate.io/) framework for blockchain development. It is built on the
extensible [`libp2p` networking stack](https://docs.libp2p.io/introduction/what-is-libp2p/). `libp2p` provides
implementations of a number of battle-tested peer-to-peer (P2P) networking protocols (e.g. [Noise](#noise) for
[key exchange](#identities--key-management), and [Yamux](#yamux) for [stream multiplexing](#stream-multiplexing)), and
also makes it possible to implement the blockchain-specific protocols defined by Substrate (e.g. [syncing](#sync) and
[finalising](#GRANDPA) blocks, and maintaining the [transaction pool](#transactions)). The purpose of this document is
to provide the information that is needed to understand the P2P networking capabilities that are implemented by
Gossamer - this includes an introduction to P2P networks and `libp2p`, as well as detailed descriptions of the Gossamer
P2P networking protocols.

## Peer-to-Peer Networking & `libp2p`

[Peer-to-peer](https://en.wikipedia.org/wiki/Peer-to-peer) networking has been a dynamic field of research for over two
decades, and P2P protocols are at the heart of blockchain networks. P2P networks can be contrasted with traditional
[client-server](https://en.wikipedia.org/wiki/Client%E2%80%93server_model) networks where there is a clear separation of
authority and privilege between the maintainers of the network and its users - in a P2P network, each participant
possesses equal authority and equal privilege. `libp2p` is a framework for implementing P2P networks that was
modularized out of [IPFS](https://ipfs.io/); there are implementations in many languages including Go (used by this
project), Rust, Javascript, C++, and more. In addition to the standard library of protocols in a `libp2p`
implementation, there is a rich ecosystem of P2P networking packages that work with the pluggable architecture of
`libp2p`. In some cases, Gossamer uses the `libp2p` networking primitives to implement custom protocols for
blockchain-specific use cases. What follows is an exploration into three concepts that underpin P2P networks:
[identity & key management](#identity--key-management), [peer discovery & management](#peer-discovery--management), and
[stream multiplexing](#stream-multiplexing).

### Identity & Key Management

Many peer-to-peer networks, including those built with Gossamer, use
[public-key cryptography](https://en.wikipedia.org/wiki/Public-key_cryptography) (also known as asymmetric cryptography)
to allow network participants to securely identify themselves and interact with one another. The term "asymmetric"
refers to the fact that in a public-key cryptography system, each participant's identity is associated with a set of two
keys, each of which serve a distinct ("asymmetric") purpose. One of the keys in an asymmetric key pair is private and is
used by the network participant to "sign" messages in order to cryptographically prove that the message originated from
the private key's owner; the other key is public, this is the key that the participant uses to identify themselves - it
is distributed to network peers to allow for the verification of messages signed by the corresponding private key. It
may be constructive to think about a public key as a username and private key as a password, such as for a banking or
social media website. Participants in P2P networks that use asymmetric cryptography must protect their private keys, as
well as keep track of the public keys that belong to the other participants in the network. Gossamer provides a
[keystore](../../lib/keystore) for securely storing one's private keys. There are a number of Gossamer processes that
manage the public keys of network peers - some of these, such as
[peer discovery and management](#peer-discovery--management), are described in this document, but there are other
packages (most notably [`peerset`](../peerset)) that also interact with the public keys of network peers. One of the
most critical details in a network that uses asymmetric cryptography is the
[key distribution](https://en.wikipedia.org/wiki/Key_distribution) mechanism, which is the process that the nodes in the
network use to securely exchange public keys - `libp2p` supports [Noise](#noise), a key distribution framework that is
based on [Diffie-Hellman key exchange](https://en.wikipedia.org/wiki/Diffie%E2%80%93Hellman_key_exchange).

### Peer Discovery & Management

In a peer-to-peer network, "[discovery](https://docs.libp2p.io/concepts/publish-subscribe/#discovery)" is the term that
is used to describe the mechanism that peers use to find one another - this is an important topic since there is not a
privileged authority that can maintain an index of known/trusted network participants. The discovery mechanisms that
peer-to-peer networks use have evolved over time - [Napster](https://en.wikipedia.org/wiki/Napster) relied on a central
database, [Gnutella](https://en.wikipedia.org/wiki/Gnutella) used a brute-force technique called "flooding",
[BitTorrent](https://en.wikipedia.org/wiki/BitTorrent) takes a performance-preserving approach that relies on a
[distributed hash table (DHT)](https://en.wikipedia.org/wiki/Distributed_hash_table). Gossamer uses a `libp2p`-based
implementation of the [Kademlia](#kademlia) DHT for peer discovery.

### Stream Multiplexing

[Multiplexing](https://en.wikipedia.org/wiki/Multiplexing) allows multiple independent logical streams to share a common
underlying transport medium, which amortizes the overhead of establishing new connections with peers in a P2P network.
In particular, `libp2p` relies on "[stream multiplexing](https://docs.libp2p.io/concepts/stream-multiplexing/)", which
uses logically distinct "paths" to route requests to the proper handlers. A familiar example of stream multiplexing
exists in the TCP/IP stack, where unique port numbers are used to distinguish logically independent streams that share a
common physical transport medium. Gossamer uses [Yamux](#yamux) for stream multiplexing.

## Gossamer Network Protocols

The types of network protocols that Gossamer uses can be separated into "core"
[peer-to-peer protocols](#peer-to-peer-protocols), which are often maintained alongside `libp2p`, and
[blockchain network protocols](#blockchain-network-protocols), which
[Substrate](https://crates.parity.io/sc_network/index.html) implements on top of the `libp2p` stack.

### Peer-to-Peer Protocols

These are the "core" peer-to-peer network protocols that are used by Gossamer.

#### `ping`

This is a simple liveness check [protocol](https://docs.libp2p.io/concepts/protocols/#ping) that peers can use to
quickly see if another peer is online - it is
[included](https://github.com/libp2p/go-libp2p/tree/master/p2p/protocol/ping) with the official Go implementation of
`libp2p`.

#### `identify`

The [`identify` protocol](https://docs.libp2p.io/concepts/protocols/#identify) allows peers to exchange information
about each other, most notably their public keys and known network addresses; like [`ping`](#ping), it is
[included with `go-libp2p`](https://github.com/libp2p/go-libp2p/tree/master/p2p/protocol/identify).

#### Noise

[Noise](http://noiseprotocol.org/) provides `libp2p` with its [key distribution](#identity--key-management)
capabilities. The Noise protocol is [well documented](https://github.com/libp2p/specs/blob/master/noise/README.md) and
the Go implementation is maintained [under the official](https://github.com/libp2p/go-libp2p-noise) `libp2p` GitHub
organization. Noise defines a
[handshake](https://github.com/libp2p/specs/blob/master/noise/README.md#the-noise-handshake) that participants in a
peer-to-peer network can use to establish message-passing channels with one another.

#### Yamux

[Yamux (Yet another Multiplexer)](https://github.com/hashicorp/yamux) is a Golang library for
[stream-oriented multiplexing](#stream-multiplexing) that is maintained by [HashiCorp](https://www.hashicorp.com/) - it
implements a well defined [specification](https://github.com/hashicorp/yamux/blob/master/spec.md). Gossamer uses
[the official `libp2p` adapter](https://github.com/libp2p/go-libp2p-yamux) for Yamux.

#### Kademlia

[Kademlia](https://en.wikipedia.org/wiki/Kademlia) is a battle-tested
[distributed hash table (DHT)](https://en.wikipedia.org/wiki/Distributed_hash_table) that defines methods for managing a
dynamic list of peers that is constantly updated in order to make a P2P network more resilient and resistant to attacks.
Kademlia calculates a logical "distance" between any two nodes in the network by applying the xor operation to the IDs
of those two peers. Although this "distance" is not correlated to the physical distance between the peers, it adheres to
three properties that are [crucial to the analysis](https://en.wikipedia.org/wiki/Kademlia#Academic_significance) of
Kademlia as a protocol - in particular, these three properties are:

- the "distance" between a peer and itself is zero
- the "distance" between two peers is the same regardless of the order in which the peers are considered (it is
  [symmetric](https://en.wikipedia.org/wiki/Symmetry_in_mathematics))
- the shortest "distance" between two peers does not include any intermediate peers (it follows the
  [triangle inequality](https://en.wikipedia.org/wiki/Triangle_inequality))

Gossamer uses [the official `libp2p` implementation of Kademlia for Go](https://github.com/libp2p/go-libp2p-kad-dht).

### Blockchain Network Protocols

The `libp2p` stack is used to implement the blockchain-specific protocols that are used to participate in
"Substrate-like" networks - these protocols are divided into two types, [notification](#notification-protocols) and
[request/response](#requestresponse-protocols). The two types of protocols are described in greater details below, along
with the specific protocols for each type.

##### Notification Protocols

[Notification protocols](https://crates.parity.io/sc_network/index.html#notifications-protocols) allow peers to
unidirectionally "push" information to other peers in the network. Although the peer receiving the information must
explicitly accept a handshake in order to open a stream for a notification protocol, this stream does not allow the
receiver to "push" information to the sender.

###### Transactions

This protocol is used to notify network peers of [transactions](https://docs.substrate.io/v3/concepts/tx-pool/) that
have been locally received and validated. Transactions are used to access the
[public APIs of blockchain runtimes](https://docs.substrate.io/v3/concepts/extrinsics/#signed-transactions).

###### Block Announces

The block announce protocol is used to notify network peers of the creation of a new block. The message for this
protocol contains a [block header](https://docs.substrate.io/v3/getting-started/glossary/#header) and associated data,
such as the [BABE pre-runtime digest](https://crates.parity.io/sp_consensus_babe/digests/enum.PreDigest.html).

###### GRANDPA

[Finality](https://wiki.polkadot.network/docs/learn-consensus#finality-gadget-grandpa) protocols ("gadgets") such as
GRANDPA are often described in terms of "games" that are played by the participants in a network. In GRANDPA, this game
relates to voting on what blocks should be part of the canonical chain. This notification protocol is used by peers to
cast votes for participation in the GRANDPA game.

##### Request/Response Protocols

[These protocols](https://crates.parity.io/sc_network/index.html#request-response-protocols) allow peers to request
specific information from one another. The requesting peer sends a protocol-specific message that describes the request
and the peer to which the request was sent replies with a message.

###### Sync

The sync protocol allows peers to request more information about a block that may have been discovered through the
[block announce notification protocol](#block-announces). The `BlockRequest` and `BlockResponse` messages for this
protocol are defined in
[the `api.v1.proto` file](https://github.com/paritytech/substrate/blob/master/client/network/src/schema/api.v1.proto)
that ships with Substrate.

###### Light

Light clients, like [Substrate Connect](https://paritytech.github.io/substrate-connect/), increase the decentralization
of blockchain networks by allowing users to interact with the network _directly_ through client applications, as opposed
to using a client application to send a request to an intermediary node in the network. This protocol allows light
clients to request information about the state of the network. The `Request` and `Response` messages for this protocol
are defined in
[the `light.v1.proto`](https://github.com/paritytech/substrate/blob/master/client/network/src/schema/light.v1.proto)
that ships with Substrate.
