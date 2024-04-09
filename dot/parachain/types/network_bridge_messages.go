package parachaintypes

import "github.com/libp2p/go-libp2p/core/peer"

type NewGossipTopology struct {
}

// UpdateAuthorityIDs is used to inform the distribution subsystems about `AuthorityDiscoveryId` key rotations.
type UpdateAuthorityIDs struct {
	// The `PeerId` of the peer that updated its `AuthorityDiscoveryId`s.
	peerID peer.ID
	authorityIDs
}
