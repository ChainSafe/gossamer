---
layout: default
title: Grid Tracker
permalink: /design/grid-tracker/
---

# Statement Distribution: Grid Tracker

Grid tracker is a knowledge data struct that stores authorities for a given relay parent.

- **received**: maps a validator index to a set of candidate hashes where each received candidate hash has a summary (like a gist) of the candidate, that is called `Manifest Summary`, in this summary we have the parent hash, group index and a statement filter.
- **confirmed backed**: maps a candidate hash, that we already have marked as backed, to a set of informations that helps us to understand the group where this candidate was originated as well as how statements already was circulated and what statements for that candidate are pending to what validator (possible recipients) and what could be the possible senders, that is called `MutualKnowledge`
- **unconfirmed**: maps candidates not backed to a list of sender (validator index) and group index
- **pending manifests**: stores a list of manifests that should be sent to a specific validator.
  - **example 1.** we've received a `Full Manifest` from some validator (that is allowed to send us that manifest due to the grid topology constraints), and the received manifest is from a candidate we already know that is backed, then we should insert in the pending manifests an `Acknowledgement Manifest` for that candidate hash for that sender validator index
  - **example 2.** we've marked a candidate as backed, so we should query every validator we can send (given the grid topology) and insert in the pending manifests a `Full Manifest`, meaning that we will send them this message, as well query all the validators who have sent to us manifests for that hash (if any) and for them insert an `Acknowledgement Manifest`.
- **pending statements**: is essentially a message queue that tracks which validator statements need to be sent to which validators. It maps a target validator, who needs to receive the information to a set o tuples containing the originator, who created the statement, and the statement (either Seconded or Valid).

##### What is a manifest?

Manifest is a kind of message that can be sent/received by validators. Currently there is two types of manifests `BackedCandidateManifest` or `Full Manifest`, and `BackedCandidateAcknowledgement` or `Acknowledgement Manifest`.

- `Full Manifest`: is a message that advertise a description of a backed candidate and stored statements. This message is send whenever a validator observers a candidate as backed.

- `Acknowledgement Manifest`: acknowledge that a backed candidate is fully known, is sent in response of a `Full Manifest` message.

The Grid Tracker relies on [`StatementFilter`](https://github.com/paritytech/polkadot-sdk/blob/c29e72a8628835e34deb6aa7db9a78a2e4eabcee/polkadot/node/network/protocol/src/lib.rs#L630) which is a struct that holds in a bitfield what CandidateStatement (Seconded or Valid) a validator index produced in a group. It is used under the Mutual Knowledge where we can easily. check if a remote peer has the same statements that our node has, and in a case one or more is missing we can provide it.

We can split the Grid Tracker Behaviours in 4 parts:

#### Manifest Handling

This section is composed of 
- **importing manifests**: should validate incoming manifests, checking if the sender is allowed to gossip informations about the group, validate the manifest format, update local knowledge and might trigger acknowledgement responses. If necessary add informations to the pending statements tracking.
  - **example**: validator V0 sends us a manifest and we noticed that for him is pending statements from validator V4 and V5 (based on our local knowledge), in this case we should keep track of that under pending statements and in the first oportunity send to V0 these infos.

- **adding backed candidates**: registers a new parachain candidate that has received sufficient backing. Creates tracking entry for the candidate in confirmed_backed
Processes any previously received unconfirmed manifests. Determines which validators should receive knowledge about this candidate. Returns list of validators to notify with appropriate manifest types

- **manifests sending**: records that we've shared candidate knowledge with a validator. Updates mutual knowledge tracking to reflect what we've told them. Queues any statements they might be missing for direct transmission. Removes this candidate from pending manifests for that validator

#### Knowledge Query Methods

- **pending manifests for**: lists all candidates we need to advertise to a specific validator. Returns pairs of candidate hashes and manifest types to be sent

- **pending statements for**: gets specific statements a validator is missing about a candidate. Returns a filtered view of statements they need to receive.

- **all pending statements for**: lists all statements waiting to be sent to a validator. Returns complete list of pending statements for that validator. Prioritizes "Seconded" statements before "Valid" 

- **advertised statements**: retrieves what a validator has told us they know about a candidate. Returns their knowledge as a StatementFilter if available.

#### Network Routing Logic
- **can request**: determines if a validator may request manifest data from us. Checks if we've sent them a manifest but haven't received theirs

- **direct statement providers**: Identifies validators who can send us a specific statement. Analyzes network topology to find authorized senders. Indicates if the sender should already know about the statement.

- **direct statement targets**: Determines validators who should receive a statement from us. Returns validators who need this statement based on mutual knowledge

#### Statement Tracking
- **learned fresh statement**: Updates tracking when we learn about a new validator statement. Records the statement in our local knowledge. If actually new, adds it to pending statements for relevant validators

- **sent or received direct statement**: Records statement transmission between validators. Updates mutual knowledge tracking between validators. Removes statement from pending queue once transmitted

## Extra helper structs
#### ReceivedManifests

This struct tracks the candidate manifests received for a given ValidatorIndex.

So a given validator index can have a set of different candidate hashes each one mapping to a Manifest Summary that contains the parent hash, group index and a statement knowledge (indicating validators that seconded and validators that validated the candidate).

The importing process of a received manifest is straightfoward.

- For new candidate hash we should check if the seconded in group respect the limit defined by the session, see [statement-distribution/src/v2/mod.rs#L2414](https://github.com/paritytech/polkadot-sdk/blob/3dc3a11cd68762c2e5feb0beba0b61f448c4fc92/polkadot/node/network/statement-distribution/src/v2/mod.rs#L2414). This means that, the imported manifest should have the amount of seconded statements below the defined limit.
- If the candidate already exists we can update it if 
  - the informations in the incoming manifest summary does not conflict with the current informations
  - the imported manifest should have the amount of seconded statements below the defined limit.


#### KnownBackedCandidate

This struct tracks:

- Which validators know about this candidate
- What statements (Seconded or Valid) each validator knows about
- What knowledge has been shared between validators about this candidate

The key behavior for this struct is to share pending statements (Second, Valid) with validators that knows about a candidate.
