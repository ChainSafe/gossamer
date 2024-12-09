# Prospective Parachains

[The prospective parachains subsystem](https://paritytech.github.io/polkadot-sdk/book/node/backing/prospective-parachains.html)
tracks and handles prospective parachain fragments and inform other backing-subsystems of work to be done.

## Subsystem Structure

The implementation must conform to the `Subsystem` interface defined in the `parachaintypes` package. It should live in
a package named `prospectiveparachains` under `dot/parachain/prospective-parachains`.

### Messages Received

The subsystem must be registered with the overseer and handle six subsystem-specific messages from it:

1. [`prospectiveparachains.IntroduceSecondedCandidate`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1379)

Inform the prospective parachains subsystem of a new seconded candidate, the response is either false if the candidate was rejected by prospective parachains, true otherwise (if it was accepted or already present)

2. [`prospectiveparachains.CandidateBacked`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1383)

Inform the prospective parachains subsystem that a previously introduced candidate has been backed. This requires that the candidate was successfully introduced in the past.

3. [`prospectiveparachains.GetBackableCandidates`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1391C2-L1391C23)

Try getting N backable candidate hashes along with their relay parents for the given parachain, under the given relay-parent hash, which is a descendant of the given ancestors. Timed out ancestors should not be included in the collection. N should represent the number of scheduled cores of this `ParaID`. A timed out ancestor frees the cores of all of its descendants, so if there's a hole in the supplied ancestor path, we'll get candidates that backfill those timed out slots first. It may also return less/no candidates, if there aren't enough backable candidates recorded.

4. [`prospectiveparachains.GetHypotheticalMembership`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1411)

Get the hypothetical or actual membership of candidates with the given properties under the specified active leave's fragment chain. For each candidate, we return a vector of leaves where the candidate is present or could be added. "Could be added" either means that the candidate can be added to the chain right now or could be added in the future (we may not have its ancestors yet). Note that even if we think it could be added in the future, we may find out that it was invalid, as time passes. If an active leaf is not in the vector, it means that there's no chance this candidate will become valid under that leaf in the future. If `fragment_chain_relay_parent` in the request is `Some()`, the return vector can only contain this relay parent (or none).

5. [`prospectiveparachains.GetMinimumRelayParents`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1428C2-L1428C24)

Get the minimum accepted relay-parent number for each para in the fragment chain for the given relay-chain block hash. That is, if the block hash is known and is an active leaf, this returns the minimum relay-parent block number in the same branch of the relay chain which is accepted in the fragment chain for each ParaID. If the block hash is not an active leaf, this will return an empty vector. ParaIDs which are omitted from this list can be assumed to have no valid candidate relay-parents under the given relay-chain block hash. ParaIDs are returned in no particular order.

6. [`prospectiveparachains.GetProspectiveValidationData`](https://github.com/paritytech/polkadot-sdk/blob/2ef2723126584dfcd6d2a9272282ee78375dbcd3/polkadot/node/subsystem-types/src/messages.rs#L1434C2-L1434C30)

Get the validation data of some prospective candidate. The candidate doesn't need to be part of any fragment chain, but this only succeeds if the parent head-data and relay-parent are part of the `CandidateStorage` (meaning that it's a candidate which is part of some fragment chain or which prospective-parachains predicted will become part of some fragment chain).

Additionally, the subsystem must handle the following general network bridge events and overseer signals:

1. `overseer.Conclude` -> should halt the subsystem
2. `overseer.ActiveLeaves` -> update the new activated leaf to the new scheduled paras, pre-populate the candidate storage with pending availability candidates and candidates from the parent leaf, populate the fragment chain, add it to the implicit view. Then mark the newly deactivated leaves as deactivated and update the implicit view. Finally, remove any relay parents that are no longer part of the implicit view.

The overseer must be modified to forward these messages to the subsystem.

### Messages Sent

When handling `ProspectiveParachainsMessage` messages, the subsystem sometimes need to retrieve informations from Runtime API Subsystem and sends a `RuntimeApiMessage::Request` message to the overseer to reach the subsystem requesting:

- `RuntimeApiRequest::ParaBackingState`.
- `RuntimeApiRequest::AvailabilityCores`

Also the prospective parachains subsystems needs informations from the relay chain, that is done through sending a message to Chain API Subsystem, the message goes to the overseer to reach the subsystem and we request:

- `ChainApiMessage::Ancestors`
- `ChainApiMessage::BlockHeader`

## Subsystem State

The subsystem state stores: 
- A relay chain block view data per-relay-parent hash. The relay chain block view data contains a hash map per parachain id of their fragment chains.
- Active leaves, which is a subset of the keys in the per-relay-parent view.
- Implicit View

## Message Handling Logic

- [`handle_active_leaves_update`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L183C10-L183C37)

Whenever a new relay chain block is imported, we should handle the `OverseerSignal::ActiveLeaves` message that contains the relay chain block of interest and a list of relay chains blocks hashes no longer of interest, a.k.a deactivated.

Here the handler should retrieve the `prospective_parachain_mode`, which is a runtime call asking for the async backing configurations of `max_candidate_depth` and `allowed_ancestry_len`. After that we should get the scheduled parachains throught the function `fetch_upcoming_paras` which should return a set of parachain ids. **IMPORTANT**: `fetch_upcoming_paras` has a fallback branch for when the runtime does not has the `ClaimQueue` runtime function, we should be aware that this branch will be removed soon as the runtime with the function is release everywhere, meaning that the branch will be useless, [check comment here](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L914).

Next the handler uses the activated hash to retrieve the amount of allowed ancestors, the first ancestor (parent of activated hash) will provide the previous fragment chains (as is one fragment chain per parachain). So for each scheduled parachain we need to fetch the current backing state of the scheduled parachain by calling `fetch_backing_state` that under the hood calls the `RuntimeApiMessage` requesting the `RuntimeApiRequest::ParaBackingState`, this will return the current parachain constraints and the pending availability candidates, which will be used to create a new scope for the active leaf that will be used to create and populate the fragment chain for each scheduled parachain.

- [`handle_introduce_seconded_candidate`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L497C10-L497C45)

This handler receives the candidate and the parachain id it belongs to, so the handler should be able to grab the correct fragment chain corresponding to that parachain id incoming, then we try to add, if the candidate is already present nothing happens, if it does not exist then we insert it, if it fails to insert then an error is logged and false is returned. 

- [`handle_candidate_backed`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L595C10-L595C33)

This handler receives the candidate hash and its parachain id and will try to mark the candidate hash as backed, the candidate should already exist in the parachain fragment chain otherwise this handler will fail, if the candidate is already marked as backed nothing happens.

- [`answer_get_backable_candidates`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L670C4-L670C34)

This handler receives a relay parent hash, parachain id, count and a set of ancestors and should return a list of backable candidate hashes based on the best chain of parachain's fragment chain

- [`answer_hypothetical_membership_request`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L752)

Given a set of candidates and a relay parent [HypotheticalMembershipRequest](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/subsystem-types/src/messages.rs#L1321), check if the candidates can be added as potentials into the fragment chain.

- [`answer_prospective_validation_data_request`](https://github.com/paritytech/polkadot-sdk/blob/fdb264d0df6fdbed32f001ba43c3282a01dd3d65/polkadot/node/core/prospective-parachains/src/lib.rs#L828C4-L828C46)

Given the parachain id, candidate relay parent, and a parent head data search in the available fragment chains (that are hold by the set of active leaves) informations to fullfil the `PersistedValidationData` and returns it. the handler should return nil if there was not enough info to build the pvd.
