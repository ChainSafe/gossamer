// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"
	"regexp"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/stretchr/testify/require"
)

const RegExVersion = "^([0-9]+).([0-9]+).([0-9]+)(?:-([0-9A-Za-z-]+(?:-[0-9A-Za-z-]+)*))?$"

func TestVersionCommand(t *testing.T) {
	rootCmd, err := NewRootCommand()
	rootCmd.AddCommand(VersionCmd)
	rootCmd.SetArgs([]string{VersionCmd.Name()})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

func TestVersionString(t *testing.T) {
	stableVersion := cfg.GetStableVersion()
	stableMatch, err := regexp.MatchString(RegExVersion, stableVersion)
	require.NoError(t, err)
	require.True(t, stableMatch)

	dirtyVersion := cfg.GetFullVersion()
	dirtyMatch, err := regexp.MatchString(RegExVersion, dirtyVersion)
	require.NoError(t, err)
	require.True(t, dirtyMatch)
}
