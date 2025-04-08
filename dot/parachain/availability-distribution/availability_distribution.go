// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-distribution"))

const leafAncestryLenWithinSession = 3

type AvailabilityDistribution struct {
	subSystemToOverseer chan<- any
	net                 Network
	blockState          BlockState
	sessionCache        SessionCache
	fetchTasks          map[parachaintypes.CandidateHash]*fetchChunkTask
	mu                  sync.Mutex
}

var _ parachaintypes.Subsystem = (*AvailabilityDistribution)(nil)

type Network interface {
	RegisterRequestHandler(subprotocolID protocol.ID, handler network.RequestHandler)
}

type BlockState interface {
	GetHeader(hash common.Hash) (*types.Header, error)
	GetRuntime(hash common.Hash) (instance runtime.Instance, err error)
}

// NewAvailabilityDistribution creates a new AvailabilityDistribution subsystem
func NewAvailabilityDistribution(
	overseerChan chan<- any,
	net Network,
	blockState BlockState,
	sessionCache SessionCache,
) *AvailabilityDistribution {
	ad := &AvailabilityDistribution{
		subSystemToOverseer: overseerChan,
		net:                 net,
		blockState:          blockState,
		sessionCache:        sessionCache,
		fetchTasks:          make(map[parachaintypes.CandidateHash]*fetchChunkTask),
	}

	cfProtoID := protocol.ID(messages.ChunkFetchingV2.String())
	net.RegisterRequestHandler(cfProtoID, ad.handleChunkFetchingRequest)

	povProtoID := protocol.ID(messages.PoVFetchingV1.String())
	net.RegisterRequestHandler(povProtoID, ad.handlePoVFetchingRequest)

	return ad
}

// Run starts the AvailabilityDistribution subsystem
func (ad *AvailabilityDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := ad.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (ad *AvailabilityDistribution) Stop() {
	logger.Tracef("Stopping %s subsystem", ad.Name())
	ad.mu.Lock()
	defer ad.mu.Unlock()

	for _, task := range ad.fetchTasks {
		task.cancel()
	}
}

// Name returns the name of the subsystem
func (ad *AvailabilityDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.AvailabilityDistribution
}

// processMessage processes messages sent to the AvailabilityDistribution subsystem
func (ad *AvailabilityDistribution) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := ad.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		return ad.ProcessBlockFinalizedSignal(msg)
	case parachaintypes.AvailabilityDistributionMessageFetchPoV:
		return ad.processAvailabilityDistributionMessageFetchPoV(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (ad *AvailabilityDistribution) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal,
) error {
	leaf := signal.Activated.Hash
	fmtErr := func(op string, err error) error {
		return fmt.Errorf("%s for block %d (%s): %w", op, signal.Activated.Number, leaf.String(), err)
	}

	leafSessionIndex, ancestorsInSession, err := ad.getBlockAncestorsInSameSession(leaf, leafAncestryLenWithinSession)
	if err != nil {
		return fmtErr("getting block ancestors in same session", err)
	}

	rt, err := ad.blockState.GetRuntime(leaf)
	if err != nil {
		return fmtErr("instantiating runtime", err)
	}

	coreStates, err := rt.ParachainHostAvailabilityCores()
	if err != nil {
		return fmtErr("getting availability cores", err)
	}

	// Start or update fetch tasks for the leaf and its ancestors in the same session. All started tasks are marked as
	// live in the leaf. This way they will be cancelled on subsequent updates if their leaf gets deactivated without
	// having to track them explicitly.
	for _, hash := range append([]common.Hash{leaf}, ancestorsInSession...) {
		cores, err := ad.getOccupiedCoresFor(hash, coreStates)
		if err != nil {
			return fmtErr("getting occupied cores", err)
		}

		if err := ad.addCores(rt, leaf, leafSessionIndex, cores); err != nil {
			return fmtErr("starting/updating fetch tasks for occupied cores", err)
		}
	}

	ad.mu.Lock()
	defer ad.mu.Unlock()

	for hash, task := range ad.fetchTasks {
		task.removeLeaves(signal.Deactivated)
		if !task.isLive() {
			delete(ad.fetchTasks, hash)
		}
	}

	return nil
}

func (ad *AvailabilityDistribution) addCores(
	rt runtime.Instance,
	leaf common.Hash,
	leafSessionIndex parachaintypes.SessionIndex,
	cores []*parachaintypes.OccupiedCore,
) error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	sessionInfo, err := ad.sessionCache.GetSessionInfo(leafSessionIndex, rt)
	if err != nil {
		return err
	}

	for coreIndex, core := range cores {
		// Don't run tasks for our backing group:
		if sessionInfo.OurGroup != nil && *sessionInfo.OurGroup == core.GroupResponsible {
			continue
		}

		candidateHash := parachaintypes.CandidateHash{Value: core.CandidateHash}
		task, ok := ad.fetchTasks[candidateHash]
		if ok {
			task.addLeaf(leaf)
			continue
		}

		chunkIndex, err := availabilityChunkIndex(
			sessionInfo.NodeFeatures,
			sessionInfo.NumberOfValidators(),
			coreIndex,
			sessionInfo.OurIndex,
		)
		if err != nil {
			return err
		}

		task = newFetchChunkTask(
			leaf,
			chunkIndex,
			sessionInfo,
			core,
			ad.subSystemToOverseer,
			ad.handleFetchTaskTermination,
		)

		ad.fetchTasks[candidateHash] = task
		go task.run()
	}
	return nil
}

