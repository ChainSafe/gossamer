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

package wasmtime

import (
	"errors"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Error("failed to generate runtime wasm file", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

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

func TestInstance_Version_NodeRuntime(t *testing.T) {
	t.Skip() // TODO: currently fails, returns all 0

	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		260,
		0,
		nil,
		1,
	)

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_PaymentQueryInfo(t *testing.T) {
	tests := []struct {
		extB   []byte
		ext    string
		err    error
		expect *types.TransactionPaymentQueryInfo
	}{
		{
			// Was made with @polkadot/api on https://github.com/danforbes/polkadot-js-scripts/tree/create-signed-tx
			ext: "0xd1018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01bc2b6e35929aabd5b8bc4e5b0168c9bee59e2bb9d6098769f6683ecf73e44c776652d947a270d59f3d37eb9f9c8c17ec1b4cc473f2f9928ffdeef0f3abd43e85d502000000012844616e20466f72626573",
			err: nil,
			expect: &types.TransactionPaymentQueryInfo{
				Weight: 1973000,
				Class:  0,
				PartialFee: &scale.Uint128{
					Upper: 0,
					Lower: uint64(1180126973000),
				},
			},
		},
		{
			// incomplete extrinsic
			ext: "0x4ccde39a5684e7a56da23b22d4d9fbadb023baa19c56495432884d0640000000000000000000000000000000",
			err: errors.New("Failed to call the `TransactionPaymentApi_query_info` exported function."), //nolint
		},
		{
			// incomplete extrinsic
			extB: nil,
			err:  errors.New("Failed to call the `TransactionPaymentApi_query_info` exported function."), //nolint
		},
	}

	for _, test := range tests {
		var err error
		var extBytes []byte

		if test.ext == "" {
			extBytes = test.extB
		} else {
			extBytes, err = common.HexToBytes(test.ext)
			require.NoError(t, err)
		}

		ins := NewTestInstance(t, runtime.NODE_RUNTIME)
		info, err := ins.PaymentQueryInfo(extBytes)

		if test.err != nil {
			require.Error(t, err)
			require.Equal(t, err.Error(), test.err.Error())
			continue
		}

		require.NoError(t, err)
		require.NotNil(t, info)
		require.Equal(t, test.expect, info)
	}
}
