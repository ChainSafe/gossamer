// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

import (
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// StorageAPI is the interface for the storage state
type StorageAPI interface {
	RegisterStorageObserver(observer state.Observer)
	UnregisterStorageObserver(observer state.Observer)
}

// BlockAPI is the interface for the block state
type BlockAPI interface {
	GetJustification(hash common.Hash) ([]byte, error)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error)
}

// TransactionStateAPI is the interface to get and free status notifier channels
type TransactionStateAPI interface {
	GetStatusNotifierChannel(ext types.Extrinsic) chan transaction.Status
	FreeStatusNotifierChannel(ch chan transaction.Status)
}

// CoreAPI is the interface for the core methods
type CoreAPI interface {
	GetRuntimeVersion(bhash *common.Hash) (runtime.Version, error)
	HandleSubmittedExtrinsic(types.Extrinsic) error
}
