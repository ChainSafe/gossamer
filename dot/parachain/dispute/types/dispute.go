package types

import (
	"bytes"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/google/btree"
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

// Less returns true if the current dispute item is less than the other item
// it uses the Comparator to determine the order
func (d *Dispute) Less(than btree.Item) bool {
	other := than.(*Dispute)

	if d.Comparator.SessionIndex == other.Comparator.SessionIndex {
		return bytes.Compare(d.Comparator.CandidateHash[:], other.Comparator.CandidateHash[:]) < 0
	}

	return d.Comparator.SessionIndex < other.Comparator.SessionIndex
}
