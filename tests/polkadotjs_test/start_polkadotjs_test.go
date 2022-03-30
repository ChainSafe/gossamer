// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package polkadotjs_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/stretchr/testify/assert"
)

var polkadotSuite = "polkadot"

func TestStartGossamerAndPolkadotAPI(t *testing.T) {
	if utils.MODE != polkadotSuite {
		t.Log("Going to skip polkadot.js/api suite tests")
		return
	}
	t.Log("starting gossamer for polkadot.js/api tests...")

	config := config.CreateDefault(t)

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	n := node.New(t, node.SetGenesis(genesisPath), node.SetConfig(config))

	ctx, cancel := context.WithCancel(context.Background())
	n.InitAndStartTest(ctx, t, cancel)

	command := "npx mocha ./test --timeout 30000"
	parts := strings.Fields(command)
	data, err := exec.CommandContext(ctx, parts[0], parts[1:]...).Output()
	assert.NoError(t, err, string(data))

	//uncomment this to see log results from javascript tests
	//fmt.Printf("%s\n", data)
}
