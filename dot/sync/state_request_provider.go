// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
)

var (
	ErrEmptyStateEntries = errors.New("empty state entries")
)

type StateRequestProvider struct {
	lastKeys           [][]byte
	collectedResponses []*messages.StateResponse
	targetHash         common.Hash
	completed          bool
	stateVersion       trie.TrieLayout
}

func NewStateRequestProvider(target common.Hash, stateVersion trie.TrieLayout) *StateRequestProvider {
	return &StateRequestProvider{
		lastKeys:           [][]byte{},
		targetHash:         target,
		collectedResponses: make([]*messages.StateResponse, 0),
		stateVersion:       stateVersion,
	}
}

func (s *StateRequestProvider) BuildRequest() *messages.StateRequest {
	return &messages.StateRequest{
		Block:   s.targetHash,
		Start:   s.lastKeys,
		NoProof: true,
	}
}

func (s *StateRequestProvider) ProcessResponse(stateResponse *messages.StateResponse) (completed bool, err error) {
	// TODO: handle merkle proofs to prevent accepting invalid state entries. Issue: #4481

	if len(stateResponse.Entries) == 0 {
		return false, ErrEmptyStateEntries
	}

	logger.Debugf("retrieved %d entries", len(stateResponse.Entries))
	for idx, entry := range stateResponse.Entries {
		logger.Debugf("#%d with %d entries (complete: %v, root: %s)",
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

	s.completed = true

	for _, state := range stateResponse.Entries {
		if !state.Complete {
			lastItemInResponse := state.StateEntries[len(state.StateEntries)-1]
			s.lastKeys = append(s.lastKeys, lastItemInResponse.Key)
			s.completed = false
		}
	}

	return s.completed, nil
}

func (s *StateRequestProvider) BuildTrie() (trie.Trie, error) {
	stateTrie := inmemory.NewEmptyTrie()
	stateTrie.SetVersion(s.stateVersion)

	for _, stateResponse := range s.collectedResponses {
		for _, stateEntry := range stateResponse.Entries {
			for _, kv := range stateEntry.StateEntries {
				if err := stateTrie.Put(kv.Key, kv.Value); err != nil {
					return nil, err
				}
			}
		}
	}

	return stateTrie, nil
}

func (s *StateRequestProvider) GetLastKeys() [][]byte {
	return s.lastKeys
}

func (s *StateRequestProvider) IsCompleted() bool {
	return s.completed
}
