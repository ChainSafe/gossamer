// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

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
	GetAuthoritiesChangesFromBlock(blockNumber uint) ([]uint, error)
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
