// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

const portsEnd = 8000

type portsQueue chan int

func (p portsQueue) get() int {
	return <-p
}

func (p portsQueue) put(port int) {
	p <- port
}

var availablePorts portsQueue

func init() {
	availablePorts = make(portsQueue, portsEnd)
	for port := 7001; port <= portsEnd; port++ {
		availablePorts <- port
	}
}

var (
	currdir, _  = os.Getwd()
	gossamerBin = filepath.Join(currdir, "../..", "bin/gossamer")
)

type node struct {
	p2pAddr, basepath, bootnodes, config, genesis, key string
	rpcport, wsport, port                              int
	babelead, enablews                                 bool
}

func initializeGossamer(t *testing.T, node *node) {
	t.Helper()

	cmdInit := exec.Command(gossamerBin, "init",
		"--config", node.config,
		"--basepath", node.basepath,
		"--genesis", node.genesis,
		"--force",
	)

	t.Logf("initialising gossamer using %s ...", cmdInit.String())

	err := cmdInit.Start()
	require.NoError(t, err)
}

func runGossamerNode(t *testing.T, args ...string) {
	t.Helper()

	gossamerCmd := exec.Command(gossamerBin, args...)

	t.Logf("starting gossamer at %v...", gossamerCmd.String())

	err := gossamerCmd.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		gossamerCmd.Process.Kill()
	})
}

func runGossamerAndGetP2PAddress(t *testing.T, args ...string) (p2p string) {
	t.Helper()

	gossamerCmd := exec.Command(gossamerBin, args...)
	stdoutPipe, err := gossamerCmd.StdoutPipe()
	// use the same output for stderr
	gossamerCmd.Stderr = gossamerCmd.Stdout

	require.NoError(t, err)

	rule := regexp.MustCompile("/ip4/[0-9.]+/tcp/[0-9]+/p2p/[a-zA-Z0-9]+")

	duration := time.Second * 15
	timeoutCtx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	done := make(chan string)
	scanner := bufio.NewScanner(stdoutPipe)

	go func() {
		for scanner.Scan() {
			if timeoutCtx.Err() != nil {
				done <- ""
				return
			}

			outputLine := scanner.Text()
			addr := rule.FindStringSubmatch(outputLine)

			if len(addr) != 0 {
				done <- addr[0]
				return
			}
		}

		done <- ""
	}()

	t.Cleanup(func() {
		gossamerCmd.Process.Kill()
	})

	gossamerCmd.Start()
	t.Logf("starting gossamer using %v...", gossamerCmd.String())
	p2p = <-done

	return p2p
}

func buildGossamerParams(t *testing.T, node *node) []string {
	params := []string{
		"--port", fmt.Sprint(node.port),
		"--config", node.config,
		"--basepath", node.basepath,
		"--rpchost", "127.0.0.1",
		"--rpcport", fmt.Sprint(node.rpcport),
		"--rpcmods", "system,author,chain,state,dev,rpc",
		"--rpc",
		"--rpc-unsafe",
		"--log", "info"}

	if node.babelead {
		params = append(params, "--babe-lead")
	}

	if node.enablews {
		params = append(params, "--ws", "--wsport", fmt.Sprint(node.wsport))
	}

	if strings.TrimSpace(node.key) != "" {
		params = append(params, "--roles", "4", "--key", node.key)
	}

	if strings.TrimSpace(node.bootnodes) != "" {
		params = append(params, "--bootnodes", node.bootnodes)
	}

	return params
}

func checkGossamerRPCIsOK(t *testing.T, node *node) error {
	const retries = 5
	for i := 0; i < retries; i++ {
		uri := fmt.Sprintf("http://localhost:%d", node.rpcport)
		err := utils.CheckNodeStarted(t, uri)

		if err == nil {
			return nil
		}

		t.Log(err)
		time.Sleep(time.Second * 5)
	}

	return errors.New("gossamer node could not be started")
}

func spinUpGossamerNode(t *testing.T, node *node) error {
	t.Helper()

	initializeGossamer(t, node)

	params := buildGossamerParams(t, node)
	runGossamerNode(t, params...)

	return checkGossamerRPCIsOK(t, node)
}

