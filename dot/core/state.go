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

package core

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	BestBlock() (*types.Block, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	AddBlockWithArrivalTime(*types.Block, uint64) error
	CompareAndSetBlockData(bd *types.BlockData) error
	GetBlockBody(common.Hash) (*types.Body, error)
	SetHeader(*types.Header) error
	GetHeader(common.Hash) (*types.Header, error)
	HasHeader(hash common.Hash) (bool, error)
	GetReceipt(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetJustification(common.Hash) ([]byte, error)
	GetBlockByNumber(*big.Int) (*types.Block, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetArrivalTime(common.Hash) (uint64, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	HighestBlockHash() common.Hash
	HighestBlockNumber() *big.Int
	GetFinalizedHeader() (*types.Header, error)
}

// StorageState interface for storage state methods
type StorageState interface {
	StorageRoot() (common.Hash, error)
	SetStorage([]byte, []byte) error
	GetStorage([]byte) ([]byte, error)
	StoreInDB() error
	LoadCode() ([]byte, error)
	LoadCodeHash() (common.Hash, error)
	SetStorageChild([]byte, *trie.Trie) error
	SetStorageIntoChild([]byte, []byte, []byte) error
	GetStorageFromChild([]byte, []byte) ([]byte, error)
	ClearStorage([]byte) error
	Entries() map[string][]byte
	SetBalance(key [32]byte, balance uint64) error
	GetBalance(key [32]byte) (uint64, error)
}

// TransactionQueue is the interface for transaction queue methods
type TransactionQueue interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	Pop() *transaction.ValidTransaction
	Peek() *transaction.ValidTransaction
	RemoveExtrinsic(ext types.Extrinsic)
}
