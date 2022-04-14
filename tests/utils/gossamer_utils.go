// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
)

// Logger is the utils package local logger.
var Logger = log.NewFromGlobal(log.AddContext("pkg", "test/utils"))
var maxRetries = 24

var (
	// KeyList is the list of built-in keys
	KeyList  = []string{"alice", "bob", "charlie", "dave", "eve", "ferdie", "george", "heather", "ian"}
	basePort = 7000

	// BaseRPCPort is the starting RPC port for test nodes
	BaseRPCPort = 8540

	// BaseWSPort is the starting Websocket port for test nodes
	BaseWSPort = 8546

	currentDir, _ = os.Getwd()
	gossamerCMD   = filepath.Join(currentDir, "../..", "bin/gossamer")

	// GenesisOneAuth is the genesis file that has 1 authority
	GenesisOneAuth = filepath.Join(currentDir, "../utils/genesis_oneauth.json")
	// GenesisThreeAuths is the genesis file that has 3 authorities
	GenesisThreeAuths = filepath.Join(currentDir, "../utils/genesis_threeauths.json")
	// GenesisTwoAuthsSecondaryVRF0_9_10 is the genesis file that has 2 authorities and block production by
	// secondary VRF slots enabled
	GenesisTwoAuthsSecondaryVRF0_9_10 = filepath.Join(currentDir, "../utils/genesis_two_auths_secondaryvrf_0_9_10.json")

	// GenesisSixAuths is the genesis file that has 6 authorities
	GenesisSixAuths = filepath.Join(currentDir, "../utils/genesis_sixauths.json")
	// GenesisDefault is the default gssmr genesis file
	GenesisDefault = filepath.Join(currentDir, "../..", "chain/gssmr/genesis.json")
	// GenesisDev is the default dev genesis file
	GenesisDev = filepath.Join(currentDir, "../..", "chain/dev/genesis-spec.json")

	// ConfigDefault is the default config file
	ConfigDefault = filepath.Join(currentDir, "../utils/config_default.toml")
	// ConfigLogGrandpa is a config file where log levels are set to CRIT except for GRANDPA
	ConfigLogGrandpa = filepath.Join(currentDir, "../utils/config_log_grandpa.toml")
	// ConfigNoBABE is a config file with BABE disabled
	ConfigNoBABE = filepath.Join(currentDir, "../utils/config_nobabe.toml")
	// ConfigNoGrandpa is a config file with grandpa disabled
	ConfigNoGrandpa = filepath.Join(currentDir, "../utils/config_nograndpa.toml")
	// ConfigNotAuthority is a config file with no authority functionality
	ConfigNotAuthority = filepath.Join(currentDir, "../utils/config_notauthority.toml")
)

// Node represents a gossamer process
type Node struct {
	Process  *exec.Cmd
	Key      string
	RPCPort  string
	Idx      int
	basePath string
	config   string
	WSPort   string
	BABELead bool
}

// InitGossamer initialises given node number and returns node reference
func InitGossamer(idx int, basePath, genesis, config string) (
	node Node, err error) {
	cmdInit := exec.Command(gossamerCMD, "init",
		"--config", config,
		"--basepath", basePath,
		"--genesis", genesis,
		"--force",
	)

	Logger.Info("initialising gossamer using " + cmdInit.String() + "...")
	stdOutInit, err := cmdInit.CombinedOutput()
	if err != nil {
		fmt.Printf("%s", stdOutInit)
		return node, err
	}

	Logger.Infof("initialised gossamer node %d!", idx)
	return Node{
		Idx:      idx,
		RPCPort:  strconv.Itoa(BaseRPCPort + idx),
		WSPort:   strconv.Itoa(BaseWSPort + idx),
		basePath: basePath,
		config:   config,
	}, nil
}

