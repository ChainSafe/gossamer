package utils

import (
	"fmt"
	scribble "github.com/nanobox-io/golang-scribble"
	"io/ioutil"
	"strconv"
	"testing"
)

type Framework struct {
	nodes []*Node
	db *scribble.Driver
	callQty int
}

func InitFramework(qtyNodes int) (*Framework, error) {
	f := &Framework{	}
	nodes, err := InitNodes(qtyNodes)
	if err != nil {
		return nil, err
	}
	f.nodes = nodes

	tempDir, err := ioutil.TempDir("", "gossamer-stress-db")
	db, err := scribble.New(tempDir, nil)
	f.db = db

	return f, nil
}

func (fw *Framework) StartNodes(t *testing.T) (errorList []error) {
	for _, node := range fw.nodes {
		err := RestartGossamer(t, node)
		if err != nil {
			errorList = append(errorList, err)
		}
	}
	return errorList
}

func (fw *Framework) KillNodes(t *testing.T) []error {
	return TearDown(t, fw.nodes)
}

//func (fw *Framework) StoreChainHeads() {
//	for _, node := range fw.nodes {
//		res, err := CallRPC(node, "chain_getHeader", "[]")
//		fmt.Errorf("error getting chain header %v", err)
//		fmt.Printf("resp %v\n", res["number"])
//		err = fw.db.Write("blocks_"+node.Key, res["number"].(string), res)
//		if err != nil {
//			fmt.Errorf("error writting to db %v", err)
//		}
//	}
//}

// TODO ed, should params be []string instead?
func (fw *Framework) CallRPC(idx int, method, params string) (respJson map[string]interface{}, err error) {
	node := fw.nodes[idx]
	respBody, err := PostRPC(method, NewEndpoint(node.RPCPort), params)
	if err != nil {
		return nil, err
	}

	respJson = make(map[string]interface{})
	err = DecodeRPC_NT(respBody, &respJson)
	err = fw.db.Write("node_"+ strconv.Itoa(node.Idx), strconv.Itoa(fw.callQty), respJson)
	fmt.Printf("RPC call fi %v se %v \n", "node_"+strconv.Itoa(node.Idx), strconv.Itoa(fw.callQty))
	if err != nil {
		fmt.Errorf("error writting to db %v", err)
	}

	fw.callQty++

	return
}

func (fw *Framework) PrintDB(idx int) {
	items, err := fw.db.ReadAll("node_" + fw.nodes[idx].Key)
	if err != nil {
		fmt.Errorf("error reading from db %v\n", err)
	}
	for _, item := range items {
		fmt.Printf("%v\n", item)
	}
}

func (fw *Framework) PrintRecord(nodeIdx int, callIdx int) {
	v := make(map[string] interface{})
	err := fw.db.Read("node_" + strconv.Itoa(nodeIdx), strconv.Itoa(callIdx), &v)
	if err != nil {
		fmt.Errorf("error reading from db %v\n", err)
	}
	fmt.Printf("collection: %v resource: %v v: %v\n", "node_"+strconv.Itoa(nodeIdx), strconv.Itoa(callIdx), v)
}