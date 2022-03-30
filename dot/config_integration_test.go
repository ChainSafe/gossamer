// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportConfig(t *testing.T) {
	cfg := NewTestConfig(t)
	configPath := filepath.Join(cfg.Global.BasePath, "config.toml")

	exportConfig(t, cfg, configPath)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	exportConfig(t, cfg, configPath)
}
