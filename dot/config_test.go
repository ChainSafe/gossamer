// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

const GssmrConfigPath = "../node/gssmr/config.toml"
const GssmrGenesisPath = "../node/gssmr/genesis.json"

const KsmccConfigPath = "../node/ksmcc/config.toml"
const KsmccGenesisPath = "../node/ksmcc/genesis.json"

// TestLoadConfig tests loading a toml configuration file
func TestLoadConfig(t *testing.T) {
	cfg, cfgFile := NewTestConfigWithFile(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	err = LoadConfig(cfg, cfgFile.Name())
	require.Nil(t, err)

	// TODO: improve dot config tests
	require.NotNil(t, cfg)
}

// TestExportConfig tests exporting a toml configuration file
func TestExportConfig(t *testing.T) {
	cfg, cfgFile := NewTestConfigWithFile(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	file := ExportConfig(cfg, cfgFile.Name())

	// TODO: improve dot config tests
	require.NotNil(t, file)
}

// Gssmr Node

// TestLoadConfigGssmr tests loading the toml configuration file for ksmcc
func TestLoadConfigGssmr(t *testing.T) {
	cfg := GssmrConfig()
	require.NotNil(t, cfg)

	cfg.Global.DataDir = utils.NewTestDir(t)
	cfg.Init.Genesis = GssmrGenesisPath

	defer utils.RemoveTestDir(t)

	err := InitNode(cfg)
	require.Nil(t, err)

	err = LoadConfig(cfg, GssmrConfigPath)
	require.Nil(t, err)

	// TODO: improve dot config tests
	require.NotNil(t, cfg)
}

// TestExportConfigGssmr tests exporting the toml configuration file
func TestExportConfigGssmr(t *testing.T) {
	cfg := GssmrConfig()
	require.NotNil(t, cfg)

	gssmrGenesis := cfg.Init.Genesis
	gssmrDataDir := cfg.Global.DataDir
	cfg.Global.DataDir = utils.NewTestDir(t)
	cfg.Init.Genesis = GssmrGenesisPath

	defer utils.RemoveTestDir(t)

	err := InitNode(cfg)
	require.Nil(t, err)

	cfg.Init.Genesis = gssmrGenesis
	cfg.Global.DataDir = gssmrDataDir

	file := ExportConfig(cfg, GssmrConfigPath)

	// TODO: improve dot config tests
	require.NotNil(t, file)
}

// Ksmcc Node

// TestLoadConfigKsmcc tests loading the toml configuration file for ksmcc
func TestLoadConfigKsmcc(t *testing.T) {
	cfg := KsmccConfig()
	require.NotNil(t, cfg)

	cfg.Global.DataDir = utils.NewTestDir(t)
	cfg.Init.Genesis = KsmccGenesisPath

	defer utils.RemoveTestDir(t)

	err := InitNode(cfg)
	require.Nil(t, err)

	err = LoadConfig(cfg, KsmccConfigPath)

	// TODO: improve dot config tests
	require.Nil(t, err)
}

// TestExportConfigKsmcc tests exporting the toml configuration file
func TestExportConfigKsmcc(t *testing.T) {
	cfg := KsmccConfig()
	require.NotNil(t, cfg)

	ksmccGenesis := cfg.Init.Genesis
	ksmccDataDir := cfg.Global.DataDir
	cfg.Global.DataDir = utils.NewTestDir(t)
	cfg.Init.Genesis = KsmccGenesisPath

	defer utils.RemoveTestDir(t)

	err := InitNode(cfg)
	require.Nil(t, err)

	cfg.Init.Genesis = ksmccGenesis
	cfg.Global.DataDir = ksmccDataDir

	file := ExportConfig(cfg, KsmccConfigPath)

	// TODO: improve dot config tests
	require.NotNil(t, file)
}
