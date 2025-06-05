// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package validationprotocol

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// A notification of a signed statement in compact form, for a given relay parent.
type Statement struct {
	RelayParent common.Hash
	Compact     parachaintypes.UncheckedSignedCompactStatement
}

// A notification of a backed candidate being known by the
// sending node, for the purpose of being requested by the receiving node
// if needed.
type BackedCandidateManifest struct {
	RelayParent   common.Hash
	CandidateHash parachaintypes.CandidateHash
	// The group index backing the candidate at the relay-parent.
	GroupIndex parachaintypes.GroupIndex

	// The para ID of the candidate. It is illegal for this to
	// be a para ID which is not assigned to the group indicated
	// in this manifest.
	ParaID             parachaintypes.ParaID
	ParentHeadDataHash common.Hash

	// A statement filter which indicates which validators in the
	// para's group at the relay-parent have validated this candidate
	// and issued statements about it, to the advertiser's knowledge.
	//
	// This MUST have exactly the minimum amount of bytes
	// necessary to represent the number of validators in the assigned
	// backing group as-of the relay-parent.
	// TODO: make statement filter public and encodable/decodable
	StatementKnowledge parachaintypes.StatementFilter
}

// An acknowledgement of a backed candidate being known.
type BackedCandidateKnown struct {
	CandidateHash parachaintypes.CandidateHash

	// A statement filter which indicates which validators in the
	// para's group at the relay-parent have validated this candidate
	// and issued statements about it, to the advertiser's knowledge.
	//
	// This MUST have exactly the minimum amount of bytes
	// necessary to represent the number of validators in the assigned
	// backing group as-of the relay-parent.
	// TODO: make statement filter public and encodable/decodable
	StatementKnowledge parachaintypes.StatementFilter
}

type StatementDistributionMessageValues interface {
	Statement | BackedCandidateManifest | BackedCandidateKnown
}

// StatementDistributionMessage represents network messages used by the statement distribution subsystem
type StatementDistributionMessage struct {
	inner any
}

// NewStatementDistributionMessage returns a new statement distribution message varying data type
func NewStatementDistributionMessage() StatementDistributionMessage {
	return StatementDistributionMessage{}
}

func setStatementDistributionMessage[Value StatementDistributionMessageValues](
	mvdt *StatementDistributionMessage, value Value,
) {
	mvdt.inner = value
}

func (mvdt *StatementDistributionMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Statement:
		setStatementDistributionMessage(mvdt, value)
		return

	case BackedCandidateManifest:
		setStatementDistributionMessage(mvdt, value)
		return

	case BackedCandidateKnown:
		setStatementDistributionMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported value of type: %v (%T)", value, value)
	}
}

func (mvdt StatementDistributionMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Statement:
		return 0, mvdt.inner, nil

	case BackedCandidateManifest:
		return 1, mvdt.inner, nil

	case BackedCandidateKnown:
		return 2, mvdt.inner, nil
	}

	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt StatementDistributionMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt StatementDistributionMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return Statement{}, nil

	case 1:
		return BackedCandidateManifest{}, nil

	case 2:
		return BackedCandidateKnown{}, nil
	}

	return nil, scale.ErrUnknownVaryingDataTypeValue
}
