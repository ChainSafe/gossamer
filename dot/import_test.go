// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func setupStateFile(t *testing.T) string {
	filename := "../lib/runtime/test_data/kusama/block1482002.out"

	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	require.NoError(t, err)
	pairs := rpcPairs["result"].([]interface{})

	bz, err := json.Marshal(pairs)
	require.NoError(t, err)

	fp := "./test_data/state.json"
	err = os.WriteFile(fp, bz, 0777)
	require.NoError(t, err)

	return fp
}

func setupHeaderFile(t *testing.T) string {
	headerStr := "{\"digest\":{\"logs\":[\"0x0642414245b501013c0000009659bd0f0000000070edad1c9064fff78cb18435223d8adaf5ea04c24b1a8766e3dc01eb03cc6a0c11b79793d4e31cc0990838229c44fed1669a7c7c79e1e6d0a96374d6496728069d1ef739e290497a0e3b728fa88fcbdd3a5504e0efde0242e7a806dd4fa9260c\",\"0x054241424501019e7f28dddcf27c1e6b328d5694c368d5b2ec5dbe0e412ae1c98f88d53be4d8502fac571f3f19c9caaf281a673319241e0c5095a683ad34316204088a36a4bd86\"]},\"extrinsicsRoot\":\"0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0\",\"number\":\"0x169d12\",\"parentHash\":\"0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975\",\"stateRoot\":\"0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb\"}"
	fp := "./test_data/header.json"
	err := os.WriteFile(fp, []byte(headerStr), 0777)
	require.NoError(t, err)
	return fp
}

func TestNewTrieFromPairs(t *testing.T) {
	fp := setupStateFile(t)
	tr, err := newTrieFromPairs(fp)
	require.NoError(t, err)

	expectedRoot := common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb")
	require.Equal(t, expectedRoot, tr.MustHash())
}

func TestNewHeaderFromFile(t *testing.T) {
	fp := setupHeaderFile(t)
	header, err := newHeaderFromFile(fp)
	require.NoError(t, err)

	digestBytes := common.MustHexToBytes("0x080642414245b501013c0000009659bd0f0000000070edad1c9064fff78cb18435223d8adaf5ea04c24b1a8766e3dc01eb03cc6a0c11b79793d4e31cc0990838229c44fed1669a7c7c79e1e6d0a96374d6496728069d1ef739e290497a0e3b728fa88fcbdd3a5504e0efde0242e7a806dd4fa9260c054241424501019e7f28dddcf27c1e6b328d5694c368d5b2ec5dbe0e412ae1c98f88d53be4d8502fac571f3f19c9caaf281a673319241e0c5095a683ad34316204088a36a4bd86")
	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest.Types))

	expected := &types.Header{
		ParentHash:     common.MustHexToHash("0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975"),
		Number:         big.NewInt(1482002),
		StateRoot:      common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		ExtrinsicsRoot: common.MustHexToHash("0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0"),
		Digest:         digest,
	}

	require.Equal(t, expected, header)
}

func TestImportState(t *testing.T) {
	basepath := t.TempDir()

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	cfg.Global.BasePath = basepath
	err := InitNode(cfg)
	require.NoError(t, err)

	stateFP := setupStateFile(t)
	headerFP := setupHeaderFile(t)

	firstSlot := uint64(262493679)
	err = ImportState(basepath, stateFP, headerFP, firstSlot)
	require.NoError(t, err)
}