func (ad *AvailabilityDistribution) getOccupiedCoresFor(
	relayParent common.Hash,
	coreStates []parachaintypes.CoreState,
) ([]*parachaintypes.OccupiedCore, error) {
	cores := make([]*parachaintypes.OccupiedCore, 0)

	for _, coreState := range coreStates {
		v, err := coreState.Value()
		if err != nil {
			return nil, err
		}

		oc, ok := v.(parachaintypes.OccupiedCore)
		if !ok || oc.CandidateDescriptor.RelayParent != relayParent {
			continue
		}

		cores = append(cores, &oc)
	}
	return cores, nil
}

func (ad *AvailabilityDistribution) handleFetchTaskTermination(
	candidateHash parachaintypes.CandidateHash,
	reason taskTerminationReason,
) {
	ad.mu.Lock()
	delete(ad.fetchTasks, candidateHash)
	ad.mu.Unlock()

	success, ok := reason.(taskSucceeded)
	if ok && len(success.badValidators) > 0 {
		err := ad.sessionCache.ReportBadValidators(success.sessionIndex, success.groupIndex, success.badValidators)
		if err != nil {
			logger.Errorf("reporting bad validators for session %d: %s", success.sessionIndex, err)
		}
	}
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (ad *AvailabilityDistribution) ProcessBlockFinalizedSignal(_ parachaintypes.BlockFinalizedSignal) error {
	return nil // nothing to do
}

func (ad *AvailabilityDistribution) processAvailabilityDistributionMessageFetchPoV(
	msg parachaintypes.AvailabilityDistributionMessageFetchPoV,
) error {
	return nil // TODO: implement #4489
}

func (ad *AvailabilityDistribution) handleChunkFetchingRequest(
	_ peer.ID,
	payload []byte,
) (network.ResponseMessage, error) {
	request := &messages.ChunkFetchingRequest{}

	err := request.Decode(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding chunk fetching request: %w", err)
	}

	query := availabilitystore.QueryChunk{
		CandidateHash:  request.CandidateHash,
		ValidatorIndex: request.Index,
		Sender:         make(chan availabilitystore.ErasureChunk),
	}

	ad.subSystemToOverseer <- query
	response := &messages.ChunkFetchingResponse{}

	// Conceptually, this is a "one shot" channel, so ideally availability store would always close the channel and not
	// send anything in case the chunk is not found.
	chunk, ok := <-query.Sender
	if !ok || chunk.Chunk == nil {
		err = response.SetValue(messages.NoSuchChunk{})
		if err != nil {
			return nil, fmt.Errorf("setting chunk response value: %w", err)
		}
	} else {
		err = response.SetValue(messages.ChunkResponse{
			Chunk: chunk.Chunk,
			Index: chunk.Index,
			// Proof: chunk.Proof,  // FIXME see #4597
		})
		if err != nil {
			return nil, fmt.Errorf("setting chunk response value: %w", err)
		}
	}

	return response, nil
}

func (ad *AvailabilityDistribution) handlePoVFetchingRequest(
	_ peer.ID,
	payload []byte,
) (network.ResponseMessage, error) {
	request := &messages.PoVFetchingRequest{}

	err := request.Decode(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding chunk fetching request: %w", err)
	}

	query := availabilitystore.QueryAvailableData{
		CandidateHash: request.CandidateHash,
		Sender:        make(chan availabilitystore.AvailableData),
	}

	ad.subSystemToOverseer <- query
	response := &messages.PoVFetchingResponse{}

	availableData := <-query.Sender
	if availableData.PoV.BlockData == nil {
		err = response.SetValue(parachaintypes.NoSuchPoV{})
		if err != nil {
			return nil, fmt.Errorf("setting PoV response value: %w", err)
		}
	} else {
		err = response.SetValue(availableData.PoV)
		if err != nil {
			return nil, fmt.Errorf("setting PoV response value: %w", err)
		}
	}

	return response, nil
}

func (ad *AvailabilityDistribution) getBlockAncestorsInSameSession(
	leafHash common.Hash,
	limit int,
) (parachaintypes.SessionIndex, []common.Hash, error) {
	leafHeader, err := ad.blockState.GetHeader(leafHash)
	if err != nil {
		return 0, nil, fmt.Errorf("getting header for activated leaf %v: %w", leafHash, err)
	}

	rt, err := ad.blockState.GetRuntime(leafHash)
	if err != nil {
		return 0, nil, fmt.Errorf("instantiating runtime for block %v: %w", leafHash, err)
	}
	defer rt.Stop()

	headSessionIndex, err := ad.sessionCache.GetSessionIndexForChild(leafHeader.ParentHash, rt)
	if err != nil {
		return 0, nil, fmt.Errorf("getting session index for activated leaf %v: %w", leafHash, err)
	}

	currentHash := leafHeader.ParentHash
	ancestors := make([]common.Hash, 0, limit)

	for i := 0; i < limit; i++ {
		header, err := ad.blockState.GetHeader(currentHash)
		if err != nil {
			return 0, nil, fmt.Errorf("getting header for leaf ancestor %v: %w", currentHash, err)
		}

		// stop at genesis
		if header.Number == 0 {
			return headSessionIndex, ancestors, nil
		}

		sessionIndex, err := ad.sessionCache.GetSessionIndexForChild(header.ParentHash, rt)
		if err != nil {
			return 0, nil, fmt.Errorf("getting session index for leaf ancestor %v: %w", currentHash, err)
		}

		if sessionIndex != headSessionIndex {
			return headSessionIndex, ancestors, nil
		}

		ancestors = append(ancestors, currentHash)
		currentHash = header.ParentHash
	}

	return headSessionIndex, ancestors, nil
}

// Compute the per-validator availability chunk index.
// WARNING: THIS FUNCTION IS CRITICAL TO PARACHAIN CONSENSUS.
// Any modification to the output of the function needs to be coordinated via the runtime.
// It's best to use minimal/no external dependencies.
func availabilityChunkIndex(
	nodeFeatures parachaintypes.BitVec,
	nValidators uint,
	coreIndex int,
	validatorIndex parachaintypes.ValidatorIndex,
) (uint32, error) {
	avChunkMapping, err := nodeFeatures.Get(uint(parachaintypes.AvailabilityChunkMapping))
	if err != nil {
		return 0, err
	}

	if avChunkMapping {
		systematicThreshold, err := erasure.SystematicRecoveryThreshold(nValidators)
		if err != nil {
			return 0, err
		}

		coreStartPos := uint32(coreIndex) * systematicThreshold
		chunkIndex := (coreStartPos + uint32(validatorIndex)) % uint32(nValidators)
		return chunkIndex, nil
	}

	return uint32(validatorIndex), nil
}
