package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"golang.org/x/exp/slices"
)

func (cb *CandidateBacking) handleCanSecondMessage(msg CanSecondMessage) {
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
			// candidate should be recognised by at least some fragment tree.
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
//
// if the candidate can be seconded, returns true and a map of the heads of active leaves to the depths,
// where the candidate is a member of the fragment tree.
// Returns false if the candidate cannot be seconded.
func (cb *CandidateBacking) secondingSanityCheck(
	hypotheticalCandidate parachaintypes.HypotheticalCandidate,
	backedInPathOnly bool, //nolint:unparam
) (bool, map[common.Hash][]uint) {
	type response struct {
		depths          []uint
		head            common.Hash
		activeLeafState ActiveLeafState
	}

	var (
		responses            []response
		candidateParaID      parachaintypes.ParaID
		candidateRelayParent common.Hash
		membership           = make(map[common.Hash][]uint)
	)

	switch v := hypotheticalCandidate.(type) {
	case parachaintypes.HCIncomplete:
		candidateParaID = v.CandidateParaID
		candidateRelayParent = v.RelayParent
	case parachaintypes.HCComplete:
		candidateParaID = parachaintypes.ParaID(v.CommittedCandidateReceipt.Descriptor.ParaID)
		candidateRelayParent = v.CommittedCandidateReceipt.Descriptor.RelayParent
	}

	for head, leafState := range cb.perLeaf {
		if leafState.ProspectiveParachainsMode.IsEnabled {
			allowedParentsForPara := cb.implicitView.knownAllowedRelayParentsUnder(head, candidateParaID)

			if !slices.Contains(allowedParentsForPara, candidateRelayParent) {
				continue
			}

			responseCh := make(chan parachaintypes.HypotheticalFrontierResponse)
			cb.SubSystemToOverseer <- parachaintypes.PPMGetHypotheticalFrontier{
				HypotheticalFrontierRequest: parachaintypes.HypotheticalFrontierRequest{
					Candidates:              []parachaintypes.HypotheticalCandidate{hypotheticalCandidate},
					FragmentTreeRelayParent: &head,
					BackedInPathOnly:        backedInPathOnly,
				},
				Ch: responseCh,
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
		} else if head == candidateRelayParent {
			if bTreeMap, ok := leafState.SecondedAtDepth[candidateParaID]; ok {
				if _, ok := bTreeMap.Get(0); ok {
					logger.Debug("Refusing to second candidate because leaf is already occupied.")
					return false, nil
				}
			}
			responses = append(responses, response{[]uint{0}, head, leafState})
		}
	}

	if len(responses) == 0 {
		return false, nil
	}

	for _, res := range responses {
		for _, depth := range res.depths {
			if bTreeMap, ok := res.activeLeafState.SecondedAtDepth[candidateParaID]; ok {
				if _, ok := bTreeMap.Get(depth); ok {
					logger.Debugf("Refusing to second candidate at depth %d - already occupied.", depth)
					return false, nil
				}
			}
		}
		membership[res.head] = res.depths
	}

	// At this point we've checked the depths of the candidate against all active leaves.
	return true, membership
}
