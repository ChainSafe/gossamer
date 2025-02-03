# Availability Distribution Subsystem

[The availability distribution subsystem](https://paritytech.github.io/polkadot-sdk/book/node/availability/availability-distribution.html)
is responsible for retrieving and distributing availability data such as PoVs and erasure chunks.

## Subsystem Structure

The implementation must conform to the `Subsystem` interface defined in the `parachaintypes` package. It should live in
a package named `availabilitydistribution` under `dot/parachain/availability-distribution`.

### Messages Received

The subsystem must be registered with the overseer and handle three subsystem-specific messages from it:

#### [`parachain.ChunkFetchingRequest`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/chunk_fetching.go#L14)

This is a network message received from other validators in the core group.

Protocol versions `v1` and `v2` exist for chunk fetching. The `ChunkFetchingRequest` message is identical in both
versions. The `ChunkFetchingResponse` message in `v1` omits the chunks index. In `v2` it contains [the full
`ErasureChunk` object](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/network/protocol/src/request_response/v2.rs#L99).
[Our implementation](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/chunk_fetching.go#L89)
only covers `v1` so far. We should probably add the chunk index to this struct and only support `v2` for the initial
implementation, analogous the approach in other subsystems.

#### [`parachain.PoVFetchingRequest`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/pov_fetching.go#L14)

This is a network message received from other validators in the core group.

#### [`AvailabilityDistributionMessageFetchPoV`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/types/overseer_message.go#L143-L144)

This is an internal message from the backing subsystem, instructing the availability distribution subsystem to fetch a
PoV for a given candidate from a specific validator in the core group.

Additionally, the subsystem must handle `parachaintypes.ActiveLeavesUpdateSignal`.

The overseer must be modified to forward these messages to the subsystem.

### Messages Sent

#### [`NetworkBridgeTxMessage::SendRequests(Requests, IfDisconnected::ImmediateError)`](https://github.com/paritytech/polkadot-sdk/blob/41b6915ecb4b5691cdeeb585e26d46c4897ae151/polkadot/node/subsystem-types/src/messages.rs#L433)

This network bridge message is used to request PoVs and erasure chunks from other validators in the core group.

#### [`availabilitystore.QueryChunk`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/availability-store/messages.go#L42)

This is an internal message sent to the availability store subsystem for retrieving a previously stored erasure chunk.

#### [`availabilitystore.StoreChunk`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/availability-store/messages.go#L72)

This is an internal message sent to the availability store subsystem containing an erasure chunk that has been fetched
from another validator in the core group.

#### [`availabilitystore.QueryAvailableData`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/availability-store/messages.go#L19)

This is an internal message sent to the availability store subsystem for retrieving [a PoV plus persisted validation
data](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/availability-store/messages.go#L94-L95).

## Subsystem State

The subsystem calls the following runtime functions:

- [`ParachainHostSessionIndexForChild()`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/lib/runtime/wazero/instance.go#L1258)

- [`ParachainHostSessionInfo()`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/lib/runtime/wazero/instance.go#L1316-L1317)

- [`ParachainHostAvailabilityCores()`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/lib/runtime/wazero/instance.go#L1211-L1212)

The Parity node implementation caches this data using an instance of [the `RuntimeInfo` struct](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/subsystem-util/src/runtime/mod.rs#L75).
Our implementation should also cache the results of the above listed runtime functions, either in a separate struct or
directly inside the subsystem struct. The cache needs to be invalidated at the start of a new session.

The Parity node also maintains two long-running tasks for fetching and receiving erasure chunks and PoVs. For an initial
implementation it is sufficient to simply start goroutines for this on demand.
The task for fetching chunks is implemented as a struct that keeps additional state. Part of that state is an instance
of [`SessionCache`](https://github.com/paritytech/polkadot-sdk/blob/523e62560eb5d9a36ea75851f2fb15b9d7993f01/polkadot/node/network/availability-distribution/src/requester/session_cache.rs#L36),
which is an LRU map from session index to [`SessionInfo`)](https://github.com/paritytech/polkadot-sdk/blob/523e62560eb5d9a36ea75851f2fb15b9d7993f01/polkadot/node/network/availability-distribution/src/requester/session_cache.rs#L47).
This seems somewhat redundant with the above-mentioned `RuntimeInfo` instance. The implementer of the subsystem should
determine whether both are needed and how to model them in our codebase.
The second important part of the state of the chunk fetcher is a map from candidate hash to a running fetch task. Given
the substantial logic, state and parameterization of this task, it makes sense to implement it as a separate type.

Finally, the subsystem needs access to ancestors of active leafs in the block state. In the Parity node, this is
accomplished via the ChainAPI utility subsystem. Since Gossamer does not have this subsystem yet, a `BlockState`
interface with the required methods should be defined, analogous to how it is done in
[the candidate backing subsystem](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/backing/candidate_backing.go#L83).
An object implementing this interface should be passed to the subsystem constructor held in its state. When querying
the state, the subsystem also needs a constant denoting how many ancestors of the active leaf it should consider.

```go
package availabilitydistribution

const leafAncestryLenWithinSession = 3

type BlockState interface {
	GetHeader(hash common.Hash) (*types.Header, error)
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

type fetchChunkTask struct {
	ctx    context.Context
	leaves []common.Hash
}

type AvailabilityDistribution struct {
	sessionCache map[SessionIndex]SessionInfo
	fetches      map[CandidateHash]*fetchChunkTask
	blockState   BlockState
}
```

## Message Handling Logic

###[`parachain.ChunkFetchingRequest`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/chunk_fetching.go#L14)

Send `availabilitystore.QueryChunk` to the availability store with the requested candidate hash and validator index.
Receive the response and forward it to the peer that sent the request.

In the Parity node this is implemented in [`answer_chunk_request()`](https://github.com/paritytech/polkadot-sdk/blob/dada6cea6447ce2730a3f3b43a3b48b7a5c26cf6/polkadot/node/network/availability-distribution/src/responder.rs#L215).

### [`parachain.PoVFetchingRequest`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/pov_fetching.go#L14)

Send `availabilitystore.QueryAvailableData` to the availability store with the requested candidate hash. Receive the
response and forward the PoV to the peer that sent the request.

In the Parity node this is implemented in [`answer_pov_request()`](https://github.com/paritytech/polkadot-sdk/blob/dada6cea6447ce2730a3f3b43a3b48b7a5c26cf6/polkadot/node/network/availability-distribution/src/responder.rs#L189).

### [`AvailabilityDistributionMessageFetchPoV`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/types/overseer_message.go#L143-L144)

Retrieve the session info either from cache or from the runtime. Retrieve the authority ID from the discovery keys in
the session info, using the validator index included in `AvailabilityDistributionMessageFetchPoV`. Send a
[`PoVFetchingRequest`](https://github.com/ChainSafe/gossamer/blob/32256782470db15efb3b7b4ba687311dd4e7cdce/dot/parachain/pov_fetching.go#L14)
via the overseer to the network bridge. Send the response back over the channel included in
`AvailabilityDistributionMessageFetchPoV`.

In the Parity node this is implemented in [`fetch_pov()`](https://github.com/paritytech/polkadot-sdk/blob/dada6cea6447ce2730a3f3b43a3b48b7a5c26cf6/polkadot/node/network/availability-distribution/src/pov_requester/mod.rs#L44).

### [`parachaintypes.ActiveLeavesUpdateSignal`](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/network/availability-distribution/src/requester/mod.rs#L109)

#### Collect ancestors of activated leaves in same session

For each newly activated leaf, retrieve the headers of up to `leafAncestryLenWithinSession` ancestors. Instantiate the
runtime for the first ancestor and call `ParachainHostSessionIndexForChild()` to get the session index of the newly
activated leaf. If no ancestors have been found, set this value to 0 instead.

If the list of ancestors is not empty, iterate through the remaining ones and check that they have the same session
index as the newly activated leaf. Stop iterating if a different session index is found and discard those ancestors.

In the Parity node this part of handling `ActiveLeavesUpdateSignal` is implemented in
[`get_block_ancestors_in_same_session()`](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/network/availability-distribution/src/requester/mod.rs#L281).

#### Start or update chunk fetching tasks

For each newly activated leaf plus the ancestors retrieved in the previous step, instantiate a runtime instance and call
`ParachainHostAvailabilityCores()`. Filter out all cores that are unoccupied or not associated with the relay parent.
For each core, check whether there is already a fetch task running for the candidate hash associated with the core. If
yes, add the newly activated leaf to the state of the fetch task.
If not, determine the chunk index for the core and start a new fetch task for this chunk.

In the Parity node this part of handling `ActiveLeavesUpdateSignal` is implemented in
[`get_occupied_cores()`](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/subsystem-util/src/runtime/mod.rs#L353)
and [`add_cores()`](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/network/availability-distribution/src/requester/mod.rs#L184).

#### Cancel obsolete chunk fetching tasks

For each deactivated leaf, remove it from the state of all running fetch tasks. Cancel all fetch tasks that have an
empty list of leaves.

In the Parity node this part of handling `ActiveLeavesUpdateSignal` is implemented in
[`stop_requesting_chunks()`](https://github.com/paritytech/polkadot-sdk/blob/4c618a83d33281fe96f0e2b68a111ed227af22c0/polkadot/node/network/availability-distribution/src/requester/mod.rs#L169)
