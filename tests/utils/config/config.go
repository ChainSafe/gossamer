package config

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/stretchr/testify/require"
)

// CreateDefault generates a default config and writes
// it to a temporary file for the current test.
func CreateDefault(t *testing.T) (configPath string) {
	cfg := generateDefaultConfig()
	return writeTestTOMLConfig(t, cfg)
}

// CreateLogGrandpa generates a grandpa config and writes
// it to a temporary file for the current test.
func CreateLogGrandpa(t *testing.T) (configPath string) {
	cfg := generateDefaultConfig()
	cfg.Log = ctoml.LogConfig{
		CoreLvl:           "crit",
		NetworkLvl:        "debug",
		RuntimeLvl:        "crit",
		BlockProducerLvl:  "info",
		FinalityGadgetLvl: "debug",
	}
	return writeTestTOMLConfig(t, cfg)
}

// CreateNoBabe generates a no-babe config and writes
// it to a temporary file for the current test.
func CreateNoBabe(t *testing.T) (configPath string) {
	cfg := generateDefaultConfig()
	cfg.Global.LogLvl = "info"
	cfg.Log = ctoml.LogConfig{
		SyncLvl:    "debug",
		NetworkLvl: "debug",
	}
	cfg.Core.BabeAuthority = false
	return writeTestTOMLConfig(t, cfg)
}

// CreateNoGrandpa generates an no-grandpa config and writes
// it to a temporary file for the current test.
func CreateNoGrandpa(t *testing.T) (configPath string) {
	t.Helper()
	cfg := generateDefaultConfig()
	cfg.Core.GrandpaAuthority = false
	cfg.Core.BABELead = true
	cfg.Core.GrandpaInterval = 1
	return writeTestTOMLConfig(t, cfg)
}

// CreateNotAuthority generates an non-authority config and writes
// it to a temporary file for the current test.
func CreateNotAuthority(t *testing.T) (configPath string) {
	t.Helper()
	cfg := generateDefaultConfig()
	cfg.Core.Roles = 1
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	return writeTestTOMLConfig(t, cfg)
}

func writeTestTOMLConfig(t *testing.T, cfg *ctoml.Config) (configPath string) {
	t.Helper()
	configPath = filepath.Join(t.TempDir(), "config.toml")
	err := dot.ExportTomlConfig(cfg, configPath)
	require.NoError(t, err)
	return configPath
}
