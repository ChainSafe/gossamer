// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/ChainSafe/gossamer/scripts/p2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func buildStateRequestMessage(target common.Hash, start [][]byte, noProof bool) *messages.StateRequest {
	return &messages.StateRequest{
		Block:   target,
		Start:   start,
		NoProof: noProof,
	}
}

func buildTrieFromFolder(folder string) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		panic(err)
	}

	tt := inmemory.NewEmptyTrie()
	tt.SetVersion(trie.V1)

	for _, file := range entries {
		content, err := os.ReadFile(filepath.Join(folder, file.Name()))
		if err != nil {
			panic(err)
		}

		stateResponse := &messages.StateResponse{}
		err = stateResponse.Decode(common.MustHexToBytes(string(content)))
		if err != nil {
			panic(err)
		}

		for _, stateEntry := range stateResponse.Entries {
			for _, kv := range stateEntry.StateEntries {
				if err := tt.Put(kv.Key, kv.Value); err != nil {
					panic(err)
				}
			}
		}
	}

	fmt.Printf("=> %s\n", tt.MustHash().String())
}

func main() {
	buildTrieFromFolder(filepath.Clean("./tmp"))
	return

	targetBlock := common.MustHexToHash("0xa603508126444a00249999a6d69d3a186377e1bb1b6c7ca9959d4bdb8f96ebb9")

	p2pHost := p2p.SetupP2PClient()
	chain := p2p.ParseChainSpec(os.Args[1])
	bootnodes := p2p.ParseBootnodes(chain.Bootnodes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protocolID := protocol.ID(fmt.Sprintf("/%s/state/2", chain.ProtocolID))

	requestMessage := buildStateRequestMessage(targetBlock, nil, true)
	keepPID := false
	var pid peer.AddrInfo
	idx := 0
	lastKeys := [][]byte{}
	complete := false

	for !complete {
		if !keepPID {
			pid = bootnodes[rand.Intn(len(bootnodes))]
			err := p2pHost.Connect(ctx, pid)
			if err != nil {
				continue
			}
			log.Printf("requesting from peer %s\n", pid.String())
		}

		stream, err := p2pHost.NewStream(ctx, pid.ID, protocolID)
		if err != nil {
			log.Printf("WARN: failed to create stream using protocol %s: %s", protocolID, err.Error())
			keepPID = false
			continue
		}

		err = p2p.WriteStream(requestMessage, stream)
		if err != nil {
			log.Println(err.Error())
			stream.Close()
			keepPID = false
			continue
		}

		output, err := p2p.ReadStream(stream)
		stream.Close()

		if len(output) == 0 {
			keepPID = false
			continue
		}

		if err != nil {
			log.Println(err.Error())
			keepPID = false
			continue
		}

		stateResponse := &messages.StateResponse{}
		err = stateResponse.Decode(output)
		if err != nil {
			log.Println(err.Error())
			keepPID = false
			continue
		}

		if len(stateResponse.Entries) == 0 {
			keepPID = false
			log.Printf("received empty state response entries from %s\n", pid.String())
			continue
		}

		log.Printf("retrieved %d entries\n", len(stateResponse.Entries))
		for idx, entry := range stateResponse.Entries {
			log.Printf("\t#%d with %d entries (complete: %v, root: %s)\n",
				idx, len(entry.StateEntries), entry.Complete, entry.StateRoot.String())
		}

		outputFile := fmt.Sprintf("%d_%s.bin", idx, targetBlock.String())
		err = os.WriteFile(filepath.Join("tmp", outputFile), []byte(common.BytesToHex(output)), os.ModePerm)
		if err != nil {
			log.Fatalf("failed to write %s: %s", outputFile, err.Error())
		}

		if len(lastKeys) == 2 && len(stateResponse.Entries[0].StateEntries) == 0 {
			// pop last item and keep the first
			// do not remove the parent trie position.
			lastKeys = lastKeys[:len(lastKeys)-1]
		} else {
			lastKeys = [][]byte{}
		}

		for _, state := range stateResponse.Entries {
			if !state.Complete {
				lastItemInResponse := state.StateEntries[len(state.StateEntries)-1]
				lastKeys = append(lastKeys, lastItemInResponse.Key)
				complete = false
			} else {
				complete = true
			}
		}

		requestMessage = buildStateRequestMessage(targetBlock, lastKeys, true)
		idx++
		keepPID = true
	}

}
