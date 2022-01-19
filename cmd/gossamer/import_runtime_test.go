// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"

	"github.com/stretchr/testify/require"
)

func TestCreateGenesisWithRuntime(t *testing.T) {
	defaultGenesisSpecPath = "../../chain/gssmr/genesis-spec.json"

	testCode := []byte("somecode")
	testHex := common.BytesToHex(testCode)

	filename := filepath.Join(t.TempDir(), "test.wasm")
	err := os.WriteFile(filename, testCode, os.ModePerm)
	require.NoError(t, err)

	out, err := createGenesisWithRuntime(filename)
	require.NoError(t, err)

	g := new(genesis.Genesis)
	err = json.Unmarshal([]byte(out), g)
	require.NoError(t, err)
	require.Equal(t, testHex, g.Genesis.Runtime["System"]["code"].(string))
}
