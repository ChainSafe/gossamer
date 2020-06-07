package sync

import (
	"fmt"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var framework utils.Framework
type testRPCCall struct {
	nodeIdx int
	method string
	params string
	delay time.Duration
}

var tests = []testRPCCall {
	{nodeIdx: 0, method:  "chain_getHeader", params:  "[]", delay: time.Second},
	{nodeIdx: 0, method:  "state_getRuntimeVersion", params:  "[]", delay: time.Second},
}
func TestMain(m *testing.M) {
	fw, err := utils.InitFramework(1)
	if err != nil {
		fmt.Errorf("error initializing test framework")
	}
	framework = *fw
	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestSyncSetup(t *testing.T) {
	err := framework.StartNodes(t)
	require.Len(t, err, 0)

	//framework.StoreChainHeads()
	framework.PrintDB(0)
	err = framework.KillNodes(t)
	require.Len(t, err, 0)
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

func TestCalls(t *testing.T) {
	err := framework.StartNodes(t)
	require.Len(t, err, 0)
	for _, call := range tests {
		time.Sleep(call.delay)
		framework.CallRPC(call.nodeIdx, call.method, call.params)
	}
	v := framework.GetRecord(0, 0)
	fmt.Printf("get Record %v\n", v)
	err = framework.KillNodes(t)
	require.Len(t, err, 0)
}