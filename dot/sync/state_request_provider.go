package sync

import (
	"errors"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
)

var (
	errEmptyStateEntries = errors.New("empty state entries")
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

func (s *StateRequestProvider) buildRequest() *messages.StateRequest {
	return &messages.StateRequest{
		Block:   s.targetHash,
		Start:   s.lastKeys,
		NoProof: true,
	}
}

func (s *StateRequestProvider) processResponse(stateResponse *messages.StateResponse) (completed bool, err error) {
	if len(stateResponse.Entries) == 0 {
		return false, errEmptyStateEntries
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

	for _, state := range stateResponse.Entries {
		if !state.Complete {
			lastItemInResponse := state.StateEntries[len(state.StateEntries)-1]
			s.lastKeys = append(s.lastKeys, lastItemInResponse.Key)
			s.completed = false
		} else {
			s.completed = true
		}
	}

	return s.completed, nil
}

func (s *StateRequestProvider) buildTrie() (trie.Trie, error) {
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

	rootHash := stateTrie.MustHash()
	if s.targetHash != rootHash {
		logger.Errorf("root mismatch, expected root hash: %s, got root hash: %s",
			s.targetHash.String(), rootHash.String())
	}

	return stateTrie, nil
}

func (s *StateRequestProvider) getLastKeys() [][]byte {
	return s.lastKeys

}
