// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"text/template"
	"time"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/require"
)

type TestExecCommand struct {
	*testing.T
	Func    template.FuncMap
	Data    interface{}
	Cleanup func()
	cmd     *exec.Cmd
	stdout  *bufio.Reader
	stdin   io.WriteCloser
	stderr  *testlog
	Err     error
}

type testgossamer struct {
	*TestExecCommand
	Datadir   string
	Etherbase string
}

type testlog struct {
	t   *testing.T
	mu  sync.Mutex
	buf bytes.Buffer
}

func init() {
	reexec.Register("gossamer-test", func() {
		if err := app.Run(os.Args); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func (tl *testlog) Write(b []byte) (n int, err error) {
	lines := bytes.Split(b, []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			tl.t.Logf("stderr: %s", line)
		}
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.buf.Write(b)
	return len(b), err
}

func (tt *TestExecCommand) Run(name string, args ...string) {
	tt.stderr = &testlog{t: tt.T}
	tt.cmd = &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{name}, args...),
		Stderr: tt.stderr,
	}
	stdout, err := tt.cmd.StdoutPipe()
	require.Nil(tt, err)
	tt.stdout = bufio.NewReader(stdout)
	if tt.stdin, err = tt.cmd.StdinPipe(); err != nil {
		require.Nil(tt, err)
	}
	if err := tt.cmd.Start(); err != nil {
		require.Nil(tt, err)
	}
}

func (tt *TestExecCommand) ExpectExit() {
	var output []byte
	tt.withKillTimeout(func() {
		output, _ = ioutil.ReadAll(tt.stdout)
	})
	tt.WaitExit()
	if tt.Cleanup != nil {
		tt.Cleanup()
	}
	if len(output) > 0 {
		tt.Errorf("stdout unmatched:\n%s", output)
	}
}

func (tt *TestExecCommand) GetOutput() (stdout []byte, stderr []byte) {
	tt.withSigTimeout(func() {
		stdout, _ = ioutil.ReadAll(tt.stdout)
		stderr = tt.stderr.buf.Bytes()
	})
	tt.WaitExit()
	if tt.Cleanup != nil {
		tt.Cleanup()
	}

	return stdout, stderr
}

func (tt *TestExecCommand) WaitExit() {
	tt.Err = tt.cmd.Wait()
}

func (tt *TestExecCommand) withKillTimeout(fn func()) {
	timeout := time.AfterFunc(5*time.Second, func() {
		tt.Log("process timeout, killing")
		tt.Kill()
	})
	defer timeout.Stop()
	fn()
}

func (tt *TestExecCommand) withSigTimeout(fn func()) {
	timeout := time.AfterFunc(5*time.Second, func() {
		tt.Log("process timeout, will signal")
		tt.Signal()
	})
	defer timeout.Stop()
	fn()
}

func (tt *TestExecCommand) Kill() {
	_ = tt.cmd.Process.Kill()
	if tt.Cleanup != nil {
		tt.Cleanup()
	}
}

func (tt *TestExecCommand) Signal() {
	err := tt.cmd.Process.Signal(syscall.SIGINT)
	require.Nil(tt.T, err)
	if tt.Cleanup != nil {
		tt.Cleanup()
	}
}

func (tt *TestExecCommand) Expect(tplsource string) {
	tpl := template.Must(template.New("").Funcs(tt.Func).Parse(tplsource))
	wantbuf := new(bytes.Buffer)
	require.Nil(tt, tpl.Execute(wantbuf, tt.Data))

	want := bytes.TrimPrefix(wantbuf.Bytes(), []byte("\n"))
	tt.matchExactOutput(want)

	tt.Logf("stdout matched:\n%s", want)
}

func (tt *TestExecCommand) matchExactOutput(want []byte) {
	buf := make([]byte, len(want))
	n := 0
	tt.withKillTimeout(func() { n, _ = io.ReadFull(tt.stdout, buf) })
	buf = buf[:n]
	if n < len(want) || !bytes.Equal(buf, want) {
		buf = append(buf, make([]byte, tt.stdout.Buffered())...)
		_, _ = tt.stdout.Read(buf[n:])
		require.Equal(tt, want, buf)
	}
}

func (tt *TestExecCommand) StderrText() string {
	tt.stderr.mu.Lock()
	defer tt.stderr.mu.Unlock()
	return tt.stderr.buf.String()
}

func newTestCommand(t *testing.T, data interface{}) *TestExecCommand {
	return &TestExecCommand{T: t, Data: data}
}

func runTestGossamer(t *testing.T, args ...string) *testgossamer {
	tt := &testgossamer{}
	tt.TestExecCommand = newTestCommand(t, tt)
	tt.Run("gossamer-test", args...)
	return tt
}

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	defaultGssmrConfigPath = "../../chain/gssmr/config.toml"
	defaultKusamaConfigPath = "../../chain/kusama/config.toml"
	defaultPolkadotConfigPath = "../../chain/polkadot/config.toml"
	os.Exit(m.Run())
}

func TestInvalidCommand(t *testing.T) {
	gossamer := runTestGossamer(t, "potato")

	gossamer.ExpectExit()

	expectedMessages := []string{
		"failed to read command argument: \"potato\"",
	}

	for _, m := range expectedMessages {
		require.Contains(t, gossamer.StderrText(), m)
	}
}

