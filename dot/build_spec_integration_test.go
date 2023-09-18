// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// hex encoding for ":code", used as key for code is raw genesis files.
const codeHex = "0x3a636f6465"

func TestWriteGenesisSpecFile_Integration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)
	config.ChainSpec = utils.GetWestendDevRawGenesisPath(t)

	expected, err := genesis.NewGenesisFromJSONRaw(config.ChainSpec)
	require.NoError(t, err)

	err = InitNode(config)
	require.NoError(t, err)

	bs, err := BuildFromGenesis(config.ChainSpec, 0)
	require.NoError(t, err)

	data, err := bs.ToJSONRaw()
	require.NoError(t, err)

	tmpFile := filepath.Join(t.TempDir(), "unique-raw-genesis.json")
	err = WriteGenesisSpecFile(data, tmpFile)
	require.NoError(t, err)

	file, err := os.Open(tmpFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := file.Close()
		require.NoError(t, err)
	})

	gen := new(genesis.Genesis)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(gen)
	require.NoError(t, err)

	require.Equal(t, expected.ChainType, gen.ChainType)
	require.Equal(t, expected.Properties, gen.Properties)

}

func TestBuildFromDB_Integration(t *testing.T) {
	// setup expected
	config := DefaultTestWestendDevConfig(t)
	config.ChainSpec = utils.GetWestendDevRawGenesisPath(t)
	expected, err := genesis.NewGenesisFromJSONRaw(config.ChainSpec)
	require.NoError(t, err)
	// initialise node (initialise state database and load genesis data)
	err = InitNode(config)
	require.NoError(t, err)

	bs, err := BuildFromDB(config.BasePath)
	require.NoError(t, err)
	res, err := bs.ToJSON()
	require.NoError(t, err)
	jGen := genesis.Genesis{}
	err = json.Unmarshal(res, &jGen)
	require.NoError(t, err)

	require.Equal(t, expected.Genesis.Raw["top"][codeHex], jGen.Genesis.Runtime.System.Code)
}
