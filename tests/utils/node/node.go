// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils/config"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/tests/utils/pathfinder"
)

// Node is a structure holding all the settings to
// configure a Gossamer node.
type Node struct {
	index      *int
	tomlConfig cfg.Config
	writer     io.Writer
	logsBuffer *bytes.Buffer
	binPath    string
}

// New returns a node configured using the
// toml configuration and options given.
func New(t *testing.T, tomlConfig cfg.Config,
	options ...Option) (node Node) {
	node.tomlConfig = cfg.Copy(&tomlConfig)
	for _, option := range options {
		option(&node)
	}
	node.setDefaults(t)
	node.setWriterPrefix()

	return node
}

func (n Node) String() string {
	indexString := fmt.Sprint(*n.index)
	return fmt.Sprintf("%s-%s", n.tomlConfig.Account.Key, indexString)
}

// RPCPort returns the rpc port of the node.
func (n Node) RPCPort() (port string) { return fmt.Sprint(n.tomlConfig.RPC.Port) }

// WSPort returns the websocket port of the node.
func (n Node) WSPort() (port string) { return fmt.Sprint(n.tomlConfig.RPC.WSPort) }

// Key returns the key of the node.
func (n Node) Key() (key string) { return n.tomlConfig.Account.Key }

func intPtr(n int) *int { return &n }

func (n *Node) setDefaults(t *testing.T) {
	if n.index == nil {
		n.index = intPtr(0)
	}

	if n.tomlConfig.BasePath == "" {
		n.tomlConfig.BasePath = t.TempDir()
	}

	if n.tomlConfig.ChainSpec == "" {
		n.tomlConfig.ChainSpec = utils.GetWestendDevRawGenesisPath(t)
	}

	if n.tomlConfig.Account.Key == "" {
		keyList := []string{
			"alice",
			"bob",
			"charlie",
			"dave",
			"eve",
			"ferdie",
			"george",
			"heather",
			"ian",
		}
		if *n.index < len(keyList) {
			n.tomlConfig.Account.Key = keyList[*n.index]
		} else {
			n.tomlConfig.Account.Key = "default-key"
		}
	}

	if n.tomlConfig.Network.Port == 0 {
		const basePort uint16 = 7004
		n.tomlConfig.Network.Port = basePort + uint16(*n.index)
	}

	if n.tomlConfig.RPC.IsRPCEnabled() && n.tomlConfig.RPC.Port == 0 {
		const basePort uint32 = 8540
		n.tomlConfig.RPC.Port = basePort + uint32(*n.index)
	}

	if n.tomlConfig.RPC.IsWSEnabled() && n.tomlConfig.RPC.WSPort == 0 {
		const basePort uint32 = 8546
		n.tomlConfig.RPC.WSPort = basePort + uint32(*n.index)
	}

	userSetWriter := n.writer != nil && n.writer != io.Discard
	if !userSetWriter {
		n.logsBuffer = bytes.NewBuffer(nil)
	}

	if n.writer == nil {
		n.writer = io.Discard
	}

	if n.binPath == "" {
		n.binPath = pathfinder.GetGossamer(t)
	}
}

// Init initialises the Gossamer node.
func (n *Node) Init() (err error) {
	// Ensure the base path exists.
	if err := cfg.EnsureRoot(n.tomlConfig.BasePath); err != nil {
		return fmt.Errorf("cannot ensure root: %w", err)
	}

	if err := n.tomlConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("cannot validate basic config: %w", err)
	}

	// Write the configuration to a file.
	return cfg.WriteConfigFile(n.tomlConfig.BasePath, &n.tomlConfig)
}

// Start starts a Gossamer node using the node configuration of
// the receiving struct. It returns a start error if the node cannot
// be started, and runs the node until the context gets canceled.
// When the node crashes or is stopped, an error (nil or not) is sent
// in the waitErrCh.
func (n *Node) Start(ctx context.Context, optArgs ...string) (runtimeError <-chan error, startErr error) {
	args := []string{
		"--base-path", n.tomlConfig.BasePath,
		"--chain", n.tomlConfig.ChainSpec,
		"--role", config.ParseNetworkRole(n.tomlConfig.Core.Role),
		"--no-telemetry"}
	args = append(args, optArgs...)
	cmd := exec.CommandContext(ctx, n.binPath, args...)

	if n.logsBuffer != nil {
		n.logsBuffer.Reset()
		n.writer = io.MultiWriter(n.writer, n.logsBuffer)
	}

	cmd.Stdout = n.writer
	cmd.Stderr = cmd.Stdout // we assume no race between stdout and stderr

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start %s: %w", cmd, err)
	}

	waitErrCh := make(chan error)
	go func(cmd *exec.Cmd, node *Node, waitErr chan<- error) {
		err = cmd.Wait()
		waitErr <- node.wrapRuntimeError(ctx, cmd, err)
	}(cmd, n, waitErrCh)

	return waitErrCh, nil
}

