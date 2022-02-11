// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package dot

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportConfigIntegration(t *testing.T) {
	cfg, cfgFile := newTestConfigWithFile(t)

	genFile := newTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	file := exportConfig(cfg, cfgFile.Name())
	require.NotNil(t, file)
	os.Remove(file.Name())
}
