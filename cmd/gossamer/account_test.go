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
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestAccountGenerate test "gossamer account --generate"
func TestAccountGenerate(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	// TODO: implement table driven tests #130 and add more test cases #685

	ctx, err := newTestContext(
		"Test gossamer account --generate",
		[]string{"datadir", "generate"},
		[]interface{}{testDir, true},
	)
	require.Nil(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.Nil(t, err)

	// TODO: add more require checks #685
}

// TestAccountGeneratePassword test "gossamer account --generate --password"
func TestAccountGeneratePassword(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	// TODO: implement table driven tests #130 and add more test cases #685

	ctx, err := newTestContext(
		"Test gossamer account --generate --password",
		[]string{"datadir", "generate", "password"},
		[]interface{}{testDir, true, "1234"},
	)
	require.Nil(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.Nil(t, err)

	// TODO: add more require checks #685
}

// TestAccountGenerateType test "gossamer account --generate --type"
func TestAccountGenerateType(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	// TODO: implement table driven tests #130 and add more test cases #685

	ctx, err := newTestContext(
		"Test gossamer account --generate --type",
		[]string{"datadir", "generate", "type"},
		[]interface{}{testDir, true, "ed25519"},
	)
	require.Nil(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.Nil(t, err)

	// TODO: add more require checks #685
}

// TestAccountImport test "gossamer account --import"
func TestAccountImport(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	// TODO: implement table driven tests #130 and add more test cases #685

	ctx, err := newTestContext(
		"Test gossamer account --import",
		[]string{"datadir", "import"},
		[]interface{}{testDir, "testfile"},
	)
	require.Nil(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.Nil(t, err)

	// TODO: add more require checks #685
}

// TestAccountList test "gossamer account --list"
func TestAccountList(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	// TODO: implement table driven tests #130 and add more test cases #685

	ctx, err := newTestContext(
		"Test gossamer account --list",
		[]string{"datadir", "list"},
		[]interface{}{testDir, true},
	)
	require.Nil(t, err)

	command := accountCommand
	err = command.Run(ctx)
	require.Nil(t, err)

	// TODO: add more require checks #685
}
