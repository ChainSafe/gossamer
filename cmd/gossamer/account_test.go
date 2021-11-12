// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestAccountGenerate test "gossamer account --generate"
func TestAccountGenerate(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --generate",
		[]string{"basepath", "generate"},
		[]interface{}{testDir, "true"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountGeneratePassword test "gossamer account --generate --password"
func TestAccountGeneratePassword(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=true"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --generate --password",
		[]string{"basepath", "generate", "password"},
		[]interface{}{testDir, "true", "1234"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountGenerateEd25519 test "gossamer account --generate --ed25519"
func TestAccountGenerateEd25519(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--ed25519"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --generate --ed25519",
		[]string{"basepath", "generate", "ed25519"},
		[]interface{}{testDir, "true", "ed25519"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountGenerateSr25519 test "gossamer account --generate --ed25519"
func TestAccountGenerateSr25519(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--sr25519"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --generate --sr25519",
		[]string{"basepath", "generate", "sr25519"},
		[]interface{}{testDir, "true", "sr25519"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountGenerateSecp256k1 test "gossamer account --generate --ed25519"
func TestAccountGenerateSecp256k1(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--secp256k1"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --generate --secp256k1",
		[]string{"basepath", "generate", "secp256k1"},
		[]interface{}{testDir, "true", "secp256k1"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account --import"
func TestAccountImport(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)

	err := app.Run([]string{"irrelevant", "account", directory, "--import=./test_inputs/test-key.key"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --import=./test_inputs/test-key.key",
		[]string{"basepath", "import"},
		[]interface{}{"./test_inputs/", "test-key.key"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountImport test "gossamer account --import-raw"
func TestAccountImportRaw(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)

	err := app.Run([]string{"irrelevant", "account", directory, `--import-raw=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09`, "--password=1234"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --import-raw=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09 --password=1234",
		[]string{"import-raw", "password"},
		[]interface{}{"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09", "1234"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}

// TestAccountList test "gossamer account --list"
func TestAccountList(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--list"})
	require.NoError(t, err)

	ctx, err := newTestContext(
		"Test gossamer account --list",
		[]string{"basepath", "list"},
		[]interface{}{testDir, "true"},
	)
	require.NoError(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.NoError(t, err)
}
