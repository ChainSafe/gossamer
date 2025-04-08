// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package prospectiveparachains

import (
	"context"
	"errors"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	"github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-prospective-parachains"))

// Initialize with empty values.
func NewView() *view {
	//nolint:lll
	return &view{
		perRelayParent: make(map[common.Hash]*relayParentData),
		activeLeaves:   make(map[common.Hash]bool),
		implicitView:   nil, // TODO: currently there's no implementation for ImplicitView, reference is: //nolint:lll
		//  https://github.com/paritytech/polkadot-sdk/blob/028e61be43f05f6f6c88c5cca94160f8db075585/polkadot/node/subsystem-util/src/backing_implicit_view.rs#L40 //nolint:lll
	}
}

type ProspectiveParachains struct {
	SubsystemToOverseer chan<- any
	View                *view
}

type view struct {
	activeLeaves   map[common.Hash]bool
	perRelayParent map[common.Hash]*relayParentData
	implicitView   backing.ImplicitView
}

type relayParentData struct {
	fragmentChains map[parachaintypes.ParaID]*fragmentChain
}

// Name returns the name of the subsystem
func (*ProspectiveParachains) Name() parachaintypes.SubSystemName {
	return parachaintypes.ProspectiveParachains
}

// NewProspectiveParachains creates a new ProspectiveParachain subsystem
func NewProspectiveParachains(overseerChan chan<- any) *ProspectiveParachains {
	prospectiveParachain := ProspectiveParachains{
		SubsystemToOverseer: overseerChan,
		View:                NewView(),
	}
	return &prospectiveParachain
}

// Run starts the ProspectiveParachains subsystem
func (pp *ProspectiveParachains) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubsystem:
			pp.processMessage(msg)
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (*ProspectiveParachains) Stop() {}

