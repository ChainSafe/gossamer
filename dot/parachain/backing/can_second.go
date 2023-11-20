package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"golang.org/x/exp/slices"
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

	hypotheticalCandidate := parachaintypes.HCIncomplete{
		CandidateHash:      msg.CandidateHash,
		CandidateParaID:    msg.CandidateParaID,
		ParentHeadDataHash: msg.ParentHeadDataHash,
		RelayParent:        msg.CandidateRelayParent,
	}

	isSecondingAllowed, membership := cb.secondingSanityCheck(hypotheticalCandidate, true)

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
func (cb *CandidateBacking) secondingSanityCheck(
	hypotheticalCandidate parachaintypes.HypotheticalCandidate,
	backedInPathOnly bool,
) (bool, map[common.Hash][]uint) {
	// TODO: Implement this
	var (
		candidateParaID      parachaintypes.ParaID
		candidateRelayParent common.Hash
		candidateHash        parachaintypes.CandidateHash
	)
	membership := make(map[common.Hash][]uint)

	type response struct {
		depths          []uint
		head            common.Hash
		activeLeafState ActiveLeafState
	}
	var responses []response

	switch v := hypotheticalCandidate.(type) {
	case parachaintypes.HCIncomplete:
		candidateParaID = v.CandidateParaID
		candidateRelayParent = v.RelayParent
		candidateHash = v.CandidateHash
	case parachaintypes.HCComplete:
		candidateParaID = parachaintypes.ParaID(v.CommittedCandidateReceipt.Descriptor.ParaID)
		candidateRelayParent = v.CommittedCandidateReceipt.Descriptor.RelayParent
		candidateHash = v.CandidateHash
	}

	for head, leafState := range cb.perLeaf {
		if leafState.ProspectiveParachainsMode.IsEnabled {
			allowedParentsForPara := cb.implicitView.knownAllowedRelayParentsUnder(head, &candidateParaID)

			if !slices.Contains(allowedParentsForPara, candidateRelayParent) {
				continue
			}

			responseCh := make(chan parachaintypes.HypotheticalFrontierResponse)
			cb.SubSystemToOverseer <- parachaintypes.ProspectiveParachainsMessage{
				Value: parachaintypes.PPMGetHypotheticalFrontier{
					HypotheticalFrontierRequest: parachaintypes.HypotheticalFrontierRequest{
						Candidates:              []parachaintypes.HypotheticalCandidate{hypotheticalCandidate},
						FragmentTreeRelayParent: &head,
						BackedInPathOnly:        backedInPathOnly,
					},
					Ch: responseCh,
				},
			}

			res, ok := <-responseCh
			if ok {
				var depths []uint
				for _, val := range res {
					for _, membership := range val.FragmentTreeMembership {
						depths = append(depths, membership.Depths...)
					}
				}
				responses = append(responses, response{depths, head, leafState})
			}
		} else {
			if head == candidateRelayParent {
				leafState.
			}
		}
	}

	return false, nil
}
