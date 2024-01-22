// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"golang.org/x/exp/slices"
)

var (
	errUnknwnRelayParent                 = errors.New("unknown relay parent")
	errProspectiveParachainsModeDisabled = errors.New("async backing is disabled")
	errCandidateNotRecognised            = errors.New("candidate not recognised by any fragment tree")
	errLeafOccupied                      = errors.New("can't second the candidate, leaf is already occupied")
	errNoActiveLeaves                    = errors.New("no active leaves found for the candidate relay parent")
	errDepthOccupied                     = errors.New("can't second the candidate, depth is already occupied")
)

// handleCanSecondMessage performs seconding sanity check for an advertisement.
func (cb *CandidateBacking) handleCanSecondMessage(msg CanSecondMessage) error {
	rpState, ok := cb.perRelayParent[msg.CandidateRelayParent]
	if !ok {
		msg.ResponseCh <- false
		return fmt.Errorf("%w: %s", errUnknwnRelayParent, msg.CandidateRelayParent.String())
	}

	if rpState == nil {
		msg.ResponseCh <- false
		return fmt.Errorf("%w; relay parent: %s", errNilRelayParentState, msg.CandidateRelayParent.String())
	}

	ppMode := rpState.prospectiveParachainsMode
	if !ppMode.IsEnabled {
		msg.ResponseCh <- false
		return fmt.Errorf("%w; relay parent: %s", errprospectiveParachainsModeDisabled, msg.CandidateRelayParent.String())
	}

	hypotheticalCandidate := parachaintypes.HypotheticalCandidateIncomplete{
		CandidateHash:      msg.CandidateHash,
		CandidateParaID:    msg.CandidateParaID,
		ParentHeadDataHash: msg.ParentHeadDataHash,
		RelayParent:        msg.CandidateRelayParent,
	}

	membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)
	if err != nil {
		msg.ResponseCh <- false
		return err
	}

	for _, fragmentTree := range membership {
		// candidate should be recognised by at least some fragment tree.
		if len(fragmentTree) != 0 {
			msg.ResponseCh <- true
			return nil
		}
	}
	return fmt.Errorf("%w; candidate hash: %s", errCandidateNotRecognised, msg.CandidateHash.Value)
}

// secondingSanityCheck checks whether a candidate can be seconded based on its
// hypothetical frontiers in the fragment tree and what we've already seconded in
// all active leaves.
//
// If the candidate can be seconded, returns nil error and a map of the heads of active leaves to the depths,
// where the candidate is a member of the fragment tree.
// Returns error if the candidate cannot be seconded.
func (cb *CandidateBacking) secondingSanityCheck(
	hypotheticalCandidate parachaintypes.HypotheticalCandidate,
	backedInPathOnly bool, //nolint:unparam
) (map[common.Hash][]uint, error) {
	var (
		candidateParaID      parachaintypes.ParaID
		candidateRelayParent common.Hash
		candidateHash        parachaintypes.CandidateHash
		membership           = make(map[common.Hash][]uint)
	)

	switch v := hypotheticalCandidate.(type) {
	case parachaintypes.HypotheticalCandidateIncomplete:
		candidateParaID = v.CandidateParaID
		candidateRelayParent = v.RelayParent
		candidateHash = v.CandidateHash
	case parachaintypes.HypotheticalCandidateComplete:
		candidateParaID = parachaintypes.ParaID(v.CommittedCandidateReceipt.Descriptor.ParaID)
		candidateRelayParent = v.CommittedCandidateReceipt.Descriptor.RelayParent
		candidateHash = v.CandidateHash
	}

	for head, leafState := range cb.perLeaf {
		if leafState.prospectiveParachainsMode.IsEnabled {

			// check that the candidate relay parent is allowed for parachain, skip the leaf otherwise.
			allowedParentsForPara := cb.implicitView.knownAllowedRelayParentsUnder(head, candidateParaID)
			if !slices.Contains(allowedParentsForPara, candidateRelayParent) {
				continue
			}

			responseCh := make(chan parachaintypes.HypotheticalFrontierResponses)
			cb.SubSystemToOverseer <- parachaintypes.ProspectiveParachainsMessageGetHypotheticalFrontier{
				HypotheticalFrontierRequest: parachaintypes.HypotheticalFrontierRequest{
					Candidates:              []parachaintypes.HypotheticalCandidate{hypotheticalCandidate},
					FragmentTreeRelayParent: &head,
					BackedInPathOnly:        backedInPathOnly,
				},
				ResponseCh: responseCh,
			}

			res, ok := <-responseCh
			if ok {
				var depths []uint
				// collect all depths from all fragment trees
				for _, val := range res {
					for _, membership := range val.Memberships {
						depths = append(depths, membership.Depths...)
					}
				}

				if isSeconded, atDepth := checkDepthsAgainstLeaftState(depths, leafState, candidateParaID); isSeconded {
					return nil, fmt.Errorf(
						"%w; candidate hash: %s; relay parent: %s; parachain id: %v; depth: %d",
						errDepthOccupied,
						candidateHash.Value.String(),
						candidateRelayParent.String(),
						candidateParaID,
						*atDepth,
					)
				}

				membership[head] = depths

			} else {
				logger.Error("prospective parachains message get hypothetical frontier's response channel is closed")
			}
		} else if head == candidateRelayParent {
			if isSeconded := isSecondedAtDepth(0, leafState, candidateParaID); isSeconded {
				return nil, fmt.Errorf(
					"%w; candidate hash: %s; relay parent: %s; parachain id: %v",
					errLeafOccupied,
					candidateHash,
					candidateRelayParent.String(),
					candidateParaID,
				)
			}
			membership[head] = []uint{0}
		}
	}

	if len(membership) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoActiveLeaves, candidateRelayParent.String())
	}

	// At this point we've checked the depths of the candidate against all active leaves.
	return membership, nil
}

func checkDepthsAgainstLeaftState(depths []uint, leafState activeLeafState, paraID parachaintypes.ParaID,
) (bool, *uint) {
	for _, depth := range depths {
		if isSeconded := isSecondedAtDepth(depth, leafState, paraID); isSeconded {
			return true, &depth
		}
	}
	return false, nil
}

func isSecondedAtDepth(depth uint, leafState activeLeafState, candidateParaID parachaintypes.ParaID) bool {
	if bTreeMap, ok := leafState.secondedAtDepth[candidateParaID]; ok {
		if _, ok := bTreeMap.Get(depth); ok {
			return true
		}
	}
	return false
}
