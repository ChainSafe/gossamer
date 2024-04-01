package triedb

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

func (t *TrieDB) String() string {
	if t.rootHash == common.EmptyHash {
		return "empty"
	}

	return fmt.Sprintf("TrieDB: %v", t.rootHash)
}
