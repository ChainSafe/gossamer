package sync

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

var _ Strategy = (*FullSyncStrategy)(nil)

type FullSyncStrategy struct{}

func (*FullSyncStrategy) IsFinished() (bool, error) {
	return false, nil
}

func (*FullSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	fmt.Printf("received block announce: %d", msg.Number)
	return nil
}

func (*FullSyncStrategy) NextActions() ([]*syncTask, error) {
	return nil, nil
}
