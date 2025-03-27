// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-availability-distribution"))

type AvailabilityDistribution struct {
	subSystemToOverseer chan<- any
	net                 Network
	blockState          BlockState
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
) *AvailabilityDistribution {
	ad := &AvailabilityDistribution{
		subSystemToOverseer: overseerChan,
		net:                 net,
		blockState:          blockState,
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
	return nil // TODO: implement #4490 & #4492
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (ad *AvailabilityDistribution) ProcessBlockFinalizedSignal(msg parachaintypes.BlockFinalizedSignal) error {
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
