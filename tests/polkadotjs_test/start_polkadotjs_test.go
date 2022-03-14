// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package polkadotjs_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var polkadotSuite = "polkadot"

func TestStartGossamerAndPolkadotAPI(t *testing.T) {
	if utils.MODE != polkadotSuite {
		t.Log("Going to skip polkadot.js/api suite tests")
		return
	}
	t.Log("starting gossamer for polkadot.js/api tests...")

	utils.CreateDefaultConfig()
	defer os.Remove(utils.ConfigDefault)

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	nodes, err := utils.InitializeAndStartNodesWebsocket(t, 1, genesisPath, utils.ConfigDefault)
	require.NoError(t, err)

	command := "npx mocha ./test --timeout 30000"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	assert.NoError(t, err, string(data))

	//uncomment this to see log results from javascript tests
	//fmt.Printf("%s\n", data)

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
