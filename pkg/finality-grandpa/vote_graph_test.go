// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type uintVoteNode uint

func (uvn *uintVoteNode) Add(other *uintVoteNode) {
	*uvn += *other
}

func (uvn *uintVoteNode) AddVote(other int) {
	*uvn += uintVoteNode(other)
}

func (uvn *uintVoteNode) String() string {
	return fmt.Sprintf("%+v", *uvn)
}

func (uvn *uintVoteNode) Copy() *uintVoteNode {
	copied := *uvn
	return &copied
}

func createUintVoteNode(i int) *uintVoteNode {
	vn := uintVoteNode(i)
	return &vn
}

func newUintVoteNode() *uintVoteNode {
	return createUintVoteNode(0)
}

func TestVoteGraph_GraphForkNotAtNode(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("A", 2, createUintVoteNode(100), c))
	assert.NoError(t, vg.Insert("E1", 6, createUintVoteNode(100), c))
	assert.NoError(t, vg.Insert("F2", 7, createUintVoteNode(100), c))

	assert.Contains(t, vg.heads.Keys(), "E1")
	assert.Contains(t, vg.heads.Keys(), "F2")
	assert.NotContains(t, vg.heads.Keys(), "A")

	var getEntry = func(key string) voteGraphEntry[string, uint, *uintVoteNode, int] {
		entry, _ := vg.entries.Get(key)
		return entry
	}

	assert.Equal(t, []string{"E1", "F2"}, getEntry("A").descendants)
	assert.Equal(t, createUintVoteNode(300), getEntry("A").cumulativeVote)

	assert.Equal(t, "A", *getEntry("E1").AncestorNode())
	assert.Equal(t, createUintVoteNode(100), getEntry("E1").cumulativeVote)

	assert.Equal(t, "A", *getEntry("F2").AncestorNode())
	assert.Equal(t, createUintVoteNode(100), getEntry("F2").cumulativeVote)
}

func TestVoteGraph_GraphForkNotAtNode1(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("A", 2, 100, c))
	assert.NoError(t, vg.Insert("E1", 6, 100, c))
	assert.NoError(t, vg.Insert("F2", 7, 100, c))

	assert.Contains(t, vg.heads.Keys(), "E1")
	assert.Contains(t, vg.heads.Keys(), "F2")
	assert.NotContains(t, vg.heads.Keys(), "A")

	var getEntry = func(key string) voteGraphEntry[string, uint, *uintVoteNode, int] {
		entry, _ := vg.entries.Get(key)
		return entry
	}

	assert.Equal(t, []string{"E1", "F2"}, getEntry("A").descendants)
	assert.Equal(t, createUintVoteNode(300), getEntry("A").cumulativeVote)

	assert.Equal(t, "A", *getEntry("E1").AncestorNode())
	assert.Equal(t, createUintVoteNode(100), getEntry("E1").cumulativeVote)

	assert.Equal(t, "A", *getEntry("F2").AncestorNode())
	assert.Equal(t, createUintVoteNode(100), getEntry("F2").cumulativeVote)
}

func TestVoteGraph_GraphForkAtNode(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2"})

	vn := uintVoteNode(0)
	vg1 := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg1.Insert("C", 4, createUintVoteNode(100), c))
	assert.NoError(t, vg1.Insert("E1", 6, createUintVoteNode(100), c))
	assert.NoError(t, vg1.Insert("F2", 7, createUintVoteNode(100), c))

	vn1 := uintVoteNode(0)
	vg2 := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn1, newUintVoteNode)
	assert.NoError(t, vg2.Insert("E1", 6, createUintVoteNode(100), c))
	assert.NoError(t, vg2.Insert("F2", 7, createUintVoteNode(100), c))
	assert.NoError(t, vg2.Insert("C", 4, createUintVoteNode(100), c))

	for _, test := range []struct {
		name string
		VoteGraph[string, uint, *uintVoteNode, int]
	}{
		{
			name:      "vg1",
			VoteGraph: vg1,
		},
		{
			name:      "vg2",
			VoteGraph: vg1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			vg := test.VoteGraph

			var getEntry = func(key string) voteGraphEntry[string, uint, *uintVoteNode, int] {
				entry, _ := vg.entries.Get(key)
				return entry
			}

			assert.Contains(t, vg.heads.Keys(), "E1")
			assert.Contains(t, vg.heads.Keys(), "F2")
			assert.NotContains(t, vg.heads.Keys(), "C")

			assert.Contains(t, vg.entries.Keys(), "C")
			assert.Contains(t, getEntry("C").descendants, "E1")
			assert.Contains(t, getEntry("C").descendants, "F2")
			assert.Equal(t, GenesisHash, *getEntry("C").AncestorNode())
			assert.Equal(t, createUintVoteNode(300), getEntry("C").cumulativeVote)

			assert.Contains(t, vg.entries.Keys(), "E1")
			assert.Equal(t, "C", *getEntry("E1").AncestorNode())
			assert.Equal(t, createUintVoteNode(100), getEntry("E1").cumulativeVote)

			assert.Contains(t, vg.entries.Keys(), "F2")
			assert.Equal(t, "C", *getEntry("F2").AncestorNode())
			assert.Equal(t, createUintVoteNode(100), getEntry("F2").cumulativeVote)
		})
	}
}

