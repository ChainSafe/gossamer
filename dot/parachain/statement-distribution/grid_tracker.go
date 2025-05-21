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

type manifestKindByCandidateHash map[parachaintypes.CandidateHash]manifestKind

type originatorStatementPair struct {
	validatorIndex parachaintypes.ValidatorIndex
	statement      any /* FIXME should be parachaintypes.CompactStatement */
}

type originatorStatementPairSet map[originatorStatementPair]struct{}

// gridTracker tracks knowledge from authorities within the grid for a particular relay-parent.
type gridTracker struct {
	received         map[parachaintypes.ValidatorIndex]receivedManifests
	confirmedBacked  map[parachaintypes.CandidateHash]knownBackedCandidate
	unconfirmed      map[parachaintypes.CandidateHash][]validatorGroupPair
	pendingManifests map[parachaintypes.ValidatorIndex]manifestKindByCandidateHash

	// maps target to (originator, statement) pairs.
	pendingStatements map[parachaintypes.ValidatorIndex]originatorStatementPairSet
}

// newGridTracker returns a new [gridTracker]
func newGridTracker() *gridTracker {
	return &gridTracker{
		received:          make(map[parachaintypes.ValidatorIndex]receivedManifests),
		confirmedBacked:   make(map[parachaintypes.CandidateHash]knownBackedCandidate),
		unconfirmed:       make(map[parachaintypes.CandidateHash][]validatorGroupPair),
		pendingManifests:  make(map[parachaintypes.ValidatorIndex]manifestKindByCandidateHash),
		pendingStatements: make(map[parachaintypes.ValidatorIndex]originatorStatementPairSet),
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
	secondingLimit uint, //nolint:unparam
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

	receivedManifests, ok := g.received[sender]
	if !ok {
		receivedManifests = *newReceivedManifests()
	}

	err := receivedManifests.importReceived(
		uint(*groupSize),
		secondingLimit,
		candidateHash,
		manifest,
	)
	if err != nil {
		return false, err
	}
	g.received[sender] = receivedManifests

	ack := false
	known, ok := g.confirmedBacked[candidateHash]
	if ok {
		if receivingFrom && !known.hasSentManifestTo(sender) {
			// due to checks above, the manifest `kind` is guaranteed to be `full`
			g.insertPendingManifest(sender, candidateHash, acknowledgement)

			ack = true
		}

		// add all statements in local_knowledge & !remote_knowledge
		// to `pendingStatements` for this validator.
		known.manifestReceivedFrom(sender, remoteKnowledge)
		pendingStatements := known.pendingStatements(sender)
		if pendingStatements != nil {
			originatorStatementPairs := decomposeStatementFilter(
				groups,
				manifest.claimedGroupIndex,
				candidateHash,
				*pendingStatements,
			)

			g.extendPendingStatements(sender, originatorStatementPairs)
		}
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
		return nil
	}

	known := knownBackedCandidate{
		groupIndex:      groupIndex,
		localKnowledge:  localKnowledge,
		mutualKnowledge: make(map[parachaintypes.ValidatorIndex]mutualKnowledge),
	}

	g.confirmedBacked[candidateHash] = known

	// Populate the entry with previously unconfirmed manifests.
	unconfirmed := g.unconfirmed[candidateHash]
	delete(g.unconfirmed, candidateHash)

	for _, pair := range unconfirmed {
		claimedGroupIndex := pair.group
		if claimedGroupIndex != groupIndex {
			// This is misbehaviour, but is handled more comprehensively elsewhere
			continue
		}

		received := g.received[pair.validator]
		statementFilter := received.candidateStatementFilter(candidateHash)
		if statementFilter == nil {
			panic("unconfirmed is only populated by validators who have sent manifest; qed")
		}

		// No need to send direct statements, because our local knowledge is nil
		known.manifestReceivedFrom(pair.validator, *statementFilter)
	}

	groupTopology, ok := sessionTopology.groupViews[groupIndex]
	if !ok {
		return nil
	}

	// advertise onwards and accept received advertisements

	// Note that order is important: if a validator is part of both the sending
	// and receiving groups, we may overwrite a `Full` manifest with a `Acknowledgement`
	// one.

	var targets []validatorManifestKindPair

	for validator := range groupTopology.sending {
		logger.Tracef("Preparing to send full manifest to validator at index %d", validator)
		g.insertPendingManifest(validator, candidateHash, full)
		targets = append(targets, validatorManifestKindPair{validator, full})
	}

	for validator := range groupTopology.receiving {
		if known.hasReceivedManifestFrom(validator) {
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
	if known, ok := g.confirmedBacked[candidateHash]; ok {
		known.manifestSentTo(validatorIndex, localKnowledge)

		if pendingStatements := known.pendingStatements(validatorIndex); pendingStatements != nil {
			originatorStatementPairs := decomposeStatementFilter(
				groups,
				known.groupIndex,
				candidateHash,
				*pendingStatements,
			)

			g.extendPendingStatements(validatorIndex, originatorStatementPairs)
		}
	}

	g.removePendingManifest(validatorIndex, candidateHash)
}

// pendingManifestsFor returns a vector of all candidates pending manifests for
// the specific validator, and the type of manifest we should send.
func (g *gridTracker) pendingManifestsFor( //nolint:unused
	validatorIndex parachaintypes.ValidatorIndex,
) manifestKindByCandidateHash {
	return maps.Clone(g.pendingManifests[validatorIndex])
}

// pendingStatementsFor returns a statement filter indicating statements that a given peer is
// awaiting concerning the given candidate, constrained by the statements we have ourselves.
func (g *gridTracker) pendingStatementsFor(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) *statementFilter {
	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return nil
	}

	return known.pendingStatements(validatorIndex)
}

// allPendingStatementsFor returns a slice of all pending statements to the validator,
// sorted with `Seconded` statements at the front.
// Statements are in the form `(Originator, Statement Kind)`.
func (g *gridTracker) allPendingStatementsFor(
	validatorIndex parachaintypes.ValidatorIndex,
) []originatorStatementPair {
	var seconded, valid []originatorStatementPair

	for pair := range g.pendingStatements[validatorIndex] {
		if _, ok := pair.statement.(parachaintypes.CompactStatement[parachaintypes.SecondedCandidateHash]); ok {
			seconded = append(seconded, pair)
		} else {
			valid = append(valid, pair)
		}
	}

	return append(seconded, valid...)
}

// canRequest indicates whether a validator can request a manifest from us.
func (g *gridTracker) canRequest( //nolint:unused
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return false
	}

	return known.hasSentManifestTo(validatorIndex) && !known.hasReceivedManifestFrom(validatorIndex)
}

// directStatementProviders determines the validators which can send a statement to us by direct broadcast.
//
// Returns a map representing each potential sender(ValidatorIndex) and if the sender
// should already know about the statement, because we just sent it to it.
func (g *gridTracker) directStatementProviders( //nolint:unused
	groups groups,
	originator parachaintypes.ValidatorIndex,
	statement any, /* FIXME should be parachaintypes.CompactStatement */
) map[parachaintypes.ValidatorIndex]bool {
	groupIndex, candidateHash, stmtKind, idxInGroup := extractStatementAndGroupInfo(groups, originator, statement)
	if groupIndex == nil {
		return nil
	}

	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return nil
	}

	return known.directStatementSenders(*groupIndex, idxInGroup, stmtKind)
}

