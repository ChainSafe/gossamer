// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/cosmos/go-bip39"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
)

// newTestGenesis returns a test genesis instance using "gssmr" raw data
func newTestGenesis(t *testing.T) *genesis.Genesis {
	fp := utils.GetGssmrGenesisRawPathTest(t)

	gssmrGen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	return &genesis.Genesis{
		Name:       "test",
		ID:         "test",
		Bootnodes:  []string(nil),
		ProtocolID: "/gossamer/test/0",
		Genesis:    gssmrGen.GenesisFields(),
	}
}

// NewTestGenesisRawFile returns a test genesis file using "gssmr" raw data
func NewTestGenesisRawFile(t *testing.T, cfg *Config) (filename string) {
	filename = filepath.Join(t.TempDir(), "genesis.json")

	fp := utils.GetGssmrGenesisRawPathTest(t)

	gssmrGen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    gssmrGen.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}

// newTestGenesisFile returns a human-readable test genesis file using "gssmr" human readable data
func newTestGenesisFile(t *testing.T, cfg *Config) (filename string) {
	fp := utils.GetGssmrGenesisPathTest(t)

	gssmrGen, err := genesis.NewGenesisFromJSON(fp, 0)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    gssmrGen.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}

// NewTestGenesisAndRuntime create a new test runtime and a new test genesis
// file with the test runtime stored in raw data and returns the genesis file
func NewTestGenesisAndRuntime(t *testing.T) (filename string) {
	_ = wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)
	runtimeFilePath := runtime.GetAbsolutePath(runtime.NODE_RUNTIME_FP)

	runtimeData, err := os.ReadFile(filepath.Clean(runtimeFilePath))
	require.NoError(t, err)

	gen := newTestGenesis(t)
	hex := hex.EncodeToString(runtimeData)

	gen.Genesis.Raw = map[string]map[string]string{}
	if gen.Genesis.Raw["top"] == nil {
		gen.Genesis.Raw["top"] = make(map[string]string)
	}
	gen.Genesis.Raw["top"]["0x3a636f6465"] = "0x" + hex
	gen.Genesis.Raw["top"]["0xcf722c0832b5231d35e29f319ff27389f5032bfc7bfc3ba5ed7839f2042fb99f"] = "0x0000000000000001"

	genData, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, genData, os.ModePerm)
	require.NoError(t, err)

	return filename
}

// NewTestConfig returns a new test configuration using the provided basepath
func NewTestConfig(t *testing.T) *Config {
	dir := t.TempDir()

	cfg := &Config{
		Global: GlobalConfig{
			Name:        GssmrConfig().Global.Name,
			ID:          GssmrConfig().Global.ID,
			BasePath:    dir,
			LogLvl:      log.Info,
			NoTelemetry: true,
		},
		Log:     GssmrConfig().Log,
		Init:    GssmrConfig().Init,
		Account: GssmrConfig().Account,
		Core:    GssmrConfig().Core,
		Network: GssmrConfig().Network,
		RPC:     GssmrConfig().RPC,
	}

	return cfg
}

// newTestConfigWithFile returns a new test configuration and a temporary configuration file
func newTestConfigWithFile(t *testing.T) (*Config, *os.File) {
	cfg := NewTestConfig(t)

	configPath := filepath.Join(cfg.Global.BasePath, "config.toml")
	err := os.WriteFile(configPath, nil, os.ModePerm)
	require.NoError(t, err)

	cfgFile := exportConfig(cfg, configPath)
	return cfg, cfgFile
}

// exportConfig exports a dot configuration to a toml configuration file
func exportConfig(cfg *Config, fp string) *os.File {
	raw, err := toml.Marshal(*cfg)
	if err != nil {
		logger.Errorf("failed to marshal configuration: %s", err)
		os.Exit(1)
	}
	return writeConfig(raw, fp)
}

// ExportTomlConfig exports a dot configuration to a toml configuration file
func ExportTomlConfig(cfg *ctoml.Config, fp string) *os.File {
	raw, err := toml.Marshal(*cfg)
	if err != nil {
		logger.Errorf("failed to marshal configuration: %s", err)
		os.Exit(1)
	}
	return writeConfig(raw, fp)
}

// writeConfig writes the config `data` in the file 'fp'.
func writeConfig(data []byte, fp string) *os.File {
	newFile, err := os.Create(filepath.Clean(fp))
	if err != nil {
		logger.Errorf("failed to create configuration file: %s", err)
		os.Exit(1)
	}

	_, err = newFile.Write(data)
	if err != nil {
		logger.Errorf("failed to write to configuration file: %s", err)
		os.Exit(1)
	}

	if err := newFile.Close(); err != nil {
		logger.Errorf("failed to close configuration file: %s", err)
		os.Exit(1)
	}

	return newFile
}

// CreateJSONRawFile will generate a JSON genesis file with raw storage
func CreateJSONRawFile(bs *BuildSpec, fp string) *os.File {
	data, err := bs.ToJSONRaw()
	if err != nil {
		logger.Errorf("failed to convert into raw json: %s", err)
		os.Exit(1)
	}
	return writeConfig(data, fp)
}

// RandomNodeName generates a new random name if there is no name configured for the node
func RandomNodeName() string {
	entropy, _ := bip39.NewEntropy(128)
	randomNamesString, _ := bip39.NewMnemonic(entropy)
	randomNames := strings.Split(randomNamesString, " ")
	number := binary.BigEndian.Uint16(entropy)
	return randomNames[0] + "-" + randomNames[1] + "-" + fmt.Sprint(number)
}
