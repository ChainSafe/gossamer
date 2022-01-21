# Gossamer `digest` Package

In general, the term "digest" refers to a summary that includes important pieces of information. In the case of
Polkadot-like networks, a "[digest](https://docs.substrate.io/v3/getting-started/glossary/#digest)" is an extensible
field of a block header that provides information related to some network protocol, such as those implemented by a
[consensus engine](https://docs.substrate.io/v3/getting-started/glossary/#consensus-algorithm) or
[blockchain runtime](https://docs.substrate.io/v3/getting-started/glossary/#runtime). This package deals exclusively
with the handling of [consensus digests](https://crates.parity.io/sp_runtime/enum.DigestItem.html#variant.Consensus),
which are messages that originate from a blockchain runtime and are consumed by consensus engines. Consensus digests are
formally described in
[Section 5.1.2 of the Polkadot Specification](https://w3f.github.io/polkadot-spec/develop/_common_consensus_structures.html).
What follows is a description of Gossamer's consensus digest `Handler`, including the consensus engines the `Handler`
supports, and the messages the `Handler` may receive & corresponding actions it can dispatch.

## Handler

The [digest `Handler`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot/digest#Handler) implements the
[`Service` interface](https://pkg.go.dev/github.com/ChainSafe/gossamer/lib/services#Service), which means that it
exposes `Start` and `Stop` functions. When a
[Gossamer `Node`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot#Node) is started, a digest `Handler` is created
with the [`NewHandler`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot/digest#NewHandler) function and started
along with the Gossamer node's other services. The digest `Handler` maintains two `goroutines`, `handleBlockImport` and
`handleBlockFinalisation`, which handle imported and finalised blocks
respectively - these `goroutines` listen for messages on channels provided by the
[`BlockState`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot/state#BlockState).

The digest `Handler` also exposes two public functions:
[`NextGrandpaAuthorityChange`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot/digest#Handler.NextGrandpaAuthorityChange)
and [`HandleDigests`](https://pkg.go.dev/github.com/ChainSafe/gossamer/dot/digest#Handler.HandleDigests).
`NextGrandpaAuthorityChange` is consumed by the
[GRANDPA service](https://pkg.go.dev/github.com/ChainSafe/gossamer/lib/grandpa#Service) 
to determine the block number of the next GRANDPA authority change.

`HandleDigests` is invoked by the core service for each new block; this function reads the consensus digests from the
block and dispatches actions based on the digest contents. The digest `Handler` supports digests from the
BABE (authorship) and GRANDPA (finalisation) consensus engines - the following sections describe the messages in these
digests and the actions Gossamer takes when it receives them.

## BABE Messages

[BABE](https://wiki.polkadot.network/docs/learn-consensus#block-production-babe) is a block production
algorithm that assigns block authorship rights over periods of time referred to as "slots", which are a subdivision of an "epoch". One of the
benefits of BABE is that it is able to provide a blockchain runtime with secure, decentralized randomness. Furthermore,
BABE seeks to ensure the liveness of a blockchain by defining multiple tiers of potential block producers for each
epoch. BABE digests may contain 1 of 3 messages, each of which are described below.

### Next Epoch

This message is issued by the runtime on the first block of every epoch - it provides the BABE consensus engine with the
authority set and randomness for the _next_ epoch.

### Disabled

A message of this type will contain the ID of an authority; this authority should cease all authority functionality and
all other authorities should ignore any authority-related messages from the identified authority.

### Next Config

Messages of this type may only be issued in the first block of an epoch. These types of messages supply configuration
parameters that should be applied from the _next_ epoch onwards. The parameters in this configuration relate to how
backup authorities are selected.

## GRANDPA Messages

Provable [finality](https://wiki.polkadot.network/docs/glossary#finality) securely assures participants in a
blockchain network that a block (and the
[transactions](https://docs.substrate.io/v3/getting-started/glossary/#transaction) it contains) will not be reverted.
Gossamer implements the [GRANDPA](https://wiki.polkadot.network/docs/learn-consensus#finality-gadget-grandpa) finality
protocol, which, like BABE, uttilizes a rotating validator set. The GRANDPA
protocol defines the following consensus digest messages:

### Scheduled Change

These messages contain a list of new authority IDs and a delay, specified as a number of _finalised_ blocks, after which
the change should be applied.

### Forced Change

This message is like that for scheduled changes, however the delay is calculated using _imported_ blocks (as opposed to
finalised blocks), which means that the change is valid for multiple candidate chains.

### Disabled

A message of this type will contain the ID of an authority; this authority should cease all authority functionality and
all other authorities should ignore any authority-related messages from the identified authority. _Note: Gossamer has
not implemented supported for messages of this type._

### Pause

This messages specifies as a delay after which the current authority set should be paused.

### Resume

This messages specifies as a delay after which the current authority set should be resumed.
