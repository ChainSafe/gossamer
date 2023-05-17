// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testChainSpec = "./test_inputs/test-chain-spec-raw.json"

// TestInitFromChainSpec test "gossamer init --chain=./test_inputs/test-chain-spec-raw.json"
func TestInitFromChainSpec(t *testing.T) {
	basepath := t.TempDir()

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(InitCmd)

	rootCmd.SetArgs([]string{InitCmd.Name(), "--base-path", basepath, "--chain", testChainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)
}
