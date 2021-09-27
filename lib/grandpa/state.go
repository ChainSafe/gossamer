// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package grandpa

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
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
	RegisterFinalizedChannel(ch chan<- *types.FinalisationInfo) (byte, error)
	UnregisterFinalisedChannel(id byte)
	SetJustification(hash common.Hash, data []byte) error
	HasJustification(hash common.Hash) (bool, error)
	GetJustification(hash common.Hash) ([]byte, error)
	GetHashByNumber(num *big.Int) (common.Hash, error)
	BestBlockNumber() (*big.Int, error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
}

// GrandpaState is the interface required by grandpa into the grandpa state
type GrandpaState interface { //nolint
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

// DigestHandler is the interface required by GRANDPA for the digest handler
type DigestHandler interface { // TODO: remove, use GrandpaState
	NextGrandpaAuthorityChange() uint64
}

// Network is the interface required by GRANDPA for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg NotificationsMessage) error
	SendBlockReqestByHash(hash common.Hash)
	SendJustificationRequest(to peer.ID, num uint32)
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
