// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package polkadotjs_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
)

var polkadotSuite = "polkadot"

func TestStartGossamerAndPolkadotAPI(t *testing.T) {
	if utils.MODE != polkadotSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip polkadot.js/api suite tests")
		return
	}
	t.Log("starting gossamer for polkadot.js/api tests...")

	utils.CreateDefaultConfig()
	defer os.Remove(utils.ConfigDefault)
	utils.GenerateGenesisOneAuth()
	defer os.Remove(utils.GenesisOneAuth)
	utils.CreateConfigBabeMaxThreshold()
	defer os.Remove(utils.ConfigBABEMaxThreshold)

	nodes, err := utils.InitializeAndStartNodesWebsocket(t, 1, utils.GenesisOneAuth, utils.ConfigBABEMaxThreshold)
	require.NoError(t, err)

	command := "npx mocha ./test"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	require.NoError(t, err, fmt.Sprintf("%s", data))

	//uncomment this to see log results from javascript tests
	//fmt.Printf("%s\n", data)

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
