package polkadotjs_test

import (
	"fmt"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestStartGossamer(t *testing.T) {
	t.Log("starting gossamer for polkadot.js/api tests...")

	utils.CreateDefaultConfig()
	defer os.Remove(utils.ConfigDefault)

	nodes, err := utils.InitializeAndStartNodesWebsocket(t, 1, utils.GenesisDefault, utils.ConfigDefault)
	require.NoError(t, err)

	command := "yarn run mocha"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	require.NoError(t, err, fmt.Sprintf("%s", data))

	// uncomment this to see log results from javascript tests
	//fmt.Printf("data %s\n", data)

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
