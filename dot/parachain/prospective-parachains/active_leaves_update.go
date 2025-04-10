package prospectiveparachains

import (
	"fmt"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/primitives"
	"github.com/ChainSafe/gossamer/lib/runtime"
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
	// TODO: metrics

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
			// TODO: add a tracing log here
			ancestryLen = DefaultSchedulingLookAhead
		}

		ancestryLen = primitives.SaturatingSub(ancestryLen, 1)

		ancestry, err := pp.fetchAncestry(
			hash,
			ancestryLen,
			sessionIndex,
			tmpHeaderCache,
			runtimeInstance,
		)
		if err != nil {
			return fmt.Errorf("fetching ancestry: %w", err)
		}

		var prevFragmentChains map[parachaintypes.ParaID]*fragmentChain
		if len(ancestry) > 0 {
			prevFragmentChains = pp.view.perRelayParent[ancestry[0].Hash()].fragmentChains
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

			pendingAvailability, err := runtimeInstance.ParachainHostCandidatesPendingAvailability(paraID)
			if err != nil {
				return fmt.Errorf("fetching pending availability candidates: %w", err)
			}

			preprocessCandidatesPendingAvailability(tmpHeaderCache, constraints, pendingAvailability)
		}
	}

	return nil
}

func (pp *ProspectiveParachains) fetchBlockInfo(
	cache map[common.Hash]*types.Header,
	hash common.Hash,
) (*types.Header, error) {
	if header, ok := cache[hash]; ok {
		return header, nil
	}

	header, err := pp.blockState.GetHeader(hash)
	if err != nil {
		return nil, fmt.Errorf("getting header: %w", err)
	}

	cache[hash] = header
	return header, nil
}

func (pp *ProspectiveParachains) fetchAncestry(
	hash common.Hash,
	ancestryLen uint32,
	requiredSession parachaintypes.SessionIndex,
	cache map[common.Hash]*types.Header,
	rt runtime.Instance,
) ([]*types.Header, error) {
	if ancestryLen == 0 {
		return []*types.Header{}, nil
	}

	ancestors, err := util.GetBlockAncestors(pp.SubsystemToOverseer, hash, ancestryLen)
	if err != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", err)
	}

	blockInfos := make([]*types.Header, len(ancestors))

	for _, ancestorHash := range ancestors {
		ancestorHead, err := pp.fetchBlockInfo(cache, ancestorHash)
		if err != nil {
			logger.Warnf("failed to fetch info for hash returned from ancestry: %s", err.Error())
			// Return, however far we got.
			break
		}

		runtimeInstance, err := pp.blockState.GetRuntime(ancestorHash)
		if err != nil {
			return nil, fmt.Errorf("getting runtime: %w", err)
		}

		acenstorSessionIndex, err := runtimeInstance.ParachainHostSessionIndexForChild()
		if err != nil {
			return nil, fmt.Errorf("fetching session index for child: %w", err)
		}

		// the relay chain cannot accept blocks backed from previous sessions, with
		// potentially previous validators. This is a technical limitation we need to
		// respect here.

		if requiredSession == acenstorSessionIndex {
			blockInfos = append(blockInfos, ancestorHead)
		} else {
			break
		}
	}

	return blockInfos, nil
}

type importablePendingAvailability struct {
	candidate parachaintypes.CommittedCandidateReceipt
	pvd       parachaintypes.PersistedValidationData
	compact   pendingAvailability
}

func preprocessCandidatesPendingAvailability(
	blockInfoCache map[common.Hash]*types.Header,
	constraints *parachaintypes.Constraints,
	pendingAvailability []parachaintypes.CommittedCandidateReceipt,
) {
}
