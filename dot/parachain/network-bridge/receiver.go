// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "network-bridge"))

type NetworkBridgeReceiver struct {
	OverseerToSubSystem <-chan any
}

func (nbr *NetworkBridgeReceiver) Run(ctx context.Context, overseerToSubSystem chan any) {
	// TODO: handle incoming messages from the network
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := nbr.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (nbr *NetworkBridgeReceiver) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeReceiver
}

func (nbr *NetworkBridgeReceiver) ProcessActiveLeavesUpdateSignal(
	signal parachaintypes.ActiveLeavesUpdateSignal) error {
	return nil
}

func (nbr *NetworkBridgeReceiver) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (nbr *NetworkBridgeReceiver) Stop() {}

func (nbr *NetworkBridgeReceiver) processMessage(msg any) error { //nolint
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case NewGossipTopology:
		// TODO
		fmt.Println(msg)
	case UpdateAuthorityIDs:
		// TODO
	}

	return nil
}

// Inform the distribution subsystems about the new
// gossip network topology formed.
//
// The only reason to have this here, is the availability of the
// authority discovery service, otherwise, the `GossipSupport`
// subsystem would make more sense.
type NewGossipTopology struct {
	// The session info this gossip topology is concerned with.
	session parachaintypes.SessionIndex //nolint
	// Our validator index in the session, if any.
	localIndex *parachaintypes.ValidatorIndex //nolint
	//  The canonical shuffling of validators for the session.
	canonicalShuffling []canonicalShuffling //nolint
	// The reverse mapping of `canonical_shuffling`: from validator index
	// to the index in `canonical_shuffling`
	shuffledIndices uint8 //nolint
}

type canonicalShuffling struct { //nolint
	authorityDiscoveryID parachaintypes.AuthorityDiscoveryID
	validatorIndex       parachaintypes.ValidatorIndex
}

// UpdateAuthorityIDs is used to inform the distribution subsystems about `AuthorityDiscoveryId` key rotations.
type UpdateAuthorityIDs struct {
	// The `PeerId` of the peer that updated its `AuthorityDiscoveryId`s.
	peerID peer.ID //nolint
	// The updated authority discovery keys of the peer.
	authorityIDs []parachaintypes.AuthorityDiscoveryID //nolint
}
