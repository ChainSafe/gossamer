package availabilitydistribution

import (
	"fmt"
	"sync"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
)

type taskTerminationReason interface {
	isTaskTerminationReason()
}

type taskSucceeded struct {
	sessionIndex  parachaintypes.SessionIndex
	groupIndex    parachaintypes.GroupIndex
	badValidators []parachaintypes.AuthorityDiscoveryID
}

func (taskSucceeded) isTaskTerminationReason() {}

type taskFailed struct{}

func (taskFailed) isTaskTerminationReason() {}

type taskCancelled struct{}

func (taskCancelled) isTaskTerminationReason() {}

type taskTerminationHandler func(
	candidateHash parachaintypes.CandidateHash,
	reason taskTerminationReason,
)

type fetchChunkTask struct {
	chunkIndex          uint32
	sessionInfo         *SessionInfo
	core                *parachaintypes.OccupiedCore
	subsystemToOverseer chan<- any
	onTermination       taskTerminationHandler

	// Set of relay chain block hashes for which the candidate associated with the given core is pending availability.
	//
	// In other words, for which relay chain parents this candidate is considered live.
	// This is updated on every `ActiveLeavesUpdate` and enables us to know when we can safely
	// stop keeping track of that candidate/chunk.
	liveIn map[common.Hash]struct{}
	mu     sync.Mutex

	// List of validators that did not have the requested chunk or sent invalid data.
	badValidators []parachaintypes.AuthorityDiscoveryID

	stop chan bool
}

func newFetchChunkTask(
	leaf common.Hash,
	chunkIndex uint32,
	sessionInfo *SessionInfo,
	core *parachaintypes.OccupiedCore,
	subsystemToOverseer chan<- any,
	onTermination taskTerminationHandler,
) *fetchChunkTask {
	return &fetchChunkTask{
		liveIn:              map[common.Hash]struct{}{leaf: {}},
		chunkIndex:          chunkIndex,
		sessionInfo:         sessionInfo,
		core:                core,
		subsystemToOverseer: subsystemToOverseer,
		onTermination:       onTermination,
		badValidators:       make([]parachaintypes.AuthorityDiscoveryID, 0),
		stop:                make(chan bool, 1),
	}
}

func (t *fetchChunkTask) run() {
	group := t.sessionInfo.ValidatorGroups[t.core.GroupResponsible]

	// Try validators in reverse order
	for i := len(group) - 1; i >= 0; i-- {
		if t.isCancelled() {
			t.cleanup(taskCancelled{})
			return
		}

		authorityID := group[i]

		request := networkbridgemessages.NewOutgoingRequest(
			authorityID,
			&networkbridgemessages.ChunkFetchingRequest{
				CandidateHash: parachaintypes.CandidateHash{Value: t.core.CandidateHash},
				Index:         t.sessionInfo.OurIndex,
			})

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.ImmediateError,
		}

		t.subsystemToOverseer <- sendRequests

		var result networkbridgemessages.ReqRespResult
		select {
		case <-t.stop:
			request.Cancel()
			t.cleanup(taskCancelled{})
			return
		case result = <-request.Result:
		}

		chunk, err := t.extractChunk(result)
		if err != nil || !t.validateChunk(chunk) {
			t.badValidators = append(t.badValidators, authorityID)
			continue
		}

		t.subsystemToOverseer <- availabilitystore.StoreChunk{
			CandidateHash: parachaintypes.CandidateHash{Value: t.core.CandidateHash},
			Chunk:         chunk,
		}

		t.cleanup(taskSucceeded{
			sessionIndex:  t.sessionInfo.SessionIndex,
			groupIndex:    t.core.GroupResponsible,
			badValidators: t.badValidators,
		})

		return
	}

	// None of the requested validators were able to provide the chunk.
	t.cleanup(taskFailed{})
}

func (t *fetchChunkTask) extractChunk(
	result networkbridgemessages.ReqRespResult,
) (availabilitystore.ErasureChunk, error) {
	if result.Error != nil {
		return availabilitystore.ErasureChunk{}, result.Error
	}

	var response *networkbridgemessages.ChunkFetchingResponse

	switch result.Response.(type) {
	case *networkbridgemessages.ChunkFetchingResponse:
		response = result.Response.(*networkbridgemessages.ChunkFetchingResponse)
	default:
		return availabilitystore.ErasureChunk{}, fmt.Errorf(
			"unexpected network message type in response: %T",
			result.Response,
		)
	}

	v, err := response.Value()
	if err != nil {
		return availabilitystore.ErasureChunk{}, err
	}

	switch chunkResponse := v.(type) {
	case networkbridgemessages.ChunkResponse:
		return availabilitystore.ErasureChunk{
			Chunk: chunkResponse.Chunk,
			Index: chunkResponse.Index,
			Proof: chunkResponse.Proof,
		}, nil
	default:
		return availabilitystore.ErasureChunk{}, fmt.Errorf("chunk not found")
	}
}

func (t *fetchChunkTask) validateChunk(chunk availabilitystore.ErasureChunk) bool {
	if chunk.Index != t.chunkIndex {
		return false
	}

	chunkHash, err := common.Blake2bHash(chunk.Chunk)
	if err != nil {
		logger.Warnf("hashing erasure chunk: %s", err.Error())
		return false
	}

	branchHash, err := erasure.BranchHash(t.core.CandidateDescriptor.ErasureRoot, chunk.Proof, chunk.Index)
	if err != nil {
		logger.Warnf("computing branch hash: %s", err.Error())
		return false
	}

	return branchHash == chunkHash
}

func (t *fetchChunkTask) addLeaf(leaf common.Hash) {
	t.mu.Lock()
	t.liveIn[leaf] = struct{}{}
	t.mu.Unlock()
}

// removeLeaves cancels the task if the removed leaves were the last ones that this task is considered relevant for.
func (t *fetchChunkTask) removeLeaves(leaves []common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, leaf := range leaves {
		delete(t.liveIn, leaf)
	}

	if len(t.liveIn) == 0 {
		t.cancel()
	}
}

func (t *fetchChunkTask) cleanup(reason taskTerminationReason) {
	if t.onTermination != nil {
		t.onTermination(parachaintypes.CandidateHash{Value: t.core.CandidateHash}, reason)
	}
}

func (t *fetchChunkTask) isLive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.liveIn) > 0
}

func (t *fetchChunkTask) cancel() {
	if !t.isCancelled() {
		t.stop <- true
		close(t.stop)
	}
}

func (t *fetchChunkTask) isCancelled() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}
