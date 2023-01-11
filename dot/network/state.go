// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	GenesisHash() common.Hash
	GetHighestFinalisedHeader() (*types.Header, error)
}

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

// TransactionHandler is the interface used by the transactions sub-protocol
type TransactionHandler interface {
	HandleTransactionMessage(peer.ID, *TransactionMessage) (bool, error)
	TransactionsCount() int
}

// peerSetHandler is the interface used by the connection manager to handle peerset.
type peerSetHandler interface {
	Start(context.Context)
	ReportPeer(peerset.ReputationChange, ...peer.ID)
	peerAdd
	peerRemove
	SortedPeers(idx int) chan peer.IDSlice
	Messages() chan peerset.Message
}

// peerAdd is the interface used by the peerSetHandler to add peers in peerSet.
type peerAdd interface {
	Incoming(int, ...peer.ID)
	AddReservedPeer(int, ...peer.ID)
	AddPeer(int, ...peer.ID)
}

// peerRemove is the interface used by the peerSetHandler to remove peers from peerSet.
type peerRemove interface {
	RemoveReservedPeer(int, ...peer.ID)
}
