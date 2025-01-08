// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/scripts/p2p"
	lip2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var (
	errZeroLengthResponse = errors.New("zero length response")

	supportedVersions = map[string]trie.TrieLayout{
		"v0": trie.V0,
		"v1": trie.V1,
	}
)

func main() {
	if len(os.Args) != 6 {
		log.Fatalf(`script usage:
	go run retrieve_state.go [block hash] [expected state root hash] [network chain spec] [v0|v1] [output file]`)
	}

	version, ok := supportedVersions[os.Args[4]]
	if !ok {
		log.Fatalf("ERR version not supported: %s", os.Args[4])
	}

	targetBlockHash := common.MustHexToHash(os.Args[1])
	expectedStorageRootHash := common.MustHexToHash(os.Args[2])
	chain := p2p.ParseChainSpec(os.Args[3])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protocolID := protocol.ID(fmt.Sprintf("/%s/state/2", chain.ProtocolID))

	p2pHost := p2p.SetupP2PClient()
	bootnodes := p2p.ParseBootnodes(chain.Bootnodes)
	provider := sync.NewStateRequestProvider(targetBlockHash, version)

	var (
		pid           peer.AddrInfo
		refreshPeerID bool = true
	)

	for !provider.IsCompleted() {
		if refreshPeerID {
			rng, err := rand.Int(rand.Reader, big.NewInt(int64(len(bootnodes))))
			if err != nil {
				panic(err)
			}

			pid = bootnodes[rng.Uint64()]
			err = p2pHost.Connect(ctx, pid)
			if err != nil {
				log.Printf("WARN: while connecting: %s\n", err.Error())
				continue
			}

			log.Printf("OK: requesting from peer %s\n", pid.String())
		}

		stream, err := p2pHost.NewStream(ctx, pid.ID, protocolID)
		if err != nil {
			log.Printf("WARN: failed to create stream using protocol %s: %s", protocolID, err.Error())
			refreshPeerID = false
			continue
		}

		err = sendAndProcessResponse(provider, stream)
		if err != nil {
			log.Printf("WARN: %s\n", err.Error())
			refreshPeerID = true
			continue
		}

		// keep using the same peer
		refreshPeerID = false
	}

	trie, err := provider.BuildTrie()
	if err != nil {
		panic(err)
	}

	if trie.MustHash() != expectedStorageRootHash {
		panic(fmt.Errorf("ERR: state root mismatch: got %s expected %s", trie.MustHash(), expectedStorageRootHash))
	}

	encodedEntries, err := json.Marshal(trie.Entries())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(os.Args[5], encodedEntries, 0o600)
	if err != nil {
		panic(err)
	}
}

func sendAndProcessResponse(provider *sync.StateRequestProvider, stream lip2pnetwork.Stream) error {
	defer stream.Close() //nolint:errcheck

	request := provider.BuildRequest()

	fmt.Printf("Sending request to peer %s\n", request.String())
	err := p2p.WriteStream(provider.BuildRequest(), stream)
	if err != nil {
		return err
	}

	output, err := p2p.ReadStream(stream)
	if err != nil {
		return err
	}

	if len(output) == 0 {
		return errZeroLengthResponse
	}

	stateResponse := &messages.StateResponse{}
	err = stateResponse.Decode(output)
	if err != nil {
		return err
	}

	_, err = provider.ProcessResponse(stateResponse)
	if err != nil {
		return err
	}

	return nil
}
