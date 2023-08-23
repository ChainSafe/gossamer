// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportStateMissingStateFile(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name()})
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "state-file must be specified")
}

func TestImportStateInvalidFirstSlot(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name(), "--first-slot", "wrong"})
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "invalid argument \"wrong\"")
}

func TestImportStateEmptyStateFile(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name(), "--state-version", "v0", "--state-file", ""})
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "state-file must be specified")
}

func TestImportStateEmptyHeaderFile(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name(), "--state-version", "v0", "--state-file", "test", "--header-file", ""}) //nolint:lll
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "header-file must be specified")
}

func TestImportStateInvalidStateVersion(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name(), "--state-version", "v999", "--state-file", "test", "--header-file", "test"}) //nolint:lll
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "failed to parse state-version: parsing version failed: \"v999\" must be one of [v0, v1]")
}

func TestImportStateErrorImportingState(t *testing.T) {
	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(ImportStateCmd)

	rootCmd.SetArgs([]string{ImportStateCmd.Name(), "--state-version", "v0", "--state-file", "test", "--header-file", "test"}) //nolint:lll
	err = rootCmd.Execute()
	assert.ErrorContains(t, err, "no such file or directory")
}
