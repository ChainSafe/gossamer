// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
)

func TestBuildFromGenesis(t *testing.T) {
	file, err := genesis.CreateTestGenesisJSONFile(false)
	defer os.Remove(file)
	require.NoError(t, err)
	bs, err := BuildFromGenesis(file, 0)

	expectedChainType := "TESTCHAINTYPE"
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
	bs, err := BuildFromGenesis("/not/exists/genesis.json", 0)
	require.Nil(t, bs)
	require.Error(t, err, os.ErrNotExist)
}

func TestWriteGenesisSpecFileWhenFileAlreadyExists(t *testing.T) {
	f, err := ioutil.TempFile("", "existing file data")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	someBytes := []byte("Testing some bytes")
	err = WriteGenesisSpecFile(someBytes, f.Name())

	require.Error(t, err,
		fmt.Sprintf("file %s already exists, rename to avoid overwritten", f.Name()))
}

func TestWriteGenesisSpecFile(t *testing.T) {
	cfg := NewTestConfig(t)
	cfg.Init.Genesis = "../chain/gssmr/genesis.json"

	expected, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
	require.NoError(t, err)

	err = InitNode(cfg)
	require.NoError(t, err)

	bs, err := BuildFromGenesis(cfg.Init.Genesis, 0)
	require.NoError(t, err)

	data, err := bs.ToJSONRaw()
	require.NoError(t, err)

	tmpFiles := []string{
		"/tmp/unique-raw-genesis.json",
		"./unique-raw-genesis.json",
	}

	for _, tmpFile := range tmpFiles {
		err = WriteGenesisSpecFile(data, tmpFile)
		require.NoError(t, err)
		require.FileExists(t, tmpFile)

		defer os.Remove(tmpFile)

		file, err := os.Open(tmpFile)
		require.NoError(t, err)
		defer file.Close()

		genesisBytes, err := ioutil.ReadAll(file)
		require.NoError(t, err)

		gen := new(genesis.Genesis)
		err = json.Unmarshal(genesisBytes, gen)
		require.NoError(t, err)

		require.Equal(t, expected.ChainType, gen.ChainType)
		require.Equal(t, expected.Properties, gen.Properties)
	}
}

func TestBuildFromDB(t *testing.T) {
	// setup expected
	cfg := NewTestConfig(t)
	cfg.Init.Genesis = "../chain/gssmr/genesis.json"
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

	require.Equal(t, expected.Genesis.Raw["top"]["0x3a636f6465"], jGen.Genesis.Runtime["system"]["code"])
}
