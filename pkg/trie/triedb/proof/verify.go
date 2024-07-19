// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
	"golang.org/x/exp/slices"
)

var (
	ErrExtraneusNode   = errors.New("extraneous node in proof")
	ErrIncompleteProof = errors.New("incomplete proof")
)

type verifyProofStep interface {
	isProofStep()
}

type (
	verifyProofStepDescend struct {
		childPrefix []byte
	}
	verifyProofStepUnwindStackStep struct{}
)

func (verifyProofStepDescend) isProofStep()         {}
func (verifyProofStepUnwindStackStep) isProofStep() {}

type verifyProofStackEntry struct {
	trieVersion   trie.TrieLayout
	prefix        []byte
	node          codec.EncodedNode
	value         codec.EncodedValue
	isInline      bool
	childIndex    int
	children      [codec.ChildrenCapacity]triedb.ChildReference
	nextValueHash common.Hash
}

func newVerifyProofStackEntry(
	trieVersion trie.TrieLayout,
	nodeData []byte,
	prefix []byte,
	isInline bool,
) (*verifyProofStackEntry, error) {
	node, err := codec.Decode(bytes.NewReader(nodeData))
	if err != nil {
		return nil, err
	}

	return &verifyProofStackEntry{
		trieVersion:   trieVersion,
		node:          node,
		value:         node.GetValue(),
		prefix:        prefix,
		isInline:      isInline,
		childIndex:    0,
		children:      [codec.ChildrenCapacity]triedb.ChildReference{},
		nextValueHash: common.EmptyHash,
	}, nil
}

func (e *verifyProofStackEntry) getValue() codec.EncodedValue {
	if e.nextValueHash != common.EmptyHash {
		return codec.HashedValue(e.nextValueHash.ToBytes())
	}
	return e.value
}

func (e *verifyProofStackEntry) encodeNode() ([]byte, error) {
	switch n := e.node.(type) {
	case codec.Empty:
		return []byte{triedb.EmptyTrieBytes}, nil
	case codec.Leaf:
		encodingBuffer := bytes.NewBuffer(nil)
		err := triedb.NewEncodedLeaf(e.node.GetPartialKey(), e.getValue(), encodingBuffer)
		if err != nil {
			return nil, err
		}

		return encodingBuffer.Bytes(), nil
	case codec.Branch:
		// Complete children
		childIndex := e.childIndex
		children := e.children
		for childIndex < codec.ChildrenCapacity {
			child := n.Children[childIndex]
			if child != nil {
				switch c := child.(type) {
				case codec.InlineNode:
					children[childIndex] = triedb.InlineChildReference(c)
				case codec.HashedNode:
					children[childIndex] = triedb.HashChildReference(common.Hash(c))
				}
			}
			childIndex++
		}

		encodingBuffer := bytes.NewBuffer(nil)
		err := triedb.NewEncodedBranch(e.node.GetPartialKey(), children, e.getValue(), encodingBuffer)
		if err != nil {
			return nil, err
		}
		return encodingBuffer.Bytes(), nil
	default:
		panic("unreachable")
	}
}

func (e *verifyProofStackEntry) advanceItem(itemsIter *Iterator[proofItem]) (verifyProofStep, error) {
	for {
		item := itemsIter.Peek()
		if item == nil {
			return verifyProofStepUnwindStackStep{}, nil
		}

		keyNibbles := nibbles.KeyLEToNibbles(item.key)
		value := item.value
		if bytes.HasPrefix(keyNibbles, e.prefix) {
			valueMatch := matchKeyToNode(keyNibbles, len(e.prefix), e.node)
			switch m := valueMatch.(type) {
			case matchesLeaf:
				if value != nil {
					e.setValue(value)
				} else {
					return nil, fmt.Errorf("value mismatch for key: %x", item.key)
				}
			case matchesBranch:
				if value != nil {
					e.setValue(value)
				} else {
					e.value = nil
				}
			case notFound:
				if value != nil {
					return nil, fmt.Errorf("value mismatch for key: %x", item.key)
				}
			case notOmitted:
				return nil, fmt.Errorf("extraneous value for key %x", item.key)
			case isChild:
				return verifyProofStepDescend(m), nil
			}

			itemsIter.Next()
			continue
		}
		return verifyProofStepUnwindStackStep{}, nil
	}
}

func (e *verifyProofStackEntry) advanceChildIndex(
	childPrefix []byte,
	proofIter *Iterator[[]byte],
) (*verifyProofStackEntry, error) {
	switch n := e.node.(type) {
	case codec.Branch:
		if len(childPrefix) <= 0 {
			panic("child prefix should be greater than 0")
		}
		childIndex := childPrefix[len(childPrefix)-1]
		for e.childIndex < int(childIndex) {
			child := n.Children[e.childIndex]
			if child != nil {
				switch c := child.(type) {
				case codec.InlineNode:
					e.children[e.childIndex] = triedb.InlineChildReference(c)
				case codec.HashedNode:
					e.children[e.childIndex] = triedb.HashChildReference(common.Hash(c))
				}
			}
			e.childIndex++
		}
		child := n.Children[childIndex]
		return e.makeChildEntry(proofIter, child, childPrefix)
	default:
		panic("cannot have children")
	}
}

func (e *verifyProofStackEntry) makeChildEntry(
	proofIter *Iterator[[]byte],
	child codec.MerkleValue,
	childPrefix []byte,
) (*verifyProofStackEntry, error) {
	switch c := child.(type) {
	case codec.InlineNode:
		if len(c) == 0 {
			nodeData := proofIter.Next()
			if nodeData == nil {
				return nil, ErrIncompleteProof
			}
			return newVerifyProofStackEntry(e.trieVersion, *nodeData, childPrefix, false)
		}
		return newVerifyProofStackEntry(e.trieVersion, c, childPrefix, true)
	case codec.HashedNode:
		if len(c) != common.HashLength {
			return nil, fmt.Errorf("invalid hash length: %x", c)
		}
		return nil, fmt.Errorf("extraneous hash reference: %x", c)
	default:
		panic("unreachable")
	}
}

