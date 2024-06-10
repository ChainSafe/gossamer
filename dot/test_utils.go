// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// NewTestGenesisRawFile returns a test genesis file using "westend-dev" raw data
func NewTestGenesisRawFile(t *testing.T, config *cfg.Config) (filename string) {
	filename = filepath.Join(t.TempDir(), "genesis.json")
	if os.Getenv("CI") == "buildjet" {
		tmpdir, err := os.MkdirTemp("..", "*_gen_raw_file")
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, os.Remove(tmpdir))
		})
		filename = filepath.Join(tmpdir, "genesis.json")
	}

	fp := utils.GetWestendDevRawGenesisPath(t)
	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       config.Name,
		ID:         config.ID,
		Bootnodes:  config.Network.Bootnodes,
		ProtocolID: config.Network.ProtocolID,
		Genesis:    westendDevGenesis.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}
