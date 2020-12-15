// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
)

// TestAccountGenerate test "gossamer account --generate"
func TestAccountGenerate(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --generate",
		[]string{"basepath", "generate"},
		[]interface{}{testDir, "true"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountGeneratePassword test "gossamer account --generate --password"
func TestAccountGeneratePassword(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=true"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --generate --password",
		[]string{"basepath", "generate", "password"},
		[]interface{}{testDir, "true", "1234"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountGenerateEd25519 test "gossamer account --generate --ed25519"
func TestAccountGenerateEd25519(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--ed25519"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --generate --ed25519",
		[]string{"basepath", "generate", "ed25519"},
		[]interface{}{testDir, "true", "ed25519"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountGenerateSr25519 test "gossamer account --generate --ed25519"
func TestAccountGenerateSr25519(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--sr25519"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --generate --sr25519",
		[]string{"basepath", "generate", "sr25519"},
		[]interface{}{testDir, "true", "sr25519"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountGenerateSecp256k1 test "gossamer account --generate --ed25519"
func TestAccountGenerateSecp256k1(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--generate=true", "--password=false", "--secp256k1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --generate --secp256k1",
		[]string{"basepath", "generate", "secp256k1"},
		[]interface{}{testDir, "true", "secp256k1"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountImport test "gossamer account --import"
func TestAccountImport(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	// TODO: Add import value for test
	err := app.Run([]string{"irrelevant", "account", directory, "--import=./test_inputs/test-key.key"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --import=./test_inputs/test-key.key",
		[]string{"basepath", "import"},
		[]interface{}{"./test_inputs/", "test-key.key"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountImport test "gossamer account --import-raw"
func TestAccountImportRaw(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	// TODO: Add import-raw value for test
	err := app.Run([]string{"irrelevant", "account", directory, `--import-raw=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09`, "--password=1234"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --import-raw=0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09 --password=1234",
		[]string{"import-raw", "password"},
		[]interface{}{"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09", "1234"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}

// TestAccountList test "gossamer account --list"
func TestAccountList(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)
	directory := fmt.Sprintf("--basepath=%s", testDir)
	err := app.Run([]string{"irrelevant", "account", directory, "--list"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := newTestContext(
		"Test gossamer account --list",
		[]string{"basepath", "list"},
		[]interface{}{testDir, "true"},
	)
	if err != nil {
		t.Fatal(err)
	}

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: check contents of data directory - improve cmd account tests
}
