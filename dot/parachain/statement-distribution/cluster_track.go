package statementdistribution

import (
	"errors"
	"fmt"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// General knowledge
type General struct {
	parachaintypes.CandidateHash
}

// Specific knowledge of a given statement (with its originator)
type Specific[T parachaintypes.CompactStatementValues] struct {
	Validator        parachaintypes.ValidatorIndex
	CompactStatement T
}

// Knowledge is a piece of knowledge about a candidate
type Knowledge[T parachaintypes.CompactStatementValues] interface {
	General | Specific[T]
}

// IncomingP2P is one possible type of TaggedKnowledge we have received from the validator on the p2p layer.
type IncomingP2P[T parachaintypes.CompactStatementValues] Knowledge[T]

// OutgoingP2P is one possible type if TaggedKnowledge we have sent to the validator on the p2p layer.
type OutgoingP2P[T parachaintypes.CompactStatementValues] Knowledge[T]

// Seconded is one possible type of TaggedKnowledge of candidates the validator has seconded.
// This is limited only to `Seconded` statements we have accepted
type Seconded struct {
	parachaintypes.CandidateHash
}

// TaggedKnowledge is Knowledge paired with its source.
type TaggedKnowledge[T parachaintypes.CompactStatementValues] interface {
	IncomingP2P[T] | OutgoingP2P[T] | Seconded
}

// OriginatorCompactStatement is a pair of (originator, statement)
type OriginatorCompactStatement[T parachaintypes.CompactStatementValues] struct {
	Validator        parachaintypes.ValidatorIndex
	CompactStatement T
}

// Accept means incoming statement was accepted.
type Accept int

const (
	// Ok is one of the Accept type value.
	// Neither the peer nor the originator have apparently exceeded limits. Candidate or statement may already be known.
	Ok Accept = iota
	// WithPrejudice is one of the Accept type value.
	// Accept the message; the peer hasn't exceeded limits but the originator has.
	WithPrejudice
)

// Incoming statement was rejected
var errRejectIncomingExcessiveSeconded = errors.New("Peer sent excessive Seconded statements.")
var errRejectIncomingNotInGroup = errors.New("Sender or originator is not in the group.")
var errRejectIncomingCandidateUnknown = errors.New("Candidate is unknown to us. Only applies to Valid statements.")
var errRejectIncomingDuplicate = errors.New("Statement is duplicate.")

// Outgoing statement was rejected
var errRejectOutgoingCandidateUnknown = errors.New("Candidate was unknown. Only applies to Valid statements.")
var errRejectOutgoingExcessiveSeconded = errors.New("We attempted to send excessive Seconded statements. indicates a bug on the local node's code.")
var errRejectOutgoingKnown = errors.New("The statement was already known to the peer.")
var errRejectOutgoingNotInGroup = errors.New("Target or originator not in the group.")

// ClusterTrack is utility for keeping track of limits on direct statements within a group.
type ClusterTrack[T parachaintypes.CompactStatementValues, F TaggedKnowledge[T]] struct {
	Validators     []parachaintypes.ValidatorIndex
	SecondingLimit uint
	Knowledge      map[parachaintypes.ValidatorIndex]map[F]struct{}
	// Statements known locally which haven't been sent to particular validators.
	// maps target validator to (originator, statement) pairs.
	Pending map[parachaintypes.ValidatorIndex]map[OriginatorCompactStatement[T]]struct{}
}

func NewClusterTrack[T parachaintypes.CompactStatementValues, F TaggedKnowledge[T]](
	clusterValidators []parachaintypes.ValidatorIndex,
	secondingLimit uint,
) (*ClusterTrack[T, F], error) {
	if len(clusterValidators) == 0 {
		return nil, fmt.Errorf("failed to create cluster track: empty validators")
	}

	return &ClusterTrack[T, F]{
		Validators:     clusterValidators,
		SecondingLimit: secondingLimit,
		Knowledge:      make(map[parachaintypes.ValidatorIndex]map[F]struct{}),
		Pending:        make(map[parachaintypes.ValidatorIndex]map[OriginatorCompactStatement[T]]struct{}),
	}, nil
}

func (c *ClusterTrack[T, F]) CanReceive(sender parachaintypes.ValidatorIndex, originator parachaintypes.ValidatorIndex, statement T) (Accept, error) {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) NoteIssued(originator parachaintypes.ValidatorIndex, statement T) {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) NoteReceived(sender parachaintypes.ValidatorIndex, originator parachaintypes.ValidatorIndex, statement T) {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) CanSend(target parachaintypes.ValidatorIndex, originator parachaintypes.ValidatorIndex, statement T) error {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) NoteSend(target parachaintypes.ValidatorIndex, originator parachaintypes.ValidatorIndex, statement T) {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) targets() []parachaintypes.ValidatorIndex {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) SendersForOriginator(originator parachaintypes.ValidatorIndex) []parachaintypes.ValidatorIndex {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) KnowsCandidate(validator parachaintypes.ValidatorIndex, candidateHash parachaintypes.CandidateHash) bool {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) CanRequest(target parachaintypes.ValidatorIndex, candidateHash parachaintypes.CandidateHash) bool {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) PendingStatementsFor(target parachaintypes.ValidatorIndex) []OriginatorCompactStatement[T] {
	panic("implement me!")
}

func (c *ClusterTrack[T, F]) WarnIfTooManyPendingStatements(parentHash common.Hash) {
	panic("implement me!")
}