func TestE2ESystemRPC(t *testing.T) {
	port := availablePorts.get()
	rpcport := availablePorts.get()

	defer func() {
		availablePorts.put(port)
		availablePorts.put(rpcport)
	}()

	aliceKey := utils.KeyList[0]

	firstTmpdir := t.TempDir()
	basepath := filepath.Join(firstTmpdir, t.Name(), aliceKey)

	firstNode := &node{
		basepath: basepath,
		genesis:  utils.GenesisDefault,
		config:   utils.ConfigDefault,
		port:     port,
		rpcport:  port,
		babelead: true,
		key:      aliceKey,
	}

	initializeGossamer(t, firstNode)
	params := buildGossamerParams(t, firstNode)

	firstP2PAddress := runGossamerAndGetP2PAddress(t, params...)
	require.NotEmpty(t, firstP2PAddress)

	err := checkGossamerRPCIsOK(t, firstNode)
	require.NoError(t, err)

	secondPort := availablePorts.get()
	secondRPCPort := availablePorts.get()

	bobKey := utils.KeyList[1]

	secondDir := t.TempDir()
	secondBasepath := filepath.Join(secondDir, t.Name(), bobKey)

	secondNode := &node{
		basepath:  secondBasepath,
		genesis:   utils.GenesisDefault,
		config:    utils.ConfigDefault,
		port:      secondPort,
		rpcport:   secondRPCPort,
		bootnodes: firstP2PAddress,
		key:       bobKey,
	}

	err = spinUpGossamerNode(t, secondNode)
	require.NoError(t, err)
}

func TestSystemRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test system_name",
			method:      "system_name",
			skip:        true,
		},
		{ //TODO
			description: "test system_version",
			method:      "system_version",
			skip:        true,
		},
		{ //TODO
			description: "test system_chain",
			method:      "system_chain",
			skip:        true,
		},
		{ //TODO
			description: "test system_properties",
			method:      "system_properties",
			skip:        true,
		},
		{
			description: "test system_health",
			method:      "system_health",
			expected: modules.SystemHealthResponse{
				Peers:           2,
				IsSyncing:       true,
				ShouldHavePeers: true,
			},
			params: "{}",
		},
		{
			description: "test system_peers",
			method:      "system_peers",
			expected:    modules.SystemPeersResponse{},
			params:      "{}",
		},
		{
			description: "test system_network_state",
			method:      "system_networkState",
			expected: modules.SystemNetworkStateResponse{
				NetworkState: modules.NetworkStateString{
					PeerID: "",
				},
			},
			params: "{}",
		},
		{ //TODO
			description: "test system_addReservedPeer",
			method:      "system_addReservedPeer",
			skip:        true,
		},
		{ //TODO
			description: "test system_removeReservedPeer",
			method:      "system_removeReservedPeer",
			skip:        true,
		},
		{ //TODO
			description: "test system_nodeRoles",
			method:      "system_nodeRoles",
			skip:        true,
		},
		{ //TODO
			description: "test system_accountNextIndex",
			method:      "system_accountNextIndex",
			skip:        true,
		},
	}

	t.Log("starting gossamer...")
	nodes, err := utils.InitializeAndStartNodes(t, 3, utils.GenesisDefault, utils.ConfigDefault)

	//use only first server for tests
	require.Nil(t, err)
	node := nodes[2]

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			target := getResponse(t, test, node.RPCPort)

			switch v := target.(type) {
			case *modules.SystemHealthResponse:
				t.Log("Will assert SystemHealthResponse", "target", target)

				require.Equal(t, test.expected.(modules.SystemHealthResponse).IsSyncing, v.IsSyncing)
				require.Equal(t, test.expected.(modules.SystemHealthResponse).ShouldHavePeers, v.ShouldHavePeers)
				require.GreaterOrEqual(t, v.Peers, test.expected.(modules.SystemHealthResponse).Peers)

			case *modules.SystemNetworkStateResponse:
				t.Log("Will assert SystemNetworkStateResponse", "target", target)

				require.NotNil(t, v.NetworkState)
				require.NotNil(t, v.NetworkState.PeerID)

			case *modules.SystemPeersResponse:
				t.Log("Will assert SystemPeersResponse", "target", target)

				require.NotNil(t, v)

				//TODO: #807
				//this assertion requires more time on init to be enabled
				//require.GreaterOrEqual(t, len(v.Peers), 2)

				for _, vv := range *v {
					require.NotNil(t, vv.PeerID)
					require.NotNil(t, vv.Roles)
					require.NotNil(t, vv.BestHash)
					require.NotNil(t, vv.BestNumber)
				}

			}

		})
	}

	t.Log("going to tear down gossamer...")

	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
