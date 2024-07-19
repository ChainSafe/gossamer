// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
	"golang.org/x/exp/slices"
)

type MerkleProof [][]byte

type nodeHandle interface {
	isNodeHandle()
}

type (
	nodeHandleHash struct {
		hash []byte
	}

	nodeHandleInline struct {
		data []byte
	}
)

func (nodeHandleHash) isNodeHandle()   {}
func (nodeHandleInline) isNodeHandle() {}

type step interface {
	isProofStep()
}

type (
	stepDescend struct {
		childPrefixLen int
		child          nodeHandle
	}
	stepFoundValue struct {
		value *[]byte
	}
	stepFoundHashedValue struct {
		hash []byte
	}
)

func (stepDescend) isProofStep()          {}
func (stepFoundValue) isProofStep()       {}
func (stepFoundHashedValue) isProofStep() {}

type stackEntry struct {
	// prefix is the nibble path to the node in the trie
	prefix []byte
	// node is the stacked node
	node codec.EncodedNode
	// encodedNode is the encoded node data
	encodedNode []byte
	// nodeHash of the node or nil if the node is inlined
	nodeHash *[]byte
	// omitValue is a flag to know if the value should be omitted in the generated proof
	omitValue bool
	// childIndex is used for branch nodes
	childIndex int
	// children contains the child references to use in constructing the proof nodes.
	children triedb.ChildReferences
	// outputIndex is the index into the proof vector that the encoding of this entry should be placed at.
	outputIndex *int
}

func newStackEntry(
	prefix []byte,
	nodeData []byte,
	nodeHash *[]byte,
	outputIndex *int) (*stackEntry, error) {
	node, err := codec.Decode(bytes.NewReader(nodeData))
	if err != nil {
		return nil, err
	}

	return &stackEntry{
		prefix:      prefix,
		node:        node,
		encodedNode: nodeData,
		nodeHash:    nodeHash,
		omitValue:   false,
		childIndex:  0,
		children:    [codec.ChildrenCapacity]triedb.ChildReference{},
		outputIndex: outputIndex,
	}, nil
}

func (e *stackEntry) encodeNode() ([]byte, error) {
	switch n := e.node.(type) {
	case codec.Empty:
		return e.encodedNode, nil
	case codec.Leaf:
		if !e.omitValue {
			return e.encodedNode, nil
		}

		encodingBuffer := bytes.NewBuffer(nil)
		err := triedb.NewEncodedLeaf(e.node.GetPartialKey(), codec.NewInlineValue([]byte{}), encodingBuffer)
		if err != nil {
			return nil, err
		}

		return encodingBuffer.Bytes(), nil
	case codec.Branch:
		var value codec.EncodedValue
		if !e.omitValue {
			value = n.Value
		}
		e.completBranchChildren(n.Children, e.childIndex)
		encodingBuffer := bytes.NewBuffer(nil)
		err := triedb.NewEncodedBranch(e.node.GetPartialKey(), e.children, value, encodingBuffer)
		if err != nil {
			return nil, err
		}
		return encodingBuffer.Bytes(), nil
	default:
		panic("unreachable")
	}
}

func (e *stackEntry) setChild(encodedChild []byte) {
	var childRef triedb.ChildReference
	switch n := e.node.(type) {
	case codec.Branch:
		if e.childIndex >= codec.ChildrenCapacity {
			panic("child index out of bounds")
		}
		if n.Children[e.childIndex] != nil {
			childRef = e.replaceChildRef(encodedChild, n.Children[e.childIndex])
		}
	default:
		panic("Empty and leaf nodes have no children, we cannot descend into")
	}
	e.children[e.childIndex] = childRef
	e.childIndex++
}

func (e *stackEntry) completBranchChildren(
	childHandles [codec.ChildrenCapacity]codec.MerkleValue,
	childIndex int,
) {
	for i := childIndex; i < codec.ChildrenCapacity; i++ {
		switch n := childHandles[i].(type) {
		case codec.InlineNode:
			e.children[i] = triedb.NewInlineChildReference(n.Data)
		case codec.HashedNode:
			e.children[i] = triedb.NewHashChildReference(common.Hash(n.Data))
		}
	}
}

