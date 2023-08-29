package parachaintypes

import (
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
)

type BackingValidators struct {
	ValidatorIndex      ValidatorIndex
	ValidityAttestation inherents.ValidityAttestation
}

type BackingValidatorsPerCandidate struct {
	CandidateReceipt  common.Hash
	BackingValidators []BackingValidators
}

type ScrapedOnChainVotes struct {
	Session           SessionIndex
	BackingValidators BackingValidatorsPerCandidate
	Disputes          inherents.MultiDisputeStatementSet
}

type ScrapedUpdates struct {
	OnChainVotes     []ScrapedOnChainVotes
	IncludedReceipts []CandidateReceipt
}
