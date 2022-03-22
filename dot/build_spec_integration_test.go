// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// hex encoding for ":code", used as key for code is raw genesis files.
const codeHex = "0x3a636f6465"

func TestBuildFromGenesis(t *testing.T) {
	t.Parallel()

	file := genesis.CreateTestGenesisJSONFile(t, false)
	bs, err := BuildFromGenesis(file, 0)

	const expectedChainType = "TESTCHAINTYPE"
	expectedProperties := map[string]interface{}{
		"ss58Format":    0.0,
		"tokenDecimals": 0.0,
		"tokenSymbol":   "TEST",
	}

	bs.genesis.ChainType = expectedChainType
	bs.genesis.Properties = expectedProperties

	require.NoError(t, err)

	// confirm human-readable fields
	hr, err := bs.ToJSON()
	require.NoError(t, err)
	jGen := genesis.Genesis{}
	err = json.Unmarshal(hr, &jGen)
	require.NoError(t, err)
	genesis.TestGenesis.Genesis = genesis.TestFieldsHR
	require.Equal(t, genesis.TestGenesis.Genesis.Runtime, jGen.Genesis.Runtime)
	require.Equal(t, expectedChainType, jGen.ChainType)
	require.Equal(t, expectedProperties, jGen.Properties)

	// confirm raw fields
	raw, err := bs.ToJSONRaw()
	require.NoError(t, err)
	jGenRaw := genesis.Genesis{}
	err = json.Unmarshal(raw, &jGenRaw)
	require.NoError(t, err)
	genesis.TestGenesis.Genesis = genesis.TestFieldsRaw
	require.Equal(t, genesis.TestGenesis.Genesis.Raw, jGenRaw.Genesis.Raw)
	require.Equal(t, expectedChainType, jGenRaw.ChainType)
	require.Equal(t, expectedProperties, jGenRaw.Properties)
}

func TestBuildFromGenesis_WhenGenesisDoesNotExists(t *testing.T) {
	t.Parallel()

	bs, err := BuildFromGenesis("/not/exists/genesis.json", 0)
	require.Nil(t, bs)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestWriteGenesisSpecFileWhenFileAlreadyExists(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "genesis.raw")
	someBytes := []byte("Testing some bytes")
	err := WriteGenesisSpecFile(someBytes, filePath)

	require.EqualError(t, err,
		fmt.Sprintf("file %s already exists, rename to avoid overwriting", filePath))
}

func TestWriteGenesisSpecFile(t *testing.T) {
	t.Parallel()

	cfg := NewTestConfig(t)
	cfg.Init.Genesis = utils.GetGssmrGenesisRawPathTest(t)

	expected, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
	require.NoError(t, err)

	err = InitNode(cfg)
	require.NoError(t, err)

	bs, err := BuildFromGenesis(cfg.Init.Genesis, 0)
	require.NoError(t, err)

	data, err := bs.ToJSONRaw()
	require.NoError(t, err)

	tmpFile := filepath.Join(t.TempDir(), "unique-raw-genesis.json")
	err = WriteGenesisSpecFile(data, tmpFile)
	require.NoError(t, err)
	require.FileExists(t, tmpFile)

	file, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer file.Close()

	gen := new(genesis.Genesis)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(gen)
	require.NoError(t, err)

	require.Equal(t, expected.ChainType, gen.ChainType)
	require.Equal(t, expected.Properties, gen.Properties)

}

func TestBuildFromDB(t *testing.T) {
	t.Parallel()

	// setup expected
	cfg := NewTestConfig(t)
	cfg.Init.Genesis = utils.GetGssmrGenesisRawPathTest(t)
	expected, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
	require.NoError(t, err)
	// initialise node (initialise state database and load genesis data)
	err = InitNode(cfg)
	require.NoError(t, err)

	bs, err := BuildFromDB(cfg.Global.BasePath)
	require.NoError(t, err)
	res, err := bs.ToJSON()
	require.NoError(t, err)
	jGen := genesis.Genesis{}
	err = json.Unmarshal(res, &jGen)
	require.NoError(t, err)

	require.Equal(t, expected.Genesis.Raw["top"][codeHex], jGen.Genesis.Runtime["system"]["code"])
}
