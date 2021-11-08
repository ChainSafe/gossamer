// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOffchainStorageGet(t *testing.T) {
	testFuncs := map[string]string{
		offchainLocal:      "GetLocal",
		offchainPersistent: "GetPersistent",
	}

	for kind, test := range testFuncs {
		expectedValue := common.BytesToHex([]byte("some-value"))
		st := new(mocks.RuntimeStorageAPI)
		st.On(test, mock.AnythingOfType("[]uint8")).Return([]byte("some-value"), nil).Once()

		m := new(OffchainModule)
		m.nodeStorage = st

		req := &OffchainLocalStorageGet{
			Kind: kind,
			Key:  "0x11111111111111",
		}

		var res StringResponse
		err := m.LocalStorageGet(nil, req, &res)
		require.NoError(t, err)
		require.Equal(t, res, StringResponse(expectedValue))
		st.AssertCalled(t, test, mock.AnythingOfType("[]uint8"))

		st.
			On(test, mock.AnythingOfType("[]uint8")).
			Return(nil, errors.New("problem to retrieve")).
			Once()

		err = m.LocalStorageGet(nil, req, nil)
		require.Error(t, err, "problem to retrieve")
		st.AssertCalled(t, test, mock.AnythingOfType("[]uint8"))
	}
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

func TestOffchainStorageSet(t *testing.T) {
	testFuncs := map[string]string{
		offchainLocal:      "SetLocal",
		offchainPersistent: "SetPersistent",
	}

	for kind, test := range testFuncs {
		st := new(mocks.RuntimeStorageAPI)
		st.On(test, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return(nil).Once()

		m := new(OffchainModule)
		m.nodeStorage = st

		req := &OffchainLocalStorageSet{
			Kind:  kind,
			Key:   "0x11111111111111",
			Value: "0x22222222222222",
		}

		var res StringResponse
		err := m.LocalStorageSet(nil, req, &res)
		require.NoError(t, err)
		require.Empty(t, res)
		st.AssertCalled(t, test, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8"))

		st.
			On(test, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).
			Return(errors.New("problem to store")).
			Once()

		err = m.LocalStorageSet(nil, req, &res)
		require.Error(t, err, "problem to store")
		require.Empty(t, res)
		st.AssertCalled(t, test, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8"))
	}
}
