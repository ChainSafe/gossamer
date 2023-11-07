// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStateFile(t *testing.T) string {
	t.Helper()

	rootPath, err := utils.GetProjectRootPath()
	require.NoError(t, err)
	filename := filepath.Join(rootPath, "lib/runtime/test_data/kusama/block1482002.out")
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	require.NoError(t, err)
	pairs := rpcPairs["result"].([]interface{})

	bz, err := json.Marshal(pairs)
	require.NoError(t, err)

	fp := filepath.Join(t.TempDir(), "state.json")
	err = os.WriteFile(fp, bz, 0777)
	require.NoError(t, err)

	return fp
}

func setupHeaderFile(t *testing.T) string {
	t.Helper()

	//nolint:lll
	const headerStr = `{
	"digest":{
		"logs":[
			"0x0642414245b501013c0000009659bd0f0000000070edad1c9064fff78cb18435223d8adaf5ea04c24b1a8766e3dc01eb03cc6a0c11b79793d4e31cc0990838229c44fed1669a7c7c79e1e6d0a96374d6496728069d1ef739e290497a0e3b728fa88fcbdd3a5504e0efde0242e7a806dd4fa9260c",
			"0x054241424501019e7f28dddcf27c1e6b328d5694c368d5b2ec5dbe0e412ae1c98f88d53be4d8502fac571f3f19c9caaf281a673319241e0c5095a683ad34316204088a36a4bd86"
		]
	},
	"extrinsicsRoot":"0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0",
	"number":"0x169d12",
	"parentHash":"0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975",
	"stateRoot":"0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"
}`

	fp := filepath.Join(t.TempDir(), "header.json")
	err := os.WriteFile(fp, []byte(headerStr), 0777)
	require.NoError(t, err)
	return fp
}

func Test_newHeaderFromFile(t *testing.T) {
	t.Parallel()

	digest := types.NewDigest()
	preRuntimeDigest := types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		// bytes for PreRuntimeDigest that was created in setupHeaderFile function
		Data: []byte{1, 60, 0, 0, 0, 150, 89, 189, 15, 0, 0, 0, 0, 112, 237, 173, 28, 144, 100, 255,
			247, 140, 177, 132, 53, 34, 61, 138, 218, 245, 234, 4, 194, 75, 26, 135, 102, 227, 220, 1, 235, 3, 204,
			106, 12, 17, 183, 151, 147, 212, 227, 28, 192, 153, 8, 56, 34, 156, 68, 254, 209, 102, 154, 124, 124,
			121, 225, 230, 208, 169, 99, 116, 214, 73, 103, 40, 6, 157, 30, 247, 57, 226, 144, 73, 122, 14, 59, 114,
			143, 168, 143, 203, 221, 58, 85, 4, 224, 239, 222, 2, 66, 231, 168, 6, 221, 79, 169, 38, 12},
	}

	preRuntimeDigestItem := types.NewDigestItem()
	err := preRuntimeDigestItem.Set(preRuntimeDigest)
	require.NoError(t, err)
	preRuntimeDigestItemValue, err := preRuntimeDigestItem.Value()
	require.NoError(t, err)
	digest.Add(preRuntimeDigestItemValue)

	sealDigest := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		// bytes for SealDigest that was created in setupHeaderFile function
		Data: []byte{158, 127, 40, 221, 220, 242, 124, 30, 107, 50, 141, 86, 148, 195, 104, 213, 178, 236, 93, 190,
			14, 65, 42, 225, 201, 143, 136, 213, 59, 228, 216, 80, 47, 172, 87, 31, 63, 25, 201, 202, 175, 40, 26,
			103, 51, 25, 36, 30, 12, 80, 149, 166, 131, 173, 52, 49, 98, 4, 8, 138, 54, 164, 189, 134},
	}

	sealDigestItem := types.NewDigestItem()
	err = sealDigestItem.Set(sealDigest)
	require.NoError(t, err)
	sealDigestItemValue, err := sealDigestItem.Value()
	require.NoError(t, err)
	digest.Add(sealDigestItemValue)

	expectedHeader := &types.Header{
		ParentHash:     common.MustHexToHash("0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975"),
		Number:         1482002,
		StateRoot:      common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		ExtrinsicsRoot: common.MustHexToHash("0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0"),
		Digest:         digest,
	}

	tests := []struct {
		name     string
		filename string
		want     *types.Header
		err      error
	}{
		{
			name: "no_arguments",
			err:  errors.New("read .: is a directory"),
		},
		{
			name:     "working example",
			filename: setupHeaderFile(t),
			want:     expectedHeader,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newHeaderFromFile(tt.filename)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_newTrieFromPairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     common.Hash
		err      error
	}{
		{
			name: "no_arguments",
			err:  errors.New("read .: is a directory"),
			want: common.Hash{},
		},
		{
			name:     "working example",
			filename: setupStateFile(t),
			want:     common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newTrieFromPairs(tt.filename)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.want.IsEmpty() {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, got.MustHash())
			}
		})
	}
}
