package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	scribble "github.com/nanobox-io/golang-scribble"
)

// Framework struct to hold references to framework data
type Framework struct {
	nodes   []*Node
	db      *scribble.Driver
	callQty int
}

// InitFramework creates given quanity of nodes
func InitFramework(qtyNodes int) (*Framework, error) {
	f := &Framework{}
	nodes, err := InitNodes(qtyNodes)
	if err != nil {
		return nil, err
	}
	f.nodes = nodes

	tempDir, err := ioutil.TempDir("", "gossamer-stress-db")
	if err != nil {
		return nil, err
	}
	db, err := scribble.New(tempDir, nil)
	if err != nil {
		return nil, err
	}
	f.db = db

	return f, nil
}

// StartNodes calls RestartGossamor for all nodes
func (fw *Framework) StartNodes(t *testing.T) (errorList []error) {
	for _, node := range fw.nodes {
		err := RestartGossamer(t, node)
		if err != nil {
			errorList = append(errorList, err)
		}
	}
	return errorList
}

// KillNodes stops all running nodes
func (fw *Framework) KillNodes(t *testing.T) []error {
	return TearDown(t, fw.nodes)
}

// CallRPC call RPC method with given params for node at idx
func (fw *Framework) CallRPC(idx int, method, params string) (respJSON interface{}, err error) {
	if idx >= len(fw.nodes) {
		return nil, fmt.Errorf("node index greater than quantity of nodes")
	}
	node := fw.nodes[idx]
	respBody, err := PostRPC(method, NewEndpoint(node.RPCPort), params)
	if err != nil {
		return nil, err
	}

	err = DecodeRPC_NT(respBody, &respJSON)
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
	items, err := fw.db.ReadAll("rpc")
	if err != nil {
		log.Fatal(fmt.Errorf("error reading from db %v", err))
	}
	for _, item := range items {
		fmt.Printf("%v\n", item)
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