func (e *stackEntry) replaceChildRef(encodedChild []byte, child codec.MerkleValue) triedb.ChildReference {
	switch child.(type) {
	case codec.HashedNode:
		return triedb.NewInlineChildReference(nil)
	case codec.InlineNode:
		return triedb.NewInlineChildReference(encodedChild)
	default:
		panic("unreachable")
	}
}

func New(db db.RWDatabase, trieVersion trie.TrieLayout, rootHash common.Hash, keys []string) (
	proof MerkleProof, err error) {
	// Sort and deduplicate keys
	keys = sortAndDeduplicateKeys(keys)

	// The stack of nodes through a path in the trie.
	// Each entry is a child node of the preceding entry.
	stack := deque.New[*stackEntry]()

	// final proof nodes
	var proofNodes MerkleProof

	// Iterate over the keys and build the proof nodes
	for i := 0; i < len(keys); i = i + 1 {
		var key = []byte(keys[i])
		var keyNibbles = nibbles.KeyLEToNibbles(key)

		err := unwindStack(stack, proofNodes, &keyNibbles)
		if err != nil {
			return nil, err
		}

		// Traverse the trie recording the visited nodes
		recorder := triedb.NewRecorder()
		trie := triedb.NewTrieDB(rootHash, db, triedb.WithRecorder(recorder))
		trie.SetVersion(trieVersion)
		trie.Get(key)

		recordedNodes := triedb.NewRecordedNodesIterator(recorder.Drain())

		// Skip over recorded nodes already on the stack.
		for i := 0; i < stack.Len(); i++ {
			nextEntry := stack.At(i)
			nextRecord := recordedNodes.Peek()

			if nextRecord == nil || !bytes.Equal(*nextEntry.nodeHash, nextRecord.Hash) {
				break
			}

			recordedNodes.Next()
		}

		// Descend in trie collecting nodes until find the value or the end of the path
	loop:
		for {
			var nextStep step
			var entry *stackEntry
			if stack.Len() > 0 {
				entry = stack.Back()
			}
			if entry == nil {
				nextStep = stepDescend{childPrefixLen: 0, child: nodeHandleHash{hash: rootHash.ToBytes()}}
			} else {
				var err error
				nextStep, err = matchKeyToNode(
					entry.node,
					&entry.omitValue,
					&entry.childIndex,
					keyNibbles,
					len(entry.prefix),
					recordedNodes,
				)

				if err != nil {
					return nil, err
				}
			}

			switch s := nextStep.(type) {
			case stepDescend:
				childPrefix := keyNibbles[:s.childPrefixLen]
				var childEntry *stackEntry
				switch child := s.child.(type) {
				case nodeHandleHash:
					childRecord := recordedNodes.Next()

					if !bytes.Equal(childRecord.Hash, child.hash) {
						panic("hash mismatch")
					}

					outputIndex := len(proofNodes)

					// Insert a placeholder into output which will be replaced when this
					// new entry is popped from the stack.
					proofNodes = append(proofNodes, []byte{})
					childEntry, err = newStackEntry(
						childPrefix,
						childRecord.Data,
						&childRecord.Hash,
						&outputIndex,
					)

					if err != nil {
						return nil, err
					}
				case nodeHandleInline:
					if len(child.data) > common.HashLength {
						return nil, errors.New("invalid hash length")
					}
					childEntry, err = newStackEntry(
						childPrefix,
						child.data,
						nil,
						nil,
					)
					if err != nil {
						return nil, err
					}
				}
				stack.PushBack(childEntry)
			default:
				recordedNodes.Next()
				break loop
			}
		}
	}

	err = unwindStack(stack, proofNodes, nil)
	if err != nil {
		return nil, err
	}
	return proofNodes, nil
}

