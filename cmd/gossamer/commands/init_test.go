// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitFromChainSpec(t *testing.T) {
	basepath := t.TempDir()
	chainSpec := "./test_inputs/test-chain-spec-raw.json"

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(InitCmd)

	rootCmd.SetArgs([]string{InitCmd.Name(), "--base-path", basepath, "--chain", chainSpec})
	err = rootCmd.Execute()
	require.NoError(t, err)
}
