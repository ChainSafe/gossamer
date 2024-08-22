// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/ChainSafe/gossamer/scripts/p2p"
	lip2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var (
	errZeroLengthResponse = errors.New("zero length response")
	errEmptyStateEntries  = errors.New("empty state entries")
)

type StateRequestProvider struct {
	lastKeys           [][]byte
	collectedResponses []*messages.StateResponse
	targetHash         common.Hash
	completed          bool
}

func NewStateRequestProvider(target common.Hash) *StateRequestProvider {
	return &StateRequestProvider{
		lastKeys:           [][]byte{},
		targetHash:         target,
		collectedResponses: make([]*messages.StateResponse, 0),
	}
}

func (s *StateRequestProvider) buildRequest() *messages.StateRequest {
	return &messages.StateRequest{
		Block:   s.targetHash,
		Start:   s.lastKeys,
		NoProof: true,
	}
}

func (s *StateRequestProvider) processResponse(stateResponse *messages.StateResponse) (err error) {
	if len(stateResponse.Entries) == 0 {
		return errEmptyStateEntries
	}

	log.Printf("retrieved %d entries\n", len(stateResponse.Entries))
	for idx, entry := range stateResponse.Entries {
		log.Printf("\t#%d with %d entries (complete: %v, root: %s)\n",
			idx, len(entry.StateEntries), entry.Complete, entry.StateRoot.String())
	}

	s.collectedResponses = append(s.collectedResponses, stateResponse)

	if len(s.lastKeys) == 2 && len(stateResponse.Entries[0].StateEntries) == 0 {
		// pop last item and keep the first
		// do not remove the parent trie position.
		s.lastKeys = s.lastKeys[:len(s.lastKeys)-1]
	} else {
		s.lastKeys = [][]byte{}
	}

	for _, state := range stateResponse.Entries {
		if !state.Complete {
			lastItemInResponse := state.StateEntries[len(state.StateEntries)-1]
			s.lastKeys = append(s.lastKeys, lastItemInResponse.Key)
			s.completed = false
		} else {
			s.completed = true
		}
	}

	return nil
}

func (s *StateRequestProvider) buildTrie(expectedStorageRootHash common.Hash, destination string) error {
	tt := inmemory.NewEmptyTrie()
	tt.SetVersion(trie.V1)

	entries := make([]string, 0)

	for _, stateResponse := range s.collectedResponses {
		for _, stateEntry := range stateResponse.Entries {
			for _, kv := range stateEntry.StateEntries {

				trieEntry := trie.Entry{Key: kv.Key, Value: kv.Value}
				encodedTrieEntry, err := scale.Marshal(trieEntry)
				if err != nil {
					return err
				}
				entries = append(entries, common.BytesToHex(encodedTrieEntry))

				if err := tt.Put(kv.Key, kv.Value); err != nil {
					return err
				}
			}
		}
	}

	rootHash := tt.MustHash()
	if expectedStorageRootHash != rootHash {
		log.Printf("\n\texpected root hash: %s\ngot root hash: %s\n",
			expectedStorageRootHash.String(), rootHash.String())
	}

	fmt.Printf("=> trie root hash: %s\n", tt.MustHash().String())
	encodedEntries, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	err = os.WriteFile(destination, encodedEntries, 0o600)
	return err
}

func main() {
	if len(os.Args) != 5 {
		log.Fatalf(`
		script usage:
			go run retrieve_state.go [hash] [expected storage root hash] [network chain spec] [output file]`)
	}

	targetBlockHash := common.MustHexToHash(os.Args[1])
	expectedStorageRootHash := common.MustHexToHash(os.Args[2])
	chain := p2p.ParseChainSpec(os.Args[3])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protocolID := protocol.ID(fmt.Sprintf("/%s/state/2", chain.ProtocolID))

	p2pHost := p2p.SetupP2PClient()
	bootnodes := p2p.ParseBootnodes(chain.Bootnodes)
	provider := NewStateRequestProvider(targetBlockHash)

	var (
		pid           peer.AddrInfo
		refreshPeerID bool = true
	)

	for !provider.completed {
		if refreshPeerID {
			pid = bootnodes[rand.Intn(len(bootnodes))]
			err := p2pHost.Connect(ctx, pid)
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

	if err := provider.buildTrie(expectedStorageRootHash, os.Args[4]); err != nil {
		panic(err)
	}
}

func sendAndProcessResponse(provider *StateRequestProvider, stream lip2pnetwork.Stream) error {
	defer stream.Close()

	err := p2p.WriteStream(provider.buildRequest(), stream)
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

	err = provider.processResponse(stateResponse)
	if err != nil {
		return err
	}

	return nil
}
