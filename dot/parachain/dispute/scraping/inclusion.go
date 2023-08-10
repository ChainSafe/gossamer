package scraping

import (
	"sort"

	"github.com/ChainSafe/gossamer/lib/common"
)

type Inclusion struct {
	BlockNumber uint32
	BlockHash   common.Hash
}

type Inclusions struct {
	inner map[common.Hash]map[uint32][]common.Hash
}

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

func (i *Inclusions) RemoveUpToHeight(blockNumber uint32, candidatesModified []common.Hash) {
	for _, candidate := range candidatesModified {
		blocksIncluding, ok := i.inner[candidate]
		if ok {
			for height := range blocksIncluding {
				if height <= blockNumber {
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

func (i *Inclusions) Get(candidateHash common.Hash) []Inclusion {
	var inclusionsAsSlice []Inclusion
	blocksIncluding, ok := i.inner[candidateHash]
	if ok {
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
	}

	return inclusionsAsSlice
}

func NewInclusions() Inclusions {
	return Inclusions{
		inner: make(map[common.Hash]map[uint32][]common.Hash),
	}
}
