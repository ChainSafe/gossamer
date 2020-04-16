// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/tests/rpc"
	"github.com/stretchr/testify/require"
)

var (
	NETWORK_SIZE_STR               = os.Getenv("NETWORK_SIZE")
	GOSSAMER_INTEGRATION_TEST_MODE = os.Getenv("GOSSAMER_INTEGRATION_TEST_MODE")
	pidList                        = make([]*exec.Cmd, 3)
	keyList                        = []string{"alice", "bob", "charlie", "dave", "eve", "fred", "george", "heather"}
	dialTimeout                    = 60 * time.Second
	httpClientTimeout              = 120 * time.Second
	rpcHost                        = "0.0.0.0"

	transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialTimeout,
		}).Dial,
	}
	httpClient = &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeout,
	}
)

// runGossamer will start a gossamer instance and check if its online and returns CMD, otherwise return err
func runGossamer(t *testing.T, nodeNumb int, dataDir string) (*exec.Cmd, error) {

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	gossamerCMD := filepath.Join(currentDir, "../..", "bin/gossamer")

	//TODO: enable genesis file to be configured via args
	//TODO: enable [core] authority and roles via args
	cmd := exec.Command(gossamerCMD, "--port", "700"+strconv.Itoa(nodeNumb),
		"--key", keyList[nodeNumb],
		"--datadir", dataDir+strconv.Itoa(nodeNumb),
		"--rpchost", rpcHost,
		"--rpcport", "854"+strconv.Itoa(nodeNumb),
		"--rpcmods", "system,author,chain",
		"--key", keyList[nodeNumb],
		"--config", currentDir+"/config.toml",
		"--rpc",
	)

	t.Log("Going to execute gossamer", "cmd", cmd)
	err = cmd.Start()
	if err != nil {
		t.Error("Could not execute gossamer cmd", "err", err)
		return nil, err
	}

	t.Log("wait few secs for node to come up", "cmd.Process.Pid", cmd.Process.Pid)
	var started bool

	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		if err := checkFunc(t, "http://"+rpcHost+":854"+strconv.Itoa(nodeNumb)); err == nil {
			started = true
			break
		} else {
			t.Log("Waiting for Gossamer to start", "err", err)
		}
	}
	if started {
		t.Log("Gossamer started :D", "cmd.Process.Pid", cmd.Process.Pid)
	} else {
		t.Fatal("Gossamer node never managed to start!")

	}

	return cmd, nil
}

// checkFunc check if gossamer node is already started
func checkFunc(t *testing.T, gossamerHost string) error {
	data := []byte(`{"jsonrpc":"2.0","method":"system_health","params":{},"id":1}`)
	buf := &bytes.Buffer{}
	_, err := buf.Write(data)
	if err != nil {
		t.Log("could not marshal json for rpc")
		return err
	}

	r, err := http.NewRequest("POST", gossamerHost, buf)
	if err != nil {
		t.Log("could not POST json to rpc")
		return err
	}

	r.Header.Set("Content-Type", rpc.ContentTypeJSON)
	r.Header.Set("Accept", rpc.ContentTypeJSON)

	resp, err := httpClient.Do(r)
	if err != nil {
		t.Log("could not Do POST json to rpc")
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log("could not ReadAll POST json from rpc")
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(respBody))
	decoder.DisallowUnknownFields()

	var response rpc.ServerResponse
	err = decoder.Decode(&response)
	if err != nil {
		t.Log("could not Decode POST json from rpc")
		return err
	}

	decoder = json.NewDecoder(bytes.NewReader(response.Result))
	decoder.DisallowUnknownFields()

	var target modules.SystemHealthResponse
	err = decoder.Decode(&target)
	if err != nil {
		t.Log("could not Decode POST json from rpc into SystemHealthResponse")
		return err
	}

	//if we get here, we assume it worked :D

	return nil
}

// killGossamer kills a instance of gossamer
func killGossamer(t *testing.T, cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if err != nil {
		t.Log("failed to kill gossamer", "cmd", cmd)
	}
	return err
}

// bootstrap will spin gossamer nodes
func bootstrap(t *testing.T, pidList []*exec.Cmd) ([]*exec.Cmd, error) {
	var newPidList []*exec.Cmd

	tempDir, err := ioutil.TempDir("", "gossamer-stress")
	if err != nil {
		t.Log("failed to create tempDir")
		return nil, err
	}

	for i, k := range pidList {
		t.Log("bootstrap gossamer ", "k", k, "i", i)
		cmd, err := runGossamer(t, i, tempDir+strconv.Itoa(i))
		if err != nil {
			t.Log("failed to runGossamer", "i", i)
			return nil, err
		}

		newPidList = append(newPidList, cmd)
	}

	return newPidList, nil
}

// tearDown will stop gossamer nodes
func tearDown(t *testing.T, pidList []*exec.Cmd) (errorList []error) {
	for i := range pidList {
		cmd := pidList[i]
		err := killGossamer(t, cmd)
		if err != nil {
			t.Log("failed to killGossamer", "i", i, "cmd", cmd)
			errorList = append(errorList, err)
		}
	}

	return errorList
}

func TestMain(m *testing.M) {
	if GOSSAMER_INTEGRATION_TEST_MODE != "stress" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip stress test")
		return
	}

	_, _ = fmt.Fprintln(os.Stdout, "Going to start stress test")

	if NETWORK_SIZE_STR != "" {
		currentNetworkSize, err := strconv.Atoi(NETWORK_SIZE_STR)
		if err == nil {
			_, _ = fmt.Fprintln(os.Stdout, "Going to custom network size ... ", "currentNetworkSize", currentNetworkSize)
			pidList = make([]*exec.Cmd, currentNetworkSize)
		}
	}

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestStressSync(t *testing.T) {
	t.Log("going to start TestStressSync")
	localPidList, err := bootstrap(t, pidList)
	require.Nil(t, err)

	errList := tearDown(t, localPidList)
	require.Len(t, errList, 0)
}
