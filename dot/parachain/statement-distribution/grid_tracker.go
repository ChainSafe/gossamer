// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

// manifestKind is the kind of backed candidate manifest we should send to a remote peer.
type manifestKind uint8

const (
	// full manifests contain information about the candidate and should be sent
	// to peers which aren't guaranteed to have the candidate already.
	full manifestKind = iota

	// acknowledgement manifests omit information which is implicit in the candidate
	// itself, and should be sent to peers which are guaranteed to have the candidate
	// already.
	acknowledgement
)

type candidateHashByManifestKind map[parachaintypes.CandidateHash]manifestKind

// gridTracker tracks knowledge from authorities within the grid for a particular relay-parent.
type gridTracker struct {
	received         map[parachaintypes.ValidatorIndex]receivedManifests
	confirmedBacked  map[parachaintypes.CandidateHash]knownBackedCandidate
	pendingManifests map[parachaintypes.ValidatorIndex]candidateHashByManifestKind
}

// newGridTracker returns a new [gridTracker]
func newGridTracker() *gridTracker {
	return &gridTracker{
		received:         make(map[parachaintypes.ValidatorIndex]receivedManifests),
		confirmedBacked:  make(map[parachaintypes.CandidateHash]knownBackedCandidate),
		pendingManifests: make(map[parachaintypes.ValidatorIndex]candidateHashByManifestKind),
	}
}

// importManifest attempts to import a manifest advertised by a remote peer.
//
// This checks whether the peer is allowed to send us manifests
// about this group at this relay-parent. This also does sanity
// checks on the format of the manifest and the amount of votes
// it contains. It assumes that the votes from disabled validators
// are already filtered out.
// It has effects on the stored state only when successful.
//
// This returns a `bool` on success, which if true indicates that an acknowledgement
// is to be sent in response to the received manifest. This only occurs when the
// candidate is already known to be confirmed and backed.
func (g *gridTracker) importManifest(
	sessionTopology *sessionTopologyView,
	groups groups,
	candidateHash parachaintypes.CandidateHash,
	secondingLimit uint,
	manifest manifestSummary,
	kind manifestKind,
	sender parachaintypes.ValidatorIndex,
) (bool, error) {
	panic("not implemented")
}

// addBackedCandidate adds a new backed candidate to the tracker.
// This yields a list of validators which we should either advertise to
// or signal that we know the candidate, along with the corresponding
// type of manifest we should send.
func (g *gridTracker) addBackedCandidate(
	sessionTopology *sessionTopologyView,
	candidateHash parachaintypes.CandidateHash,
	groupIndex parachaintypes.GroupIndex,
	localKnowledge statementFilter,
) map[parachaintypes.ValidatorIndex]manifestKind {
	panic("not implemented")
}

// mainfestSentTo notes that a backed candidate has been advertised to a given validator.
func (g *gridTracker) mainfestSentTo(
	groups groups,
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	localKnowledge statementFilter,
) {
	panic("not implemented")
}

// pendingManifestsFor returns a vector of all candidates pending manifests for
// the specific validator, and the type of manifest we should send.
func (g *gridTracker) pendingManifestsFor(
	validatorIndex parachaintypes.ValidatorIndex,
) map[parachaintypes.CandidateHash]manifestKind {
	panic("not implemented")
}

// pendingStatementsFor returns a statement filter indicating statements that a given peer is
// awaiting concerning the given candidate, constrained by the statements we have ourselves.
func (g *gridTracker) pendingStatementsFor(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) *statementFilter {
	panic("not implemented")
}

type validatorIndexWithCompactStatement[T parachaintypes.CompactStatementValues] struct {
	validatorIndex   parachaintypes.ValidatorIndex
	compactStatement parachaintypes.CompactStatement[T]
}

// allPendingStatementsFor returns a slice of all pending statements to the validator,
// sorted with `Seconded` statements at the front.
// Statements are in the form `(Originator, Statement Kind)`.
func (g *gridTracker) allPendingStatementsFor(
	validatorIndex parachaintypes.ValidatorIndex,
) []validatorIndexWithCompactStatement[FIXME] {
	panic("not implemented")
}

// canRequest indicates whether a validator can request a manifest from us.
func (g *gridTracker) canRequest(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	panic("not implemented")
}

// directStatementProviders determines the validators which can send a statement to us by direct broadcast.
//
// Returns a list of tuples representing each potential sender(ValidatorIndex) and if
// the sender should already know about the statement, because we just sent it to it.
func (g *gridTracker) directStatementProviders() map[parachaintypes.ValidatorIndex]bool {
	panic("not implemented")
}

// directStatementTargets determines the validators which can receive a statement from us by direct broadcast.
func (g *gridTracker) directStatementTargets(
	groups groups,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement[FIXME],
) []parachaintypes.ValidatorIndex {
	panic("not implemented")
}

// learnedFreshStatement notes that we have learned about a statement.
// This will update [pendingStatementsFor] for any relevant validators
// if actually fresh.
func (g *gridTracker) learnedFreshStatement(
	groups groups,
	sessionTopology *sessionTopologyView,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement[FIXME],
) {
	panic("not implemented")
}

// / sentOrReceivedDirectStatement notes that a direct statement about a
// given candidate was sent to or received from the given validator.
func (g *gridTracker) sentOrReceivedDirectStatement(
	groups groups,
	originator parachaintypes.ValidatorIndex,
	counterparty parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement[FIXME],
	received bool,
) {
	panic("not implemented")
}

// advertisedStatements returns the advertised statement filter of a validator for a candidate.
func (g *gridTracker) advertisedStatements(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) *statementFilter {
	panic("not implemented")
}
