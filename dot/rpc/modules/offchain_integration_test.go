// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_OffchainModule_LocalStorageGet(t *testing.T) {
	t.Run("get_local_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		const keyHex = "0x11111111111111"
		request := &OffchainLocalStorageGet{
			Kind: offchainLocal,
			Key:  keyHex,
		}
		errTest := errors.New("test error")
		runtimeStorage.EXPECT().GetLocal(common.MustHexToBytes(keyHex)).
			Return(nil, errTest)

		err := offchainModule.LocalStorageGet(nil, request, nil)
		assert.ErrorIs(t, err, errTest)
	})

	t.Run("local_kind", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		const keyHex = "0x11111111111111"
		request := &OffchainLocalStorageGet{
			Kind: offchainLocal,
			Key:  keyHex,
		}
		runtimeStorage.EXPECT().GetLocal(common.MustHexToBytes(keyHex)).
			Return([]byte("some-value"), nil)
		var response StringResponse
		err := offchainModule.LocalStorageGet(nil, request, &response)
		require.NoError(t, err)
		expectedResponse := StringResponse(common.BytesToHex([]byte("some-value")))
		assert.Equal(t, response, expectedResponse)
	})

	t.Run("persistent_kind", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		const keyHex = "0x11111111111111"
		request := &OffchainLocalStorageGet{
			Kind: offchainPersistent,
			Key:  keyHex,
		}
		runtimeStorage.EXPECT().GetPersistent(common.MustHexToBytes(keyHex)).
			Return([]byte("some-value"), nil)
		var response StringResponse
		err := offchainModule.LocalStorageGet(nil, request, &response)
		require.NoError(t, err)
		expectedResponse := StringResponse(common.BytesToHex([]byte("some-value")))
		assert.Equal(t, response, expectedResponse)
	})
}

func TestOffchainStorage_OtherKind(t *testing.T) {
	m := new(OffchainModule)
	setReq := &OffchainLocalStorageSet{
		Kind:  "another kind",
		Key:   "0x11111111111111",
		Value: "0x22222222222222",
	}
	getReq := &OffchainLocalStorageGet{
		Kind: "another kind",
		Key:  "0x11111111111111",
	}
	err := m.LocalStorageSet(nil, setReq, nil)
	require.Error(t, err, "storage kind not found: another kind")

	err = m.LocalStorageGet(nil, getReq, nil)
	require.Error(t, err, "storage kind not found: another kind")
}

func Test_OffchainModule_LocalStorageSet(t *testing.T) {
	const keyHex, valueHex = "0x11111111111111", "0x22222222222222"

	t.Run("set_local_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		request := &OffchainLocalStorageSet{
			Kind:  offchainLocal,
			Key:   keyHex,
			Value: valueHex,
		}
		errTest := errors.New("test error")
		runtimeStorage.EXPECT().SetLocal(
			common.MustHexToBytes(keyHex), common.MustHexToBytes(valueHex)).
			Return(errTest)

		err := offchainModule.LocalStorageSet(nil, request, nil)
		assert.ErrorIs(t, err, errTest)
	})

	t.Run("local_kind", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		request := &OffchainLocalStorageSet{
			Kind:  offchainLocal,
			Key:   keyHex,
			Value: valueHex,
		}
		runtimeStorage.EXPECT().SetLocal(
			common.MustHexToBytes(keyHex), common.MustHexToBytes(valueHex)).
			Return(nil)
		var response StringResponse
		err := offchainModule.LocalStorageSet(nil, request, &response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})

	t.Run("persistent_kind", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		runtimeStorage := mocks.NewMockRuntimeStorageAPI(ctrl)
		offchainModule := &OffchainModule{
			nodeStorage: runtimeStorage,
		}

		request := &OffchainLocalStorageSet{
			Kind:  offchainPersistent,
			Key:   keyHex,
			Value: valueHex,
		}
		runtimeStorage.EXPECT().SetPersistent(
			common.MustHexToBytes(keyHex), common.MustHexToBytes(valueHex)).
			Return(nil)
		var response StringResponse
		err := offchainModule.LocalStorageSet(nil, request, &response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})
}
