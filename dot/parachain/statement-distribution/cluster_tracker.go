// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// accept signifies that an incoming statement was accepted.
type accept interface {
	isAccept()
}

// ok means neither the peer nor the originator have apparently exceeded limits.
// Candidate or statement may already be known.
type ok struct{}

func (ok) isAccept() {}

// / withPrejudice means accept the message; the peer hasn't exceeded limits but the originator has.
type withPrejudice struct{}

func (withPrejudice) isAccept() {}

// / rejectIncoming signifies that an incoming statement was rejected.
type rejectIncoming interface {
	isRejectIncoming()
}

// excessiveSecondingIncoming means peer sent excessive `Seconded` statements.
type excessiveSecondingIncoming struct{}

func (excessiveSecondingIncoming) isRejectIncoming() {}

// notInGroupIncoming means sender or originator is not in the group.
type notInGroupIncoming struct{}

func (notInGroupIncoming) isRejectIcoming() {}

// candidateUnknownIncoming means the candidate is unknown to us. Only applies to `Valid` statements.
type candidateUnknownIncoming struct{}

func (candidateUnknownIncoming) isRejectIncoming() {}

// duplicateIncoming means the statement is a duplicate.
type duplicateIncoming struct{}

func (duplicateIncoming) isRejectIncoming() {}

// / rejectOutgoing signifies that an outgoing statement was rejected.
type rejectOutgoing interface {
	isRejectOutgoing()
}

// candidateUnknownOutgoing means the candidate was unknown. Only applies to `Valid` statements.
type candidateUnknownOutgoing struct{}

func (candidateUnknownOutgoing) isRejectOutgoing() {}

// excessiveSecondingOutgoing means we attempted to send excessive `Seconded` statements.
// Indicates a bug on the local node's code.
type excessiveSecondingOutgoing struct{}

func (excessiveSecondingOutgoing) isRejectOutgoing() {}

// knownOutgoing means the statement was already known to the peer.
type knownOutgoing struct{}

func (knownOutgoing) isRejectOutgoing() {}

// notInGroupOutgoing means the target or originator are not in the group.
type notInGroupOutgoing struct{}

func (notInGroupOutgoing) isRejectOutgoing() {}

type acceptOrRejectIncoming interface {
	isAcceptOrRejectIncoming()
}

func (ok) isAcceptOrRejectIncoming()                         {}
func (withPrejudice) isAcceptOrRejectIncoming()              {}
func (excessiveSecondingIncoming) isAcceptOrRejectIncoming() {}
func (notInGroupIncoming) isAcceptOrRejectIncoming()         {}
func (candidateUnknownIncoming) isAcceptOrRejectIncoming()   {}
func (duplicateIncoming) isAcceptOrRejectIncoming()          {}

type acceptOrRejectOutgoing interface {
	isAcceptOrRejectOutgoing()
}

func (ok) isAcceptOrRejectOutgoing()                         {}
func (withPrejudice) isAcceptOrRejectOutgoing()              {}
func (candidateUnknownOutgoing) isAcceptOrRejectOutgoing()   {}
func (excessiveSecondingOutgoing) isAcceptOrRejectOutgoing() {}
func (knownOutgoing) isAcceptOrRejectOutgoing()              {}
func (notInGroupOutgoing) isAcceptOrRejectOutgoing()         {}

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
) acceptOrRejectIncoming {
	if !c.isInGroup(sender) || !c.isInGroup(originator) {
		return notInGroupIncoming{}
	}

	if c.theySent(sender, specific{statement, originator}) {
		return duplicateIncoming{}
	}

	switch statement.(type) {
	case *parachaintypes.CompactSeconded:
		// check whether the sender has not sent too many seconded statements for the
		// originator. we know by the duplicate check above that this iteration doesn't
		// include the statement itself.
		otherSecondedForOrigFromRemote := uint(0)
		for taggedKnowldg, _ := range c.knowledge[sender] {
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

		if otherSecondedForOrigFromRemote > c.secondingLimit {
			return excessiveSecondingIncoming{}
		}

		// at this point, it doesn't seem like the remote has done anything wrong.
		if c.secondedAlreadyOrWithinLimit(originator, statement.CandidateHash()) {
			return ok{}
		} else {
			return withPrejudice{}
		}
	case *parachaintypes.CompactValid:
		if !c.knowsCandidate(sender, statement.CandidateHash()) {
			return candidateUnknownIncoming{}
		}
		return ok{}
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
	for tk, _ := range c.knowledge[validator] {
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
	for tk, _ := range c.knowledge[validator] {
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

	for tk, _ := range c.knowledge[validator] {
		if seconded, ok := tk.(seconded); ok && seconded.candidateHash != candidateHash {
			secondedOtherCandidates += 1
		}
	}

	return secondedOtherCandidates < c.secondingLimit
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
	for tk, _ := range c.knowledge[validator] {
		if tk.equals(seconded{candidateHash}) {
			return true
		}
	}
	return false
}
