package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func handleBitfieldDistribution(_ peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a bitfield distribution message", msg)
	return false, nil
}
