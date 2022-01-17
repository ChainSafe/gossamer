// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot"

	"github.com/stretchr/testify/require"
)

const GssmrConfigPath = "../../chain/gssmr/config.toml"
const GssmrGenesisPath = "../../chain/gssmr/genesis.json"

const KusamaConfigPath = "../../chain/kusama/config.toml"
const KusamaGenesisPath = "../../chain/kusama/genesis.json"

// TestLoadConfig tests loading a toml configuration file
func TestLoadConfig(t *testing.T) {
	cfg, cfgFile := newTestConfigWithFile(t)
	require.NotNil(t, cfg)

	genFile := dot.NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

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

	cfg.Global.BasePath = t.TempDir()
	cfg.Init.Genesis = GssmrGenesisPath

	err := dot.InitNode(cfg)
	require.Nil(t, err)

	err = loadConfig(dotConfigToToml(cfg), GssmrConfigPath)
	require.Nil(t, err)
	require.NotNil(t, cfg)
}

func TestLoadConfigKusama(t *testing.T) {
	cfg := dot.KusamaConfig()
	require.NotNil(t, cfg)

	cfg.Global.BasePath = t.TempDir()
	cfg.Init.Genesis = KusamaGenesisPath

	err := dot.InitNode(cfg)
	require.Nil(t, err)

	err = loadConfig(dotConfigToToml(cfg), KusamaConfigPath)
	require.Nil(t, err)
}
