package types

import (
	"bytes"
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Comparator for ordering of disputes.
type Comparator struct {
	SessionIndex  parachainTypes.SessionIndex `scale:"1"`
	CandidateHash common.Hash                 `scale:"2"`
}

// Dispute is a dispute for a candidate.
// It is used as an item in the btree.BTree ordered by the Comparator.
type Dispute struct {
	Comparator    Comparator    `scale:"1"`
	DisputeStatus DisputeStatus `scale:"2"`
}

// NewDispute creates a new dispute for a candidate.
func NewDispute() (*Dispute, error) {
	disputeStatus, err := NewDisputeStatus()
	if err != nil {
		return nil, err
	}

	return &Dispute{
		Comparator:    Comparator{},
		DisputeStatus: disputeStatus,
	}, nil
}

// DisputeComparator compares two disputes.
func DisputeComparator(a, b any) bool {
	d1, d2 := a.(*Dispute), b.(*Dispute)

	if d1.Comparator.SessionIndex == d2.Comparator.SessionIndex {
		return bytes.Compare(d1.Comparator.CandidateHash[:], d2.Comparator.CandidateHash[:]) < 0
	}

	return d1.Comparator.SessionIndex < d2.Comparator.SessionIndex
}

type SendDispute struct {
	DisputeMessage UncheckedDisputeMessage
}

func (SendDispute) Index() uint {
	return 0
}

// DisputeDistributionMessage is the message sent to the collator to distribute the dispute
type DisputeDistributionMessage scale.VaryingDataType

// NewDisputeDistributionMessage returns a new dispute distribution message
func NewDisputeDistributionMessage() (DisputeDistributionMessage, error) {
	vdt, err := scale.NewVaryingDataType(SendDispute{})
	if err != nil {
		return DisputeDistributionMessage{}, fmt.Errorf("failed to create new varying data type: %w", err)
	}
	return DisputeDistributionMessage(vdt), nil
}
