// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSpec(t *testing.T) {
	chainSpec := "./test_inputs/test-chain-spec-raw.json"

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--chain", chainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

func TestBuildSpecRaw(t *testing.T) {
	chainSpec := "./test_inputs/test-chain-spec-raw.json"

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--chain", chainSpec, "--raw"})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

func TestBuildSpecFromDB(t *testing.T) {
	chainSpec := "./test_inputs/test-chain-spec-raw.json"
	basepath := t.TempDir()

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd, InitCmd)

	// Init the node
	rootCmd.SetArgs([]string{InitCmd.Name(), "--base-path", basepath, "--chain", chainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--base-path", basePath})
	err = rootCmd.Execute()
	require.NoError(t, err)
}
