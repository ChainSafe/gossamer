// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

// mutualKnowledge that we have about a remote peer concerning a candidate, and that they have about us
// concerning the candidate.
type mutualKnowledge struct {
	// Knowledge the remote peer has about the candidate, as far as we're aware.
	// Non-nil only if they have advertised, acknowledged, or requested the candidate.
	remoteKnowledge *statementFilter

	// Knowledge we have indicated to the remote peer about the candidate.
	// Non-nil only if we have advertised, acknowledged, or requested the candidate
	// from them.
	localKnowledge *statementFilter

	// Knowledge peer circulated to us, this is different from `localKnowledge` and
	// `remoteKnowledge`, through the fact that includes only statements that we received from
	// peer while the other two, after manifest exchange part will include both what we sent to
	// the peer and what we received from peer, see [sentOrReceivedDirectStatement] for more
	// details.
	receivedKnowledge *statementFilter
}

// knownBackedCandidate is a utility struct for keeping track of metadata about candidates
// we have confirmed as having been backed.
type knownBackedCandidate struct {
	groupIndex      parachaintypes.GroupIndex
	localKnowledge  statementFilter
	mutualKnowledge map[parachaintypes.ValidatorIndex]mutualKnowledge
}

func (kbc *knownBackedCandidate) hasReceivedManifestFrom(validator parachaintypes.ValidatorIndex) bool {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return false
	}

	return mk.remoteKnowledge != nil
}

func (kbc *knownBackedCandidate) hasSentManifestTo(validator parachaintypes.ValidatorIndex) bool {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return false
	}

	return mk.localKnowledge != nil
}

func (kbc *knownBackedCandidate) sentManifestTo(
	validator parachaintypes.ValidatorIndex,
	localKnowledge statementFilter,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		mk = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}
	}

	groupSize := uint(localKnowledge.secondedInGroup.Len())
	receivedKnowledge, err := newStatementFilter(groupSize, false)
	if err != nil {
		logger.Warnf("failed to create statement filter instance with group size %d: %v", groupSize, err)
	}

	mk.receivedKnowledge = receivedKnowledge
	mk.localKnowledge = &localKnowledge
	kbc.mutualKnowledge[validator] = mk
}

func (kbc *knownBackedCandidate) manifestReceivedFrom(
	validator parachaintypes.ValidatorIndex,
	remoteKnowledge statementFilter,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		mk = mutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}
	}

	mk.remoteKnowledge = &remoteKnowledge
	kbc.mutualKnowledge[validator] = mk
}

// directStatementSenders returns a map representing each potential sender(ValidatorIndex)
// and if the sender should already know about the statement, because we just sent it to it.
func (kbc *knownBackedCandidate) directStatementSenders(
	groupIndex parachaintypes.GroupIndex,
	originatorIndexInGroup uint,
	statementKind statementKind,
) map[parachaintypes.ValidatorIndex]bool {
	senders := make(map[parachaintypes.ValidatorIndex]bool)

	if groupIndex != kbc.groupIndex {
		return senders
	}

	for validatorIndex, mk := range kbc.mutualKnowledge {
		if mk.remoteKnowledge == nil {
			continue
		}

		if mk.receivedKnowledge == nil || !mk.receivedKnowledge.contains(originatorIndexInGroup, statementKind) {
			continue
		}

		if mk.localKnowledge != nil && mk.localKnowledge.contains(originatorIndexInGroup, statementKind) {
			senders[validatorIndex] = true
		}
	}

	return senders
}

func (kbc *knownBackedCandidate) directStatementRecipients(
	groupIndex parachaintypes.GroupIndex,
	originatorIndexInGroup uint,
	statementKind statementKind,
) []parachaintypes.ValidatorIndex {
	recipients := make([]parachaintypes.ValidatorIndex, 0, len(kbc.mutualKnowledge))

	if groupIndex != kbc.groupIndex {
		return recipients
	}

	for validatorIndex, mk := range kbc.mutualKnowledge {
		if mk.localKnowledge == nil {
			continue
		}

		if mk.remoteKnowledge == nil || !mk.remoteKnowledge.contains(originatorIndexInGroup, statementKind) {
			recipients = append(recipients, validatorIndex)
		}
	}

	return recipients
}

func (kbc *knownBackedCandidate) noteFreshStatement(statementIndexInGroup uint, statementKind statementKind) bool {
	reallyFresh := !kbc.localKnowledge.contains(statementIndexInGroup, statementKind)

	kbc.localKnowledge.set(statementIndexInGroup, statementKind)

	return reallyFresh
}

func (kbc *knownBackedCandidate) sentOrReceivedDirectStatement(
	validator parachaintypes.ValidatorIndex,
	statementIndexInGroup uint,
	statementKind statementKind,
	received bool,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return
	}

	if mk.localKnowledge != nil && mk.remoteKnowledge != nil {
		mk.localKnowledge.set(statementIndexInGroup, statementKind)
		mk.remoteKnowledge.set(statementIndexInGroup, statementKind)
	}

	if received {
		mk.receivedKnowledge.set(statementIndexInGroup, statementKind)
	}
}

func (kbc *knownBackedCandidate) isPendingStatement(
	validator parachaintypes.ValidatorIndex,
	statementIndexInGroup uint,
	statementKind statementKind,
) bool {
	// existence of both remote & local knowledge indicate we have exchanged
	// manifests.
	// then, everything that is not in the remote knowledge is pending
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return false
	}

	if mk.localKnowledge == nil || mk.remoteKnowledge == nil {
		return false
	}

	return !mk.remoteKnowledge.contains(statementIndexInGroup, statementKind)
}

func (kbc *knownBackedCandidate) pendingStatements(validator parachaintypes.ValidatorIndex) *statementFilter {
	// existence of both remote & local knowledge indicate we have exchanged
	// manifests.
	// then, everything that is not in the remote knowledge is pending, and we
	// further limit this by what is in the local knowledge itself. we use the
	// full local knowledge, as the local knowledge stored here may be outdated.
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return nil
	}

	if mk.localKnowledge == nil || mk.remoteKnowledge == nil {
		return nil
	}

	seconded := kbc.localKnowledge.secondedInGroup.Clone()
	seconded.Mask(mk.remoteKnowledge.secondedInGroup)

	validated := kbc.localKnowledge.validatedInGroup.Clone()
	validated.Mask(mk.remoteKnowledge.validatedInGroup)

	return &statementFilter{
		secondedInGroup:  seconded,
		validatedInGroup: validated,
	}
}
