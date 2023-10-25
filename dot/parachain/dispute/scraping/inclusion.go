package scraping

import (
	"sort"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Inclusion the block number and hash of a block that includes a candidate.
type Inclusion struct {
	BlockNumber uint32
	BlockHash   common.Hash
}

// Inclusions stores the candidates that are included in blocks.
type Inclusions struct {
	// CandidateHash -> [BlockNumber -> BlockHash]
	inner map[common.Hash]map[uint32][]common.Hash
}

// Insert inserts a new inclusion into the Inclusions.
func (i *Inclusions) Insert(candidateHash, blockHash common.Hash, blockNumber uint32) {
	if _, ok := i.inner[candidateHash]; !ok {
		i.inner[candidateHash] = make(map[uint32][]common.Hash)
		i.inner[candidateHash][blockNumber] = []common.Hash{blockHash}
		return
	}

	if _, ok := i.inner[candidateHash][blockNumber]; !ok {
		i.inner[candidateHash][blockNumber] = []common.Hash{blockHash}
		return
	}

	i.inner[candidateHash][blockNumber] = append(i.inner[candidateHash][blockNumber], blockHash)
}

// RemoveUpToHeight removes all inclusions up to the given block number.
func (i *Inclusions) RemoveUpToHeight(blockNumber uint32, candidatesModified []common.Hash) {
	for _, candidate := range candidatesModified {
		blocksIncluding, ok := i.inner[candidate]
		if ok {
			for height := range blocksIncluding {
				if height < blockNumber {
					delete(blocksIncluding, height)
				}
			}

			// Clean up empty inner maps
			if len(blocksIncluding) == 0 {
				delete(i.inner, candidate)
			}
		}
	}
}

// Get returns all inclusions for the given candidate hash.
func (i *Inclusions) Get(candidateHash common.Hash) []Inclusion {
	var inclusionsAsSlice []Inclusion
	blocksIncluding, ok := i.inner[candidateHash]
	if !ok {
		return inclusionsAsSlice
	}

	// Convert the map to a sorted slice for iteration
	var sortedKeys []uint32
	for h := range blocksIncluding {
		sortedKeys = append(sortedKeys, h)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] < sortedKeys[j]
	})

	// Extract inclusions as a slice of structs
	for _, height := range sortedKeys {
		blocksAtHeight := blocksIncluding[height]
		for _, block := range blocksAtHeight {
			inclusionsAsSlice = append(inclusionsAsSlice, Inclusion{
				BlockNumber: height,
				BlockHash:   block,
			})
		}
	}

	return inclusionsAsSlice
}

// NewInclusions creates a new Inclusions.
func NewInclusions() Inclusions {
	return Inclusions{
		inner: make(map[common.Hash]map[uint32][]common.Hash),
	}
}
