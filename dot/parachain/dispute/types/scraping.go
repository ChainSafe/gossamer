package types

import (
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
)

type BackingValidators struct {
	ValidatorIndex      parachainTypes.ValidatorIndex
	ValidityAttestation inherents.ValidityAttestation
}

type BackingValidatorsPerCandidate struct {
	CandidateReceipt  common.Hash
	BackingValidators []BackingValidators
}

type ScrapedOnChainVotes struct {
	Session           parachainTypes.SessionIndex
	BackingValidators BackingValidatorsPerCandidate
	Disputes          inherents.MultiDisputeStatementSet
}

type ScrappedUpdates struct {
	OnChainVotes     []ScrapedOnChainVotes
	IncludedReceipts []parachainTypes.CandidateReceipt
}
