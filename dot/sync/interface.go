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

package sync

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
)

// BlockState is the interface for the block state
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeaderVdt() (*types.HeaderVdt, error)
	BestBlockNumber() (*big.Int, error)
	//AddBlock(*types.Block) error
	AddBlockVdt(*types.Block) error
	//CompareAndSetBlockData(bd *types.BlockData) error
	CompareAndSetBlockDataVdt(bd *types.BlockDataVdt) error
	//GetBlockByNumber(*big.Int) (*types.Block, error)
	GetBlockByNumberVdt(*big.Int) (*types.Block, error)
	HasBlockBody(hash common.Hash) (bool, error)
	GetBlockBody(common.Hash) (*types.Body, error)
	//SetHeader(*types.Header) error
	SetHeaderNew(*types.HeaderVdt) error
	//GetHeader(common.Hash) (*types.Header, error)
	GetHeaderVdt(common.Hash) (*types.HeaderVdt, error)
	HasHeader(hash common.Hash) (bool, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetReceipt(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetJustification(common.Hash) ([]byte, error)
	SetJustification(hash common.Hash, data []byte) error
	SetFinalisedHash(hash common.Hash, round, setID uint64) error
	//AddBlockToBlockTree(header *types.Header) error
	AddBlockToBlockTreeVdt(header *types.HeaderVdt) error
	GetHashByNumber(*big.Int) (common.Hash, error)
	//GetBlockByHash(common.Hash) (*types.Block, error)
	GetBlockByHashVdt(common.Hash) (*types.Block, error)
	GetRuntime(*common.Hash) (runtime.Instance, error)
	StoreRuntime(common.Hash, runtime.Instance)
}

// StorageState is the interface for the storage state
type StorageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	LoadCodeHash(*common.Hash) (common.Hash, error)
	SetSyncing(bool)
	Lock()
	Unlock()
}

// CodeSubstitutedState interface to handle storage of code substitute state
type CodeSubstitutedState interface {
	LoadCodeSubstitutedBlockHash() common.Hash
	StoreCodeSubstitutedBlockHash(hash common.Hash) error
}

// TransactionState is the interface for transaction queue methods
type TransactionState interface {
	RemoveExtrinsic(ext types.Extrinsic)
}

// Verifier deals with block verification
type Verifier interface {
	//VerifyBlock(header *types.Header) error
	VerifyBlockVdt(header *types.HeaderVdt) error
}

// FinalityGadget implements justification verification functionality
type FinalityGadget interface {
	VerifyBlockJustification(common.Hash, []byte) error
}

// BlockImportHandler is the interface for the handler of newly imported blocks
type BlockImportHandler interface {
	HandleBlockImport(block *types.Block, state *rtstorage.TrieState) error
}
