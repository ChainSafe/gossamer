// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAccountGenerate test "gossamer account generate"
func TestAccountGenerate(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "generate", directory})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGeneratePassword test "gossamer account generate --password=VerySecurePassword"
func TestAccountGeneratePassword(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "generate", directory, "--password=VerySecurePassword"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateEd25519 test "gossamer account generate --scheme=ed25519"
func TestAccountGenerateEd25519(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "generate", directory, "--scheme=ed25519"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateSr25519 test "gossamer account generate --scheme=sr25519"
func TestAccountGenerateSr25519(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "generate", directory, "--scheme=sr25519"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateSecp256k1 test "gossamer account generate --scheme=secp256k1"
func TestAccountGenerateSecp256k1(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "generate", directory, "--scheme=secp256k1"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account import"
func TestAccountImport(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "import", directory, "--keystore-file=./test_inputs/test-key.key"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account import-raw --password --key"
func TestAccountImportRaw(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account",
		"import-raw",
		directory,
		"--keystore-file=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"--password=VerySecurePassword"})

	err = rootCmd.Execute()
	require.NoError(t, err)
}

// TestAccountList test "gossamer account --list"
func TestAccountList(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	rootCmd, err := NewRootCommand()
	require.NoError(t, err)
	rootCmd.AddCommand(AccountCmd)

	rootCmd.SetArgs([]string{"account", "list", directory})

	err = rootCmd.Execute()
	require.NoError(t, err)
}
