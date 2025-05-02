// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

// MutualKnowledge that we have about a remote peer concerning a candidate, and that they have about us
// concerning the candidate.
type MutualKnowledge struct {
	// Knowledge the remote peer has about the candidate, as far as we're aware.
	// Non-nil only if they have advertised, acknowledged, or requested the candidate.
	remoteKnowledge *StatementFilter

	// Knowledge we have indicated to the remote peer about the candidate.
	// Non-nil only if we have advertised, acknowledged, or requested the candidate
	// from them.
	localKnowledge *StatementFilter

	// Knowledge peer circulated to us, this is different from `localKnowledge` and
	// `remoteKnowledge`, through the fact that includes only statements that we received from
	// peer while the other two, after manifest exchange part will include both what we sent to
	// the peer and what we received from peer, see [sentOrReceivedDirectStatement] for more
	// details.
	receivedKnowledge *StatementFilter
}

// KnownBackedCandidate is a utility struct for keeping track of metadata about candidates
// we have confirmed as having been backed.
type KnownBackedCandidate struct {
	groupIndex      parachaintypes.GroupIndex
	localKnowledge  StatementFilter
	mutualKnowledge map[parachaintypes.ValidatorIndex]MutualKnowledge
}

func (kbc *KnownBackedCandidate) hasReceivedManifestFrom(validator parachaintypes.ValidatorIndex) bool {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return false
	}

	return mk.remoteKnowledge != nil
}

func (kbc *KnownBackedCandidate) hasSentManifestTo(validator parachaintypes.ValidatorIndex) bool {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return false
	}

	return mk.localKnowledge != nil
}

func (kbc *KnownBackedCandidate) sentManifestTo(
	validator parachaintypes.ValidatorIndex,
	localKnowledge StatementFilter,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		mk = MutualKnowledge{
			remoteKnowledge:   nil,
			localKnowledge:    nil,
			receivedKnowledge: nil,
		}
	}

	groupSize := uint(localKnowledge.secondedInGroup.Len())
	receivedKnowledge, err := NewStatementFilter(groupSize, false)
	if err != nil {
		logger.Warnf("failed to create statement filter instance with group size %d: %v", groupSize, err)
	}

	mk.receivedKnowledge = receivedKnowledge
	mk.localKnowledge = &localKnowledge
	kbc.mutualKnowledge[validator] = mk
}

func (kbc *KnownBackedCandidate) manifestReceivedFrom(
	validator parachaintypes.ValidatorIndex,
	remoteKnowledge StatementFilter,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		mk = MutualKnowledge{
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
func (kbc *KnownBackedCandidate) directStatementSenders(
	groupIndex parachaintypes.GroupIndex,
	originatorIndexInGroup uint,
	statementKind StatementKind,
) map[parachaintypes.ValidatorIndex]bool {
	senders := make(map[parachaintypes.ValidatorIndex]bool)

	if groupIndex != kbc.groupIndex {
		return senders
	}

	for validatorIndex, mk := range kbc.mutualKnowledge {
		if mk.remoteKnowledge == nil {
			continue
		}

		if mk.receivedKnowledge == nil || !mk.receivedKnowledge.Contains(originatorIndexInGroup, statementKind) {
			continue
		}

		if mk.localKnowledge != nil && mk.localKnowledge.Contains(originatorIndexInGroup, statementKind) {
			senders[validatorIndex] = true
		}
	}

	return senders
}

func (kbc *KnownBackedCandidate) directStatementRecipients(
	groupIndex parachaintypes.GroupIndex,
	originatorIndexInGroup uint,
	statementKind StatementKind,
) []parachaintypes.ValidatorIndex {
	recipients := make([]parachaintypes.ValidatorIndex, 0)

	if groupIndex != kbc.groupIndex {
		return recipients
	}

	for validatorIndex, mk := range kbc.mutualKnowledge {
		if mk.localKnowledge == nil {
			continue
		}

		if mk.remoteKnowledge == nil || !mk.remoteKnowledge.Contains(originatorIndexInGroup, statementKind) {
			recipients = append(recipients, validatorIndex)
		}
	}

	return recipients
}

func (kbc *KnownBackedCandidate) noteFreshStatement(statementIndexInGroup uint, statementKind StatementKind) bool {
	reallyFresh := !kbc.localKnowledge.Contains(statementIndexInGroup, statementKind)

	kbc.localKnowledge.Set(statementIndexInGroup, statementKind)

	return reallyFresh
}

func (kbc *KnownBackedCandidate) sentOrReceivedDirectStatement(
	validator parachaintypes.ValidatorIndex,
	statementIndexInGroup uint,
	statementKind StatementKind,
	received bool,
) {
	mk, ok := kbc.mutualKnowledge[validator]
	if !ok {
		return
	}

	if mk.localKnowledge != nil && mk.remoteKnowledge != nil {
		mk.localKnowledge.Set(statementIndexInGroup, statementKind)
		mk.remoteKnowledge.Set(statementIndexInGroup, statementKind)
	}

	if received {
		mk.receivedKnowledge.Set(statementIndexInGroup, statementKind)
	}
}

func (kbc *KnownBackedCandidate) isPendingStatement(
	validator parachaintypes.ValidatorIndex,
	statementIndexInGroup uint,
	statementKind StatementKind,
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

	return !mk.remoteKnowledge.Contains(statementIndexInGroup, statementKind)
}

func (kbc *KnownBackedCandidate) pendingStatements(validator parachaintypes.ValidatorIndex) *StatementFilter {
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

	return &StatementFilter{
		secondedInGroup:  seconded,
		validatedInGroup: validated,
	}
}
