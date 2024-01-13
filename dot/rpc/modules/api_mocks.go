// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	modulesmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"go.uber.org/mock/gomock"
)

// NewMockAnyStorageAPI creates and return an rpc StorageAPI interface mock
func NewMockAnyStorageAPI(ctrl *gomock.Controller) *modulesmocks.MockStorageAPI {
	m := modulesmocks.NewMockStorageAPI(ctrl)
	m.EXPECT().GetStorage(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().GetStorageFromChild(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()
	m.EXPECT().Entries(gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().GetStorageByBlockHash(gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()
	m.EXPECT().RegisterStorageObserver(gomock.Any()).AnyTimes()
	m.EXPECT().UnregisterStorageObserver(gomock.Any()).AnyTimes()
	m.EXPECT().GetStateRootFromBlock(gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().GetKeysWithPrefix(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	return m
}

// NewMockAnyBlockAPI creates and return an rpc BlockAPI interface mock
func NewMockAnyBlockAPI(ctrl *gomock.Controller) *modulesmocks.MockBlockAPI {
	m := modulesmocks.NewMockBlockAPI(ctrl)
	m.EXPECT().GetHeader(gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().BestBlockHash().Return(common.Hash{}).AnyTimes()
	m.EXPECT().GetBlockByHash(gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().GetHashByNumber(gomock.Any()).Return(common.Hash{}, nil).AnyTimes()
	m.EXPECT().GetFinalisedHash(gomock.Any(), gomock.Any()).
		Return(common.Hash{}, nil).AnyTimes()
	m.EXPECT().GetHighestFinalisedHash().Return(common.Hash{}, nil).AnyTimes()
	m.EXPECT().GetImportedBlockNotifierChannel().Return(make(chan *types.Block, 5)).AnyTimes()
	m.EXPECT().FreeImportedBlockNotifierChannel(gomock.Any()).AnyTimes()
	m.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo, 5)).AnyTimes()
	m.EXPECT().FreeFinalisedNotifierChannel(gomock.Any()).AnyTimes()
	m.EXPECT().GetJustification(gomock.Any()).Return(make([]byte, 10), nil).AnyTimes()
	m.EXPECT().HasJustification(gomock.Any()).Return(true, nil).AnyTimes()
	m.EXPECT().RegisterRuntimeUpdatedChannel(gomock.Any()).
		Return(uint32(0), nil).AnyTimes()
	return m
}

// NewMockAnyAPI creates and return an rpc CoreAPI interface mock
func NewMockAnyAPI(ctrl *gomock.Controller) *modulesmocks.MockCoreAPI {
	m := modulesmocks.NewMockCoreAPI(ctrl)
	m.EXPECT().InsertKey(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().HasKey(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	m.EXPECT().GetRuntimeVersion(gomock.Any()).
		Return(runtime.Version{SpecName: []byte(`mock-spec`)}, nil).AnyTimes()
	m.EXPECT().HandleSubmittedExtrinsic(gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().GetMetadata(gomock.Any()).Return(nil, nil).AnyTimes()
	return m
}
