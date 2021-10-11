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

package wasmer

import (
	"io/ioutil"
	"path"
	"path/filepath"
	r "runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wasm "github.com/wasmerio/wasmer-go/wasmer"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
)


func testGetBytes(moduleFileName string) []byte {
	_, filename, _, _ := r.Caller(0)
	modulePath := path.Join(path.Dir(filename), "test_data", moduleFileName)
	bytes, _ := ioutil.ReadFile(modulePath)

	return bytes
}


func TestFoo(t *testing.T) {
	engine := wasm.NewEngine()
	store := wasm.NewStore(engine)
	module, err := wasm.NewModule(store, testGetBytes("test_wasm.wasm"))
	assert.NoError(t, err)

	limits, _ := wasm.NewLimits(16, 18)
	memory := wasm.NewMemory(store, wasm.NewMemoryType(limits))

	f1 := wasm.NewFunction(
		store,
		wasm.NewFunctionType(
			wasm.NewValueTypes(wasm.I64),
			wasm.NewValueTypes(),
		),
		func(args []wasm.Value) ([]wasm.Value, error) {
			return []wasm.Value{}, nil
		},
	)

	f2 := wasm.NewFunction(
		store,
		wasm.NewFunctionType(
			wasm.NewValueTypes(wasm.I64),
			wasm.NewValueTypes(wasm.I32),
		),
		func(args []wasm.Value) ([]wasm.Value, error) {
			return []wasm.Value{wasm.NewI32(0)}, nil
		},
	)

	importObject := wasm.NewImportObject()
	importObject.Register(
		"env",
		map[string]wasm.IntoExtern{
			"memory":                           memory,
			"ext_misc_print_utf8_version_1":    f1,
			"ext_hashing_blake2_256_version_1": f2,
		},
	)

	_, err = wasm.NewInstance(module, importObject)
	assert.NoError(t, err)
}

func Test_NewRuntime(t *testing.T) {
	code := testGetBytes("test_wasm.wasm")

	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	rtCfg := &Config{}
	rtCfg.LogLvl = 5
	rtCfg.Imports = ImportsNodeRuntime
	rtCfg.Storage = genState

	rt, err := NewInstance(code, rtCfg)
	require.NoError(t, err)
	require.NotNil(t, rt)
}

// test used for ensuring runtime exec calls can me made concurrently
func TestConcurrentRuntimeCalls(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	// execute 2 concurrent calls to the runtime
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
}

func TestPointerSize(t *testing.T) {
	in := int64(8) + int64(32)<<32
	ptr, length := int64ToPointerAndSize(in)
	require.Equal(t, int32(8), ptr)
	require.Equal(t, int32(32), length)
	res := pointerAndSizeToInt64(ptr, length)
	require.Equal(t, in, res)
}

func TestInstance_CheckRuntimeVersion(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	_, err := runtime.GetRuntimeBlob(runtime.POLKADOT_RUNTIME_FP, runtime.POLKADOT_RUNTIME_URL)
	require.NoError(t, err)
	fp, err := filepath.Abs(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)
	code, err := ioutil.ReadFile(fp)
	require.NoError(t, err)
	version, err := instance.CheckRuntimeVersion(code)
	require.NoError(t, err)

	expected := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		nil,
		5,
	)

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}
