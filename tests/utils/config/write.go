// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"os"
	"path/filepath"
	"testing"

	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
)

// Write writes the toml configuration to a file
// in a temporary test directory which gets removed at
// the end of the test.
func Write(t *testing.T, cfg cfg.Config) (configPath string) {
	t.Helper()
	configPath = filepath.Join(t.TempDir(), "config.toml")
	raw, err := toml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(configPath, raw, os.ModePerm)
	require.NoError(t, err)
	return configPath
}