// directStatementTargets determines the validators which can receive a statement from us by direct broadcast.
func (g *gridTracker) directStatementTargets( //nolint:unused
	groups groups,
	originator parachaintypes.ValidatorIndex,
	statement any, /* FIXME should be parachaintypes.CompactStatement */
) []parachaintypes.ValidatorIndex {
	groupIndex, candidateHash, stmtKind, idxInGroup := extractStatementAndGroupInfo(groups, originator, statement)
	if groupIndex == nil {
		return nil
	}

	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return nil
	}

	return known.directStatementRecipients(*groupIndex, idxInGroup, stmtKind)
}

// learnedFreshStatement notes that we have learned about a statement.
// This will update [pendingStatementsFor] for any relevant validators
// if actually fresh.
func (g *gridTracker) learnedFreshStatement(
	groups groups,
	sessionTopology *sessionTopologyView,
	originator parachaintypes.ValidatorIndex,
	statement any, /* FIXME should be parachaintypes.CompactStatement */
) {
	groupIndex, candidateHash, stmtKind, idxInGroup := extractStatementAndGroupInfo(groups, originator, statement)
	if groupIndex == nil {
		return
	}

	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return
	}

	if !known.noteFreshStatement(idxInGroup, stmtKind) {
		return
	}

	// Add to `pendingStatements` for all validators we communicate with
	// who have exchanged manifests.
	subView, ok := sessionTopology.groupViews[*groupIndex]
	if !ok {
		return
	}

	var allGroupValidators []parachaintypes.ValidatorIndex

	for validatorIndex := range subView.sending {
		allGroupValidators = append(allGroupValidators, validatorIndex)
	}

	for validatorIndex := range subView.receiving {
		allGroupValidators = append(allGroupValidators, validatorIndex)
	}

	for _, validatorIndex := range allGroupValidators {
		if known.isPendingStatement(validatorIndex, idxInGroup, stmtKind) {
			g.insertPendingStatement(validatorIndex, originatorStatementPair{originator, statement})
		}
	}
}

