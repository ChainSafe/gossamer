// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// BlockState is the interface required by GRANDPA into the block state
type BlockState interface {
	GenesisHash() common.Hash
	HasHeader(hash common.Hash) (bool, error)
	GetHeader(hash common.Hash) (*types.Header, error)
	GetHeaderByNumber(num uint) (*types.Header, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error)
	HasFinalisedBlock(round, setID uint64) (bool, error)
	GetFinalisedHeader(round, setID uint64) (*types.Header, error)
	GetFinalisedRound() uint64
	GetFinalisedSetID() uint64
	GetFinalisedHash(round, setID uint64) (common.Hash, error)
	SetFinalisedHash(common.Hash, uint64, uint64) error
	BestBlockHeader() (*types.Header, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	SetJustification(hash common.Hash, data []byte) error
	BestBlockNumber() (blockNumber uint, err error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
	BestBlockHash() common.Hash
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// GrandpaState is the interface required by grandpa into the grandpa state
type GrandpaState interface {
	GetCurrentSetID() (uint64, error)
	GetAuthorities(setID uint64) ([]types.GrandpaVoter, error)
	GetSetIDByBlockNumber(num uint) (uint64, error)
	SetLatestRound(round uint64) error
	GetLatestRound() (uint64, error)
	SetPrevotes(round, setID uint64, data []SignedVote) error
	SetPrecommits(round, setID uint64, data []SignedVote) error
	GetPrevotes(round, setID uint64) ([]SignedVote, error)
	GetPrecommits(round, setID uint64) ([]SignedVote, error)
	NextGrandpaAuthorityChange(bestBlockHash common.Hash, bestBlockNumber uint) (blockHeight uint, err error)
}

// Network is the interface required by GRANDPA for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID network.MessageType,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
}
