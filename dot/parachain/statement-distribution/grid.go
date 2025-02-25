package statementdistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type ManifestKind = byte

const (
	ManifestFull ManifestKind = iota
	ManifestAcknowledgement
)

// statementFilter indicates the statement that are
// known or undesired about a candidate
type statementFilter interface {
	// TODO: define the interface
}

// TODO: maybe move to types.go?
type validatorAndGroupIndex struct {
	validatorIndex parachaintypes.ValidatorIndex
	groupIndex     parachaintypes.GroupIndex
}

type originatorAndStatement struct {
	originator       parachaintypes.ValidatorIndex
	compactStatement uint // TODO this should be from type compactStatement
}

type gridTracker struct {
	received          map[parachaintypes.ValidatorIndex]*receivedManifests
	confirmedBacked   map[parachaintypes.CandidateHash]*knownBackedCandidate
	unconfirmed       map[parachaintypes.CandidateHash][]*validatorAndGroupIndex
	pendingManifests  map[parachaintypes.ValidatorIndex]map[parachaintypes.CandidateHash]ManifestKind
	pengingStatements map[parachaintypes.ValidatorIndex]map[originatorAndStatement]struct{}
}

type receivedManifests struct {
	received       map[parachaintypes.CandidateHash]*manifestSummary
	secondedCounts map[parachaintypes.GroupIndex][]uint
}

type manifestSummary struct {
	claimedParentHash common.Hash
	claimedGroupIndex parachaintypes.GroupIndex
	stmtKnowledge     statementFilter
}

// knownBackedCandidate holds information about a candidate we
// have confirmed as backed
type knownBackedCandidate struct {
	groupIndex      parachaintypes.GroupIndex
	localKnowledge  statementFilter
	mutualKnowledge map[parachaintypes.ValidatorIndex]mutualKnowledge
}

// mutualKnowledge stores information that was advertised from
// a given remote peer and that we advertised for a candidate
type mutualKnowledge struct {
	// information a remote peer has about a candidate,
	// nil if the remote peer has not advertised, acknowledged or
	// requested the candidate
	remoteKnowledge statementFilter

	// information we have indicated to the remote peer about the
	// candidate, nil if we have not advertised, acknowledged, or requested the candidate
	// from them.
	localKnowledge statementFilter

	// TODO: need to understand more about it
	receivedKnowledge statementFilter
}
