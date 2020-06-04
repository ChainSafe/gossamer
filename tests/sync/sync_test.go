package sync

import (
	"fmt"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var framework utils.Framework

func TestMain(m *testing.M) {
	fw, err := utils.InitFramework(3)
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

	framework.StoreChainHeads()

	err = framework.KillNodes(t)
	require.Len(t, err, 0)
}
