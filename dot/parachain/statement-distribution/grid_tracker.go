// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"maps"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

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

type validatorGroupPair struct {
	validator parachaintypes.ValidatorIndex
	group     parachaintypes.GroupIndex
}

type candidateHashByManifestKind map[parachaintypes.CandidateHash]manifestKind

type originatorStatementPair /* [T parachaintypes.CompactStatementValues] */ struct {
	validatorIndex parachaintypes.ValidatorIndex
	statement      any /* FIXME parachaintypes.CompactStatement[T] */
}

type originatorStatementPairSet map[originatorStatementPair]struct{}

// gridTracker tracks knowledge from authorities within the grid for a particular relay-parent.
type gridTracker struct {
	received         map[parachaintypes.ValidatorIndex]receivedManifests
	confirmedBacked  map[parachaintypes.CandidateHash]knownBackedCandidate
	unconfirmed      map[parachaintypes.CandidateHash][]validatorGroupPair
	pendingManifests map[parachaintypes.ValidatorIndex]candidateHashByManifestKind

	// maps target to (originator, statement) pairs.
	pendingStatements map[parachaintypes.ValidatorIndex]originatorStatementPairSet
}

// newGridTracker returns a new [gridTracker]
func newGridTracker() *gridTracker {
	return &gridTracker{
		received:         make(map[parachaintypes.ValidatorIndex]receivedManifests),
		confirmedBacked:  make(map[parachaintypes.CandidateHash]knownBackedCandidate),
		unconfirmed:      make(map[parachaintypes.CandidateHash][]validatorGroupPair),
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
	groupTopology, ok := sessionTopology.groupViews[manifest.claimedGroupIndex]
	if !ok {
		return false, errManifestImportDisallowed
	}

	_, receivingFrom := groupTopology.receiving[sender]
	_, sendingTo := groupTopology.sending[sender]
	manifestAllowed := false

	// Peers can send manifests _if_:
	//   * They are in the receiving set for the group AND the manifest is full OR
	//   * They are in the sending set for the group AND we have sent them a manifest AND
	//     the received manifest is partial.
	switch kind {
	case full:
		manifestAllowed = receivingFrom
	case acknowledgement:
		confirmed, ok := g.confirmedBacked[candidateHash]
		if !ok {
			manifestAllowed = false
			break
		}

		manifestAllowed = sendingTo && confirmed.hasSentManifestTo(sender)
	default:
		panic("unreachable")
	}

	if !manifestAllowed {
		return false, errManifestImportDisallowed
	}

	groupSize, backingThreshold := groups.getSizeAndBackingThreshold(manifest.claimedGroupIndex)
	if groupSize == nil || backingThreshold == nil {
		return false, errManifestImportMalformed
	}

	remoteKnowledge := manifest.statementKnowledge.clone()
	if !remoteKnowledge.hasLen(int(*groupSize)) {
		return false, errManifestImportMalformed
	}

	if !remoteKnowledge.hasSeconded() {
		return false, errManifestImportMalformed
	}

	// ensure votes are sufficient to back.
	if remoteKnowledge.backingValidators() < int(*backingThreshold) {
		return false, errManifestImportInsufficient
	}

	receivedManifest := g.received[sender]
	err := receivedManifest.importReceived(
		uint(*groupSize),
		secondingLimit,
		candidateHash,
		manifest,
	)
	if err != nil {
		return false, err
	}
	g.received[sender] = receivedManifest

	ack := false
	confirmed, ok := g.confirmedBacked[candidateHash]
	if ok && receivingFrom && confirmed.hasSentManifestTo(sender) {
		// due to checks above, the manifest `kind` is guaranteed to be `full`
		g.insertPendingManifest(sender, candidateHash, acknowledgement)

		ack = true
	}

	// add all statements in local_knowledge & !remote_knowledge
	// to `pendingStatements` for this validator.
	confirmed.manifestReceivedFrom(sender, remoteKnowledge)
	pendingStatements := confirmed.pendingStatements(sender)

	if pendingStatements != nil {
		originatorStatementPairs := decomposeStatementFilter(
			groups,
			manifest.claimedGroupIndex,
			candidateHash,
			*pendingStatements,
		)

		g.extendPendingStatements(sender, originatorStatementPairs)
	} else {
		// `received` prevents conflicting manifests so this is max 1 per validator.
		g.addUnconfirmed(candidateHash, sender, manifest.claimedGroupIndex)
	}

	return ack, nil
}

type validatorManifestKindPair struct {
	validator parachaintypes.ValidatorIndex
	kind      manifestKind
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
) []validatorManifestKindPair {
	if _, ok := g.confirmedBacked[candidateHash]; ok {
		return []validatorManifestKindPair{}
	}

	confirmedBacked := knownBackedCandidate{
		groupIndex:      groupIndex,
		localKnowledge:  localKnowledge,
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	// Populate the entry with previously unconfirmed manifests.
	unconfirmed := g.unconfirmed[candidateHash]
	delete(g.unconfirmed, candidateHash)

	for _, pair := range unconfirmed {
		claimedGroupIndex := pair.group
		if claimedGroupIndex != groupIndex {
			// This is misbehavior, but is handled more comprehensively elsewhere
			continue
		}

		received := g.received[pair.validator]
		statementFilter := received.candidateStatementFilter(candidateHash)
		if statementFilter == nil {
			panic("unconfirmed is only populated by validators who have sent manifest; qed")
		}

		// No need to send direct statements, because our local knowledge is nil
		confirmedBacked.manifestReceivedFrom(pair.validator, *statementFilter)

		g.confirmedBacked[candidateHash] = confirmedBacked
	}

	groupTopology, ok := sessionTopology.groupViews[groupIndex]
	if !ok {
		return []validatorManifestKindPair{}
	}

	// advertise onwards and accept received advertisements

	// Note that order is important: if a validator is part of both the sending
	// and receiving groups, we may overwrite a `Full` manifest with a `Acknowledgement`
	// one.

	var targets []validatorManifestKindPair

	for validator, _ := range groupTopology.sending {
		logger.Tracef("Preparing to send full manifest to validator at index %d", validator)
		g.insertPendingManifest(validator, candidateHash, full)
		targets = append(targets, validatorManifestKindPair{validator, full})
	}

	for validator, _ := range groupTopology.receiving {
		if confirmedBacked.hasReceivedManifestFrom(validator) {
			logger.Tracef("Preparing to send manifest acknowledgement to validator at index %d", validator)
			g.insertPendingManifest(validator, candidateHash, acknowledgement)
			targets = append(targets, validatorManifestKindPair{validator, acknowledgement})
		}
	}

	return targets
}

// manifestSentTo notes that a backed candidate has been advertised to a given validator.
func (g *gridTracker) manifestSentTo(
	groups groups,
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	localKnowledge statementFilter,
) {
	if confirmedBacked, ok := g.confirmedBacked[candidateHash]; ok {
		confirmedBacked.sentManifestTo(validatorIndex, localKnowledge)

	}
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

// allPendingStatementsFor returns a slice of all pending statements to the validator,
// sorted with `Seconded` statements at the front.
// Statements are in the form `(Originator, Statement Kind)`.
//func (g *gridTracker) allPendingStatementsFor(
//	validatorIndex parachaintypes.ValidatorIndex,
//) []validatorIndexWithCompactStatement[FIXME] {
//	panic("not implemented")
//}

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
//func (g *gridTracker) directStatementTargets(
//	groups groups,
//	originator parachaintypes.ValidatorIndex,
//	statement parachaintypes.CompactStatement[FIXME],
//) []parachaintypes.ValidatorIndex {
//	panic("not implemented")
//}

// learnedFreshStatement notes that we have learned about a statement.
// This will update [pendingStatementsFor] for any relevant validators
// if actually fresh.
//func (g *gridTracker) learnedFreshStatement(
//	groups groups,
//	sessionTopology *sessionTopologyView,
//	originator parachaintypes.ValidatorIndex,
//	statement parachaintypes.CompactStatement[FIXME],
//) {
//	panic("not implemented")
//}

// / sentOrReceivedDirectStatement notes that a direct statement about a
// given candidate was sent to or received from the given validator.
//func (g *gridTracker) sentOrReceivedDirectStatement(
//	groups groups,
//	originator parachaintypes.ValidatorIndex,
//	counterparty parachaintypes.ValidatorIndex,
//	statement parachaintypes.CompactStatement[FIXME],
//	received bool,
//) {
//	panic("not implemented")
//}

// advertisedStatements returns the advertised statement filter of a validator for a candidate.
func (g *gridTracker) advertisedStatements(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) *statementFilter {
	panic("not implemented")
}

func (g *gridTracker) insertPendingManifest(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	kind manifestKind,
) {
	pendingManifests := g.pendingManifests[validatorIndex]
	pendingManifests[candidateHash] = kind
	g.pendingManifests[validatorIndex] = pendingManifests
}

func (g *gridTracker) extendPendingStatements(
	validatorIndex parachaintypes.ValidatorIndex,
	originatorStatementPairs originatorStatementPairSet,
) {
	pendingStatements := g.pendingStatements[validatorIndex]
	maps.Copy(pendingStatements, originatorStatementPairs)
	g.pendingStatements[validatorIndex] = pendingStatements
}

func (g *gridTracker) addUnconfirmed(
	candidateHash parachaintypes.CandidateHash,
	validatorIndex parachaintypes.ValidatorIndex,
	groupIndex parachaintypes.GroupIndex,
) {
	unconfirmed := g.unconfirmed[candidateHash]
	unconfirmed = append(unconfirmed, validatorGroupPair{validatorIndex, groupIndex})
	g.unconfirmed[candidateHash] = unconfirmed
}

func decomposeStatementFilter(
	groups groups,
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	statementFilter statementFilter,
) originatorStatementPairSet {
	result := make(originatorStatementPairSet)
	group := groups.get(groupIndex)
	if group == nil {
		return result
	}

	for i, bit := range statementFilter.secondedInGroup.Bits() {
		if bit == true {
			validatorIndex := group[i]
			value := parachaintypes.SecondedCandidateHash(candidateHash)

			pair := originatorStatementPair{
				validatorIndex: validatorIndex,
				statement: parachaintypes.CompactStatement[parachaintypes.SecondedCandidateHash]{
					Value: value,
				},
			}

			result[pair] = struct{}{}
		}
	}

	for i, bit := range statementFilter.validatedInGroup.Bits() {
		if bit == true {
			validatorIndex := group[i]
			value := parachaintypes.Valid(candidateHash)

			pair := originatorStatementPair{
				validatorIndex: validatorIndex,
				statement: parachaintypes.CompactStatement[parachaintypes.Valid]{
					Value: value,
				},
			}

			result[pair] = struct{}{}
		}
	}

	return result
}