// / Unwind the stack until the given key is prefixed by the entry at the top of the stack. If the
// / key is nil, unwind the stack completely. As entries are popped from the stack, they are
// / encoded into proof nodes and added to the finalized proof.
func unwindStack(
	stack *deque.Deque[*stackEntry],
	proofNodes [][]byte,
	maybeKey *[]byte,
) error {
	for stack.Len() > 0 {
		entry := stack.PopBack()
		if maybeKey != nil && bytes.HasPrefix(*maybeKey, entry.prefix) {
			stack.PushBack(entry)
			break
		}

		if stack.Len() > 0 {
			parentEntry := stack.Back()
			if parentEntry != nil {
				encoded, err := entry.encodeNode()
				if err != nil {
					return err
				}
				parentEntry.setChild(encoded)
			}
		}

		index := entry.outputIndex
		if index != nil {
			encoded, err := entry.encodeNode()
			if err != nil {
				return err
			}
			proofNodes[*index] = encoded
		}
	}
	return nil
}

func sortAndDeduplicateKeys(keys []string) []string {
	slices.Sort(keys)

	if len(keys) == 0 {
		return keys
	}

	result := []string{keys[0]}
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[i-1] {
			result = append(result, keys[i])
		}
	}

	return result
}

func matchKeyToNode(
	node codec.EncodedNode,
	omitValue *bool,
	childIndex *int,
	key []byte,
	prefixlen int,
	recordedNodes *triedb.RecordedNodesIterator,
) (step, error) {
	switch n := node.(type) {
	case codec.Empty:
		return stepFoundValue{nil}, nil
	case codec.Leaf:
		if bytes.Contains(key, n.PartialKey) && len(key) == prefixlen+len(n.PartialKey) {
			switch v := n.Value.(type) {
			case codec.InlineValue:
				*omitValue = true
				return stepFoundValue{&v.Data}, nil
			case codec.HashedValue:
				*omitValue = true
				return resolveValue(recordedNodes)
			}
		}
		return stepFoundValue{nil}, nil
	case codec.Branch:
		return matchKeyToBranchNode(
			n.Value,
			n.Children,
			childIndex,
			omitValue,
			key,
			prefixlen,
			n.PartialKey,
			recordedNodes,
		)
	default:
		panic("unreachable")
	}
}

func matchKeyToBranchNode(
	value codec.EncodedValue,
	childHandles [codec.ChildrenCapacity]codec.MerkleValue,
	childIndex *int,
	omitValue *bool,
	key []byte,
	prefixlen int,
	nodePartialKey []byte,
	recordedNodes *triedb.RecordedNodesIterator,
) (step, error) {
	if !bytes.Contains(key, nodePartialKey) {
		return stepFoundValue{nil}, nil
	}

	if len(key) == prefixlen+len(nodePartialKey) {
		if value == nil {
			return stepFoundValue{nil}, nil
		}

		switch v := value.(type) {
		case codec.InlineValue:
			*omitValue = true
			return stepFoundValue{&v.Data}, nil
		case codec.HashedValue:
			*omitValue = true
			return resolveValue(recordedNodes)
		}
	}

	newIndex := int(key[prefixlen+len(nodePartialKey)])

	if newIndex < *childIndex {
		panic("newIndex out of bounds")
	}

	*childIndex = newIndex

	if childHandles[newIndex] != nil {
		var child nodeHandle
		switch c := childHandles[newIndex].(type) {
		case codec.HashedNode:
			child = nodeHandleHash{hash: c.Data}
		case codec.InlineNode:
			child = nodeHandleInline{data: c.Data}
		}

		return stepDescend{
			childPrefixLen: len(nodePartialKey) + prefixlen + 1,
			child:          child,
		}, nil
	}
	return stepFoundValue{nil}, nil
}

func resolveValue(recordedNodes *triedb.RecordedNodesIterator) (step, error) {
	value := recordedNodes.Next()
	if value != nil {
		return stepFoundHashedValue{value.Data}, nil
	} else {
		return nil, triedb.ErrIncompleteDB
	}
}