func TestGossamerCommand(t *testing.T) {
	t.Skip() // TODO: not sure how relevant this is anymore, it also slows down the tests a lot

	basePort := 7000
	genesisPath := utils.GetGssmrGenesisRawPath()

	tempDir, err := ioutil.TempDir("", "gossamer-maintest-")
	require.Nil(t, err)

	gossamer := runTestGossamer(t,
		"init",
		"--basepath", tempDir,
		"--genesis", genesisPath,
		"--force",
	)

	stdout, stderr := gossamer.GetOutput()
	t.Log("init gossamer output, ", "stdout", string(stdout), "stderr", string(stderr))

	expectedMessages := []string{
		"node initialised",
	}

	for _, m := range expectedMessages {
		require.Contains(t, string(stdout), m)
	}

	for i := 0; i < 10; i++ {
		t.Log("Going to gossamer cmd", "iteration", i)

		// start
		gossamer = runTestGossamer(t,
			"--port", strconv.Itoa(basePort),
			"--key", "alice",
			"--basepath", tempDir,
			"--roles", "4",
		)

		time.Sleep(10 * time.Second)

		stdout, stderr = gossamer.GetOutput()
		log.Println("Run gossamer output, ", "stdout", string(stdout), "stderr", string(stderr))

		expectedMessages = []string{
			"SIGABRT: abort",
		}

		for _, m := range expectedMessages {
			require.NotContains(t, string(stderr), m)
		}
	}
}

func TestInitCommand_RenameNodeWhenCalled(t *testing.T) {
	genesisPath := utils.GetGssmrGenesisRawPath()

	tempDir, err := ioutil.TempDir("", "gossamer-maintest-")
	require.Nil(t, err)

	nodeName := dot.RandomNodeName()
	init := runTestGossamer(t,
		"init",
		"--basepath", tempDir,
		"--genesis", genesisPath,
		"--name", nodeName,
		"--config", defaultGssmrConfigPath,
		"--force",
	)

	stdout, stderr := init.GetOutput()
	require.Nil(t, err)

	t.Log("init gossamer output, ", "stdout", string(stdout), "stderr", string(stderr))

	// should contains the name defined in name flag
	require.Contains(t, string(stdout), nodeName)

	init = runTestGossamer(t,
		"init",
		"--basepath", tempDir,
		"--genesis", genesisPath,
		"--config", defaultGssmrConfigPath,
		"--force",
	)

	stdout, stderr = init.GetOutput()
	require.Nil(t, err)

	t.Log("init gossamer output, ", "stdout", string(stdout), "stderr", string(stderr))

	// should not contains the name from the last init
	require.NotContains(t, string(stdout), nodeName)
}

func TestBuildSpecCommandWithOutput(t *testing.T) {
	tmpOutputfile := "/tmp/raw-genesis-spec-output.json"
	buildSpecCommand := runTestGossamer(t,
		"build-spec",
		"--raw",
		"--genesis-spec", "../../chain/gssmr/genesis-spec.json",
		"--output", tmpOutputfile)

	time.Sleep(5 * time.Second)

	_, err := os.Stat(tmpOutputfile)
	require.False(t, os.IsNotExist(err))
	defer os.Remove(tmpOutputfile)

	outb, errb := buildSpecCommand.GetOutput()
	require.Empty(t, outb)
	require.Empty(t, errb)
}

// TODO: TestExportCommand test "gossamer export" does not error

// TODO: TestInitCommand test "gossamer init" does not error

// TODO: TestAccountCommand test "gossamer account" does not error

func TestPruneState(t *testing.T) {
	const (
		bloomSize      = 256
		retainBlockNum = 5
	)

	chainDBPath := "/tmp/TestSync_ProduceBlocks/alice"
	opts := badger.DefaultOptions(chainDBPath)
	currDB, err := badger.Open(opts)
	require.NoError(t, err)

	txn := currDB.NewTransaction(false)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)

	keyMap := make(map[string]interface{})
	for itr.Rewind(); itr.Valid(); itr.Next() {
		key := string(itr.Item().Key())

		if !strings.HasPrefix(key, state.StoragePrefix) {
			keyMap[key] = nil
		}
	}

	t.Log("Total keys in old DB", len(keyMap))
	currDB.Close()

	pruner, err := newPruner(chainDBPath, bloomSize, retainBlockNum)
	require.NoError(t, err)

	// key with storage prefix of last 256 blocks
	err = pruner.setBloomFilter()
	require.NoError(t, err)

	// close pruner inputDB so it can be used again
	pruner.inputDB.Close()

	newBadgerDBPath := fmt.Sprintf("%s/%s", t.TempDir(), "badger")
	_ = runTestGossamer(t,
		"prune-state",
		"--basepath", chainDBPath,
		"--badger-path", newBadgerDBPath,
		"--bloom-size", "256",
		"--retain-block", "5")

	time.Sleep(10 * time.Second)

	t.Logf("new badger DB path %s", newBadgerDBPath)

	prunedDB, err := badger.Open(badger.DefaultOptions(newBadgerDBPath))
	require.NoError(t, err)

	txn = prunedDB.NewTransaction(false)
	itr = txn.NewIterator(badger.DefaultIteratorOptions)

	storageKeyMap := make(map[string]interface{})
	otherKeyMap := make(map[string]interface{})

	for itr.Rewind(); itr.Valid(); itr.Next() {
		key := string(itr.Item().Key())
		if strings.HasPrefix(key, state.StoragePrefix) {
			key = strings.TrimPrefix(key, state.StoragePrefix)
			storageKeyMap[key] = nil
			continue
		}
		otherKeyMap[key] = nil
	}

	for k := range keyMap {
		_, ok := otherKeyMap[k]
		require.True(t, ok)
	}

	for k := range storageKeyMap {
		ok := pruner.bloom.contain([]byte(k))
		require.True(t, ok)
	}
}