// startGossamer starts given node
func startGossamer(t *testing.T, node Node, websocket bool) (
	updatedNode Node, err error) {
	var key string
	var params = []string{"--port", strconv.Itoa(basePort + node.Idx),
		"--config", node.config,
		"--basepath", node.basePath,
		"--rpchost", HOSTNAME,
		"--rpcport", node.RPCPort,
		"--rpcmods", "system,author,chain,state,dev,rpc",
		"--rpc",
		"--rpc-unsafe",
		"--no-telemetry",
		"--log", "info"}

	if node.BABELead {
		params = append(params, "--babe-lead")
	}

	if node.Idx >= len(KeyList) {
		params = append(params, "--roles", "1")
	} else {
		key = KeyList[node.Idx]
		params = append(params, "--roles", "4",
			"--key", key)
	}

	if websocket {
		params = append(params, "--ws", "--ws-unsafe",
			"--wsport", node.WSPort)
	}
	node.Process = exec.Command(gossamerCMD, params...)

	node.Key = key

	Logger.Infof("node basepath: %s", node.basePath)
	// create log file
	outfile, err := os.Create(filepath.Join(node.basePath, "log.out"))
	if err != nil {
		Logger.Errorf("Error when trying to set a log file for gossamer output: %s", err)
		return node, err
	}

	// create error log file
	errfile, err := os.Create(filepath.Join(node.basePath, "error.out"))
	if err != nil {
		Logger.Errorf("Error when trying to set a log file for gossamer output: %s", err)
		return node, err
	}

	t.Cleanup(func() {
		time.Sleep(time.Second) // wait for goroutine to finish writing
		err = outfile.Close()
		assert.NoError(t, err)
		err = errfile.Close()
		assert.NoError(t, err)
	})

	stdoutPipe, err := node.Process.StdoutPipe()
	if err != nil {
		Logger.Errorf("failed to get stdoutPipe from node %d: %s", node.Idx, err)
		return node, err
	}

	stderrPipe, err := node.Process.StderrPipe()
	if err != nil {
		Logger.Errorf("failed to get stderrPipe from node %d: %s", node.Idx, err)
		return node, err
	}

	Logger.Infof("starting gossamer at %s...", node.Process)
	err = node.Process.Start()
	if err != nil {
		Logger.Errorf("Could not execute gossamer cmd: %s", err)
		return node, err
	}

	writer := bufio.NewWriter(outfile)
	go func() {
		_, err := io.Copy(writer, stdoutPipe)
		if err != nil {
			Logger.Errorf("failed copying stdout to writer: %s", err)
		}
	}()
	errWriter := bufio.NewWriter(errfile)
	go func() {
		_, err := io.Copy(errWriter, stderrPipe)
		if err != nil {
			Logger.Errorf("failed copying stderr to writer: %s", err)
		}
	}()

	ctx := context.Background()

	var started bool
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Second * 5)

		const checkNodeStartedTimeout = time.Second
		checkNodeCtx, cancel := context.WithTimeout(ctx, checkNodeStartedTimeout)

		addr := fmt.Sprintf("http://%s:%s", HOSTNAME, node.RPCPort)
		err = checkNodeStarted(checkNodeCtx, t, addr)

		cancel()

		if err == nil {
			started = true
			break
		}
	}

	if started {
		Logger.Infof("node started with key %s and cmd.Process.Pid %d", key, node.Process.Process.Pid)
	} else {
		Logger.Criticalf("node didn't start: %s", err)
		errFileContents, _ := os.ReadFile(errfile.Name())
		t.Logf("%s\n", errFileContents)
		return node, err
	}

	return node, nil
}

// RunGossamer will initialise and start a gossamer instance
func RunGossamer(t *testing.T, idx int, basepath, genesis, config string, websocket, babeLead bool) (
	node Node, err error) {
	node, err = InitGossamer(idx, basepath, genesis, config)
	if err != nil {
		return node, fmt.Errorf("could not initialise gossamer: %w", err)
	}

	if idx == 0 || babeLead {
		node.BABELead = true
	}

	node, err = startGossamer(t, node, websocket)
	if err != nil {
		return node, fmt.Errorf("could not start gossamer: %w", err)
	}

	return node, nil
}

// checkNodeStarted check if gossamer node is started
func checkNodeStarted(ctx context.Context, t *testing.T, gossamerHost string) error {
	const method = "system_health"
	const params = "{}"
	respBody, err := PostRPC(ctx, gossamerHost, method, params)
	if err != nil {
		return err
	}

	target := new(modules.SystemHealthResponse)
	err = DecodeRPC(t, respBody, target)
	if err != nil {
		return err
	}

	if !target.ShouldHavePeers {
		return fmt.Errorf("no peers")
	}

	return nil
}

// killProcess kills a instance of gossamer
func killProcess(t *testing.T, cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if err != nil {
		t.Log("failed to kill process", "cmd", cmd)
	}
	return err
}

