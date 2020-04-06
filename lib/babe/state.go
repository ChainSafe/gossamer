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

package babe

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	BestBlock() (*types.Block, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	AddBlock(*types.Block) error
	AddBlockWithArrivalTime(*types.Block, uint64) error
	GetHeader(common.Hash) (*types.Header, error)
	GetBlockByNumber(*big.Int) (*types.Block, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetArrivalTime(common.Hash) (uint64, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	HighestBlockHash() common.Hash
	HighestBlockNumber() *big.Int
}

// StorageState interface for storage state methods
type StorageState interface {
	StorageRoot() (common.Hash, error)
	SetStorage([]byte, []byte) error
}

// TransactionQueue is the interface for transaction queue methods
type TransactionQueue interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	Pop() *transaction.ValidTransaction
	Peek() *transaction.ValidTransaction
}