// StartAndWait starts a Gossamer node using the node configuration of
// the receiving struct. It returns a start error if the node cannot
// be started, and runs the node until the context gets canceled.
// When the node crashes or is stopped, an error (nil or not) is sent
// in the waitErrCh.
// It waits for the node to respond to an RPC health call before returning.
func (n *Node) StartAndWait(ctx context.Context, args ...string) (
	runtimeError <-chan error, startErr error) {
	runtimeError, startErr = n.Start(ctx, args...)
	if startErr != nil {
		return nil, startErr
	}

	err := waitForNode(ctx, n.RPCPort())
	if err != nil {
		return nil, fmt.Errorf("failed waiting: %s", err)
	}

	return runtimeError, nil
}

// InitAndStartTest is a test helper method to initialise and start the node,
// as well as registering appriopriate test handlers.
// If initialising or starting fails, cleanup is done and the test fails instantly.
// If the node crashes during runtime, the passed `signalTestToStop` argument is
// called since the test cannot be failed from outside the main test goroutine.
func (n Node) InitAndStartTest(ctx context.Context, t *testing.T,
	signalTestToStop context.CancelFunc, args ...string) {
	t.Helper()

	err := n.Init()
	require.NoError(t, err)

	nodeCtx, nodeCancel := context.WithCancel(ctx)

	waitErr, err := n.StartAndWait(nodeCtx, args...)
	if err != nil {
		t.Errorf("failed to start node %s: %s", n, err)
		// Release resources and fail the test
		nodeCancel()
		t.FailNow()
	}

	t.Logf("Node %s is ready", n)

	// watch for runtime fatal node error
	watchDogCtx, watchDogCancel := context.WithCancel(ctx)
	watchDogDone := make(chan struct{})
	go func() {
		defer close(watchDogDone)
		select {
		case <-watchDogCtx.Done():
			return
		case err := <-waitErr: // the node crashed
			if watchDogCtx.Err() != nil {
				// make sure the runtime watchdog is not meant
				// to be disengaged, in case of signal racing.
				return
			}
			t.Errorf("node %s crashed: %s", n, err)
			// Release resources
			nodeCancel()
			// we cannot stop the test with t.FailNow() from a goroutine
			// other than the test goroutine, so we call the following function
			// to signal the test goroutine to stop the test.
			signalTestToStop()
		}
	}()

	t.Cleanup(func() {
		t.Helper()
		// Disengage node watchdog goroutine
		watchDogCancel()
		<-watchDogDone
		// Stop the node and wait for it to exit
		nodeCancel()
		<-waitErr
		t.Logf("Node %s terminated", n)
	})
}

func (n *Node) setWriterPrefix() {
	if n.writer == io.Discard {
		return // no need to wrap it
	}

	n.writer = &prefixedWriter{
		prefix: []byte(n.String() + " "),
		writer: n.writer,
	}
}

// wrapRuntimeError wraps the error given using the context available
// such as the command string or the log buffer. It returns nil if the
// argument error is nil.
func (n *Node) wrapRuntimeError(ctx context.Context, cmd *exec.Cmd,
	waitErr error) (wrappedErr error) {
	if waitErr == nil {
		return nil
	}

	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w: %s", n, ctx.Err(), waitErr)
	}

	var logInformation string
	if n.logsBuffer != nil {
		// Add log information to error if no writer is set
		// for this node.
		logInformation = "\nLogs:\n" + n.logsBuffer.String()
	}

	return fmt.Errorf("%s encountered a runtime error: %w\ncommand: %s\n\n%s\n\n%s",
		n, waitErr, cmd, n.tomlConfig.BasePath, logInformation)
}
