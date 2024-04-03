package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

// GetChangedNodeHashes returns the two sets of hashes for all nodes
// inserted and deleted in the state trie since the last snapshot.
// Returned inserted map is safe for mutation, but deleted is not safe for mutation.
func (t *TrieDB) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	panic("not implemented yet")
}

// HandleTrackedDeltas sets the pending deleted node hashes in
// the trie deltas tracker if and only if success is true.
func (t *TrieDB) HandleTrackedDeltas(success bool, pendingDeltas tracking.Getter) {
	panic("not implemented yet")
}
