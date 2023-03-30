package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Service struct {
	blockState     interface{}
	chainSync      interface{}
	chainProcessor interface{}
	network        interface{}

	warpSync *WarpSync
}

// Start begins the chainSync and chainProcessor modules. It begins syncing in bootstrap mode
func (s *Service) Start() error {
	go s.warpSync.sync()
	return nil
}

// Stop stops the chainSync and chainProcessor modules
func (s *Service) Stop() error {
	return nil
}

// HandleBlockAnnounceHandshake notifies the `chainSync` module that
// we have received a BlockAnnounceHandshake from the given peer.
func (s *Service) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	return nil
}

// HandleBlockAnnounce notifies the `chainSync` module that we have received a block announcement from the given peer.
func (s *Service) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	return nil
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return false
}

// HighestBlock gets the highest known block number
func (s *Service) HighestBlock() uint {
	return 0
}
