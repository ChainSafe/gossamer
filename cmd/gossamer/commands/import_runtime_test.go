// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"encoding/json"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateGenesisWithRuntime test "gossamer import-runtime"
func TestCreateGenesisWithRuntime(t *testing.T) {
	testCode := []byte("somecode")
	testHex := common.BytesToHex(testCode)

	filename := filepath.Join(t.TempDir(), "test.wasm")
	err := os.WriteFile(filename, testCode, os.ModePerm)
	require.NoError(t, err)

	westendGenesisFile := utils.GetWestendDevHumanReadableGenesisPath(t)
	out, err := createGenesisWithRuntime(filename, westendGenesisFile)
	require.NoError(t, err)

	g := new(genesis.Genesis)
	err = json.Unmarshal([]byte(out), g)
	require.NoError(t, err)
	require.Equal(t, testHex, g.Genesis.Runtime["system"]["code"].(string))
}
