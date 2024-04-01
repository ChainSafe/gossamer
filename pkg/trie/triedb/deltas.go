package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

func (t *TrieDB) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	panic("implement me")
}

func (t *TrieDB) HandleTrackedDeltas(success bool, pendingDeltas tracking.Getter) {
	panic("implement me")
}
