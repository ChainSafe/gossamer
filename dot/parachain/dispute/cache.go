package dispute

import (
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type runtimeInfo struct {
	sessionInfoCache  map[parachainTypes.SessionIndex]parachainTypes.SessionInfo
	sessionIndexCache map[common.Hash]parachainTypes.SessionIndex

	runtime parachain.RuntimeInstance
}

func (r runtimeInfo) ParachainHostPersistedValidationData(parachaidID uint32, assumption parachainTypes.OccupiedCoreAssumption) (*parachainTypes.PersistedValidationData, error) {
	return r.runtime.ParachainHostPersistedValidationData(parachaidID, assumption)
}

func (r runtimeInfo) ParachainHostValidationCode(parachaidID uint32, assumption parachainTypes.OccupiedCoreAssumption) (*parachainTypes.ValidationCode, error) {
	return r.runtime.ParachainHostValidationCode(parachaidID, assumption)
}

func (r runtimeInfo) ParachainHostCheckValidationOutputs(parachainID uint32, outputs parachainTypes.CandidateCommitments) (bool, error) {
	return r.runtime.ParachainHostCheckValidationOutputs(parachainID, outputs)
}

func (r runtimeInfo) ParachainHostValidationCodeByHash(blockHash common.Hash, validationCodeHash parachainTypes.ValidationCodeHash) (*parachainTypes.ValidationCode, error) {
	return r.runtime.ParachainHostValidationCodeByHash(blockHash, validationCodeHash)
}

func (r runtimeInfo) ParachainHostOnChainVotes(blockHash common.Hash) (*parachainTypes.ScrapedOnChainVotes, error) {
	return r.runtime.ParachainHostOnChainVotes(blockHash)
}

func (r runtimeInfo) ParachainHostCandidateEvents(blockHash common.Hash) (*scale.VaryingDataTypeSlice, error) {
	return r.runtime.ParachainHostCandidateEvents(blockHash)
}

func (r runtimeInfo) ParachainHostSessionInfo(blockHash common.Hash, sessionIndex parachainTypes.SessionIndex) (*parachainTypes.SessionInfo, error) {
	if info, ok := r.sessionInfoCache[sessionIndex]; ok {
		return &info, nil
	}

	info, err := r.runtime.ParachainHostSessionInfo(blockHash, sessionIndex)
	if err != nil {
		return nil, err
	}

	r.sessionInfoCache[sessionIndex] = *info
	return info, nil
}

func (r runtimeInfo) ParachainHostSessionIndexForChild(blockHash common.Hash) (parachainTypes.SessionIndex, error) {
	if index, ok := r.sessionIndexCache[blockHash]; ok {
		return index, nil
	}

	index, err := r.runtime.ParachainHostSessionIndexForChild(blockHash)
	if err != nil {
		return 0, err
	}

	r.sessionIndexCache[blockHash] = index
	return index, nil
}

func newRuntimeInfo(runtime parachain.RuntimeInstance) *runtimeInfo {
	return &runtimeInfo{
		sessionInfoCache:  make(map[parachainTypes.SessionIndex]parachainTypes.SessionInfo),
		sessionIndexCache: make(map[common.Hash]parachainTypes.SessionIndex),
		runtime:           runtime,
	}
}

var _ parachain.RuntimeInstance = &runtimeInfo{}