func (e *verifyProofStackEntry) setValue(value []byte) {
	if len(value) <= e.trieVersion.MaxInlineValue() {
		e.value = codec.InlineValue(value)
	} else {
		hashedValue := common.MustBlake2bHash(value)
		e.nextValueHash = hashedValue
		e.value = nil
	}
}

type valueMatch interface {
	isValueMatch()
}

type (
	// The key matches a leaf node, so the value at the key must be present.
	matchesLeaf struct{}
	// The key matches a branch node, so the value at the key may or may not be present.
	matchesBranch struct{}
	// The key was not found to correspond to value in the trie, so must not be present.
	notFound struct{}
	// The key matches a location in trie, but the value was not omitted.
	notOmitted struct{}
	// The key may match below a child of this node. Parameter is the prefix of the child node.
	isChild struct {
		childPrefix []byte
	}
)

func (matchesLeaf) isValueMatch()   {}
func (matchesBranch) isValueMatch() {}
func (notFound) isValueMatch()      {}
func (notOmitted) isValueMatch()    {}
func (isChild) isValueMatch()       {}

func matchKeyToNode(keyNibbles []byte, prefixLen int, node codec.EncodedNode) valueMatch {
	switch n := node.(type) {
	case codec.Empty:
		return notFound{}
	case codec.Leaf:
		if bytes.Contains(keyNibbles, n.PartialKey) && len(keyNibbles) == prefixLen+len(n.PartialKey) {
			switch v := n.Value.(type) {
			case codec.HashedValue:
				return notOmitted{}
			case codec.InlineValue:
				if len(v) == 0 {
					return matchesLeaf{}
				}
				return notOmitted{}
			}
		}
		return notFound{}
	case codec.Branch:
		if bytes.Contains(keyNibbles, n.PartialKey) {
			return matchKeyToBranchNode(keyNibbles, prefixLen+len(n.PartialKey), n.Children, n.Value)
		} else {
			return notFound{}
		}
	default:
		panic("unreachable")
	}
}

func matchKeyToBranchNode(
	key []byte,
	prefixPlusPartialLen int,
	children [codec.ChildrenCapacity]codec.MerkleValue,
	value codec.EncodedValue,
) valueMatch {
	if len(key) == prefixPlusPartialLen {
		if value == nil {
			return matchesBranch{}
		}
		return notOmitted{}
	}
	index := key[prefixPlusPartialLen]
	if children[index] != nil {
		return isChild{childPrefix: key[:prefixPlusPartialLen+1]}
	}

	return notFound{}
}

type proofItem struct {
	key   []byte
	value []byte
}

func (proof MerkleProof) Verify(
	trieVersion trie.TrieLayout,
	root common.Hash,
	items []proofItem,
) error {
	// sort items
	slices.SortFunc(items, func(a, b proofItem) int {
		return bytes.Compare(a.key, b.key)
	})

	if len(items) == 0 {
		if len(proof) == 0 {
			return nil
		}
		return ErrExtraneusNode
	}

	// Check for duplicates
	for i := 0; i < len(items)-1; i++ {
		if bytes.Equal(items[i].key, items[i+1].key) {
			return fmt.Errorf("duplicate key in items: %x", items[i].key)
		}
	}

	// Iterate simultaneously in order through proof nodes and key-value pairs to verify.
	proofIter := NewIterator(proof)
	itemsIter := NewIterator(items)

	// A stack of child references to fill in omitted branch children for later trie nodes in the
	// proof.
	stack := deque.New[verifyProofStackEntry]()

	rootNode := proofIter.Next()
	if rootNode == nil {
		return ErrIncompleteProof
	}

	lastEntry, err := newVerifyProofStackEntry(trieVersion, *rootNode, []byte{}, false)
	if err != nil {
		return err
	}

loop:
	for {
		step, err := lastEntry.advanceItem(itemsIter)
		if err != nil {
			return err
		}

		switch s := step.(type) {
		case verifyProofStepDescend:
			nextEntry, err := lastEntry.advanceChildIndex(s.childPrefix, proofIter)
			if err != nil {
				return err
			}
			stack.PushBack(*lastEntry)
			lastEntry = nextEntry
		case verifyProofStepUnwindStackStep:
			isInline := lastEntry.isInline
			nodeData, err := lastEntry.encodeNode()
			if err != nil {
				return err
			}

			var childRef triedb.ChildReference
			if isInline {
				if len(nodeData) > common.HashLength {
					return fmt.Errorf("invalid child reference: %x", nodeData)
				}
				childRef = triedb.InlineChildReference(nodeData)
			} else {
				hash := common.MustBlake2bHash(nodeData)
				childRef = triedb.HashChildReference(hash)
			}

			if stack.Len() > 0 {
				entry := stack.PopBack()
				lastEntry = &entry
				lastEntry.children[lastEntry.childIndex] = childRef
				lastEntry.childIndex++
			} else {
				nextProof := proofIter.Next()
				if nextProof != nil {
					return ErrExtraneusNode
				}
				var computedRoot common.Hash
				switch c := childRef.(type) {
				case triedb.HashChildReference:
					computedRoot = common.Hash(c)
				case triedb.InlineChildReference:
					panic("unreachable")
				}

				if !bytes.Equal(computedRoot.ToBytes(), root.ToBytes()) {
					return fmt.Errorf("root hash mismatch: %x != %x", computedRoot, root)
				}
				break loop
			}
		}
	}

	return nil
}
