// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfielddistribution

import (
	"context"
	"fmt"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-distribution"))

var _ parachaintypes.Subsystem = (*BitfieldDistribution)(nil)

type BitfieldDistribution struct {
	subSystemToOverseer chan<- any
}

func NewBitfieldDistribution(overseerChan chan<- any) *BitfieldDistribution {
	return &BitfieldDistribution{
		subSystemToOverseer: overseerChan,
	}
}

func (b *BitfieldDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := b.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

// processMessage processes messages sent to the BitfieldDistribution subsystem
func (b *BitfieldDistribution) processMessage(msg any) error {
	switch msg := msg.(type) {
	case validationprotocol.BitfieldDistributionMessage:
		err := b.ProcessBitfieldDistributionMessageSignal(msg)
		if err != nil {
			return fmt.Errorf("processing bitfield distribution message signal: %w", err)
		}
	case networkbridgeevents.PeerConnected:
		err := b.ProcessPeerConnectedSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer connected signal: %w", err)
		}
	case networkbridgeevents.PeerDisconnected:
		err := b.ProcessPeerDisconnectedSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer disconnected signal: %w", err)
		}
	case networkbridgeevents.NewGossipTopology:
		err := b.ProcessNewGossipTopologySignal(msg)
		if err != nil {
			return fmt.Errorf("processing new gossip topology signal: %w", err)
		}
	case networkbridgeevents.PeerViewChange:
		err := b.ProcessPeerViewChangeSignal(msg)
		if err != nil {
			return fmt.Errorf("processing peer view change signal: %w", err)
		}
	case networkbridgeevents.OurViewChange:
		err := b.ProcessOurViewChangeSignal(msg)
		if err != nil {
			return fmt.Errorf("processing our view change signal: %w", err)
		}
	case networkbridgeevents.UpdatedAuthorityIDs:
		err := b.ProcessUpdatedAuthorityIDsSignal(msg)
		if err != nil {
			return fmt.Errorf("processing updated authority IDs signal: %w", err)
		}
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := b.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		return b.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (b *BitfieldDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.BitfieldDistribution
}

func (b *BitfieldDistribution) ProcessPeerConnectedSignal(signal networkbridgeevents.PeerConnected) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessPeerDisconnectedSignal(signal networkbridgeevents.PeerDisconnected) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessNewGossipTopologySignal(signal networkbridgeevents.NewGossipTopology) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessPeerViewChangeSignal(signal networkbridgeevents.PeerViewChange) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessOurViewChangeSignal(signal networkbridgeevents.OurViewChange) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessUpdatedAuthorityIDsSignal(signal networkbridgeevents.UpdatedAuthorityIDs) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessBitfieldDistributionMessageSignal(signal validationprotocol.BitfieldDistributionMessage) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitfieldDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (b *BitfieldDistribution) Stop() {
	logger.Infof("Stopping BitfieldDistribution subsystem")
}
