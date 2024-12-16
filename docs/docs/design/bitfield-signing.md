---
layout: default
title: Bitfield Signing Subsystem
permalink: /design/bitfield-signing/
---

# Bitfield Signing Subsystem

[The bitfield signing subsystem](https://paritytech.github.io/polkadot-sdk/book/node/availability/bitfield-signing.html)
is responsible for producing signed availability bitfields and passing them to
[the bitfield distribution subsystem](./bitfield-distribution.md). It only does that when running on a validator.

## Subsystem Structure

The implementation must conform to the `Subsystem` interface defined in the `parachaintypes` package. It should live in
a package named `bitfieldsigning` under `dot/parachain/bitfield-signing`.

### Messages Received

The subsystem must be registered with the overseer. It only handles `parachaintypes.ActiveLeavesUpdateSignal`. The
overseer must be modified to forward this message to the subsystem.

### Messages Sent

1. [`DistributeBitfield`](https://github.com/paritytech/polkadot-sdk/blob/1e3b8e1639c1cf784eabf0a9afcab1f3987e0ca4/polkadot/node/subsystem-types/src/messages.rs#L522)

```go
package parachaintypes

import "github.com/ChainSafe/gossamer/lib/common"

type DistributeBitfield struct {
	RelayParent common.Hash
	Bitfield    UncheckedSignedAvailabilityBitfield
}
```

As mentioned in [bitfield distribution](./bitfield-distribution.md), it makes sense to duplicate the type
`UncheckedSignedAvailabilityBitfield` as `CheckedSignedAvailabilityBitfield` or just `SignedAvailabilityBitfield` and
use it in `DistributeBitfield`.

2. [`AvailabilityStore::QueryChunkAvailability(CandidateHash, ValidatorIndex, response_channel)`](https://github.com/paritytech/polkadot-sdk/blob/1e3b8e1639c1cf784eabf0a9afcab1f3987e0ca4/polkadot/node/subsystem-types/src/messages.rs#L556)

This message is sent once for each occupied core, whenever the subsystem is notified of a new active leaf.

The type is already defined as `availabilitystore.QueryChunkAvailability` in
`dot/parachain/availability-store/messages.go`.

## Subsystem State

The subsystem needs access to the parachain session keys for signing bitfields. The appropriate `Keystore` should be
passed to the subsystem constructor as a dependency to facilitate testing. Since each new active leave results in a
goroutine being spawned, the subsystem needs to maintain state to receive data from them and to cancel them when
appropriate.

```go
package bitfieldsigning

import (
	"context"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

type signingTask struct {
	ctx      context.Context
	response <-chan parachaintypes.UncheckedSignedAvailabilityBitfield
}

type BitfieldSigning struct {
	keystore keystore.Keystore
	tasks    map[common.Hash]*signingTask
}
```

Ideally the implementation should avoid lock contention around the keystore. Since the signing key remains the same for
the duration of the signing task, the subsystem could pass in the key pair or just the private key. The implementer
should double-check that this approach is thread-safe.

Again, this should probably use a (to be created) type `CheckedSignedAvailabilityBitfield`.  

## Message Handling Logic

### `parachaintypes.ActiveLeavesUpdateSignal`

For all leaves that have been deactivated, cancel the signing tasks associated with them.

For all activated leaves, spawn a new signing task. The task needs to perform the following steps:

1. Wait a while to let the availability store get populated with information about erasure chunks. In the Parity node,
this delay is currently set to [1500ms](https://github.com/paritytech/polkadot-sdk/blob/1e3b8e1639c1cf784eabf0a9afcab1f3987e0ca4/polkadot/node/core/bitfield-signing/src/lib.rs#L49).

2. Query the set of [availability cores](https://paritytech.github.io/polkadot-sdk/book/runtime-api/availability-cores.html)
for the given leaf from the runtime.

3. For each core, concurrently check whether the core is occupied and if so, query the availability store using
`QueryChunkAvailability`.

4. Collect the results of the queries into a bitfield, sign it and send a `DistributeBitfield` message to the overseer.
