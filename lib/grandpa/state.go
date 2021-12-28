// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math/big"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// BlockState is the interface required by GRANDPA into the block state
type BlockState interface {
	GenesisHash() common.Hash
	HasHeader(hash common.Hash) (bool, error)
	GetHeader(hash common.Hash) (*types.Header, error)
	GetHeaderByNumber(num *big.Int) (*types.Header, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	HighestCommonAncestor(a, b common.Hash) (common.Hash, error)
	HasFinalisedBlock(round, setID uint64) (bool, error)
	GetFinalisedHeader(uint64, uint64) (*types.Header, error)
	SetFinalisedHash(common.Hash, uint64, uint64) error
	BestBlockHeader() (*types.Header, error)
	BestBlockHash() common.Hash
	Leaves() []common.Hash
	BlocktreeAsString() string
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	SetJustification(hash common.Hash, data []byte) error
	HasJustification(hash common.Hash) (bool, error)
	GetJustification(hash common.Hash) ([]byte, error)
	GetHashByNumber(num *big.Int) (common.Hash, error)
	BestBlockNumber() (*big.Int, error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
}

// GrandpaState is the interface required by grandpa into the grandpa state
type GrandpaState interface { //nolint:revive
	GetCurrentSetID() (uint64, error)
	GetAuthorities(setID uint64) ([]types.GrandpaVoter, error)
	GetSetIDByBlockNumber(num *big.Int) (uint64, error)
	SetLatestRound(round uint64) error
	GetLatestRound() (uint64, error)
	SetPrevotes(round, setID uint64, data []SignedVote) error
	SetPrecommits(round, setID uint64, data []SignedVote) error
	GetPrevotes(round, setID uint64) ([]SignedVote, error)
	GetPrecommits(round, setID uint64) ([]SignedVote, error)
}

//go:generate mockery --name DigestHandler --structname DigestHandler --case underscore --keeptree

// DigestHandler is the interface required by GRANDPA for the digest handler
type DigestHandler interface { // TODO: use GrandpaState instead (#1871)
	NextGrandpaAuthorityChange() uint64
}

//go:generate mockery --name Network --structname Network --case underscore --keeptree

// Network is the interface required by GRANDPA for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID byte,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
	) error
}
