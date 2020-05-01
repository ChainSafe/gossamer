// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package modules

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateModule_GetRuntimeVersion(t *testing.T) {
	// expected results based on responses from prior tests
	expected := StateRuntimeVersionResponse{
		SpecName:         "test",
		ImplName:         "parity-test",
		AuthoringVersion: 1,
		SpecVersion:      1,
		ImplVersion:      1,
		Apis: []interface{}{[]interface{}{"0xdf6acb689907609b", int32(2)},
			[]interface{}{"0x37e397fc7c91f5e4", int32(1)},
			[]interface{}{"0xd2bc9897eed08f15", int32(1)},
			[]interface{}{"0x40fe3ad401f8959a", int32(3)},
			[]interface{}{"0xc6e9a76309f39b09", int32(1)},
			[]interface{}{"0xdd718d5cc53262d4", int32(1)},
			[]interface{}{"0xcbca25e39f142387", int32(1)},
			[]interface{}{"0xf78b278be53f454c", int32(1)},
			[]interface{}{"0xab3c0572291feb8b", int32(1)}},
	}
	sm := setupStateModule(t)
	res := StateRuntimeVersionResponse{}
	err := sm.GetRuntimeVersion(nil, nil, &res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestStateModule_GetStorage(t *testing.T) {
	sm := setupStateModule(t)
	expected := []byte(`value1`)
	req := []string{"0x3a6b657931"}  // :key1
	var res  interface{}

	sm.GetStorage(nil, &req, &res)

	hex, err := common.HexToBytes(fmt.Sprintf("%v", res))

	require.NoError(t, err)
	require.Equal(t, expected, hex)
}

func TestStateModule_GetStorage_NotFound(t *testing.T) {
	sm := setupStateModule(t)

	req := []string{"0x666f6f"}  // foo
	var res  interface{}

	sm.GetStorage(nil, &req, &res)

	require.Nil(t, res)
}

func setupStateModule(t *testing.T) *StateModule {
	// setup service
	net := newNetworkService(t)
	chain := newChainService(t)
	// init storage with test data
	err := chain.Storage.SetStorage([]byte(`:key1`), []byte(`value1`))
	require.NoError(t, err)
	err = chain.Storage.SetStorage([]byte(`:key2`), []byte(`value2`))
	require.NoError(t, err)

	core := newCoreService(t)
	return NewStateModule(net, chain.Storage, core)
}
