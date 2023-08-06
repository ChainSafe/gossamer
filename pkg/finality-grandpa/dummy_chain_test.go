// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

const GenesisHash = "genesis"
const nullHash = "NULL"

type blockRecord struct {
	hash   string
	number uint32
	parent string
}

// dummyChain is translation of finality_grandpa::testing::chain::DummyChain
type dummyChain struct {
	inner     map[string]blockRecord
	leaves    []blockRecord
	finalized struct {
		hash   string
		number uint32
	}
}

func newDummyChain() *dummyChain {
	dc := &dummyChain{
		inner:  make(map[string]blockRecord),
		leaves: make([]blockRecord, 0),
	}
	dc.inner[GenesisHash] = blockRecord{
		number: 1,
		parent: nullHash,
		hash:   GenesisHash,
	}
	dc.leaves = append(dc.leaves, dc.inner[GenesisHash])
	dc.finalized.hash = GenesisHash
	dc.finalized.number = 1
	return dc
}

func (dc *dummyChain) Ancestry(base, block string) (ancestors []string, err error) {
	ancestors = make([]string, 0)
loop:
	for {
		br, ok := dc.inner[block]
		if !ok {
			// TODO: make this sentinel error on entire package
			return nil, fmt.Errorf("Block not descendent of base")
		}
		block = br.parent

		switch block {
		case nullHash:
			return nil, fmt.Errorf("Block not descendent of base")
		case base:
			break loop
		}
		ancestors = append(ancestors, block)
	}
	return ancestors, nil
}

func (dc *dummyChain) IsEqualOrDescendantOf(base, block string) bool {
	if base == block {
		return true
	}

	_, err := dc.Ancestry(base, block)
	return err == nil
}

func (dc *dummyChain) PushBlocks(parent string, blocks []string) {
	br, ok := dc.inner[parent]
	if !ok {
		panic("could not find parent hash")
	}
	baseNumber := br.number + 1

	for i, leaf := range dc.leaves {
		if leaf.hash == parent {
			dc.leaves = append(dc.leaves[:i], dc.leaves[i+1:]...)
		}
	}

	for i, descendant := range blocks {
		dc.inner[descendant] = blockRecord{
			hash:   descendant,
			number: baseNumber + uint32(i),
			parent: parent,
		}
		parent = descendant
	}

	newLeafHash := blocks[len(blocks)-1]
	newLeaf := dc.inner[newLeafHash]
	insertionIndex, _ := slices.BinarySearchFunc(dc.leaves, newLeaf, func(a, b blockRecord) int {
		switch {
		case a.number == b.number:
			return 0
		case a.number > b.number:
			return -1
		case b.number > a.number:
			return 1
		default:
			panic("huh?")
		}
	})

	switch {
	case len(dc.leaves) == 0 && insertionIndex == 0:
		dc.leaves = append(dc.leaves, newLeaf)
	case insertionIndex == len(dc.leaves):
		dc.leaves = append(dc.leaves, newLeaf)
	default:
		dc.leaves = append(
			dc.leaves[:insertionIndex],
			append([]blockRecord{newLeaf}, dc.leaves[insertionIndex:]...)...)
	}
}

func (dc *dummyChain) Number(hash string) uint32 {
	e, ok := dc.inner[hash]
	if !ok {
		panic("huh?")
	}
	return e.number
}

func (dc *dummyChain) LastFinalized() (string, uint32) {
	return dc.finalized.hash, dc.finalized.number
}

func (dc *dummyChain) SetLastFinalized(hash string, number uint32) {
	dc.finalized.hash = hash
	dc.finalized.number = number
}

func (dc *dummyChain) BestChainContaining(base string) *HashNumber[string, uint32] {
	baseRecord, ok := dc.inner[base]
	if !ok {
		return nil
	}
	baseNumber := baseRecord.number

	for _, leaf := range dc.leaves {
		// leaves are in descending order.
		leafNumber := leaf.number
		if leafNumber < baseNumber {
			break
		}

		if leaf.hash == base {
			return &HashNumber[string, uint32]{leaf.hash, leafNumber}
		}

		_, err := dc.Ancestry(base, leaf.hash)
		if err == nil {
			return &HashNumber[string, uint32]{leaf.hash, leafNumber}
		}
	}

	return nil
}

func TestDummyGraphPushBlocks(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{
		"A", "B", "C",
	})
	c.PushBlocks(GenesisHash, []string{
		"A'", "B'", "C'",
	})
	assert.Equal(t, []blockRecord{
		{hash: "C'", number: 4, parent: "B'"},
		{hash: "C", number: 4, parent: "B"},
	}, c.leaves)
	assert.Equal(t, c.inner, map[string]blockRecord{
		GenesisHash: {GenesisHash, 1, nullHash},
		"A":         {"A", 2, GenesisHash},
		"A'":        {"A'", 2, GenesisHash},
		"B":         {"B", 3, "A"},
		"B'":        {"B'", 3, "A'"},
		"C":         {"C", 4, "B"},
		"C'":        {"C'", 4, "B'"},
	})
}
