// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type acceptOrReject byte

const (
	// ok means neither the peer nor the originator have apparently exceeded limits.
	// Candidate or statement may already be known.
	ok acceptOrReject = iota

	// withPrejudice means accept the message; the peer hasn't exceeded limits but the originator has.
	withPrejudice

	// excessiveSeconded means peer sent excessive `Seconded` statements or we attempted to send excessive `Seconded`
	// statements. The latter indicates a bug on the local node's code.
	excessiveSeconded

	// notInGroup means sender/target or originator is not in the group.
	notInGroup

	// candidateUnknown means the candidate is unknown to us. Only applies to `Valid` statements.
	candidateUnknown

	// duplicate means the statement is a duplicate.
	duplicate

	// known means the statement was already known to the peer.
	known
)

// knowledge about a candidate
type knowledge interface {
	isKnowledge()
	equals(other knowledge) bool
}

// general knowledge
type general struct {
	candidateHash parachaintypes.CandidateHash
}

func (general) isKnowledge() {}

func (g general) equals(other knowledge) bool {
	if otherGeneral, ok := other.(general); ok {
		return g.candidateHash.Value == otherGeneral.candidateHash.Value
	}
	return false
}

// specific knowledge of a given statement (with its originator)
type specific struct {
	statement parachaintypes.CompactStatement
	validator parachaintypes.ValidatorIndex
}

func (specific) isKnowledge() {}

func (s specific) equals(other knowledge) bool {
	if otherSpecific, ok := other.(specific); ok {
		return s.validator == otherSpecific.validator && s.statement.Equals(otherSpecific.statement)
	}
	return false
}

// taggedKnowledge is knowledge paired with its source.
type taggedKnowledge interface {
	isTaggedKnowledge()
	equals(other taggedKnowledge) bool
}

// incomingP2P is knowledge we have received from the validator on the p2p layer.
type incomingP2P struct {
	knowledge knowledge
}

func (incomingP2P) isTaggedKnowledge() {}

func (i incomingP2P) equals(other taggedKnowledge) bool {
	if otherIncoming, ok := other.(incomingP2P); ok {
		return i.knowledge.equals(otherIncoming.knowledge)
	}
	return false
}

// outgoingP2P is knowledge we have sent to the validator on the p2p layer.
type outgoingP2P struct {
	knowledge knowledge
}

func (outgoingP2P) isTaggedKnowledge() {}

func (o outgoingP2P) equals(other taggedKnowledge) bool {
	if otherOutgoing, ok := other.(outgoingP2P); ok {
		return o.knowledge.equals(otherOutgoing.knowledge)
	}
	return false
}

// seconded is knowledge of candidates the validator has seconded.
// This is limited only to `Seconded` statements we have accepted
// _without prejudice_.
type seconded struct {
	candidateHash parachaintypes.CandidateHash
}

func (seconded) isTaggedKnowledge() {}

func (s seconded) equals(other taggedKnowledge) bool {
	if otherSeconded, ok := other.(seconded); ok {
		return s.candidateHash.Value == otherSeconded.candidateHash.Value
	}
	return false
}

// clusterTracker is a utility for keeping track of limits on direct statements within a group.
type clusterTracker struct {
	validators     []parachaintypes.ValidatorIndex
	secondingLimit uint
	knowledge      map[parachaintypes.ValidatorIndex]map[taggedKnowledge]struct{}

	// pending contains statements known locally which haven't been sent to particular validators.
	// maps target validator to (originator, statement) pairs.
	pending map[parachaintypes.ValidatorIndex]originatorStatementPairSet
}

func newClusterTracker(
	clusterValidators []parachaintypes.ValidatorIndex,
	secondingLimit uint,
) *clusterTracker {
	return &clusterTracker{
		validators:     clusterValidators,
		secondingLimit: secondingLimit,
		knowledge:      make(map[parachaintypes.ValidatorIndex]map[taggedKnowledge]struct{}),
		pending:        make(map[parachaintypes.ValidatorIndex]originatorStatementPairSet),
	}
}

// canReceive queries whether we can receive some statement from the given validator.
//
// This does no deduplication of `Valid` statements.
func (c *clusterTracker) canReceive(
	sender parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) acceptOrReject {
	if !c.isInGroup(sender) || !c.isInGroup(originator) {
		return notInGroup
	}

	if c.theySent(sender, specific{statement, originator}) {
		return duplicate
	}

	switch statement.(type) {
	case *parachaintypes.CompactSeconded:
		// check whether the sender has not sent too many seconded statements for the
		// originator. we know by the duplicate check above that this iteration doesn't
		// include the statement itself.
		otherSecondedForOrigFromRemote := uint(0)
		for taggedKnowldg := range c.knowledge[sender] {
			incp2p, ok := taggedKnowldg.(incomingP2P)
			if !ok {
				continue
			}

			spec, ok := incp2p.knowledge.(specific)
			if !ok {
				continue
			}

			if _, ok = spec.statement.(*parachaintypes.CompactSeconded); !ok {
				continue
			}

			if spec.validator == originator {
				otherSecondedForOrigFromRemote += 1
			}
		}

		if otherSecondedForOrigFromRemote == c.secondingLimit {
			return excessiveSeconded
		}

		// at this point, it doesn't seem like the remote has done anything wrong.
		if c.secondedAlreadyOrWithinLimit(originator, statement.CandidateHash()) {
			return ok
		} else {
			return withPrejudice
		}
	case *parachaintypes.CompactValid:
		if !c.knowsCandidate(sender, statement.CandidateHash()) {
			return candidateUnknown
		}
		return ok
	default:
		panic("unreachable")
	}
}

