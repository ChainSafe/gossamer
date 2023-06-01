// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"context"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"

	"github.com/klauspost/compress/zstd"
)

// test used for ensuring runtime exec calls can be made concurrently
func TestConcurrentRuntimeCalls(t *testing.T) {
	instance := NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929)

	// execute 2 concurrent calls to the runtime
	go func() {
		_, _ = instance.Exec(runtime.CoreVersion, []byte{})
	}()
	go func() {
		_, _ = instance.Exec(runtime.CoreVersion, []byte{})
	}()
}

func Test_GetRuntimeVersion(t *testing.T) {
	polkadotRuntimeFilepath, err := runtime.GetRuntime(
		context.Background(), runtime.POLKADOT_RUNTIME_v0929)
	require.NoError(t, err)
	code, err := os.ReadFile(polkadotRuntimeFilepath)
	require.NoError(t, err)
	version, err := GetRuntimeVersion(code)
	require.NoError(t, err)

	expected := runtime.Version{
		SpecName:         []byte("polkadot"),
		ImplName:         []byte("parity-polkadot"),
		AuthoringVersion: 0,
		SpecVersion:      9290,
		ImplVersion:      0,
		APIItems: []runtime.APIItem{
			{Name: [8]uint8{0xdf, 0x6a, 0xcb, 0x68, 0x99, 0x7, 0x60, 0x9b}, Ver: 0x4},
			{Name: [8]uint8{0x37, 0xe3, 0x97, 0xfc, 0x7c, 0x91, 0xf5, 0xe4}, Ver: 0x1},
			{Name: [8]uint8{0x40, 0xfe, 0x3a, 0xd4, 0x1, 0xf8, 0x95, 0x9a}, Ver: 0x6},
			{Name: [8]uint8{0x17, 0xa6, 0xbc, 0xd, 0x0, 0x62, 0xae, 0xb3}, Ver: 0x1},
			{Name: [8]uint8{0xd2, 0xbc, 0x98, 0x97, 0xee, 0xd0, 0x8f, 0x15}, Ver: 0x3},
			{Name: [8]uint8{0xf7, 0x8b, 0x27, 0x8b, 0xe5, 0x3f, 0x45, 0x4c}, Ver: 0x2},
			{Name: [8]uint8{0xaf, 0x2c, 0x2, 0x97, 0xa2, 0x3e, 0x6d, 0x3d}, Ver: 0x2},
			{Name: [8]uint8{0x49, 0xea, 0xaf, 0x1b, 0x54, 0x8a, 0xc, 0xb0}, Ver: 0x1},
			{Name: [8]uint8{0x91, 0xd5, 0xdf, 0x18, 0xb0, 0xd2, 0xcf, 0x58}, Ver: 0x1},
			{Name: [8]uint8{0xed, 0x99, 0xc5, 0xac, 0xb2, 0x5e, 0xed, 0xf5}, Ver: 0x3},
			{Name: [8]uint8{0xcb, 0xca, 0x25, 0xe3, 0x9f, 0x14, 0x23, 0x87}, Ver: 0x2},
			{Name: [8]uint8{0x68, 0x7a, 0xd4, 0x4a, 0xd3, 0x7f, 0x3, 0xc2}, Ver: 0x1},
			{Name: [8]uint8{0xab, 0x3c, 0x5, 0x72, 0x29, 0x1f, 0xeb, 0x8b}, Ver: 0x1},
			{Name: [8]uint8{0xbc, 0x9d, 0x89, 0x90, 0x4f, 0x5b, 0x92, 0x3f}, Ver: 0x1},
			{Name: [8]uint8{0x37, 0xc8, 0xbb, 0x13, 0x50, 0xa9, 0xa2, 0xa8}, Ver: 0x1},
			{Name: [8]uint8{0xf3, 0xff, 0x14, 0xd5, 0xab, 0x52, 0x70, 0x59}, Ver: 0x1},
		},
		TransactionVersion: 14,
	}

	require.Equal(t, expected, version)
}

func Benchmark_GetRuntimeVersion(b *testing.B) {
	polkadotRuntimeFilepath, err := runtime.GetRuntime(
		context.Background(), runtime.POLKADOT_RUNTIME_v0929)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code, _ := os.ReadFile(polkadotRuntimeFilepath)
		_, _ = GetRuntimeVersion(code)
	}
}

func TestDecompressWasm(t *testing.T) {
	encoder, err := zstd.NewWriter(nil)
	require.NoError(t, err)

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
			append([]byte{82, 188, 83, 118, 70, 219, 142, 5}, encoder.EncodeAll([]byte("compressed"), nil)...),
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
