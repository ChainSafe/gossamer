---
layout: default
title: Candidate Backing
permalink: /design/candidate-backing/
---

# Candidate backing subsystem

## Overview

* The Candidate Backing subsystem ensures every parablock considered for relay block inclusion has been
    * seconded by at least one parachain validator
    * approved by a quorum of parachain validators

* Parablocks that fail to secure enough validator votes for correctness are rejected.

* If a block that has received a valid vote later proves to be invalid, the validators who initially voted for it are subject to slashing.

---

* The role of this subsystem it to produce backable candidates for inclusion in new relay-chain blocks.
it does so by
    * issuing signed Statements
    * tracking received statements signed by other validators
    * Once enough statements are received, they can be combined into backing for specific candidates

* it does not attempt to choose a single authoritative one. The choice of which candidate actually gets included is ultimately up to the block author.

* Once a sufficient quorum has agreed that a candidate is valid, this subsystem notifies the [Provisioner](https://paritytech.github.io/polkadot-sdk/book/node/utility/provisioner.html) to engage the block production mechanisms to include the parablock into relay chain block

---
* There are two types of statements: `Seconded` and `Valid`.
`Seconded` implies `Valid`, and nothing should be stated as `Valid` unless its already been `Seconded`.

* Only parachain validators may Second the candidates, and they may only second one candidate per depth per active leaf.
    * before async backing, depth is always 0

 
## candidate backing message
### * Second
```
type SecondMessage struct {
	RelayParent             common.Hash
	CandidateReceipt        parachaintypes.CandidateReceipt
	PersistedValidationData parachaintypes.PersistedValidationData
	PoV                     parachaintypes.PoV
}
```
- second: validate the candidate and put forward by a parachain validator to other validators for validating the candidate.

- Collator Protocol Subsystem sends `SecondMessage` to the Candidate Backing Subsystem.

**Functionality:**
* candiate backing subsystem request validation from validation subsystem and generates appropriate statement.

* If the candidate is found Invalid, and it was recommended to us to second by our own Collator Protocol subsystem, a message is sent to the Collator Protocol subsystem with the candidate's hash so that the collator which recommended it can be penalized.

* If the candidate found Valid
    * sign and dispatch a Seconded statement to be gossiped to peers only if we have not seconded any other candidate and have not signed a Valid statement for the requested candidate. 
    * store our issued statement into our statement table.
    * Signing both a Seconded and Valid message is a double-voting misbehavior with a heavy penalty.
        * It's just a theory, as far as I have noticed in codebase, they ignores missbehaviour

### * Statement
```
type StatementMessage struct {
	RelayParent         common.Hash
	SignedFullStatement parachaintypes.SignedFullStatementWithPVD
}
```
- StatementMessage represents a validator's assessment of a specific candidate received from other parachain validators.

- Statement Distribution Subsystem sends `StatementMessage` to the Candidate Backing subsystem.

**Functionality:**
- If the statement in the message is `Valid`, store it in our statement table.

- If the statement in the message is `Seconded` and it contains a candidate that belongs to our assignment, request the corresponding `PoV` from the backing node via `AvailabilityDistribution`.
    - backing nodes are the nodes who have issued the statement for a candidate. 

- after getting `POV`, launch validation and Issue our own Valid or Invalid statement as a result.

- If there are disagreements regarding the validity of this assessment, they should be addressed through the Disputes Subsystem
- Meanwhile, agreements are straightforwardly count to the quorum.


### * GetBackableCandidates
```
type GetBackableCandidatesMessage struct {
	Candidates []*CandidateHashAndRelayParent
	ResCh      chan []*parachaintypes.BackedCandidate
}
```

- Provisioner Subsystem(Utility Subsystem) sends `GetBackableCandidatesMessage` to the Candidate Backing subsystem.

- `GetBackableCandidatesMessage` represents a request to get a set of backable candidates that could be backed in a child of the given relay-parent.

**Functionality:**
- from the list of candidate we have received in the message, identify those who have received enough supporting statement by the parachain validators. and return them using response channel.
- Minimum supporting statements required in legasy backing(Before async): 2

---
<br/>

**[Candidate Backing - Elastic Scaling](./candidate-backing-elastic-scaling.md) contains the updated design of candidate backing subsystem to support elastic scaling.**