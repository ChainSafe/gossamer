// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"os"
	"time"
)

func fetchWithTimeout(ctx context.Context,
	method, params string, target interface{}) {

	getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
	defer getResponseCancel()
	err := getResponse(getResponseCtx, method, params, target)
	if err != nil {
		panic(fmt.Sprintf("error getting response %v", err))
	}
}

func getResponse(ctx context.Context, method, params string, target interface{}) (err error) {
	//const rpcPort = "8545"
	//endpoint := rpc.NewEndpoint(rpcPort)
	//"http://18.222.22.42:8545"
	endpoint := "http://18.222.22.42:8545"
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
1)  ./bin/gossamer init --chain westend-dev --key alice
1)  ./bin/gossamer --chain westend-dev --key alice --rpc-external=true --unsafe-rpc=true
finalised block number 512 with hash 0x5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e
*/
func main() {
	// Get block hash from cli
	blockHash := os.Args[1]

	// Goal, take hash as input and then write to file
	ctx, _ := context.WithCancel(context.Background())

	// Starting with just genesis info to get working
	//const westendDevGenesisHash = "0x276bfa91f70859348285599321ea96afd3ae681f0be47d36196bac8075ea32e8"
	//const westendDevStateRoot = "0x953044ba4386a72ae434d2a2fbdfca77640a28ac3841a924674cbfe7a8b9a81c"
	const westend512StateRoot = "0xe6e996b91f9ec0c2900099754777c13644bb742dd512530dc6230aeaee2c19f9"
	params := fmt.Sprintf(`["%s"]`, blockHash)

	var response modules.StateTrieResponse
	fetchWithTimeout(ctx, "state_trie", params, &response)

	// THis is needed stuff
	encResponse, err := json.Marshal(response)
	if err != nil {
		panic(fmt.Sprintf("json marshalling response %v", err))
	}

	err = os.WriteFile("trie_state_data", encResponse, 0o600)
	if err != nil {
		panic(fmt.Sprintf("writing to file %v", err))
	}
	// Below is for testing correctness

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
	if westend512StateRoot != trieHash.String() {
		panic(fmt.Sprintf("westendDevStateRoot does not match trieHash"))
	}
}
