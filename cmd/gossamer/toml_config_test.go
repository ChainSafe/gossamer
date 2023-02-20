// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestLoadConfig tests loading a toml configuration file
func TestLoadConfig(t *testing.T) {
	polkadotConfig := dot.PolkadotConfig()
	cfg, cfgFile := newTestConfigWithFile(t, polkadotConfig)

	genFile := dot.NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := dot.InitNode(cfg)
	require.NoError(t, err)

	err = loadConfig(dotConfigToToml(cfg), cfgFile)
	require.NoError(t, err)
}

// TestLoadConfigWestendDev tests loading the toml configuration file for westend-dev
func TestLoadConfigWestendDev(t *testing.T) {
	cfg := dot.WestendDevConfig()
	require.NotNil(t, cfg)

	cfg.Global.BasePath = t.TempDir()
	cfg.Init.Genesis = utils.GetWestendDevRawGenesisPath(t)

	err := dot.InitNode(cfg)
	require.NoError(t, err)

	projectRootPath := utils.GetProjectRootPathTest(t)
	configPath := filepath.Join(projectRootPath, "./chain/westend-dev/config.toml")

	err = loadConfig(dotConfigToToml(cfg), configPath)
	require.NoError(t, err)
}

func TestLoadConfigKusama(t *testing.T) {
	cfg := dot.KusamaConfig()
	require.NotNil(t, cfg)

	cfg.Global.BasePath = t.TempDir()
	cfg.Init.Genesis = utils.GetKusamaGenesisPath(t)

	err := dot.InitNode(cfg)
	require.NoError(t, err)

	projectRootPath := utils.GetProjectRootPathTest(t)
	kusamaConfigPath := filepath.Join(projectRootPath, "./chain/kusama/config.toml")

	err = loadConfig(dotConfigToToml(cfg), kusamaConfigPath)
	require.NoError(t, err)
}
