// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package network

import (
	"math/big"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

//go:generate mockery --name BlockState --structname MockBlockState --case underscore --inpackage

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	GenesisHash() common.Hash
	HasBlockBody(common.Hash) (bool, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetHashByNumber(num *big.Int) (common.Hash, error)
}

//go:generate mockery --name Syncer --structname MockSyncer --case underscore --inpackage

// Syncer is implemented by the syncing service
type Syncer interface {
	HandleBlockAnnounceHandshake(from peer.ID, msg *BlockAnnounceHandshake) error

	// HandleBlockAnnounce is called upon receipt of a BlockAnnounceMessage to process it.
	// If a request needs to be sent to the peer to retrieve the full block, this function will return it.
	HandleBlockAnnounce(from peer.ID, msg *BlockAnnounceMessage) error

	// IsSynced exposes the internal synced state
	IsSynced() bool

	// CreateBlockResponse is called upon receipt of a BlockRequestMessage to create the response
	CreateBlockResponse(*BlockRequestMessage) (*BlockResponseMessage, error)
}

//go:generate mockery --name TransactionHandler --structname MockTransactionHandler --case underscore --inpackage

// TransactionHandler is the interface used by the transactions sub-protocol
type TransactionHandler interface {
	HandleTransactionMessage(peer.ID, *TransactionMessage) (bool, error)
	TransactionsCount() int
}

// PeerSetHandler is the interface used by the connection manager to handle peerset.
type PeerSetHandler interface {
	Start()
	Stop()
	ReportPeer(peerset.ReputationChange, ...peer.ID)
	PeerAdd
	PeerRemove
	Peer
}

// PeerAdd is the interface used by the PeerSetHandler to add peers in peerSet.
type PeerAdd interface {
	Incoming(int, ...peer.ID)
	AddReservedPeer(int, ...peer.ID)
	AddPeer(int, ...peer.ID)
	SetReservedPeer(int, ...peer.ID)
}

// PeerRemove is the interface used by the PeerSetHandler to remove peers from peerSet.
type PeerRemove interface {
	DisconnectPeer(int, ...peer.ID)
	RemoveReservedPeer(int, ...peer.ID)
	RemovePeer(int, ...peer.ID)
}

// Peer is the interface used by the PeerSetHandler to get the peer data from peerSet.
type Peer interface {
	PeerReputation(peer.ID) (peerset.Reputation, error)
	SortedPeers(idx int) chan peer.IDSlice
	Messages() chan interface{}
}
