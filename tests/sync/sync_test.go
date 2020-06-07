package sync

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
)

var framework utils.Framework

type testRPCCall struct {
	nodeIdx int
	method  string
	params  string
	delay   time.Duration
}

var tests = []testRPCCall{
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: time.Second * 10},
	{nodeIdx: 1, method: "chain_getHeader", params: "[]", delay: time.Second * 10},
}

func TestMain(m *testing.M) {
	fw, err := utils.InitFramework(3)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing test framework"))
	}
	framework = *fw
	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestCallRPC(t *testing.T) {
	err := framework.StartNodes(t)
	require.Len(t, err, 0)
	framework.CallRPC(0, "chain_getHeader", "[]")
	framework.CallRPC(0, "chain_getHeader", "[]")
	framework.PrintDB(0)
	err = framework.KillNodes(t)
	require.Len(t, err, 0)
}

// this starts nodes and runs RPC calls (which loads db)
func TestCalls(t *testing.T) {
	err := framework.StartNodes(t)
	require.Len(t, err, 0)
	for _, call := range tests {
		time.Sleep(call.delay)
		_, err := framework.CallRPC(call.nodeIdx, call.method, call.params)
		require.NoError(t, err)
	}
	err = framework.KillNodes(t)
	require.Len(t, err, 0)
}