func (pp *ProspectiveParachains) processMessage(msg any) {
	switch msg := msg.(type) {
	case parachaintypes.Conclude:
		pp.Stop()
	case parachaintypes.ActiveLeavesUpdateSignal:
		_ = pp.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		_ = pp.ProcessBlockFinalizedSignal(msg)
	case messages.IntroduceSecondedCandidate:
		pp.introduceSecondedCandidate(
			pp.View,
			msg.Request,
			msg.Response,
		)
	case messages.CandidateBacked:
		pp.handleCandidateBacked(msg)
	case messages.GetBackableCandidates:
		pp.getBackableCandidates(msg)
	case messages.GetHypotheticalMembership:
		panic("not implemented yet: see issue #4311")
	case messages.GetMinimumRelayParents:
		// Directly use the msg since it's already of type GetMinimumRelayParents
		pp.getMinimumRelayParents(msg.RelayChainBlockHash, msg.Sender)
	case messages.GetProspectiveValidationData:
		pp.answerProspectiveValidationDataRequest(msg.ProspectiveValidationDataRequest, msg.Sender)
	default:
		logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
}

func (pp *ProspectiveParachains) introduceSecondedCandidate(
	view *view,
	request messages.IntroduceSecondedCandidateRequest,
	response chan bool,
) {
	defer close(response)

	para := request.CandidateParaID
	candidate := request.CandidateReceipt
	pvd := request.PersistedValidationData

	hash, err := candidate.Hash()
	if err != nil {
		logger.Tracef("hashing candidate: %s", err.Error())
		response <- false
		return
	}

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	entry, err := newCandidateEntry(
		candidateHash,
		candidate,
		pvd,
		seconded,
	)
	if err != nil {
		logger.Tracef("adding seconded candidate error: %s para: %v", err.Error(), para)
		response <- false
		return
	}

	added := make([]common.Hash, 0, len(view.perRelayParent))
	paraScheduled := false

	for relayParent, rpData := range view.perRelayParent {
		chain, exists := rpData.fragmentChains[para]
		if !exists {
			continue
		}

		_, isActiveLeaf := view.activeLeaves[relayParent]

		paraScheduled = true

		err = chain.tryAddingSecondedCandidate(entry)
		if err != nil {
			if errors.Is(err, errCandidateAlreadyKnown) {
				logger.Tracef(
					"attempting to introduce an already known candidate with hash: %s, para: %v relayParent: %v isActiveLeaf: %v",
					candidateHash,
					para,
					relayParent,
					isActiveLeaf,
				)
				added = append(added, relayParent)
			} else {
				logger.Tracef(
					"adding seconded candidate with hash: %s error: %s para: %v relayParent: %v isActiveLeaf: %v",
					candidateHash,
					err.Error(),
					para,
					relayParent,
					isActiveLeaf,
				)
			}
		} else {
			added = append(added, relayParent)
		}
	}

	if !paraScheduled {
		logger.Warnf(
			"received seconded candidate with hash: %s for inactive para: %v",
			candidateHash,
			para,
		)
	}

	if len(added) == 0 {
		logger.Debugf(
			"newly-seconded candidate cannot be kept under any relay parent: %s",
			candidateHash,
		)
	} else {
		logger.Tracef("added seconded candidate to %d relay parents: %s", len(added), candidateHash)
	}

	response <- len(added) > 0
}

func (pp *ProspectiveParachains) handleCandidateBacked(msg messages.CandidateBacked) {
	para := msg.ParaID
	candidateHash := msg.CandidateHash

	foundCandidate := false
	foundPara := false

	for relayParent, rpData := range pp.View.perRelayParent {
		chain, ok := rpData.fragmentChains[para]
		if !ok {
			continue
		}

		_, isActiveLeaf := pp.View.activeLeaves[relayParent]

		foundPara = true
		if chain.isCandidateBacked(candidateHash) {
			logger.Debugf(
				"para = %s, candidateHash = %s, isActiveLeaf = %s, "+
					"Received redundant instruction to mark as backed an already backed candidate",
				para,
				candidateHash,
				isActiveLeaf,
			)
			foundCandidate = true
		} else if chain.containsUnconnectedCandidate(candidateHash) {
			foundCandidate = true
			chain.candidateBacked(candidateHash)

			var candidatedHashes []parachaintypes.CandidateHash

			for _, candidateEntry := range chain.unconnected.byCandidateHash {
				candidatedHashes = append(candidatedHashes, candidateEntry.candidateHash)
			}

			logger.Tracef("relayParent = %s, para = %s, candidateHash = %s, "+
				"isActiveLeaf = %s, Candidate backed. Candidate chain for para: %v",
				relayParent, para, candidateHash, isActiveLeaf, chain.bestChainVec())

			logger.Tracef("relayParent = %s, para = %s, candidateHash = %s, "+
				"isActiveLeaf = %s, Potential candidate storage for para: %v",
				relayParent, para, candidateHash, isActiveLeaf, candidatedHashes)
		}

		if !foundPara {
			logger.Warnf(
				"para = %s, candidateHash = %s, Received instruction to back a candidate for unscheduled para",
				para,
				candidateHash,
			)
			return
		}

		if !foundCandidate {
			logger.Debugf(
				"para = %s, candidateHash = %s, Received instruction to back unknown candidate",
				para,
				candidateHash,
			)
		}
	}
}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (pp *ProspectiveParachains) ProcessActiveLeavesUpdateSignal(
	parachaintypes.ActiveLeavesUpdateSignal,
) error {
	panic("not implemented yet: see issue #4305")
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (*ProspectiveParachains) ProcessBlockFinalizedSignal(
	parachaintypes.BlockFinalizedSignal,
) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}

func (pp *ProspectiveParachains) getMinimumRelayParents(
	relayChainBlockHash common.Hash,
	sender chan []messages.ParaIDBlockNumber,
) {
	var result []messages.ParaIDBlockNumber

	// Check if the relayChainBlockHash exists in active_leaves
	if exists := pp.View.activeLeaves[relayChainBlockHash]; exists {
		// Retrieve data associated with the relayChainBlockHash
		if leafData, found := pp.View.perRelayParent[relayChainBlockHash]; found {
			// Iterate over fragment_chains and collect the data
			for paraID, fragmentChain := range leafData.fragmentChains {
				result = append(result, messages.ParaIDBlockNumber{
					ParaId:      paraID,
					BlockNumber: fragmentChain.scope.relayParent.Number,
				})
			}
		}
	}

	// Send the result through the sender channel
	sender <- result
}

func (pp *ProspectiveParachains) getBackableCandidates(
	msg messages.GetBackableCandidates,
) {
	// Extract details from the message
	relayParentHash := msg.RelayParentHash
	paraId := msg.ParaId
	requestedQty := msg.RequestedQty
	ancestors := msg.Ancestors
	responseChan := msg.Response

	// Check if the relay parent is active
	if _, exists := pp.View.activeLeaves[relayParentHash]; !exists {
		logger.Debugf(
			"Requested backable candidates for inactive relay-parent. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve data for the relay parent
	data, ok := pp.View.perRelayParent[relayParentHash]
	if !ok {
		logger.Debugf(
			"Requested backable candidates for nonexistent relay-parent. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve the fragment chain for the ParaID
	chain, ok := data.fragmentChains[paraId]
	if !ok {
		logger.Debugf(
			"Requested backable candidates for inactive ParaID. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve backable candidates from the fragment chain
	backableCandidates := chain.findBackableChain(ancestors, requestedQty)
	if len(backableCandidates) == 0 {
		logger.Debugf(
			"No backable candidates found. RelayParentHash: %v, ParaId: %v, Ancestors: %v",
			relayParentHash, paraId, ancestors,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	logger.Debugf(
		"Found backable candidates: %v. RelayParentHash: %v, ParaId: %v, Ancestors: %v",
		backableCandidates, relayParentHash, paraId, ancestors,
	)

	// Convert backable candidates to the expected response format
	candidateHashes := make([]parachaintypes.CandidateHashAndRelayParent, len(backableCandidates))
	for i, candidate := range backableCandidates {
		candidateHashes[i] = parachaintypes.CandidateHashAndRelayParent{
			CandidateHash:        candidate.candidateHash,
			CandidateRelayParent: candidate.realyParentHash,
		}
	}

	// Send the result through the response channel
	responseChan <- candidateHashes
}

func (pp *ProspectiveParachains) answerProspectiveValidationDataRequest(
	request messages.ProspectiveValidationDataRequest,
	response chan<- *parachaintypes.PersistedValidationData,
) {
	var headData *parachaintypes.HeadData
	var parentHeadDataHash common.Hash

	// extracting informations from the request depending on the incoming type.
	switch value := request.ParentHeadData.(type) {
	case messages.OnlyHash:
		parentHeadDataHash = common.Hash(value)
	case messages.ParentHeadDataWithHash:
		headData = &value.Data
		parentHeadDataHash = value.Hash
	}

	var relayParentInfo *relayChainBlockInfo
	var maxPovSize *uint32

	// // iterate over active leaves
	for leaf := range pp.View.activeLeaves {
		relayBlockViewData, exists := pp.View.perRelayParent[leaf]

		if !exists {
			continue
		}

		fragmentChain, exists := relayBlockViewData.fragmentChains[request.ParaId]
		if !exists {
			continue
		}

		// stop the iteration once we retrieve all the informations
		if headData != nil && relayParentInfo != nil && maxPovSize != nil {
			break
		}

		if relayParentInfo == nil {
			relayParentInfo = fragmentChain.scope.ancestor(request.CandidateRelayParent)
		}

		if headData == nil {
			var err error
			headData, err = fragmentChain.getHeadDataByHash(parentHeadDataHash)
			if err != nil {
				response <- nil
				return
			}
		}

		if maxPovSize == nil {
			containAncestor := fragmentChain.scope.ancestor(request.CandidateRelayParent) != nil

			if containAncestor {
				maxPovSize = &fragmentChain.scope.baseConstraints.MaxPoVSize
			}
		}
	}

	if headData != nil && relayParentInfo != nil && maxPovSize != nil {
		response <- &parachaintypes.PersistedValidationData{
			ParentHead:             *headData,
			RelayParentNumber:      uint32(relayParentInfo.Number),
			RelayParentStorageRoot: relayParentInfo.StorageRoot,
			MaxPovSize:             *maxPovSize,
		}
	} else {
		response <- nil
	}
}
