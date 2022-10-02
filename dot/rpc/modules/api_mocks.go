// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"testing"

	modulesmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/mock"
)

// NewMockeryStorageAPI creates and return an rpc StorageAPI interface mock
func NewMockeryStorageAPI(t *testing.T) *modulesmocks.StorageAPI {
	m := modulesmocks.NewStorageAPI(t)
	m.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil).Maybe()
	m.On("GetStorageFromChild", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("[]uint8")).Return(nil, nil).Maybe()
	m.On("Entries", mock.AnythingOfType("*common.Hash")).Return(nil, nil).Maybe()
	m.On("GetStorageByBlockHash", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).
		Return(nil, nil).Maybe()
	m.On("RegisterStorageObserver", mock.Anything).Maybe()
	m.On("UnregisterStorageObserver", mock.Anything).Maybe()
	m.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil).Maybe()
	m.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil).Maybe()
	return m
}

// NewMockeryBlockAPI creates and return an rpc BlockAPI interface mock
func NewMockeryBlockAPI(t *testing.T) *modulesmocks.BlockAPI {
	m := modulesmocks.NewBlockAPI(t)
	m.On("GetHeader", mock.AnythingOfType("common.Hash")).Return(nil, nil).Maybe()
	m.On("BestBlockHash").Return(common.Hash{}).Maybe()
	m.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(nil, nil).Maybe()
	m.On("GetHashByNumber", mock.AnythingOfType("uint")).Return(nil, nil).Maybe()
	m.On("GetFinalisedHash", mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).
		Return(common.Hash{}, nil).Maybe()
	m.On("GetHighestFinalisedHash").Return(common.Hash{}, nil).Maybe()
	m.On("GetImportedBlockNotifierChannel").Return(make(chan *types.Block, 5)).Maybe()
	m.On("FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block")).Maybe()
	m.On("GetFinalisedNotifierChannel").Return(make(chan *types.FinalisationInfo, 5)).Maybe()
	m.On("FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo")).Maybe()
	m.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(make([]byte, 10), nil).Maybe()
	m.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil).Maybe()
	m.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).
		Return(make([]common.Hash, 0), nil).Maybe()
	m.On("RegisterRuntimeUpdatedChannel", mock.AnythingOfType("chan<- runtime.Version")).
		Return(uint32(0), nil).Maybe()

	return m
}

// NewMockCoreAPI creates and return an rpc CoreAPI interface mock
func NewMockCoreAPI(t *testing.T) *modulesmocks.CoreAPI {
	m := modulesmocks.NewCoreAPI(t)
	m.On("InsertKey", mock.AnythingOfType("crypto.Keypair"), mock.AnythingOfType("string")).Return(nil).Maybe()
	m.On("HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(false, nil).Maybe()
	m.On("GetRuntimeVersion", mock.AnythingOfType("*common.Hash")).
		Return(runtime.Version{SpecName: []byte(`mock-spec`)}, nil).Maybe()
	m.On("IsBlockProducer").Return(false).Maybe()
	m.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(nil).Maybe()
	m.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(nil, nil).Maybe()
	return m
}
