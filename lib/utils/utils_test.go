// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPathExists tests the NewTestDir method
func TestPathExists(t *testing.T) {
	require.Equal(t, PathExists("../utils"), true)
	require.Equal(t, PathExists("../utilzzz"), false)
}

// TestHomeDir tests the HomeDir method
func TestHomeDir(t *testing.T) {
	require.NotEqual(t, HomeDir(), "")
}

// TestExpandDir tests the ExpandDir method
func TestExpandDir(t *testing.T) {
	testDirA := "~/.gossamer-test"

	homeDir := HomeDir()
	expandedDirA := ExpandDir(testDirA)

	require.NotEqual(t, testDirA, expandedDirA)
	require.Equal(t, strings.Contains(expandedDirA, homeDir), true)

	testDirB := NewTestBasePath(t, "test")
	defer RemoveTestDir(t)

	expandedDirB := ExpandDir(testDirB)

	require.Equal(t, testDirB, expandedDirB)
	require.Equal(t, strings.Contains(expandedDirB, homeDir), false)
}

// TestBasePath tests the BasePath method
func TestBasePath(t *testing.T) {
	testDir := NewTestBasePath(t, "test")
	defer RemoveTestDir(t)

	homeDir := HomeDir()
	basePath := BasePath(testDir)

	require.NotEqual(t, testDir, basePath)
	require.Equal(t, strings.Contains(basePath, homeDir), true)
}

// TestKeystoreDir tests the KeystoreDir method
func TestKeystoreDir(t *testing.T) {
	testDir := NewTestBasePath(t, "test")
	defer RemoveTestDir(t)

	homeDir := HomeDir()
	basePath := BasePath(testDir)

	keystoreDir, err := KeystoreDir(testDir)
	require.Nil(t, err)

	require.NotEqual(t, testDir, basePath)
	require.Equal(t, strings.Contains(keystoreDir, homeDir), true)
}
