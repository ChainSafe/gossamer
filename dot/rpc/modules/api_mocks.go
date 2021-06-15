package modules

import (
	modulesmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
)

// NewMockStorageAPI creates and return an rpc StorageAPI interface mock
func NewMockStorageAPI() *modulesmocks.MockStorageAPI {
	m := new(modulesmocks.MockStorageAPI)
	m.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	m.On("Entries", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	m.On("GetStorageByBlockHash", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	m.On("RegisterStorageObserver", mock.Anything)
	m.On("UnregisterStorageObserver", mock.Anything)
	m.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	m.On("GetKeysWithPrefix", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	return m
}

// NewMockBlockAPI creates and return an rpc BlockAPI interface mock
func NewMockBlockAPI() *modulesmocks.MockBlockAPI {
	m := new(modulesmocks.MockBlockAPI)
	m.On("GetHeader", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	m.On("BestBlockHash").Return(common.Hash{})
	m.On("GetBlockByHash", mock.AnythingOfType("common.Hash")).Return(nil, nil)
	m.On("GetBlockHash", mock.AnythingOfType("*big.Int")).Return(nil, nil)
	m.On("GetFinalizedHash", mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).Return(common.Hash{}, nil)
	m.On("RegisterImportedChannel", mock.AnythingOfType("chan<- *types.Block")).Return(byte(0), nil)
	m.On("UnregisterImportedChannel", mock.AnythingOfType("uint8"))
	m.On("RegisterFinalizedChannel", mock.AnythingOfType("chan<- *types.FinalisationInfo")).Return(byte(0), nil)
	m.On("UnregisterFinalizedChannel", mock.AnythingOfType("uint8"))
	m.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(make([]byte, 10), nil)
	m.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	m.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(make([]common.Hash, 0), nil)
	return m
}

// NewMockCoreAPI creates and return an rpc CoreAPI interface mock
func NewMockCoreAPI() *modulesmocks.MockCoreAPI {
	m := new(modulesmocks.MockCoreAPI)
	m.On("InsertKey", mock.AnythingOfType("crypto.Keypair"))
	m.On("HasKey", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(false, nil)
	m.On("GetRuntimeVersion", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	m.On("IsBlockProducer").Return(false)
	m.On("HandleSubmittedExtrinsic", mock.AnythingOfType("types.Extrinsic")).Return(nil)
	m.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	return m
}
