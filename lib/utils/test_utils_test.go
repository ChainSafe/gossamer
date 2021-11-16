// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewTestDir tests the NewTestDir method
func TestNewTestDir(t *testing.T) {
	testDir := NewTestDir(t)

	expected := path.Join(TestDir, t.Name())

	require.Equal(t, expected, testDir)
	require.Equal(t, PathExists(testDir), true)

	RemoveTestDir(t)
}

// TestNewTestBasePath tests the NewTestBasePath method
func TestNewTestBasePath(t *testing.T) {
	basePath := "test"

	testDir := NewTestBasePath(t, basePath)

	expected := path.Join(TestDir, t.Name(), basePath)

	require.Equal(t, expected, testDir)
	require.Equal(t, PathExists(testDir), true)

	RemoveTestDir(t)
}

// TestRemoveTestDir tests the RemoveTestDir method
func TestRemoveTestDir(t *testing.T) {
	testDir := NewTestDir(t)

	expected := path.Join(TestDir, t.Name())

	require.Equal(t, expected, testDir)
	require.Equal(t, PathExists(testDir), true)

	RemoveTestDir(t)
	require.Equal(t, PathExists(testDir), false)
}
