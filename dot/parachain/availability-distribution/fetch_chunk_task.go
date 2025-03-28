package availabilitydistribution

import (
	"fmt"
	"sync"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type taskTerminationReason uint

const (
	taskSucceeded taskTerminationReason = iota
	taskFailed
	taskCancelled
)

type taskTerminationHandler func(
	candidateHash parachaintypes.CandidateHash,
	reason taskTerminationReason,
	badValidators []parachaintypes.AuthorityDiscoveryID,
)

type fetchChunkTask struct {
	chunkIndex          uint32
	ourIndex            parachaintypes.ValidatorIndex
	core                *parachaintypes.OccupiedCore
	group               []parachaintypes.AuthorityDiscoveryID
	subsystemToOverseer chan<- any

	// Set of relay chain block hashes for which the candidate associated with the given core is pending availability.
	//
	// In other words, for which relay chain parents this candidate is considered live.
	// This is updated on every `ActiveLeavesUpdate` and enables us to know when we can safely
	// stop keeping track of that candidate/chunk.
	liveIn map[common.Hash]struct{}
	mu     sync.Mutex

	// List of validators that did not have the requested chunk or sent invalid data.
	badValidators []parachaintypes.AuthorityDiscoveryID

	onTermination taskTerminationHandler
	stop          chan bool
}

func newFetchChunkTask(
	leaf common.Hash,
	chunkIndex uint32,
	ourIndex parachaintypes.ValidatorIndex,
	core *parachaintypes.OccupiedCore,
	group []parachaintypes.AuthorityDiscoveryID,
	subsystemToOverseer chan<- any,
	onTermination taskTerminationHandler,
) *fetchChunkTask {
	return &fetchChunkTask{
		liveIn:              map[common.Hash]struct{}{leaf: {}},
		chunkIndex:          chunkIndex,
		ourIndex:            ourIndex,
		core:                core,
		group:               group,
		subsystemToOverseer: subsystemToOverseer,
		onTermination:       onTermination,
		badValidators:       make([]parachaintypes.AuthorityDiscoveryID, 0),
		stop:                make(chan bool, 1),
	}
}

func (t *fetchChunkTask) run() {
	// Try validators in reverse order
	for i := len(t.group) - 1; i >= 0; i-- {
		if t.isCancelled() {
			t.cleanup(taskCancelled)
			return
		}

		authorityID := t.group[i]

		request := networkbridgemessages.NewOutgoingRequest(
			authorityID,
			&networkbridgemessages.ChunkFetchingRequest{
				CandidateHash: parachaintypes.CandidateHash{Value: t.core.CandidateHash},
				Index:         t.ourIndex,
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
			t.cleanup(taskCancelled)
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
		t.cleanup(taskSucceeded)
		return
	}

	// None of the requested validators were able to provide the chunk.
	t.cleanup(taskFailed)
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
			// Proof: chunkResponse.Proof, // FIXME see #4597
		}, nil
	default:
		return availabilitystore.ErasureChunk{}, fmt.Errorf("chunk not found")
	}
}

func (t *fetchChunkTask) validateChunk(chunk availabilitystore.ErasureChunk) bool {
	if chunk.Index != t.chunkIndex { //nolint:gosimple
		return false
	}
	return true // TODO check the proof against erasure root (blocked by #4597)
}

//nolint:unused
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
		t.onTermination(parachaintypes.CandidateHash{Value: t.core.CandidateHash}, reason, t.badValidators)
	}
}

func (t *fetchChunkTask) cancel() {
	t.stop <- true
	close(t.stop)
}

func (t *fetchChunkTask) isCancelled() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}
