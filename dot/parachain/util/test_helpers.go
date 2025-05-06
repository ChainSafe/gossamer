package util

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var genesisHash = common.Hash{0}

// GetBlockHeader returns a block header for the given hash from the chain of hashes.
func GetBlockHeader(t *testing.T, chain []common.Hash, hash common.Hash) *types.Header {
	t.Helper()

	idx := -1
	for i, h := range chain {
		if h == hash {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}

	var parentHash common.Hash
	if idx > 0 {
		parentHash = chain[idx-1]
	} else {
		parentHash = genesisHash
	}

	var number uint
	if hash == genesisHash {
		number = 0
	} else {
		number = uint(idx) + 1
	}

	return &types.Header{
		ParentHash: parentHash,
		Number:     number,
	}
}
