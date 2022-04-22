// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/stretchr/testify/require"
)

// Write writes the toml configuration to a file
// in a temporary test directory which gets removed at
// the end of the test.
func Write(t *testing.T, cfg ctoml.Config) (configPath string) {
	t.Helper()
	configPath = filepath.Join(t.TempDir(), "config.toml")
	err := dot.ExportTomlConfig(&cfg, configPath)
	require.NoError(t, err)
	return configPath
}
