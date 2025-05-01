package prospectiveparachains

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/primitives"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"golang.org/x/exp/maps"
)

const DefaultSchedulingLookAhead uint32 = 3

type BlockState interface {
	GetHeader(hash common.Hash) (header *types.Header, err error)
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (pp *ProspectiveParachains) ProcessActiveLeavesUpdateSignal(
	msg parachaintypes.ActiveLeavesUpdateSignal,
) error {
	// ensure there activated is not nil and it is not in the deactivated list (sanity check)
	if msg.Activated != nil && !slices.Contains(msg.Deactivated, msg.Activated.Hash) {
		tmpHeaderCache := make(map[common.Hash]*types.Header)

		hash := msg.Activated.Hash

		runtimeInstance, err := pp.blockState.GetRuntime(hash)
		if err != nil {
			return fmt.Errorf("getting runtime: %w", err)
		}

		claimQueue, err := runtimeInstance.ParachainHostClaimQueue()
		if err != nil {
			return fmt.Errorf("fetching claim queue: %w", err)
		}

		transposed := claimQueue.ToTransposed()

		blockInfo, err := pp.fetchBlockInfo(tmpHeaderCache, hash)
		if err != nil {
			return fmt.Errorf("fetching block info: %w", err)
		}

		sessionIndex, err := runtimeInstance.ParachainHostSessionIndexForChild()
		if err != nil {
			return fmt.Errorf("requesting session index for child: %w", err)
		}

		ancestryLen, err := runtimeInstance.ParachainHostSchedulingLookAhead()
		if err != nil {
			logger.Warnf("fetching scheduling lookahead: %s", err.Error())
			ancestryLen = DefaultSchedulingLookAhead
		}

		ancestryLen = primitives.SaturatingSub(ancestryLen, 1)

		ancestry, err := pp.fetchAncestry(
			hash,
			ancestryLen,
			sessionIndex,
			tmpHeaderCache,
		)
		if err != nil {
			return fmt.Errorf("fetching ancestry: %w", err)
		}

		ancestorsHashes := make([]string, len(ancestry))
		ancestryBlockInfo := make([]relayChainBlockInfo, len(ancestry))
		for idx, a := range ancestry {
			ancestryBlockInfo[idx] = relayChainBlockInfo{
				Hash:        a.Hash,
				Number:      a.Number,
				StorageRoot: a.StorageRoot,
			}
			ancestorsHashes[idx] = a.Hash.String()
		}

		var prevFragmentChains map[parachaintypes.ParaID]*fragmentChain
		if len(ancestry) > 0 {
			prevFragmentChains = pp.view.perRelayParent[ancestry[0].Hash].fragmentChains
		}

		fragmentChains := make(map[parachaintypes.ParaID]*fragmentChain)
		for paraID, claimsByDepth := range transposed {
			constraints, err := runtimeInstance.ParachainHostBackingConstraints(paraID)
			if err != nil {
				return fmt.Errorf("fetching backing constraints: %w", err)
			}

			if constraints == nil {
				logger.Warnf("failed to get inclusion backing constraints")
				continue
			}

			pendingAvailabilityCandidates, err := runtimeInstance.
				ParachainHostCandidatesPendingAvailability(
					paraID,
				)
			if err != nil {
				return fmt.Errorf("fetching pending availability candidates: %w", err)
			}

			importablePending, err := pp.preprocessCandidatesPendingAvailability(
				tmpHeaderCache,
				constraints,
				pendingAvailabilityCandidates,
			)
			if err != nil {
				return fmt.Errorf("preprocessing candidates pending availability: %w", err)
			}

			compactPending := make([]*pendingAvailability, 0, len(importablePending))
			pendingAvailabilityStorage := newCandidateStorage()

			for _, pending := range importablePending {
				candidateHash := pending.compact.candidateHash

				err := pendingAvailabilityStorage.addPendingAvailabilityCandidate(
					candidateHash,
					pending.candidate,
					pending.pvd,
				)

				if err != nil && !errors.Is(err, errCandidateAlreadyKnown) {
					logger.Warnf("scraped invalid candidate pending availability, "+
						"candidate hash=%s, para id=%d, err=%s", candidateHash, paraID, err.Error())
					break
				}

				compactPending = append(compactPending, &pending.compact)
			}

			maxBackableChainLen := len(maps.Values(claimsByDepth))

			scope, err := newScopeWithAncestors(
				relayChainBlockInfo{
					Hash:        blockInfo.Hash,
					Number:      blockInfo.Number,
					StorageRoot: blockInfo.StorageRoot,
				},
				constraints,
				compactPending,
				uint(maxBackableChainLen),
				ancestryBlockInfo,
			)
			if err != nil {
				logger.Warnf(
					"relay chain ancestors have wrong order, "+
						"para id=%d, max backable=%d, ancestry=%s, leaf=%s, err=%s",
					paraID,
					maxBackableChainLen,
					strings.Join(ancestorsHashes, ","),
					hash,
					err.Error(),
				)
				continue
			}

			logger.Tracef(
				"creating fragment chain, "+
					"relay parent=%s, min relay parent=%s, max backable=%d, para id=%d, ancestors=%s",
				hash,
				scope.earliestRelayParent().Number,
				maxBackableChainLen,
				paraID,
				strings.Join(ancestorsHashes, ","),
			)

			numOfPendingCandidates := pendingAvailabilityStorage.len()
			// Init the fragment chain with the pending availability candidates.
			chain := newFragmentChain(scope, pendingAvailabilityStorage)
			if chain.bestChainLen() < numOfPendingCandidates {
				logger.Warnf(
					"not all pending availability candidates could be introduced, "+
						"expected=%d, actual=%d, para id=%d, relay parent=%d",
					chain.bestChainLen(),
					numOfPendingCandidates,
					paraID,
					hash,
				)
			}

			// If we know the previous fragment chain, use that for further populating the fragment
			// chain.

			if prevFragmentChains != nil {
				if prev, ok := prevFragmentChains[paraID]; ok {
					chain.populateFromPrevious(prev)
				}
			}

			logger.Tracef("populated fragment chain with %d candidates: %v, relay parent=%s, para id=%d",
				chain.bestChainLen(), chain.bestChainVec(), hash, paraID)

			unconnectedHashes := make([]string, 0)
			for hash := range chain.unconnected.byCandidateHash {
				unconnectedHashes = append(unconnectedHashes, hash.String())
			}

			logger.Tracef("potential candidate storage for para: %v, relay parent=%s, para id=%d",
				strings.Join(unconnectedHashes, ","), hash, paraID)

			fragmentChains[paraID] = chain
		}

		pp.view.perRelayParent[hash] = &relayParentData{
			fragmentChains: fragmentChains,
		}

		pp.view.activeLeaves[hash] = true
		pp.view.implicitView.ActivateLeafFromProspectiveParachains(blockInfo, ancestry)
	}

	for _, deactivated := range msg.Deactivated {
		delete(pp.view.activeLeaves, deactivated)
		pp.view.implicitView.DeactivateLeaf(deactivated)
	}

	// keep in our state only the relay parents that are still active
	// under implicitly view
	allowedRelayParents := pp.view.implicitView.AllAllowedRelayParents()

	for rp := range pp.view.perRelayParent {
		if !slices.Contains(allowedRelayParents, rp) {
			delete(pp.view.perRelayParent, rp)
		}
	}

	return nil
}

func (pp *ProspectiveParachains) fetchBlockInfo(
	cache map[common.Hash]*types.Header,
	hash common.Hash,
) (*util.BlockInfoProspectiveParachains, error) {
	if header, ok := cache[hash]; ok {
		return &util.BlockInfoProspectiveParachains{
			Hash:        header.Hash(),
			ParentHash:  header.ParentHash,
			Number:      parachaintypes.BlockNumber(header.Number),
			StorageRoot: header.StateRoot,
		}, nil
	}

	header, err := pp.blockState.GetHeader(hash)
	if err != nil {
		return nil, fmt.Errorf("getting header: %w", err)
	}

	cache[hash] = header

	return &util.BlockInfoProspectiveParachains{
		Hash:        header.Hash(),
		ParentHash:  header.ParentHash,
		Number:      parachaintypes.BlockNumber(header.Number),
		StorageRoot: header.StateRoot,
	}, nil
}

func (pp *ProspectiveParachains) fetchAncestry(
	hash common.Hash,
	ancestryLen uint32,
	requiredSession parachaintypes.SessionIndex,
	cache map[common.Hash]*types.Header,
) ([]*util.BlockInfoProspectiveParachains, error) {
	if ancestryLen == 0 {
		return []*util.BlockInfoProspectiveParachains{}, nil
	}

	curr, err := pp.fetchBlockInfo(cache, hash)
	if err != nil {
		return nil, fmt.Errorf("fetching block info: %w", err)
	}

	parentHash := curr.ParentHash
	ancestorsBlockInfo := make([]*util.BlockInfoProspectiveParachains, 0, ancestryLen)
	for i := 0; i < int(ancestryLen); i++ {
		ancestorInfo, err := pp.fetchBlockInfo(cache, parentHash)
		if err != nil {
			return nil, fmt.Errorf("fetching ancestor block info: %w", err)
		}

		runtimeInstance, err := pp.blockState.GetRuntime(ancestorInfo.Hash)
		if err != nil {
			return nil, fmt.Errorf("getting runtime: %w", err)
		}

		ancestorSessionIndex, err := runtimeInstance.ParachainHostSessionIndexForChild()
		if err != nil {
			return nil, fmt.Errorf("fetching session index for child: %w", err)
		}

		// the relay chain cannot accept blocks backed from previous sessions, with
		// potentially previous validators. This is a technical limitation we need to
		// respect here.

		if requiredSession == ancestorSessionIndex {
			ancestorsBlockInfo = append(ancestorsBlockInfo, ancestorInfo)
		} else {
			break
		}

		parentHash = ancestorInfo.ParentHash
	}

	return ancestorsBlockInfo, nil
}

type importablePendingAvailability struct {
	candidate parachaintypes.CommittedCandidateReceiptV2
	pvd       parachaintypes.PersistedValidationData
	compact   pendingAvailability
}

func (pp *ProspectiveParachains) preprocessCandidatesPendingAvailability(
	blockInfoCache map[common.Hash]*types.Header,
	constraints *parachaintypes.VStagingConstraints,
	pendingAvailabilityCandidates []parachaintypes.CommittedCandidateReceiptV2,
) ([]importablePendingAvailability, error) {
	requiredParent := constraints.RequiredParent
	importable := make([]importablePendingAvailability, 0)
	expectedCount := len(pendingAvailabilityCandidates)

	for i, pending := range pendingAvailabilityCandidates {
		candidateHash, err := pending.Hash()
		if err != nil {
			return nil, fmt.Errorf("hashing pending candidate: %w", err)
		}

		relayParent, err := pp.fetchBlockInfo(blockInfoCache, pending.Descriptor.RelayParent)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("fetching block info: %w", err)
		}

		// if the block could not be fetch from database we should break and return
		// what we have accumulated so far
		if relayParent == nil || errors.Is(err, database.ErrNotFound) {
			logger.Errorf("had to stop processing pending "+
				"candidates early due to missing info, "+
				"candidate hash=%s, para id=%d, index=%d, expected count=%d, error=%s",
				candidateHash, pending.Descriptor.ParaID, i, expectedCount, err,
			)
			break
		}

		nextRequiredParent := pending.Commitments.HeadData
		importable = append(importable, importablePendingAvailability{
			candidate: parachaintypes.CommittedCandidateReceiptV2{
				Descriptor:  pending.Descriptor,
				Commitments: pending.Commitments,
			},
			pvd: parachaintypes.PersistedValidationData{
				ParentHead:             requiredParent,
				MaxPovSize:             constraints.MaxPoVSize,
				RelayParentNumber:      uint32(relayParent.Number),
				RelayParentStorageRoot: relayParent.StorageRoot,
			},
			compact: pendingAvailability{
				candidateHash: parachaintypes.CandidateHash{
					Value: candidateHash,
				},
				relayParent: relayChainBlockInfo{
					Hash:        relayParent.Hash,
					Number:      relayParent.Number,
					StorageRoot: relayParent.StorageRoot,
				},
			},
		})

		requiredParent = nextRequiredParent
	}

	return importable, nil
}
