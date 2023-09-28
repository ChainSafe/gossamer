package parachaintypes

import (
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
)

// BackingValidators backing validators for a candidate
type BackingValidators struct {
	ValidatorIndex      ValidatorIndex
	ValidityAttestation inherents.ValidityAttestation
}

// BackingValidatorsPerCandidate Set of backing validators for each candidate, represented by its candidate receipt.
type BackingValidatorsPerCandidate struct {
	CandidateReceipt  CandidateReceipt
	BackingValidators []BackingValidators
}

// ScrapedOnChainVotes scraped runtime backing votes and resolved disputes
type ScrapedOnChainVotes struct {
	Session           SessionIndex
	BackingValidators []BackingValidatorsPerCandidate
	Disputes          inherents.MultiDisputeStatementSet
}

// ScrapedUpdates Updates to `OnChainVotes` and included receipts for new active leaf and its unprocessed ancestors.
type ScrapedUpdates struct {
	// New votes as seen on chain
	OnChainVotes []ScrapedOnChainVotes
	// Newly included parachain block candidate receipts as seen on chain
	IncludedReceipts []CandidateReceipt
}
