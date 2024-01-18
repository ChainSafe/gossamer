// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
)

func fetchWithTimeout(ctx context.Context,
	method, params string, target interface{}) {

	getResponseCtx, getResponseCancel := context.WithTimeout(ctx, 1000000*time.Second)
	defer getResponseCancel()
	err := getResponse(getResponseCtx, method, params, target)
	if err != nil {
		panic(fmt.Sprintf("error getting response %v", err))
	}
}

func getResponse(ctx context.Context, method, params string, target interface{}) (err error) {
	const rpcPort = "8545"
	endpoint := rpc.NewEndpoint(rpcPort)
	respBody, err := rpc.Post(ctx, endpoint, method, params)
	if err != nil {
		return fmt.Errorf("cannot RPC post: %w", err)
	}

	err = rpc.Decode(respBody, &target)
	if err != nil {
		return fmt.Errorf("cannot decode RPC response: %w", err)
	}

	return nil
}

/*
This is a script to query the trie state from a specific block height from a running node.

Example commands to run a node:
 1. ./bin/gossamer init --chain westend-dev --key alice
 2. ./bin/gossamer --chain westend-dev --key alice --rpc-external=true --unsafe-rpc=true

Once the node has started and processed the block whose state you need, can execute the script like so:
 1. go run trieStateScript.go <block hash>
*/
func main() {
	// Get block hash from cli (i.e. 0x276bfa91f70859348285599321ea96afd3ae681f0be47d36196bac8075ea32e8)
	blockHash := os.Args[1]

	ctx, _ := context.WithCancel(context.Background()) //nolint

	// Plug in the expected state root if you wish to assert the calculated state root
	const expectedStateRoot = ""
	params := fmt.Sprintf(`["%s"]`, blockHash)

	var response modules.StateTrieResponse
	fetchWithTimeout(ctx, "state_trie", params, &response)

	encResponse, err := json.Marshal(response)
	if err != nil {
		panic(fmt.Sprintf("json marshalling response %v", err))
	}

	err = os.WriteFile("trie_state_data", encResponse, 0o600)
	if err != nil {
		panic(fmt.Sprintf("writing to file %v", err))
	}

	// Below is for testing correctness, can be commented out if this is not desired
	entries := make(map[string]string, len(response))
	for _, encodedEntry := range response {
		bytesEncodedEntry := common.MustHexToBytes(encodedEntry)

		entry := trie.Entry{}
		err := scale.Unmarshal(bytesEncodedEntry, &entry)
		if err != nil {
			panic(fmt.Sprintf("error unmarshalling into trie entry %v", err))
		}
		entries[common.BytesToHex(entry.Key)] = common.BytesToHex(entry.Value)
	}

	newTrie, err := trie.LoadFromMap(entries)
	if err != nil {
		panic(fmt.Sprintf("loading trie from map %v", err))
	}

	trieHash := newTrie.MustHash(trie.V0.MaxInlineValue())
	if expectedStateRoot != trieHash.String() {
		panic("westendDevStateRoot does not match trieHash")
	}
}
