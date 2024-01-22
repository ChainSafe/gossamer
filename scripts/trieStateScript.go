// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
)

func fetchWithTimeout(ctx context.Context,
	method, params string, target interface{}) {

	// Can adjust timeout as desired, default is very long
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

func writeTrieState(response modules.StateTrieResponse, destination string) {
	encResponse, err := json.Marshal(response)
	if err != nil {
		panic(fmt.Sprintf("json marshalling response %v", err))
	}

	err = os.WriteFile(destination, encResponse, 0o600)
	if err != nil {
		panic(fmt.Sprintf("writing to file %v", err))
	}
}

func fetchTrieState(ctx context.Context, blockHash common.Hash, destination string) modules.StateTrieResponse {
	params := fmt.Sprintf(`["%s"]`, blockHash)
	var response modules.StateTrieResponse
	fetchWithTimeout(ctx, "state_trie", params, &response)

	writeTrieState(response, destination)
	return response
}

func compareStateRoots(response modules.StateTrieResponse, expectedStateRoot common.Hash, trieVersion int) {
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

	trieHash := common.Hash{} //nolint
	if trieVersion == 0 {
		trieHash = trie.V0.MustHash(newTrie)
	} else if trieVersion == 1 {
		trieHash = trie.V1.MustHash(newTrie)
	} else {
		panic("invalid trie version")
	}

	if expectedStateRoot != trieHash {
		panic("westendDevStateRoot does not match trieHash")
	}
}

/*
This is a script to query the trie state from a specific block height from a running node.

Example commands to run a node:

 1. ./bin/gossamer init --chain westend-dev --key alice

 2. ./bin/gossamer --chain westend-dev --key alice --rpc-external=true --unsafe-rpc=true

Once the node has started and processed the block whose state you need, can execute the script like so:
 1. go run trieStateScript.go <block hash> <destination file> <optional: expected state root> <optional: trie version>
*/
func main() {
	if len(os.Args) < 3 {
		panic("expected more arguments, block hash and destination file required")
	}

	blockHash, err := common.HexToHash(os.Args[1])
	if err != nil {
		panic("block hash must be in hex format")
	}

	destinationFile := os.Args[2]
	expectedStateRoot := common.Hash{}
	var trieVersion int
	if len(os.Args) == 5 {
		expectedStateRoot, err = common.HexToHash(os.Args[3])
		if err != nil {
			panic("expected state root must be in hex format")
		}

		trieVersion, err = strconv.Atoi(os.Args[4])
		if err != nil {
			panic("trie version must be an integer")
		}
	} else if len(os.Args) != 3 {
		panic("invalid number of arguments")
	}

	ctx, _ := context.WithCancel(context.Background()) //nolint
	response := fetchTrieState(ctx, blockHash, destinationFile)

	if !expectedStateRoot.IsEmpty() {
		compareStateRoots(response, expectedStateRoot, trieVersion)
	}
}
