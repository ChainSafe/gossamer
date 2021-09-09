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

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/libp2p/go-libp2p-core/peer"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	GenesisHash() common.Hash
	HasBlockBody(common.Hash) (bool, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetHashByNumber(num *big.Int) (common.Hash, error)
}

// Syncer is implemented by the syncing service
type Syncer interface {
	HandleBlockAnnounceHandshake(from peer.ID, msg *BlockAnnounceHandshake) error

	// HandleBlockAnnounce is called upon receipt of a BlockAnnounceMessage to process it.
	// If a request needs to be sent to the peer to retrieve the full block, this function will return it.
	HandleBlockAnnounce(from peer.ID, msg *BlockAnnounceMessage) error

	// IsSynced exposes the internal synced state // TODO: use syncQueue for this
	IsSynced() bool

	// CreateBlockResponse is called upon receipt of a BlockRequestMessage to create the response
	CreateBlockResponse(*BlockRequestMessage) (*BlockResponseMessage, error)
}

// TransactionHandler is the interface used by the transactions sub-protocol
type TransactionHandler interface {
	HandleTransactionMessage(*TransactionMessage) (bool, error)
	TransactionsCount() int
}
