// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	scribble "github.com/nanobox-io/golang-scribble"
)

// Framework struct to hold references to framework data
type Framework struct {
	nodes   node.Nodes
	db      *scribble.Driver
	callQty int
}

// NewFramework creates a new framework.
func NewFramework() (framework *Framework) {
	return &Framework{}
}

// InitFramework creates given quantity of nodes
func InitFramework(ctx context.Context, t *testing.T, qtyNodes int,
	tomlConfig cfg.Config) (*Framework, error) {
	f := &Framework{}

	f.nodes = node.MakeNodes(t, qtyNodes, tomlConfig)

	err := f.nodes.Init()
	if err != nil {
		return nil, fmt.Errorf("cannot init nodes: %w", err)
	}

	db, err := scribble.New(t.TempDir(), nil)
	if err != nil {
		return nil, err
	}
	f.db = db

	return f, nil
}

// StartNodes calls RestartGossamer for all nodes
func (fw *Framework) StartNodes(ctx context.Context, t *testing.T) (
	runtimeErrors []<-chan error, startErr error) {
	return fw.nodes.Start(ctx)
}

// CallRPC call RPC method with given params for node at idx
func (fw *Framework) CallRPC(ctx context.Context, idx int, method, params string) (
	respJSON interface{}, err error) {
	if idx >= len(fw.nodes) {
		return nil, fmt.Errorf("node index greater than quantity of nodes")
	}
	node := fw.nodes[idx]
	respBody, err := rpc.Post(ctx, rpc.NewEndpoint(node.RPCPort()), method, params)
	if err != nil {
		return nil, err
	}

	err = rpc.Decode(respBody, &respJSON)
	if err != nil {
		return nil, fmt.Errorf("error making RPC call %v", err)
	}
	err = fw.db.Write("rpc", strconv.Itoa(fw.callQty), respJSON)
	if err != nil {
		return nil, fmt.Errorf("error writing to db %v", err)
	}

	fw.callQty++

	return
}

// PrintDB prints all records for given node
func (fw *Framework) PrintDB() {
	for i := 0; i < fw.callQty; i++ {
		fmt.Printf("Call: %v: Val: %v\n", i, fw.GetRecord(i))
	}
}

// GetRecord return value of record for node and call index
func (fw *Framework) GetRecord(callIdx int) interface{} {
	var v interface{}
	err := fw.db.Read("rpc", strconv.Itoa(callIdx), &v)
	if err != nil {
		return fmt.Errorf("error reading from db %v", err)
	}
	return v
}

// CheckEqual returns true if the field values are equal
func (fw *Framework) CheckEqual(c1, c2 int, field string) bool {
	var r1 map[string]interface{}
	err := fw.db.Read("rpc", strconv.Itoa(c1), &r1)
	if err != nil {
		return false
	}

	var r2 map[string]interface{}
	err = fw.db.Read("rpc", strconv.Itoa(c2), &r2)
	if err != nil {
		return false
	}

	return r1[field] == r2[field]
}
