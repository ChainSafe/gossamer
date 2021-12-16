// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package polkadotjs_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/assert"
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

	nodes, err := utils.InitializeAndStartNodesWebsocket(t, 1, utils.GenesisDev, utils.ConfigDefault)
	require.NoError(t, err)

	node := nodes[0]

	const command = "npx mocha ./test --timeout 30000"

	err = os.Setenv("WSHOST", fmt.Sprintf("ws://127.0.0.1:%d", node.WSPort))
	require.NoError(t, err)

	defer os.Unsetenv("WSHOST")

	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()

	fmt.Println(len(data))
	//uncomment this to see log results from javascript tests
	fmt.Printf("%s\n", data)

	assert.NoError(t, err, string(data))

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
