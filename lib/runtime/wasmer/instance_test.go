// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"

	"github.com/klauspost/compress/zstd"
)

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
	ptr, length := runtime.Int64ToPointerAndSize(in)
	require.Equal(t, int32(8), ptr)
	require.Equal(t, int32(32), length)
	res := runtime.PointerAndSizeToInt64(ptr, length)
	require.Equal(t, in, res)
}

func TestInstance_CheckRuntimeVersion(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	err := runtime.GetRuntimeBlob(runtime.POLKADOT_RUNTIME_FP, runtime.POLKADOT_RUNTIME_URL)
	require.NoError(t, err)
	fp, err := filepath.Abs(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)
	code, err := os.ReadFile(fp)
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

func TestDecompressWasm(t *testing.T) {
	encoder, _ := zstd.NewWriter(nil)
	cases := []struct {
		in       []byte
		expected []byte
		msg      string
	}{
		{
			[]byte{82, 188, 83, 118, 70, 219, 142},
			[]byte{82, 188, 83, 118, 70, 219, 142},
			"partial compression flag",
		},
		{
			[]byte{82, 188, 83, 118, 70, 219, 142, 6},
			[]byte{82, 188, 83, 118, 70, 219, 142, 6},
			"wrong compression flag",
		},
		{
			[]byte{82, 188, 83, 118, 70, 219, 142, 6, 221},
			[]byte{82, 188, 83, 118, 70, 219, 142, 6, 221},
			"wrong compression flag with data",
		},
		{
			append([]byte{82, 188, 83, 118, 70, 219, 142, 5}, encoder.EncodeAll([]byte("compressed"), make([]byte, 0))...),
			[]byte("compressed"),
			"compressed data",
		},
	}

	for _, test := range cases {
		actual, err := decompressWasm(test.in)
		require.NoError(t, err)
		require.Equal(t, test.expected, actual)
	}
}
