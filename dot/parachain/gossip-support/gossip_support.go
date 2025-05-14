// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossipsupport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-gossip-support"))

const (
	LowConnectivityWarnDelay = 600 * time.Second
)

// GossipSupport is the parachain subsystem that is responsible for keeping track of session changes and issuing a
// connection request to all validators in the next, current and a few past sessions if we are a validator
// in these sessions.
type GossipSupport struct {
	subSystemToOverseer chan<- any

	keystore               keystore.Keystore
	lastSessionIndex       *parachaintypes.SessionIndex
	minKnownSession        parachaintypes.SessionIndex
	lastFailure            *time.Time
	lastConnectionRequest  *time.Time
	failureStart           *time.Time
	resolvedAuthorities    map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}
	connectedAuthorities   map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID
	connectedPeers         map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}
	authorityDiscovery     networkbridge.AuthorityDiscoveryService
	finalizedNeededSession *uint32
}

func NewGossipSupport(
	ks keystore.Keystore,
	overseerChan chan<- any,
) *GossipSupport {
	return &GossipSupport{
		subSystemToOverseer: overseerChan,

		keystore:              ks,
		lastSessionIndex:      nil,
		minKnownSession:       math.MaxUint32,
		lastFailure:           nil,
		lastConnectionRequest: nil,
		failureStart:          nil,
		resolvedAuthorities:   make(map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}),
		connectedAuthorities:  make(map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID),
		connectedPeers:        make(map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}),
		// TODO: authorityDiscovery needs to be passed in from param, however; AuthorityDiscoveryService is not
		// implemented yet on overseer level.
		// NetworkBridgeSender, NetworkBridgeReceiver, DisputeDistribution subsyestems are also depending on it.
		authorityDiscovery:     nil,
		finalizedNeededSession: nil,
	}
}

func (*GossipSupport) Name() parachaintypes.SubSystemName {
	return parachaintypes.GossipSupport
}

func (*GossipSupport) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO implement in #4507
	return nil
}

func (*GossipSupport) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// TODO implement #4507
	return nil
}

func (*GossipSupport) Stop() {
	logger.Tracef("Stopping GossipSupport subsystem")
}

// Run starts the GossipSupport subsystem
func (gs *GossipSupport) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	checkConnectivityTicker := time.NewTicker(LowConnectivityWarnDelay)
	for {
		select {
		case <-checkConnectivityTicker.C:
			gs.checkConnectivity()
		case msg := <-overseerToSubsystem:
			err := gs.processMessage(msg)
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

func (gs *GossipSupport) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := gs.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case networkbridgeevents.PeerConnected:
		gs.processPeerConnectedEvent(msg)
	case networkbridgeevents.PeerDisconnected:
		gs.processPeerDisconnectedEvent(msg)
	case parachaintypes.BlockFinalizedSignal:
		return gs.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (*GossipSupport) processPeerConnectedEvent(event networkbridgeevents.PeerConnected) {
	// TODO implement in #4509
}

func (*GossipSupport) processPeerDisconnectedEvent(event networkbridgeevents.PeerDisconnected) {
	// TODO implement in #4509
}

func (*GossipSupport) checkConnectivity() {
	// TODO implement in #4507
}
