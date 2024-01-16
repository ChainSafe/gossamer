// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildSpec test "gossamer build-spec --chain=chain-spec-raw.json"
func TestBuildSpec(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--chain", testChainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestBuildSpecRaw test "gossamer build-spec --chain=chain-spec-raw.json --raw"
func TestBuildSpecRaw(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--chain", testChainSpec, "--raw"})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestBuildSpecFromDB test init and build-spec
//
//	"gossamer init --chain chain-spec-raw.json --base-path=basepath && \
//		gossamer build-spec --base-path=basepath"
func TestBuildSpecFromDB(t *testing.T) {
	basepath := t.TempDir()

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(BuildSpecCmd, InitCmd)

	// Init the node
	rootCmd.SetArgs([]string{InitCmd.Name(), "--base-path", basepath, "--chain", testChainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)

	rootCmd.SetArgs([]string{BuildSpecCmd.Name(), "--base-path", basepath})
	err = rootCmd.Execute()
	require.NoError(t, err)
}