// / sentOrReceivedDirectStatement notes that a direct statement about a
// given candidate was sent to or received from the given validator.
func (g *gridTracker) sentOrReceivedDirectStatement( //nolint:unused
	groups groups,
	originator parachaintypes.ValidatorIndex,
	counterparty parachaintypes.ValidatorIndex,
	statement any, /* FIXME should be parachaintypes.CompactStatement */
	received bool,
) {
	groupIndex, candidateHash, stmtKind, idxInGroup := extractStatementAndGroupInfo(groups, originator, statement)
	if groupIndex == nil {
		return
	}

	known, ok := g.confirmedBacked[candidateHash]
	if !ok {
		return
	}

	known.sentOrReceivedDirectStatement(counterparty, idxInGroup, stmtKind, received)
	g.confirmedBacked[candidateHash] = known

	delete(g.pendingStatements, counterparty)
}

// advertisedStatements returns the advertised statement filter of a validator for a candidate.
func (g *gridTracker) advertisedStatements( //nolint:unused
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) *statementFilter {
	manifests, ok := g.received[validator]
	if !ok {
		return nil
	}

	return manifests.candidateStatementFilter(candidateHash)
}

func (g *gridTracker) insertPendingManifest(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
	kind manifestKind,
) {
	pm := g.pendingManifests[validatorIndex]
	if pm == nil {
		pm = make(manifestKindByCandidateHash)
	}

	pm[candidateHash] = kind
	g.pendingManifests[validatorIndex] = pm
}

func (g *gridTracker) removePendingManifest(
	validatorIndex parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) {
	delete(g.pendingManifests[validatorIndex], candidateHash)
}

func (g *gridTracker) insertPendingStatement(
	validatorIndex parachaintypes.ValidatorIndex,
	pair originatorStatementPair,
) {
	ps := g.pendingStatements[validatorIndex]
	if ps == nil {
		ps = make(originatorStatementPairSet)
	}

	ps[pair] = struct{}{}
	g.pendingStatements[validatorIndex] = ps
}

func (g *gridTracker) extendPendingStatements(
	validatorIndex parachaintypes.ValidatorIndex,
	originatorStatementPairs originatorStatementPairSet,
) {
	ps := g.pendingStatements[validatorIndex]
	if ps == nil {
		ps = make(originatorStatementPairSet)
	}

	maps.Copy(ps, originatorStatementPairs)
	g.pendingStatements[validatorIndex] = ps
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
		if bit {
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
		if bit {
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

// If the first return value is nil, the others are invalid as well (i.e. nil-equivalent).
func extractStatementAndGroupInfo(
	groups groups,
	originator parachaintypes.ValidatorIndex,
	statement any, /* FIXME should be parachaintypes.CompactStatement */
) (gi *parachaintypes.GroupIndex, ch parachaintypes.CandidateHash, sk statementKind, i uint) {
	switch s := statement.(type) {
	case parachaintypes.CompactStatement[parachaintypes.SecondedCandidateHash]:
		ch = parachaintypes.CandidateHash(s.Value)
		sk = seconded
	case parachaintypes.CompactStatement[parachaintypes.Valid]:
		ch = parachaintypes.CandidateHash(s.Value)
		sk = valid
	default:
		panic("unreachable")
	}

	// gi is the index of the *group* that the originator is in
	gi = groups.byValidatorIndex(originator)
	if gi == nil {
		return
	}

	// i is the index of the *originator* in its group
	for indexInGroup, validator := range groups.get(*gi) {
		if validator == originator {
			i = uint(indexInGroup)
		}
	}
	return
}
