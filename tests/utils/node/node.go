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

	"github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/pathfinder"
	"github.com/stretchr/testify/require"
)

// Node is a structure holding all the settings to
// configure a Gossamer node.
type Node struct {
	index      *int
	configPath string
	tomlConfig toml.Config
	writer     io.Writer
	logsBuffer *bytes.Buffer
	binPath    string
}

// New returns a node configured using the
// toml configuration and options given.
func New(t *testing.T, tomlConfig toml.Config,
	options ...Option) (node Node) {
	node.tomlConfig = tomlConfig
	for _, option := range options {
		option(&node)
	}
	node.setDefaults(t)
	node.setWriterPrefix()
	node.configPath = config.Write(t, node.tomlConfig)
	return node
}

func (n Node) String() string {
	indexString := fmt.Sprint(*n.index)
	return fmt.Sprintf("%s-%s", n.tomlConfig.Account.Key, indexString)
}

// GetRPCPort returns the rpc port of the node.
func (n Node) GetRPCPort() (port string) { return fmt.Sprint(n.tomlConfig.RPC.Port) }

// GetWSPort returns the websocket port of the node.
func (n Node) GetWSPort() (port string) { return fmt.Sprint(n.tomlConfig.RPC.WSPort) }

// GetKey returns the key of the node.
func (n Node) GetKey() (key string) { return n.tomlConfig.Account.Key }

func intPtr(n int) *int { return &n }

func (n *Node) setDefaults(t *testing.T) {
	if n.index == nil {
		n.index = intPtr(0)
	}

	if n.tomlConfig.Global.BasePath == "" {
		n.tomlConfig.Global.BasePath = t.TempDir()
	}

	if n.tomlConfig.Init.Genesis == "" {
		n.tomlConfig.Init.Genesis = utils.GetGssmrGenesisRawPathTest(t)
	}

	if n.tomlConfig.Account.Key == "" {
		keyList := []string{"alice", "bob", "charlie", "dave", "eve", "ferdie", "george", "heather", "ian"}
		if *n.index < len(keyList) {
			n.tomlConfig.Account.Key = keyList[*n.index]
		} else {
			n.tomlConfig.Account.Key = "default-key"
		}
	}

	if n.tomlConfig.Network.Port == 0 {
		const basePort uint16 = 7000
		n.tomlConfig.Network.Port = basePort + uint16(*n.index)
	}

	if n.tomlConfig.RPC.Enabled && n.tomlConfig.RPC.Port == 0 {
		const basePort uint32 = 8540
		n.tomlConfig.RPC.Port = basePort + uint32(*n.index)
	}

	if n.tomlConfig.RPC.WS && n.tomlConfig.RPC.WSPort == 0 {
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
func (n *Node) Init(ctx context.Context) (err error) {
	cmdInit := exec.CommandContext(ctx, n.binPath, "init", //nolint:gosec
		"--config", n.configPath,
	)

	if n.logsBuffer != nil {
		n.writer = io.MultiWriter(n.writer, n.logsBuffer)
	}

	cmdInit.Stdout = n.writer
	cmdInit.Stderr = n.writer

	err = cmdInit.Start()
	if err != nil {
		return fmt.Errorf("cannot start command: %w", err)
	}

	err = cmdInit.Wait()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// Start starts a Gossamer node using the node configuration of
// the receiving struct. It returns a start error if the node cannot
// be started, and runs the node until the context gets canceled.
// When the node crashes or is stopped, an error (nil or not) is sent
// in the waitErrCh.
func (n *Node) Start(ctx context.Context, waitErrCh chan<- error) (startErr error) {
	cmd := exec.CommandContext(ctx, n.binPath, //nolint:gosec
		"--config", n.configPath,
		"--no-telemetry")

	cmd.Stdout = n.writer
	cmd.Stderr = cmd.Stdout // we assume no race between stdout and stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("cannot start %s: %w", cmd, err)
	}

	go func(cmd *exec.Cmd, node *Node, waitErr chan<- error) {
		err = cmd.Wait()
		if err != nil {
			if ctx.Err() != nil {
				err = fmt.Errorf("%s: %w: %s", node, ctx.Err(), err)
			} else {
				var logInformation string
				if node.logsBuffer != nil {
					// Add log information to error if no writer is set
					// for this node.
					logInformation = "\nLogs:\n" + node.logsBuffer.String()
				}
				err = fmt.Errorf("%s encountered a runtime error: %w\ncommand: %s%s", n, err, cmd, logInformation)
			}
		}
		waitErr <- err
	}(cmd, n, waitErrCh)

	return nil
}

// StartAndWait starts a Gossamer node using the node configuration of
// the receiving struct. It returns a start error if the node cannot
// be started, and runs the node until the context gets canceled.
// When the node crashes or is stopped, an error (nil or not) is sent
// in the waitErrCh.
// It waits for the node to respond to an RPC health call before returning.
func (n *Node) StartAndWait(ctx context.Context, waitErrCh chan<- error) (startErr error) {
	startErr = n.Start(ctx, waitErrCh)
	if startErr != nil {
		return startErr
	}

	err := waitForNode(ctx, n.GetRPCPort())
	if err != nil {
		return fmt.Errorf("failed waiting: %s", err)
	}

	return nil
}

// InitAndStartTest is a test helper method to initialise and start the node,
// as well as registering appriopriate test handlers.
// If initialising or starting fails, cleanup is done and the test fails instantly.
// If the node crashes during runtime, the passed `signalTestToStop` argument is
// called since the test cannot be failed from outside the main test goroutine.
func (n Node) InitAndStartTest(ctx context.Context, t *testing.T,
	signalTestToStop context.CancelFunc) {
	t.Helper()

	err := n.Init(ctx)
	require.NoError(t, err)

	nodeCtx, nodeCancel := context.WithCancel(ctx)
	waitErr := make(chan error)

	err = n.StartAndWait(nodeCtx, waitErr)
	if err != nil {
		t.Errorf("failed to start node %s: %s", n, err)
		// Release resources and fail the test
		nodeCancel()
		close(waitErr)
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
			close(waitErr)
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
