// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// Logger is the utils package local logger.
var Logger = log.NewFromGlobal(log.AddContext("pkg", "test/utils"))

// GenerateGenesisAuths generates a genesis file with numAuths authorities
// and returns the file path to the genesis file. The genesis file is
// automatically removed when the test ends.
func GenerateGenesisAuths(t *testing.T, numAuths int) (genesisPath string) {
	westendGenesisPath := utils.GetWestendDevRawGenesisPath(t)

	buildSpec, err := dot.BuildFromGenesis(westendGenesisPath, numAuths)
	require.NoError(t, err)

	buildSpecJSON, err := buildSpec.ToJSONRaw()
	require.NoError(t, err)

	genesisPath = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(genesisPath, buildSpecJSON, os.ModePerm)
	require.NoError(t, err)

	return genesisPath
}
