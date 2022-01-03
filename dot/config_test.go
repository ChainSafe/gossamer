// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExportConfig tests exporting a toml configuration file
func TestExportConfig(t *testing.T) {
	cfg, cfgFile := newTestConfigWithFile(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	file := exportConfig(cfg, cfgFile.Name())
	require.NotNil(t, file)
}