func TestVoteGraph_GhostMergeAtNode(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("B", 3, createUintVoteNode(0), c))
	assert.NoError(t, vg.Insert("C", 4, createUintVoteNode(100), c))
	assert.NoError(t, vg.Insert("E1", 6, createUintVoteNode(100), c))
	assert.NoError(t, vg.Insert("F2", 7, createUintVoteNode(100), c))

	assert.Equal(t, &HashNumber[string, uint]{"C", 4}, vg.FindGHOST(nil, func(i *uintVoteNode) bool { return *i >= 250 }))
	assert.Equal(t, &HashNumber[string, uint]{"C", 4},
		vg.FindGHOST(&HashNumber[string, uint]{"C", 4}, func(i *uintVoteNode) bool { return *i >= 250 }))
	assert.Equal(t, &HashNumber[string, uint]{"C", 4},
		vg.FindGHOST(&HashNumber[string, uint]{"B", 3}, func(i *uintVoteNode) bool { return *i >= 250 }))
}

func TestVoteGraph_GhostMergeNoteAtNodeOneSideWeighted(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	c.PushBlocks("F", []string{"G1", "H1", "I1"})
	c.PushBlocks("F", []string{"G2", "H2", "I2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("B", 3, createUintVoteNode(0), c))
	assert.NoError(t, vg.Insert("G1", 8, createUintVoteNode(100), c))
	assert.NoError(t, vg.Insert("H2", 9, createUintVoteNode(150), c))

	assert.Equal(t, &HashNumber[string, uint]{"F", 7}, vg.FindGHOST(nil, func(i *uintVoteNode) bool { return *i >= 250 }))
	assert.Equal(t, &HashNumber[string, uint]{"F", 7},
		vg.FindGHOST(&HashNumber[string, uint]{"F", 7}, func(i *uintVoteNode) bool { return *i >= 250 }))
	assert.Equal(t, &HashNumber[string, uint]{"F", 7},
		vg.FindGHOST(&HashNumber[string, uint]{"C", 4}, func(i *uintVoteNode) bool { return *i >= 250 }))
	assert.Equal(t, &HashNumber[string, uint]{"F", 7},
		vg.FindGHOST(&HashNumber[string, uint]{"B", 3}, func(i *uintVoteNode) bool { return *i >= 250 }))
}

func TestVoteGraph_GhostIntroduceBranch(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	c.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	c.PushBlocks("F", []string{"FA", "FB", "FC"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("FC", 10, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("ED", 10, createUintVoteNode(7), c))

	var getEntry = func(key string) voteGraphEntry[string, uint, *uintVoteNode, int] {
		entry, _ := vg.entries.Get(key)
		return entry
	}

	assert.Equal(t, &HashNumber[string, uint]{"E", 6}, vg.FindGHOST(nil, func(x *uintVoteNode) bool { return *x >= 10 }))
	assert.Equal(t, []string{"FC", "ED"}, getEntry(GenesisHash).descendants)

	// introduce a branch in the middle.
	assert.NoError(t, vg.Insert("E", 6, createUintVoteNode(3), c))

	assert.Equal(t, []string{"E"}, getEntry(GenesisHash).descendants)
	assert.Equal(t, 2, len(getEntry("E").descendants))
	assert.Contains(t, getEntry("E").descendants, "ED")
	assert.Contains(t, getEntry("E").descendants, "FC")

	assert.Equal(t, &HashNumber[string, uint]{"E", 6}, vg.FindGHOST(nil, func(x *uintVoteNode) bool { return *x >= 10 }))
	assert.Equal(t, &HashNumber[string, uint]{"E", 6},
		vg.FindGHOST(&HashNumber[string, uint]{"C", 4}, func(x *uintVoteNode) bool { return *x >= 10 }))
	assert.Equal(t, &HashNumber[string, uint]{"E", 6},
		vg.FindGHOST(&HashNumber[string, uint]{"E", 6}, func(x *uintVoteNode) bool { return *x >= 10 }))
}

func TestVoteGraph_WalkBackFromBlockInEdgeForkBelow(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1", "G1", "H1", "I1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2", "G2", "H2", "I2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("B", 3, createUintVoteNode(10), c))
	assert.NoError(t, vg.Insert("F1", 7, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("G2", 8, createUintVoteNode(5), c))

	for _, block := range []string{"D1", "D2", "E1", "E2", "F1", "F2", "G2"} {
		number := c.Number(block)
		assert.Equal(t, &HashNumber[string, uint]{"C", 4},
			vg.FindAncestor(block, uint(number), func(x *uintVoteNode) bool { return *x > 5 }))
	}
}

func TestVoteGraph_WalkBackFromForkBlockNodeBelow(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C", "D"})
	c.PushBlocks("D", []string{"E1", "F1", "G1", "H1", "I1"})
	c.PushBlocks("D", []string{"E2", "F2", "G2", "H2", "I2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("B", 3, createUintVoteNode(10), c))
	assert.NoError(t, vg.Insert("F1", 7, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("G2", 8, createUintVoteNode(5), c))

	assert.Equal(t, &HashNumber[string, uint]{"D", 5},
		vg.FindAncestor("G2", 8, func(x *uintVoteNode) bool { return *x > 5 }))
	for _, block := range []string{"E1", "E2", "F1", "F2", "G2"} {
		number := c.Number(block)
		assert.Equal(t, &HashNumber[string, uint]{"D", 5},
			vg.FindAncestor(block, uint(number), func(x *uintVoteNode) bool { return *x > 5 }))
	}
}

func TestVoteGraph_WalkBackAtNode(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C"})
	c.PushBlocks("C", []string{"D1", "E1", "F1", "G1", "H1", "I1"})
	c.PushBlocks("C", []string{"D2", "E2", "F2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(1), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("C", 4, createUintVoteNode(10), c))
	assert.NoError(t, vg.Insert("F1", 7, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("F2", 7, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("I1", 10, createUintVoteNode(1), c))

	for _, block := range []string{"C", "D1", "D2", "E1", "E2", "F1", "F2", "I1"} {
		number := c.Number(block)
		assert.Equal(t, &HashNumber[string, uint]{"C", 4},
			vg.FindAncestor(block, uint(number), func(x *uintVoteNode) bool { return *x >= 20 }))
	}
}

func TestVoteGraph_AdjustBase(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A", "B", "C", "D", "E", "F"})
	c.PushBlocks("E", []string{"EA", "EB", "EC", "ED"})
	c.PushBlocks("F", []string{"FA", "FB", "FC"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int]("E", uint(6), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("FC", 10, createUintVoteNode(5), c))
	assert.NoError(t, vg.Insert("ED", 10, createUintVoteNode(7), c))

	assert.Equal(t, HashNumber[string, uint]{"E", 6}, vg.Base())

	vg.AdjustBase([]string{"D", "C", "B", "A"})

	assert.Equal(t, HashNumber[string, uint]{"A", 2}, vg.Base())

	c.PushBlocks("A", []string{"3", "4", "5"})

	vg.AdjustBase([]string{GenesisHash})
	assert.Equal(t, HashNumber[string, uint]{GenesisHash, 1}, vg.Base())

	var getEntry = func(key string) voteGraphEntry[string, uint, *uintVoteNode, int] {
		entry, _ := vg.entries.Get(key)
		return entry
	}

	assert.Equal(t, createUintVoteNode(12), getEntry(GenesisHash).cumulativeVote)

	assert.NoError(t, vg.Insert("5", 5, createUintVoteNode(3), c))

	assert.Equal(t, int(15), int(*getEntry(GenesisHash).cumulativeVote))
}

func TestVoteGraph_FindAncestorIsLargest(t *testing.T) {
	c := newDummyChain()
	c.PushBlocks(GenesisHash, []string{"A"})
	c.PushBlocks(GenesisHash, []string{"B"})
	c.PushBlocks("A", []string{"A1"})
	c.PushBlocks("A", []string{"A2"})
	c.PushBlocks("B", []string{"B1"})
	c.PushBlocks("B", []string{"B2"})

	vn := uintVoteNode(0)
	vg := NewVoteGraph[string, uint, *uintVoteNode, int](GenesisHash, uint(0), &vn, newUintVoteNode)
	assert.NoError(t, vg.Insert("B1", 2, createUintVoteNode(1), c))
	assert.NoError(t, vg.Insert("B2", 2, createUintVoteNode(1), c))
	assert.NoError(t, vg.Insert("A1", 2, createUintVoteNode(1), c))
	assert.NoError(t, vg.Insert("A2", 2, createUintVoteNode(1), c))

	assert.Equal(t, &HashNumber[string, uint]{"A", 1},
		vg.FindAncestor("A", 1, func(x *uintVoteNode) bool { return *x >= 2 }))
}
