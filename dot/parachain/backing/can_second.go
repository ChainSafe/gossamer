// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"
	"sync"

	prospectiveparachains "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"golang.org/x/exp/slices"
)

var errUnknownRelayParent = errors.New("unknown relay parent")

// handleCanSecondMessage performs seconding sanity check for an advertisement.
func (cb *CandidateBacking) handleCanSecondMessage(msg CanSecondMessage) error {
	_, ok := cb.perRelayParent[msg.CandidateRelayParent]
	if !ok {
		msg.ResponseCh <- false
		return fmt.Errorf("%w: %s", errUnknownRelayParent, msg.CandidateRelayParent.String())
	}

	hypotheticalCandidate := parachaintypes.HypotheticalCandidateIncomplete{
		ClaimedCandidateHash: msg.CandidateHash,
		CandidateParaID:      msg.CandidateParaID,
		ParentHeadDataHash:   msg.ParentHeadDataHash,
		RelayParent:          msg.CandidateRelayParent,
	}

	leavesForSeconding := cb.secondingSanityCheck(hypotheticalCandidate)

	// true if `leavesForSeconding` is not empty
	msg.ResponseCh <- len(leavesForSeconding) != 0
	return nil
}

// secondingSanityCheck checks whether a candidate can be seconded
// based on its hypothetical membership in the fragment chain.
func (cb *CandidateBacking) secondingSanityCheck(
	hypotheticalCandidate parachaintypes.HypotheticalCandidate,
) []common.Hash {
	activeLeaves := cb.implicitView.Leaves()
	leavesForSecondingCh := make(chan common.Hash, len(activeLeaves))
	var wg sync.WaitGroup

	for _, head := range activeLeaves {
		wg.Add(1)
		go func(head common.Hash) {
			defer wg.Done()
			cb.checkLeafForSeconding(head, hypotheticalCandidate, leavesForSecondingCh)
		}(head)
	}

	wg.Wait()
	close(leavesForSecondingCh)

	leavesForSeconding := make([]common.Hash, 0, len(activeLeaves))
	for leaf := range leavesForSecondingCh {
		leavesForSeconding = append(leavesForSeconding, leaf)
	}

	return leavesForSeconding
}

func (cb *CandidateBacking) checkLeafForSeconding(
	activeLeaf common.Hash,
	hypotheticalCandidate parachaintypes.HypotheticalCandidate,
	leavesForSecondingCh chan<- common.Hash,
) {
	candidateParaID := hypotheticalCandidate.ParaID()
	candidateRelayParent := hypotheticalCandidate.RelayParentHash()
	candidateHash := hypotheticalCandidate.CandidateHash()

	// Check that the candidate relay parent is allowed for para, skip the leaf otherwise.
	allowedParentsForPara := cb.implicitView.KnownAllowedRelayParentsUnder(
		activeLeaf,
		&candidateParaID,
	)
	if !slices.Contains(allowedParentsForPara, candidateRelayParent) {
		return
	}

	responseCh := make(chan []*prospectiveparachains.HypotheticalMembershipResponseItem)
	cb.SubSystemToOverseer <- prospectiveparachains.GetHypotheticalMembership{
		Candidates: []parachaintypes.HypotheticalCandidate{hypotheticalCandidate},
		Response:   responseCh,
	}

	memberships, ok := <-responseCh
	if !ok {
		logger.Error("failed to receive hypothetical membership response: response channel is closed")
		return
	}

	if isMemberOrPotentialMember(memberships, candidateHash, activeLeaf) {
		leavesForSecondingCh <- activeLeaf
	}
}

// isMemberOrPotentialMember checks if the given candidate hash is a member or
// potential member of the candidate memberships.
func isMemberOrPotentialMember(
	memberships []*prospectiveparachains.HypotheticalMembershipResponseItem,
	hypotheticalCandidateHash parachaintypes.CandidateHash,
	activeLeaf common.Hash,
) bool {
	for _, item := range memberships {
		if item.HypotheticalCandidate.CandidateHash() == hypotheticalCandidateHash {
			if slices.Contains(item.HypotheticalMembership, activeLeaf) {
				return true
			}
		}
	}
	return false
}
