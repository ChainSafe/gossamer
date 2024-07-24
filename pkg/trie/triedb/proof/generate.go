// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
	"golang.org/x/exp/slices"
)

type nodeHandle interface {
	isNodeHandle()
}

type (
	nodeHandleHash   common.Hash
	nodeHandleInline []byte
)

func (nodeHandleHash) isNodeHandle()   {}
func (nodeHandleInline) isNodeHandle() {}

type genProofStep interface {
	isProofStep()
}

type (
	genProofStepDescend struct {
		childPrefixLen int
		child          nodeHandle
	}
	genProofStepFoundValue struct {
		value *[]byte
	}
	genProofStepFoundHashedValue struct {
		hash []byte
	}
)

func (genProofStepDescend) isProofStep()          {}
func (genProofStepFoundValue) isProofStep()       {}
func (genProofStepFoundHashedValue) isProofStep() {}

type genProofStackEntry struct {
	// prefix is the nibble path to the node in the trie
	prefix []byte
	// node is the stacked node
	node codec.EncodedNode
	// encodedNode is the encoded node data
	encodedNode []byte
	// nodeHash of the node or nil if the node is inlined
	nodeHash *common.Hash
	// omitValue is a flag to know if the value should be omitted in the generated proof
	omitValue bool
	// childIndex is used for branch nodes
	childIndex int
	// children contains the child references to use in constructing the proof nodes.
	children triedb.ChildReferences
	// outputIndex is the index into the proof vector that the encoding of this entry should be placed at.
	outputIndex *int
}

func newGenProofStackEntry(
	prefix []byte,
	nodeData []byte,
	nodeHash *common.Hash,
	outputIndex *int) (*genProofStackEntry, error) {
	node, err := codec.Decode(bytes.NewReader(nodeData))
	if err != nil {
		return nil, err
	}

	return &genProofStackEntry{
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

func (e *genProofStackEntry) encodeNode() ([]byte, error) {
	switch n := e.node.(type) {
	case codec.Empty:
		return e.encodedNode, nil
	case codec.Leaf:
		if !e.omitValue {
			return e.encodedNode, nil
		}

		encodingBuffer := bytes.NewBuffer(nil)
		err := triedb.NewEncodedLeaf(e.node.GetPartialKey(), codec.InlineValue{}, encodingBuffer)
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

func (e *genProofStackEntry) setChild(encodedChild []byte) {
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

func (e *genProofStackEntry) completBranchChildren(
	childHandles [codec.ChildrenCapacity]codec.MerkleValue,
	childIndex int,
) {
	for i := childIndex; i < codec.ChildrenCapacity; i++ {
		switch n := childHandles[i].(type) {
		case codec.InlineNode:
			e.children[i] = triedb.InlineChildReference(n)
		case codec.HashedNode:
			e.children[i] = triedb.HashChildReference(common.Hash(n))
		}
	}
}

func (e *genProofStackEntry) replaceChildRef(encodedChild []byte, child codec.MerkleValue) triedb.ChildReference {
	switch child.(type) {
	case codec.HashedNode:
		return triedb.InlineChildReference(nil)
	case codec.InlineNode:
		return triedb.InlineChildReference(encodedChild)
	default:
		panic("unreachable")
	}
}

// / Unwind the stack until the given key is prefixed by the entry at the top of the stack. If the
// / key is nil, unwind the stack completely. As entries are popped from the stack, they are
// / encoded into proof nodes and added to the finalized proof.
func unwindStack(
	stack *deque.Deque[*genProofStackEntry],
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
	deduplicatedkeys := slices.Compact(keys)
	return deduplicatedkeys
}

func genProofMatchKeyToNode(
	node codec.EncodedNode,
	omitValue *bool,
	childIndex *int,
	key []byte,
	prefixlen int,
	recordedNodes *Iterator[triedb.Record],
) (genProofStep, error) {
	switch n := node.(type) {
	case codec.Empty:
		return genProofStepFoundValue{nil}, nil
	case codec.Leaf:
		if bytes.Contains(key, n.PartialKey) && len(key) == prefixlen+len(n.PartialKey) {
			switch v := n.Value.(type) {
			case codec.InlineValue:
				*omitValue = true
				value := []byte(v)
				return genProofStepFoundValue{&value}, nil
			case codec.HashedValue:
				*omitValue = true
				return resolveValue(recordedNodes)
			}
		}
		return genProofStepFoundValue{nil}, nil
	case codec.Branch:
		return genProofMatchKeyToBranchNode(
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

func genProofMatchKeyToBranchNode(
	value codec.EncodedValue,
	childHandles [codec.ChildrenCapacity]codec.MerkleValue,
	childIndex *int,
	omitValue *bool,
	key []byte,
	prefixlen int,
	nodePartialKey []byte,
	recordedNodes *Iterator[triedb.Record],
) (genProofStep, error) {
	if !bytes.Contains(key, nodePartialKey) {
		return genProofStepFoundValue{nil}, nil
	}

	if len(key) == prefixlen+len(nodePartialKey) {
		if value == nil {
			return genProofStepFoundValue{nil}, nil
		}

		switch v := value.(type) {
		case codec.InlineValue:
			*omitValue = true
			value := []byte(v)
			return genProofStepFoundValue{&value}, nil
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
			child = nodeHandleHash(c)
		case codec.InlineNode:
			child = nodeHandleInline(c)
		}

		return genProofStepDescend{
			childPrefixLen: len(nodePartialKey) + prefixlen + 1,
			child:          child,
		}, nil
	}
	return genProofStepFoundValue{nil}, nil
}

func resolveValue(recordedNodes *Iterator[triedb.Record]) (genProofStep, error) {
	value := recordedNodes.Next()
	if value != nil {
		return genProofStepFoundHashedValue{value.Data}, nil
	} else {
		return nil, triedb.ErrIncompleteDB
	}
}
