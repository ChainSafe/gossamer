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

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	BestBlockStateRoot() (common.Hash, error)
	BestBlock() (*types.Block, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	GetBlockByHash(common.Hash) (*types.Block, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	GetFinalizedHeader(uint64, uint64) (*types.Header, error)
	GetFinalizedHash(uint64, uint64) (common.Hash, error)
	SetFinalizedHash(common.Hash, uint64, uint64) error
	RegisterImportedChannel(ch chan<- *types.Block) (byte, error)
	UnregisterImportedChannel(id byte)
	RegisterFinalizedChannel(ch chan<- *types.Header) (byte, error)
	UnregisterFinalizedChannel(id byte)
	HighestCommonAncestor(a, b common.Hash) (common.Hash, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
}

// StorageState interface for storage state methods
type StorageState interface {
	LoadCode(root *common.Hash) ([]byte, error)
	LoadCodeHash(root *common.Hash) (common.Hash, error)
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
}

// TransactionState is the interface for transaction state methods
type TransactionState interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	AddToPool(vt *transaction.ValidTransaction) common.Hash
	RemoveExtrinsic(ext types.Extrinsic)
	RemoveExtrinsicFromPool(ext types.Extrinsic)
	PendingInPool() []*transaction.ValidTransaction
}

// FinalityGadget is the interface that a finality gadget must implement
type FinalityGadget interface {
	services.Service

	UpdateAuthorities(ad []*types.Authority)
	Authorities() []*types.Authority
}

// BlockProducer is the interface that a block production service must implement
type BlockProducer interface {
	GetBlockChannel() <-chan types.Block
	SetOnDisabled(authorityIndex uint32)
}

// Verifier is the interface for the block verifier
type Verifier interface {
	SetOnDisabled(authorityIndex uint32, block *types.Header) error
}

// Network is the interface for the network service
type Network interface {
	SendMessage(network.NotificationsMessage)
}

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetEpochData(epoch uint64, info *types.EpochData) error
	SetConfigData(epoch uint64, info *types.ConfigData) error
	SetCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)
}
