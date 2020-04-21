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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
)

var (
	keyList = []string{"alice", "bob", "charlie", "dave", "eve", "fred", "george", "heather"}
)

// RunGossamer will start a gossamer instance and check if its online and returns CMD, otherwise return err
func RunGossamer(t *testing.T, nodeNumb int, dataDir string) (*exec.Cmd, error) {

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	gossamerCMD := filepath.Join(currentDir, "../..", "bin/gossamer")

	//nolint
	cmdInit := exec.Command(gossamerCMD, "init",
		"--datadir", dataDir+strconv.Itoa(nodeNumb),
		"--genesis", filepath.Join(currentDir, "../..", "node/gssmr/genesis.json"),
		"--force",
	)

	//add step for init
	t.Log("Going to init gossamer", "cmdInit", cmdInit)
	stdOutInit, err := cmdInit.CombinedOutput()
	if err != nil {
		t.Error("Could not init gossamer", "err", err, "output", string(stdOutInit))
		return nil, err
	}

	t.Log("Gossamer init ok")

	//TODO: could we enable genesis file to be configured via args without init?
	//nolint
	cmd := exec.Command(gossamerCMD, "--port", "700"+strconv.Itoa(nodeNumb),
		"--key", keyList[nodeNumb],
		"--datadir", dataDir+strconv.Itoa(nodeNumb),
		"--rpchost", GOSSAMER_NODE_HOST,
		"--rpcport", "854"+strconv.Itoa(nodeNumb),
		"--rpcmods", "system,author,chain",
		"--key", keyList[nodeNumb],
		"--roles", "4",
		"--rpc",
	)

	f, err := os.Create(filepath.Join(dataDir+strconv.Itoa(nodeNumb), "gossamer.log"))
	if err != nil {
		t.Fatalf("Error when trying to set a log file for gossamer output: %v", err)
	}

	multiWriter := io.MultiWriter(f, os.Stdout)

	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	t.Log("Going to execute gossamer", "cmd", cmd)
	err = cmd.Start()
	if err != nil {
		t.Error("Could not execute gossamer cmd", "err", err)
		return nil, err
	}

	t.Log("Gossamer start", "err", err)

	t.Log("wait few secs for node to come up", "cmd.Process.Pid", cmd.Process.Pid)
	var started bool

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if err = CheckFunc(t, "http://"+GOSSAMER_NODE_HOST+":854"+strconv.Itoa(nodeNumb)); err == nil {
			started = true
			break
		} else {
			t.Log("Waiting for Gossamer to start", "err", err)
		}
	}
	if started {
		t.Log("Gossamer started :D", "cmd.Process.Pid", cmd.Process.Pid)
	} else {
		t.Fatal("Gossamer node never managed to start!", "err", err)
	}

	return cmd, nil
}

// CheckFunc check if gossamer node is already started
func CheckFunc(t *testing.T, gossamerHost string) error {
	method := "system_health"

	respBody, err := PostRPC(t, method, gossamerHost, "{}")
	if err != nil {
		return err
	}

	target := DecodeRPC(t, respBody, method)

	if !target.(*modules.SystemHealthResponse).Health.ShouldHavePeers {
		return fmt.Errorf("no peers")
	}

	//if we get here, we assume it worked :D

	return nil
}

// KillGossamer kills a instance of gossamer
func KillGossamer(t *testing.T, cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if err != nil {
		t.Log("failed to kill gossamer", "cmd", cmd)
	}
	return err
}

// Bootstrap will spin gossamer nodes
func Bootstrap(t *testing.T, pidList []*exec.Cmd) ([]*exec.Cmd, error) {
	var newPidList []*exec.Cmd

	tempDir, err := ioutil.TempDir("", "gossamer-stress")
	if err != nil {
		t.Log("failed to create tempDir")
		return nil, err
	}

	for i, cmd := range pidList {
		t.Log("bootstrap gossamer ", "cmd", cmd, "i", i)
		cmd, err := RunGossamer(t, i, tempDir+strconv.Itoa(i))
		if err != nil {
			t.Log("failed to runGossamer", "i", i)
			return nil, err
		}

		newPidList = append(newPidList, cmd)
	}

	return newPidList, nil
}

// TearDown will stop gossamer nodes
func TearDown(t *testing.T, pidList []*exec.Cmd) (errorList []error) {
	for i := range pidList {
		cmd := pidList[i]
		err := KillGossamer(t, cmd)
		if err != nil {
			t.Log("failed to killGossamer", "i", i, "cmd", cmd)
			errorList = append(errorList, err)
		}
	}

	return errorList
}
