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

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestExportCommand test "gossamer export --config"
func TestExportCommand(t *testing.T) {
	_, testConfig := dot.NewTestConfigWithFile(t)
	defer utils.RemoveTestDir(t)

	testName := "testnode"
	testBootnode := "bootnode"
	testProtocol := "/gossamer/test/0"
	testConfigPath := testConfig.Name()

	ctx, err := newTestContext(
		"Test gossamer export --config --name --bootnodes --protocol --force",
		[]string{"config", "name", "bootnodes", "protocol", "force"},
		[]interface{}{testConfigPath, testName, testBootnode, testProtocol, true},
	)
	require.Nil(t, err)

	err = exportCommand.Run(ctx)
	require.Nil(t, err)

	configExists := utils.PathExists(testConfigPath)
	require.Equal(t, true, configExists)

	testCfg := new(dot.Config)

	err = dot.LoadConfig(testCfg, testConfigPath)
	require.Nil(t, err)

	expected := DefaultCfg
	expected.Global.Name = testName
	expected.Network.Bootnodes = []string{testBootnode}
	expected.Network.ProtocolID = testProtocol

	require.Equal(t, expected, testCfg)
}
