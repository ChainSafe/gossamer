// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	modulesmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/stretchr/testify/mock"
)

// NewMockeryStorageAPI creates and return an rpc StorageAPI interface mock
func NewMockeryStorageAPI() *modulesmocks.StorageAPI {
	m := new(modulesmocks.StorageAPI)
	m.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	m.On("GetStorageFromChild", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("[]uint8")).Return(nil, nil)
	m.On("Entries", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	m.On("GetStorageByBlockHash", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	m.On("RegisterStorageObserver", mock.Anything)
	m.On("UnregisterStorageObserver", mock.Anything)
	m.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	m.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	return m
}

// NewMockeryBlockAPI creates and return an rpc BlockAPI interface mock
func NewMockeryBlockAPI() *modulesmocks.BlockAPI {
	m := new(modulesmocks.BlockAPI)
	m.On("GetHeader", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	m.On("BestBlockHash").Return(common.Hash{})
	m.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	m.On("GetHashByNumber", mock.AnythingOfType("uint")).Return(nil, nil)
	m.On("GetFinalisedHash", mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).Return(common.Hash{}, nil)
	m.On("GetHighestFinalisedHash").Return(common.Hash{}, nil)
	m.On("GetImportedBlockNotifierChannel").Return(make(chan *types.Block, 5))
	m.On("FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block"))
	m.On("GetFinalisedNotifierChannel").Return(make(chan *types.FinalisationInfo, 5))
	m.On("FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))
	m.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(make([]byte, 10), nil)
	m.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	m.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).
		Return(make([]common.Hash, 0), nil)
	m.On("RegisterRuntimeUpdatedChannel", mock.AnythingOfType("chan<- runtime.Version")).Return(uint32(0), nil)

	return m
}

// NewMockTransactionStateAPI creates and return an rpc TransactionStateAPI interface mock
func NewMockTransactionStateAPI() *modulesmocks.TransactionStateAPI {
	m := new(modulesmocks.TransactionStateAPI)
	m.On("FreeStatusNotifierChannel", mock.AnythingOfType("chan transaction.Status"))
	m.On("GetStatusNotifierChannel", mock.AnythingOfType("types.Extrinsic")).Return(make(chan transaction.Status))
	m.On("AddToPool", mock.AnythingOfType("transaction.ValidTransaction")).Return(common.Hash{})
	return m
}

// NewMockCoreAPI creates and return an rpc CoreAPI interface mock
func NewMockCoreAPI() *modulesmocks.CoreAPI {
	m := new(modulesmocks.CoreAPI)
	m.On("InsertKey", mock.AnythingOfType("crypto.Keypair"), mock.AnythingOfType("string")).Return(nil)
	m.On("HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(false, nil)
	m.On("GetRuntimeVersion", mock.AnythingOfType("*common.Hash")).
		Return(runtime.VersionData{SpecName: []byte(`mock-spec`)}, nil)
	m.On("IsBlockProducer").Return(false)
	m.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(nil)
	m.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	return m
}
