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

func (nbr *NetworkBridgeReceiver) Run(ctx context.Context, OverseerToSubSystem chan any,
	SubSystemToOverseer chan any) {

	// TODO: handle incoming messages from the network
	for msg := range nbr.OverseerToSubSystem {
		err := nbr.processMessage(msg)
		if err != nil {
			logger.Errorf("processing overseer message: %w", err)
		}
	}
}

func (nbr *NetworkBridgeReceiver) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeReceiver
}

func (nbr *NetworkBridgeReceiver) ProcessActiveLeavesUpdateSignal() {}

func (nbr *NetworkBridgeReceiver) ProcessBlockFinalizedSignal() {}

func (nbr *NetworkBridgeReceiver) Stop() {}

func (nbr *NetworkBridgeReceiver) processMessage(msg any) error {
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
	session parachaintypes.SessionIndex
	// Our validator index in the session, if any.
	localIndex *parachaintypes.ValidatorIndex
	//  The canonical shuffling of validators for the session.
	canonicalShuffling []canonicalShuffling
	// The reverse mapping of `canonical_shuffling`: from validator index
	// to the index in `canonical_shuffling`
	shuffledIndices uint8
}

type canonicalShuffling struct {
	authorityDiscoveryID parachaintypes.AuthorityDiscoveryID
	validatorIndex       parachaintypes.ValidatorIndex
}

// UpdateAuthorityIDs is used to inform the distribution subsystems about `AuthorityDiscoveryId` key rotations.
type UpdateAuthorityIDs struct {
	// The `PeerId` of the peer that updated its `AuthorityDiscoveryId`s.
	peerID peer.ID
	// The updated authority discovery keys of the peer.
	authorityIDs []parachaintypes.AuthorityDiscoveryID
}
