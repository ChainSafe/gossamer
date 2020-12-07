package network

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
)

type mockSyncer struct {
	highestSeen *big.Int
	synced      bool
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{
		highestSeen: big.NewInt(0),
		synced:      false,
	}
}

func (s *mockSyncer) CreateBlockResponse(msg *BlockRequestMessage) (*BlockResponseMessage, error) {
	return nil, nil
}

func (s *mockSyncer) HandleBlockResponse(msg *BlockResponseMessage) *BlockRequestMessage {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *BlockAnnounceMessage) *BlockRequestMessage {
	if msg.Number.Cmp(s.highestSeen) > 0 {
		s.highestSeen = msg.Number
	}

	startBlock, _ := variadic.NewUint64OrHash(1)
	return &BlockRequestMessage{
		ID:            99,
		StartingBlock: startBlock,
		Max:           optional.NewUint32(false, 0),
	}
}

func (s *mockSyncer) IsSynced() bool {
	return s.synced
}

func (s *mockSyncer) SetSyncedState(newState bool) {
	s.synced = newState
}
