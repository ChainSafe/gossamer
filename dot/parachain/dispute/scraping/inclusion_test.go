package scraping

import (
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/lib/common"
)

func TestInclusions_Insert(t *testing.T) {
	t.Parallel()
	inclusions := &Inclusions{inner: make(map[common.Hash]map[uint32][]common.Hash)}
	candidateHash := types.GetRandomHash()
	blockHash1 := types.GetRandomHash()
	blockHash2 := types.GetRandomHash()

	inclusions.Insert(candidateHash, blockHash1, 1)
	require.Equal(t, map[common.Hash]map[uint32][]common.Hash{
		candidateHash: {
			1: {blockHash1},
		},
	}, inclusions.inner)

	inclusions.Insert(candidateHash, blockHash2, 1)
	require.Equal(t, map[common.Hash]map[uint32][]common.Hash{
		candidateHash: {
			1: {blockHash1, blockHash2},
		},
	}, inclusions.inner)

	inclusions.Insert(candidateHash, blockHash1, 2)
	require.Equal(t, map[common.Hash]map[uint32][]common.Hash{
		candidateHash: {
			1: {blockHash1, blockHash2},
			2: {blockHash1},
		},
	}, inclusions.inner)
}

func TestInclusions_RemoveUpToHeight(t *testing.T) {
	t.Parallel()
	inclusions := &Inclusions{inner: make(map[common.Hash]map[uint32][]common.Hash)}
	candidateHash1 := types.GetRandomHash()
	candidateHash2 := types.GetRandomHash()
	blockHash1 := types.GetRandomHash()
	blockHash2 := types.GetRandomHash()

	inclusions.inner[candidateHash1] = map[uint32][]common.Hash{
		1: {blockHash1},
		2: {blockHash2},
	}
	inclusions.inner[candidateHash2] = map[uint32][]common.Hash{
		1: {blockHash1},
	}

	inclusions.RemoveUpToHeight(1, []common.Hash{candidateHash1, candidateHash2})
	require.Equal(t, 2, len(inclusions.inner))

	inclusions.RemoveUpToHeight(2, []common.Hash{candidateHash1, candidateHash2})
	require.Equal(t, 1, len(inclusions.inner))

	inclusions.RemoveUpToHeight(3, []common.Hash{candidateHash1, candidateHash2})
	require.Equal(t, 0, len(inclusions.inner))
}

func TestInclusions_Get(t *testing.T) {
	t.Parallel()
	inclusions := &Inclusions{inner: make(map[common.Hash]map[uint32][]common.Hash)}
	candidateHash := types.GetRandomHash()
	blockHash1 := types.GetRandomHash()
	blockHash2 := types.GetRandomHash()

	inclusions.inner[candidateHash] = map[uint32][]common.Hash{
		2: {blockHash1},
		1: {blockHash2},
	}

	expected := []Inclusion{
		{BlockNumber: 1, BlockHash: blockHash2},
		{BlockNumber: 2, BlockHash: blockHash1},
	}

	require.Equal(t, expected, inclusions.Get(candidateHash))
}
