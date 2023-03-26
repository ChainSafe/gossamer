package commands

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestAccountGenerate test "gossamer account generate"
func TestAccountGenerate(t *testing.T) {
	testDir := t.TempDir()

	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	accountCmd.SetArgs([]string{"generate", directory})
	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGeneratePassword test "gossamer account generate --password=VerySecurePassword"
func TestAccountGeneratePassword(t *testing.T) {
	testDir := t.TempDir()

	directory := fmt.Sprintf("--keystore-path=%s", testDir)
	accountCmd.SetArgs([]string{"generate", directory, "--password=VerySecurePassword"})

	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateEd25519 test "gossamer account generate --scheme=ed25519"
func TestAccountGenerateEd25519(t *testing.T) {
	testDir := t.TempDir()

	directory := fmt.Sprintf("--keystore-path=%s", testDir)
	accountCmd.SetArgs([]string{"generate", directory, "--scheme=ed25519"})

	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateSr25519 test "gossamer account generate --scheme=sr25519"
func TestAccountGenerateSr25519(t *testing.T) {
	testDir := t.TempDir()

	directory := fmt.Sprintf("--keystore-path=%s", testDir)
	accountCmd.SetArgs([]string{"generate", directory, "--scheme=sr25519"})

	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountGenerateSecp256k1 test "gossamer account generate --scheme=secp256k1"
func TestAccountGenerateSecp256k1(t *testing.T) {
	testDir := t.TempDir()

	directory := fmt.Sprintf("--keystore-path=%s", testDir)
	accountCmd.SetArgs([]string{"generate", directory, "--scheme=secp256k1"})

	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account import"
func TestAccountImport(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)
	accountCmd.SetArgs([]string{"import", directory, "--keystore-file=./test_inputs/test-key.key"})

	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account import-raw --password --key"
func TestAccountImportRaw(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	accountCmd.SetArgs([]string{"import-raw",
		directory,
		"--keystore-file=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"--password=VerySecurePassword"})
	err := accountCmd.Execute()
	require.NoError(t, err)
}

// TestAccountList test "gossamer account --list"
func TestAccountList(t *testing.T) {
	testDir := t.TempDir()
	directory := fmt.Sprintf("--keystore-path=%s", testDir)

	accountCmd.SetArgs([]string{"list", directory})
	err := accountCmd.Execute()
	require.NoError(t, err)
}