// InitNodes initialises given number of nodes
func InitNodes(num int, config string) (nodes []Node, err error) {
	tempDir, err := os.MkdirTemp("", "gossamer-stress-")
	if err != nil {
		return nil, err
	}

	for i := 0; i < num; i++ {
		node, err := InitGossamer(i, tempDir+strconv.Itoa(i), GenesisDefault, config)
		if err != nil {
			Logger.Errorf("failed to initialise Gossamer for node index %d", i)
			return nil, err
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

// StartNodes starts given array of nodes
func StartNodes(t *testing.T, nodes []Node) (err error) {
	for i, n := range nodes {
		nodes[i], err = startGossamer(t, n, false)
		if err != nil {
			return fmt.Errorf("node %d of %d: %w",
				i+1, len(nodes), err)
		}
	}
	return nil
}

// InitializeAndStartNodes will spin up `num` gossamer nodes
func InitializeAndStartNodes(t *testing.T, num int, genesis, config string) (
	nodes []Node, err error) {
	var wg sync.WaitGroup
	var nodesMutex, errMutex sync.Mutex
	wg.Add(num)

	for i := 0; i < num; i++ {
		go func(i int) {
			defer wg.Done()
			name := strconv.Itoa(i)
			if i < len(KeyList) {
				name = KeyList[i]
			}
			node, runErr := RunGossamer(t, i, TestDir(t, name), genesis, config, false, false)
			if runErr != nil {
				errMutex.Lock()
				if err == nil {
					err = fmt.Errorf("failed to run Gossamer for node index %d: %w", i, runErr)
				}
				errMutex.Unlock()
				return
			}

			nodesMutex.Lock()
			nodes = append(nodes, node)
			nodesMutex.Unlock()
		}(i)
	}

	wg.Wait()

	if err != nil {
		_ = StopNodes(t, nodes)
		return nil, err
	}

	return nodes, nil
}

// InitializeAndStartNodesWebsocket will spin up `num` gossamer nodes running with Websocket rpc enabled
func InitializeAndStartNodesWebsocket(t *testing.T, num int, genesis, config string) (
	nodes []Node, err error) {
	var nodesMutex, errMutex sync.Mutex
	var wg sync.WaitGroup

	wg.Add(num)

	for i := 0; i < num; i++ {
		go func(i int) {
			defer wg.Done()
			name := strconv.Itoa(i)
			if i < len(KeyList) {
				name = KeyList[i]
			}
			node, runErr := RunGossamer(t, i, TestDir(t, name), genesis, config, true, false)
			if runErr != nil {
				errMutex.Lock()
				if err == nil {
					err = fmt.Errorf("failed to run Gossamer for node index %d: %w", i, runErr)
				}
				errMutex.Unlock()
				return
			}

			nodesMutex.Lock()
			nodes = append(nodes, node)
			nodesMutex.Unlock()
		}(i)
	}

	wg.Wait()

	if err != nil {
		_ = StopNodes(t, nodes)
		return nil, err
	}

	return nodes, nil
}

// StopNodes stops the given nodes
func StopNodes(t *testing.T, nodes []Node) (errs []error) {
	for i := range nodes {
		cmd := nodes[i].Process
		err := killProcess(t, cmd)
		if err != nil {
			Logger.Errorf("failed to kill Gossamer (cmd %s) for node index %d", cmd, i)
			errs = append(errs, err)
		}
	}

	return errs
}

// TearDown stops the given nodes and remove their datadir
func TearDown(t *testing.T, nodes []Node) (errorList []error) {
	for i, node := range nodes {
		cmd := nodes[i].Process
		err := killProcess(t, cmd)
		if err != nil {
			Logger.Errorf("failed to kill Gossamer (cmd %s) for node index %d", cmd, i)
			errorList = append(errorList, err)
		}

		err = os.RemoveAll(node.basePath)
		if err != nil {
			Logger.Error("failed to remove base path directory " + node.basePath)
			errorList = append(errorList, err)
		}
	}

	return errorList
}

// TestDir returns the test directory path <current-directory>/test_data/<test-name>/<name>
func TestDir(t *testing.T, name string) string {
	return filepath.Join("/tmp/", t.Name(), name)
}

// GenerateGenesisThreeAuth generates Genesis file with three authority.
func GenerateGenesisThreeAuth() {
	genesisPath, err := utils.GetGssmrGenesisPath()
	if err != nil {
		panic(err)
	}

	bs, err := dot.BuildFromGenesis(genesisPath, 3)
	if err != nil {
		Logger.Errorf("genesis file not found: %s", err)
		os.Exit(1)
	}
	dot.CreateJSONRawFile(bs, GenesisThreeAuths)
}

// GenerateGenesisSixAuth generates Genesis file with six authority.
func GenerateGenesisSixAuth(t *testing.T) {
	bs, err := dot.BuildFromGenesis(utils.GetGssmrGenesisPathTest(t), 6)
	if err != nil {
		Logger.Errorf("genesis file not found: %s", err)
		os.Exit(1)
	}
	dot.CreateJSONRawFile(bs, GenesisSixAuths)
}

func generateDefaultConfig() *ctoml.Config {
	return &ctoml.Config{
		Global: ctoml.GlobalConfig{
			Name:           "Gossamer",
			ID:             "gssmr",
			LogLvl:         "crit",
			MetricsAddress: "localhost:9876",
			RetainBlocks:   256,
			Pruning:        "archive",
		},
		Log: ctoml.LogConfig{
			CoreLvl: "info",
			SyncLvl: "info",
		},
		Init: ctoml.InitConfig{
			Genesis: "./chain/gssmr/genesis.json",
		},
		Account: ctoml.AccountConfig{
			Key:    "",
			Unlock: "",
		},
		Core: ctoml.CoreConfig{
			Roles:            4,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  1,
		},
		Network: ctoml.NetworkConfig{
			Bootnodes:   nil,
			ProtocolID:  "/gossamer/gssmr/0",
			NoBootstrap: false,
			NoMDNS:      false,
			MinPeers:    1,
			MaxPeers:    3,
		},
		RPC: ctoml.RPCConfig{
			Enabled:  false,
			Unsafe:   true,
			WSUnsafe: true,
			Host:     "localhost",
			Modules:  []string{"system", "author", "chain", "state"},
			WS:       false,
		},
	}
}

// CreateDefaultConfig generates and creates default config file.
func CreateDefaultConfig() {
	cfg := generateDefaultConfig()
	dot.ExportTomlConfig(cfg, ConfigDefault)
}

func generateConfigLogGrandpa() *ctoml.Config {
	cfg := generateDefaultConfig()
	cfg.Log = ctoml.LogConfig{
		CoreLvl:           "crit",
		NetworkLvl:        "debug",
		RuntimeLvl:        "crit",
		BlockProducerLvl:  "info",
		FinalityGadgetLvl: "debug",
	}
	return cfg
}

// CreateConfigLogGrandpa generates and creates grandpa config file.
func CreateConfigLogGrandpa() {
	cfg := generateConfigLogGrandpa()
	dot.ExportTomlConfig(cfg, ConfigLogGrandpa)
}

func generateConfigNoBabe() *ctoml.Config {
	cfg := generateDefaultConfig()
	cfg.Global.LogLvl = "info"
	cfg.Log = ctoml.LogConfig{
		SyncLvl:    "debug",
		NetworkLvl: "debug",
	}

	cfg.Core.BabeAuthority = false
	return cfg
}

// CreateConfigNoBabe generates and creates no babe config file.
func CreateConfigNoBabe() {
	cfg := generateConfigNoBabe()
	dot.ExportTomlConfig(cfg, ConfigNoBABE)
}

func generateConfigNoGrandpa() *ctoml.Config {
	cfg := generateDefaultConfig()
	cfg.Core.GrandpaAuthority = false
	cfg.Core.BABELead = true
	cfg.Core.GrandpaInterval = 1
	return cfg
}

// CreateConfigNoGrandpa generates and creates no grandpa config file.
func CreateConfigNoGrandpa() {
	cfg := generateConfigNoGrandpa()
	dot.ExportTomlConfig(cfg, ConfigNoGrandpa)
}

func generateConfigNotAuthority() *ctoml.Config {
	cfg := generateDefaultConfig()
	cfg.Core.Roles = 1
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	return cfg
}

// CreateConfigNotAuthority generates and creates non-authority config file.
func CreateConfigNotAuthority() {
	cfg := generateConfigNotAuthority()
	dot.ExportTomlConfig(cfg, ConfigNotAuthority)
}
