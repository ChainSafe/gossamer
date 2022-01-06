// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/dot"
	"github.com/ChainSafe/gossamer/internal/lib/utils"

	"github.com/stretchr/testify/require"
)

const GssmrConfigPath = "../../internal/chain/gssmr/config.toml"
const GssmrGenesisPath = "../../internal/chain/gssmr/genesis.json"

const KusamaConfigPath = "../../internal/chain/kusama/config.toml"
const KusamaGenesisPath = "../../internal/chain/kusama/genesis.json"

// TestLoadConfig tests loading a toml configuration file
func TestLoadConfig(t *testing.T) {
	cfg, cfgFile := newTestConfigWithFile(t)
	require.NotNil(t, cfg)

	genFile := dot.NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := dot.InitNode(cfg)
	require.Nil(t, err)

	err = loadConfig(dotConfigToToml(cfg), cfgFile.Name())
	require.Nil(t, err)
	require.NotNil(t, cfg)
}

// TestLoadConfigGssmr tests loading the toml configuration file for gssmr
func TestLoadConfigGssmr(t *testing.T) {
	cfg := dot.GssmrConfig()
	require.NotNil(t, cfg)

	cfg.Global.BasePath = utils.NewTestDir(t)
	cfg.Init.Genesis = GssmrGenesisPath

	defer utils.RemoveTestDir(t)

	err := dot.InitNode(cfg)
	require.Nil(t, err)

	err = loadConfig(dotConfigToToml(cfg), GssmrConfigPath)
	require.Nil(t, err)
	require.NotNil(t, cfg)
}

func TestLoadConfigKusama(t *testing.T) {
	cfg := dot.KusamaConfig()
	require.NotNil(t, cfg)

	cfg.Global.BasePath = utils.NewTestDir(t)
	cfg.Init.Genesis = KusamaGenesisPath

	defer utils.RemoveTestDir(t)

	err := dot.InitNode(cfg)
	require.Nil(t, err)

	err = loadConfig(dotConfigToToml(cfg), KusamaConfigPath)
	require.Nil(t, err)
}