// noteReceived notes that we accepted an incoming statement. This updates internal structures.
//
// Should only be called after a successful [canReceive] call.
func (c *clusterTracker) noteReceived(
	sender parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) {
	for _, clusterMember := range c.validators {
		if clusterMember == sender {
			if pending, ok := c.pending[sender]; ok {
				pending.remove(originator, statement)
			}
		} else if !c.theyKnowStatement(clusterMember, originator, statement) {
			// add the statement to pending knowledge for all peers
			// which don't know the statement.
			m := c.pending[clusterMember]
			if m == nil {
				m = make(originatorStatementPairSet)
			}
			m.insert(originator, statement)
			c.pending[clusterMember] = m
		}
	}

	senderKnowledge := c.knowledge[sender]
	if senderKnowledge == nil {
		senderKnowledge = make(map[taggedKnowledge]struct{})
		c.knowledge[sender] = senderKnowledge
	}
	senderKnowledge[incomingP2P{specific{statement, originator}}] = struct{}{}

	if _, ok := statement.(*parachaintypes.CompactSeconded); ok {
		senderKnowledge[incomingP2P{general{statement.CandidateHash()}}] = struct{}{}

		// since we accept additional `Seconded` statements beyond the limits
		// 'with prejudice', we must respect the limit here.
		if c.secondedAlreadyOrWithinLimit(originator, statement.CandidateHash()) {
			originatorKnowledge := c.knowledge[originator]
			if originatorKnowledge == nil {
				originatorKnowledge = make(map[taggedKnowledge]struct{})
				c.knowledge[originator] = originatorKnowledge
			}
			originatorKnowledge[seconded{statement.CandidateHash()}] = struct{}{}
		}
	}
}

func (c *clusterTracker) theyKnowStatement(
	validator parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) bool {
	knowledge := specific{statement, originator}
	return c.weSent(validator, knowledge) || c.theySent(validator, knowledge)
}

func (c *clusterTracker) isInGroup(validator parachaintypes.ValidatorIndex) bool {
	return slices.Contains(c.validators, validator)
}

func (c *clusterTracker) theySent(
	validator parachaintypes.ValidatorIndex,
	knowledge knowledge,
) bool {
	for tk := range c.knowledge[validator] {
		if tk.equals(incomingP2P{knowledge}) {
			return true
		}
	}
	return false
}

func (c *clusterTracker) weSent(
	validator parachaintypes.ValidatorIndex,
	knowledge knowledge,
) bool {
	for tk := range c.knowledge[validator] {
		if tk.equals(outgoingP2P{knowledge}) {
			return true
		}
	}
	return false
}

// secondedAlreadyOrWithinLimit returns true if it's legal to accept a new `Seconded` message from this validator.
// This is either
//  1. because we've already accepted it.
//  2. because there's space for more seconding.
func (c *clusterTracker) secondedAlreadyOrWithinLimit(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	secondedOtherCandidates := uint(0)

	for tk := range c.knowledge[validator] {
		if seconded, ok := tk.(seconded); ok && seconded.candidateHash != candidateHash {
			secondedOtherCandidates += 1
		}
	}

	return secondedOtherCandidates < c.secondingLimit
}

// sendersForOriginator returns all possible senders for the given originator.
// Returns the empty slice in the case that the originator
// is not part of the cluster.
// note: this API is future-proofing for a case where we may
// extend clusters beyond just the assigned group, for optimization
// purposes.
// The method returns an internal datastructure of the object which
// should be copied before mutating it.
func (c *clusterTracker) sendersForOriginator( //nolint:unused
	originator parachaintypes.ValidatorIndex,
) []parachaintypes.ValidatorIndex {
	if slices.Contains(c.validators, originator) {
		return c.validators
	}
	return nil
}

// knowsCandidate queries whether a validator knows the candidate is `Seconded`.
func (c *clusterTracker) knowsCandidate(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	// we sent, they sent, or they signed and we received from someone else.

	return c.weSentSeconded(validator, candidateHash) || c.theySentSeconded(validator, candidateHash) ||
		c.validatorSeconded(validator, candidateHash)
}

