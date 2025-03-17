// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
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
	return &AvailabilityDistribution{
		subSystemToOverseer: overseerChan,
		net:                 net,
		blockState:          blockState,
	}
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

//nolint:unused
func (ad *AvailabilityDistribution) handleChunkFetchingRequest(
	who peer.ID,
	payload []byte,
) (network.ResponseMessage, error) {
	return nil, nil // TODO: implement #4487
}

//nolint:unused
func (ad *AvailabilityDistribution) handlePoVFetchingRequest(
	who peer.ID,
	payload []byte,
) (network.ResponseMessage, error) {
	return nil, nil // TODO: implement #4488
}
