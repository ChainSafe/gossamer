// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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

// NewTestGenesisRawFile returns a test genesis file using "westend_dev" raw data
func NewTestGenesisRawFile(t *testing.T, cfg *Config) (filename string) {
	filename = filepath.Join(t.TempDir(), "genesis.json")

	fp := utils.GetWestendDevRawGenesisPath(t)

	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    westendDevGenesis.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}
