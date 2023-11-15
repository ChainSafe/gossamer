package backing

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

func (cb *CandidateBacking) handleCanSecondMessage(msg CanSecondMessage) {
	// TODO: Implement this #3505

	rpState, ok := cb.perRelayParent[msg.CandidateRelayParent]
	if !ok {
		// Relay parent is unknown
		msg.resCh <- false
		return
	}

	ppMode := rpState.ProspectiveParachainsMode
	if !ppMode.IsEnabled {
		// async backing is disabled.
		msg.resCh <- false
		return
	}

	hypotheticalCandidate := HypotheticalCandidate{
		Value: HCIncomplete{
			CandidateHash:      msg.CandidateHash,
			CandidateParaID:    msg.CandidateParaID,
			ParentHeadDataHash: msg.ParentHeadDataHash,
			RelayParent:        msg.CandidateRelayParent,
		},
	}

	isSecondingAllowed, membership, err := secondingSanityCheck(hypotheticalCandidate)
	if err != nil {
		logger.Errorf("checking seconding sanity: %w", err)
		msg.resCh <- false
		return
	}

	if isSecondingAllowed {
		for _, v := range membership {
			// candidate should be recognized by at least some fragment tree.
			if v != nil {
				msg.resCh <- true
				return
			}
		}
	}

	msg.resCh <- false
}

// secondingSanityCheck checks whether a candidate can be seconded based on its
// hypothetical frontiers in the fragment tree and what we've already seconded in
// all active leaves.
func (cb *CandidateBacking) secondingSanityCheck(hypotheticalCandidate HypotheticalCandidate) (bool, map[common.Hash][]uint, error) {
	// TODO: Implement this
	var (
		candidateParaID      parachaintypes.ParaID
		candidateRelayParent common.Hash
		candidateHash        parachaintypes.CandidateHash
	)
	membership := make(map[common.Hash][]uint)

	switch v := hypotheticalCandidate.Value.(type) {
	case HCIncomplete:
		candidateParaID = v.CandidateParaID
		candidateRelayParent = v.RelayParent
		candidateHash = v.CandidateHash
	case HCComplete:
		candidateParaID = parachaintypes.ParaID(v.CommittedCandidateReceipt.Descriptor.ParaID)
		candidateRelayParent = v.CommittedCandidateReceipt.Descriptor.RelayParent
		candidateHash = v.CandidateHash
	default:
		return false, nil, fmt.Errorf("unexpected hypothetical candidate type: %T", v)
	}

	for head, leafState := range cb.perLeaf {
		if leafState.ProspectiveParachainsMode.IsEnabled {
			allowedParents := knownAllowedRelayParents
		}
	}

	return false, nil, nil
}

// HypotheticalCandidate represents a candidate to be evaluated for membership
// in the prospective parachains subsystem.
//
// Hypothetical candidates can be categorized into two types: complete and incomplete.
//   - Complete candidates have already had their potentially heavy candidate receipt
//     fetched, making them suitable for stricter evaluation.
//   - Incomplete candidates are simply claims about properties that a fetched candidate
//     would have and are evaluated less strictly.
type HypotheticalCandidate struct {
	Value any
}

// HCIncomplete represents an incomplete hypothetical candidate.
// this
type HCIncomplete struct {
	// CandidateHash is the claimed hash of the candidate.
	CandidateHash parachaintypes.CandidateHash
	// ParaID is the claimed para-ID of the candidate.
	CandidateParaID parachaintypes.ParaID
	// ParentHeadDataHash is the claimed head-data hash of the candidate.
	ParentHeadDataHash common.Hash
	// RelayParent is the claimed relay parent of the candidate.
	RelayParent common.Hash
}

// HCComplete represents a complete candidate, including its hash, committed candidate receipt,
// and persisted validation data.
type HCComplete struct {
	CandidateHash             parachaintypes.CandidateHash
	CommittedCandidateReceipt parachaintypes.CommittedCandidateReceipt
	PersistedValidationData   parachaintypes.PersistedValidationData
}
