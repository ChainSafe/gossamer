package proof

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
	"golang.org/x/exp/slices"
)

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
	children []*triedb.ChildReference
	// outputIndex is the index into the proof vector that the encoding of this entry should be placed at.
	outputIndex *int
}

func NewStackEntry(
	prefix []byte,
	nodeData []byte,
	nodeHash *[]byte,
	outputIndex *int) (*stackEntry, error) {
	node, err := codec.Decode(bytes.NewReader(nodeData))
	if err != nil {
		return nil, err
	}

	var childrenLen int
	switch node.(type) {
	case codec.Empty, codec.Leaf:
		childrenLen = 0
	case codec.Branch:
		childrenLen = codec.ChildrenCapacity
	}

	return &stackEntry{
		prefix:      prefix,
		node:        node,
		encodedNode: nodeData,
		nodeHash:    nodeHash,
		omitValue:   false,
		childIndex:  0,
		children:    make([]*triedb.ChildReference, childrenLen),
		outputIndex: outputIndex,
	}, nil
}

func (e *stackEntry) setChild(encodedChild []byte) {
	var childRef triedb.ChildReference
	switch n := e.node.(type) {
	case codec.Empty, codec.Leaf:
		panic("Empty and leaf nodes have no children, we cannot descend into")
	case codec.Branch:
		if e.childIndex >= codec.ChildrenCapacity {
			panic("child index out of bounds")
		}
		if n.Children[e.childIndex] != nil {
			childRef = e.replaceChildRef(encodedChild, n.Children[e.childIndex])
		}
	}
	e.children[e.childIndex] = &childRef
	e.childIndex++
}

func (e *stackEntry) replaceChildRef(encodedChild []byte, child codec.MerkleValue) triedb.ChildReference {
	switch child.(type) {
	case codec.HashedNode:
		return triedb.InlineChildReference{}
	case codec.InlineNode:
		return triedb.InlineChildReference{EncodedNode: encodedChild}
	default:
		panic("unreachable")
	}
}

func GenerateProof(db db.RWDatabase, trieVersion trie.TrieLayout, rootHash common.Hash, keys []string) (
	proof [][]byte, err error) {
	// Sort and deduplicate keys
	keys = sortAndDeduplicateKeys(keys)

	// The stack of nodes through a path in the trie.
	// Each entry is a child node of the preceding entry.
	stack := deque.New[*stackEntry]()

	// final proof nodes
	var proofNodes [][]byte

	for i := 0; i <= len(keys); i = i + 1 {
		var key = keys[i]
		unwindStack(stack, proofNodes, &key)

		recorder := triedb.NewRecorder()
		trie := triedb.NewTrieDB(rootHash, db, nil, recorder)
		trie.SetVersion(trieVersion)

		trie.Get([]byte(key))

		recordedNodes := triedb.NewRecordedNodesIterator(recorder.Drain())

		// Skip over recorded nodes already on the stack.
		recordedNodesIdx := 0
		stackIdx := 0
		for stackIdx < stack.Len() && recordedNodesIdx < len(recordedNodes) {
			nextEntry := stack.At(stackIdx)
			nextRecord := recordedNodes[recordedNodesIdx]

			if nextEntry.nodeHash != nil && bytes.Equal(*nextEntry.nodeHash, nextRecord.Hash) {
				stackIdx++
				recordedNodesIdx++
			} else {
				break
			}

			stack.PopBack()
		}
		recordedNodes = recordedNodes[recordedNodesIdx:]

		for {
			var nextStep step
			entry := stack.Back()
			if entry == nil {
				nextStep = stepDescend{childPrefixLen: 0, child: nodeHandleHash{hash: rootHash.ToBytes()}}
			} else {
				var err error
				nextStep, err = matchKeyToNode(
					entry.node,
					&entry.omitValue,
					entry.childIndex,
					entry.children,
					[]byte(key),
					len(entry.prefix),
					recordedNodes,
				)

				if err != nil {
					return nil, err
				}
			}

			switch s := nextStep.(type) {
			case stepDescend:
				childPrefix := []byte(key[s.childPrefixLen:])
				var childEntry *stackEntry
				switch child := s.child.(type) {
				case nodeHandleHash:
					// TODO: use recordedNodes iterator
					childRecord := recordedNodes[0]
					recordedNodes = recordedNodes[1:]
					outputIndex := len(proofNodes)

					// Insert a placeholder into output which will be replaced when this
					// new entry is popped from the stack.
					proofNodes = append(proofNodes, []byte{})
					childEntry, err = NewStackEntry(
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
						return nil, errors.New("Invalid hash length")
					}
					childEntry, err = NewStackEntry(
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
				break
			}
		}
	}

	unwindStack(stack, proofNodes, nil)
	return proofNodes, nil
}

// / Unwind the stack until the given key is prefixed by the entry at the top of the stack. If the
// / key is NIL, unwind the stack completely. As entries are popped from the stack, they are
// / encoded into proof nodes and added to the finalized proof.
func unwindStack(
	stack *deque.Deque[*stackEntry],
	proofNodes [][]byte,
	maybeKey *string,
) {
	for entry := stack.PopBack(); entry != nil; entry = stack.PopBack() {
		if maybeKey != nil {
			key := *maybeKey
			if bytes.HasPrefix([]byte(key), entry.prefix) {
				stack.PushBack(entry)
				break
			}
		}

		parentEntry := stack.Back()
		if parentEntry != nil {
			parentEntry.setChild(entry.encodedNode)
		}

		index := entry.outputIndex
		if index != nil {
			proofNodes[*index] = entry.encodedNode
		}
	}
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
	childIndex int,
	children []*triedb.ChildReference,
	key []byte,
	prefixlen int,
	recordedNodes []triedb.Record,
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
			children,
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
	childIndex int,
	omitValue *bool,
	children []*triedb.ChildReference,
	key []byte,
	prefixlen int,
	nodePartialKey []byte,
	recordedNodes []triedb.Record,
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

	if newIndex > childIndex {
		panic("newIndex out of bounds")
	}
	for childIndex < newIndex {
		// TODO: convert branch child into child reference
		//children[childIndex] = childHandles[childIndex]
		childIndex++
	}

	if childHandles[childIndex] != nil {
		return stepDescend{
			childPrefixLen: len(nodePartialKey) + prefixlen + 1,
			// TODO: encode child handle
			//child:          encode(childHandles[childIndex]),
		}, nil
	}
	return stepFoundValue{nil}, nil
}

// TODO: use an iterator to consume recordedNodes
func resolveValue(recordedNodes []triedb.Record) (step, error) {
	if len(recordedNodes) > 0 {
		value := recordedNodes[0].Data
		recordedNodes = recordedNodes[1:] // TODO: this wont change original recordedNodes
		return stepFoundHashedValue{value}, nil
	} else {
		return nil, triedb.ErrIncompleteDB
	}
}
