// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPathExists tests the PathExists method
func TestPathExists(t *testing.T) {
	require.Equal(t, PathExists("../utils"), true)
	require.Equal(t, PathExists("../utilzzz"), false)
}

// TestHomeDir tests the HomeDir method
func TestHomeDir(t *testing.T) {
	const envHomeValue = "/home/test"
	t.Setenv("HOME", envHomeValue)
	homeDir := HomeDir()
	assert.Equal(t, envHomeValue, homeDir)

	t.Setenv("HOME", "")
	homeDir = HomeDir()
	assert.NotEmpty(t, homeDir)
}

// TestExpandDir tests the ExpandDir method
func TestExpandDir(t *testing.T) {
	homeDir := HomeDir()

	const tildePath = "~/.local/share/gossamer/test"
	expandedTildePath := ExpandDir(tildePath)
	assert.Equal(t, homeDir+"/.local/share/gossamer/test", expandedTildePath)

	const absPath = "/tmp/absolute"
	expandedAbsPath := ExpandDir(absPath)
	assert.Equal(t, absPath, expandedAbsPath)
}

func TestBasePath(t *testing.T) {
	const pathSuffix = "sometestdirectory"

	basePath := BasePath(pathSuffix)

	assert.NotEqual(t, pathSuffix, basePath)
	assert.True(t, strings.HasSuffix(basePath, pathSuffix))
	assert.True(t, strings.HasPrefix(basePath, HomeDir()))
}

func TestKeystoreDir(t *testing.T) {
	testDir := t.TempDir()

	keystoreDir, err := KeystoreDir(testDir)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(testDir, "keystore"), keystoreDir)
}
