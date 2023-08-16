// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupAndClearDatabase(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup database and execute some operations
	db, err := LoadDatabase(tmpDir, false)
	require.NoError(t, err)

	err = db.Put([]byte("key"), []byte("value"))
	require.NoError(t, err)

	value, err := db.Get([]byte("key"))
	require.NoError(t, err)
	require.Equal(t, []byte("value"), value)

	err = db.Close()
	require.NoError(t, err)

	shouldExists := true
	checkDatbaseDirectory(t, tmpDir, shouldExists)

	ClearDatabase(tmpDir)

	shouldExists = false
	checkDatbaseDirectory(t, tmpDir, shouldExists)

	// Setup database after an clear operation
	// should be okay
	db, err = LoadDatabase(tmpDir, false)
	require.NoError(t, err)

	shouldExists = true
	checkDatbaseDirectory(t, tmpDir, shouldExists)

	err = db.Close()
	require.NoError(t, err)
}

func checkDatbaseDirectory(t *testing.T, dir string, shouldExists bool) {
	t.Helper()

	databaseDir := filepath.Join(dir, DefaultDatabaseDir)
	entries, err := os.ReadDir(databaseDir)
	if !shouldExists {
		require.True(t, os.IsNotExist(err))
		return
	}

	require.NoError(t, err)
	if shouldExists {
		require.Greater(t, len(entries), 0)
	}
}
