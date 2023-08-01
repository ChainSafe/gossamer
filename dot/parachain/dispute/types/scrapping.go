package types

import (
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	parachain "github.com/ChainSafe/gossamer/lib/parachain/types"
)

type BackingValidators struct {
	ValidatorIndex      parachain.ValidatorIndex
	ValidityAttestation inherents.ValidityAttestation
}

type BackingValidatorsPerCandidate struct {
	CandidateReceipt  common.Hash
	BackingValidators []BackingValidators
}

type ScrapedOnChainVotes struct {
	Session           parachain.SessionIndex
	BackingValidators BackingValidatorsPerCandidate
	Disputes          inherents.MultiDisputeStatementSet
}

type ScrappedUpdates struct {
	OnChainVotes     []ScrapedOnChainVotes
	IncludedReceipts []common.Hash
}
