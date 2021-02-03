package polkadotjs_test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os/exec"
	"strings"
	"testing"
)

//func TestMain(m *testing.M) {
//	_, _ = fmt.Fprintln(os.Stdout, "Going to start RPC suite test")
//
//	utils.CreateDefaultConfig()
//	defer os.Remove(utils.ConfigDefault)
//
//	// Start all tests
//	//code := m.Run()
//	//os.Exit(code)
//	m.Run()
//}
//var stopChan = make (chan string)

func TestStartGossamer(t *testing.T) {
	//t.Log("starting gossamer...")
	//utils.CreateDefaultConfig()
	//	defer os.Remove(utils.ConfigDefault)
	//
	//nodes, err := utils.InitializeAndStartNodes(t, 3, utils.GenesisDefault, utils.ConfigDefault)
	//fmt.Printf("nodes: %v\n error %v\n", nodes, err)
	//for {
	//	stop := <- stopChan
	//	fmt.Printf("stop %v\n", stop)
	//}

	command := "npm test --prefix tests/polkadotjs_test/"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	require.NoError(t, err)
	fmt.Printf("data %s\n", data)
}

func TestStopGossamer(t *testing.T) {
	//stopChan <- "foo"
}