// canRequest queries whether a validator can request a candidate from us.
func (c *clusterTracker) canRequest( //nolint:unused
	target parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	return slices.Contains(c.validators, target) &&
		c.weSentSeconded(target, candidateHash) &&
		!c.theySentSeconded(target, candidateHash)
}

// noteIssued notes that we issued a statement. This updates internal structures.
func (c *clusterTracker) noteIssued(
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) {
	for _, clusterMember := range c.validators {
		if !c.theyKnowStatement(clusterMember, originator, statement) {
			// add the statement to pending knowledge for all peers
			// which don't know the statement.
			pending, ok := c.pending[clusterMember]
			if !ok {
				pending = make(originatorStatementPairSet)
				c.pending[clusterMember] = pending
			}
			pending.insert(originator, statement)
		}
	}
}

func (c *clusterTracker) weSentSeconded(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	return c.weSent(validator, general{candidateHash})
}

func (c *clusterTracker) theySentSeconded(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	return c.theySent(validator, general{candidateHash})
}

func (c *clusterTracker) validatorSeconded(
	validator parachaintypes.ValidatorIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	for tk := range c.knowledge[validator] {
		if tk.equals(seconded{candidateHash}) {
			return true
		}
	}
	return false
}

// canSend queries whether we can send a statement to a given validator.
func (c *clusterTracker) canSend(
	target parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) acceptOrReject {
	if !c.isInGroup(target) || !c.isInGroup(originator) {
		return notInGroup
	}

	if c.theyKnowStatement(target, originator, statement) {
		return known
	}

	switch statement.(type) {
	case *parachaintypes.CompactSeconded:
		// we send the same `Seconded` statements to all our peers, and only the first `k`
		// from each originator.
		if !c.secondedAlreadyOrWithinLimit(originator, statement.CandidateHash()) {
			return excessiveSeconded
		}
		return ok
	case *parachaintypes.CompactValid:
		if !c.knowsCandidate(target, statement.CandidateHash()) {
			return candidateUnknown
		}
		return ok
	default:
		panic("unreachable")
	}
}

// noteSent notes that we sent an outgoing statement to a peer in the group.
// This must be preceded by a successful `can_send` call.
func (c *clusterTracker) noteSent(
	target parachaintypes.ValidatorIndex,
	originator parachaintypes.ValidatorIndex,
	statement parachaintypes.CompactStatement,
) {
	targetKnowledge, ok := c.knowledge[target]
	if !ok {
		targetKnowledge = map[taggedKnowledge]struct{}{}
		c.knowledge[target] = targetKnowledge
	}

	targetKnowledge[outgoingP2P{specific{statement, originator}}] = struct{}{}
	c.knowledge[target] = targetKnowledge

	if _, ok := statement.(*parachaintypes.CompactSeconded); ok {
		targetKnowledge[outgoingP2P{general{statement.CandidateHash()}}] = struct{}{}

		originatorKnowledge, ok := c.knowledge[originator]
		if !ok {
			originatorKnowledge = make(map[taggedKnowledge]struct{})
		}

		originatorKnowledge[seconded{statement.CandidateHash()}] = struct{}{}
		c.knowledge[originator] = originatorKnowledge
	}

	if pending, ok := c.pending[target]; ok {
		pending.remove(originator, statement)
	}
}

// pendingStatementsFor returns a slice of pending statements to be sent to a particular validator.
// `Seconded` statements are sorted to the front of the slice.
func (c *clusterTracker) pendingStatementsFor(target parachaintypes.ValidatorIndex) []originatorStatementPair {
	var seconded, valid []originatorStatementPair

	pending, ok := c.pending[target]
	if !ok {
		return nil
	}

	for pair := range pending {
		switch pair.compactStmt.(type) {
		case *parachaintypes.CompactSeconded:
			seconded = append(seconded, pair)
		case *parachaintypes.CompactValid:
			valid = append(valid, pair)
		default:
			panic("unreachable")
		}
	}

	return append(seconded, valid...)
}

// warnIfTooManyStatements dumps pending statement for this cluster.
//
// Normally we should not have pending statements to validators in our cluster,
// but if we do for all validators in our cluster, then we don't participate
// in backing. Occasional pending statements are expected if two authorities
// can't detect each other or after restart, where it takes a while to discover
// the whole network.
func (c *clusterTracker) warningIfTooManyPendingStatements(parentHash common.Hash) { //nolint:unused
	count := 0
	for _, set := range c.pending {
		if len(set) > 0 {
			count += 1
		}
	}

	numValidators := len(c.validators)
	if count >= numValidators &&
		// No reason to warn if we are the only node in the cluster.
		numValidators > 1 {
		logger.Warnf(
			"Cluster has too many pending statements, something wrong with our connection to our group peers "+
				"Restart might be needed if validator gets 0 backing rewards for more than 3-4 consecutive sessions "+
				"count=%d parentHash=%s",
			count,
			parentHash.String(),
		)
	}
}
