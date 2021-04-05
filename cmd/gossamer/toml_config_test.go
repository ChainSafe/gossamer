package main

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/lib/utils"

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
	t.Skip() // TODO: fix by updating kusama runtime
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
