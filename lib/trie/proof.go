// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/chaindb"
)

var (
	// ErrEmptyTrieRoot occurs when trying to craft a prove with an empty trie root
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")

	ErrEmptyNibbles = errors.New("empty nibbles provided from key")
)

// StackEntry is a entry on the nodes that is prooved already
// it stores the necessary infos to keep going and prooving the children nodes
type StackEntry struct {
	keyOffset   int
	key         []byte
	nodeHash    []byte
	node        node
	nodeRawData []byte
	outputIndex int
}

//
func NewStackEntry(n node, hash, rd, prefix []byte, outputIndex, keyOffset int) (*StackEntry, error) {
	_, _, err := n.encodeAndHash()
	if err != nil {
		return nil, err
	}

	return &StackEntry{
		keyOffset:   keyOffset,
		nodeHash:    hash,
		key:         prefix,
		node:        n,
		outputIndex: outputIndex,
		nodeRawData: rd,
	}, nil
}

type Stack []*StackEntry

// Push adds a new item to the top of the stack
func (s *Stack) Push(e *StackEntry) {
	(*s) = append((*s), e)
}

// Pop removes and returns the top of the stack if there is some element there otherwise return nil
func (s *Stack) Pop() *StackEntry {
	if len(*s) < 1 {
		return nil
	}

	// gets the top of the stack
	entry := (*s)[len(*s)-1]
	// removes the top of the stack
	*s = (*s)[:len(*s)-1]
	return entry
}

// Last returns the top of the stack without remove from the stack
func (s *Stack) Last() *StackEntry {
	if len(*s) < 1 {
		return nil
	}
	return (*s)[len(*s)-1]
}

type StackIterator struct {
	index int
	set   []*StackEntry
}

func (i *StackIterator) Next() *StackEntry {
	if i.HasNext() {
		t := i.set[i.index]
		i.index++
		return t
	}

	return nil
}

func (i *StackIterator) Peek() *StackEntry {
	if i.HasNext() {
		return i.set[i.index]
	}
	return nil
}

func (i *StackIterator) HasNext() bool {
	return i.index < len(i.set)
}

func (s *Stack) iter() *StackIterator {
	iter := &StackIterator{index: 0}
	iter.set = make([]*StackEntry, len(*s))
	copy(iter.set, *s)

	return iter
}

//
type Step struct {
	Found     bool
	Value     []byte
	NextNode  []byte
	KeyOffset int
}

// GenerateProof receive the keys to proof, the trie root and a reference to database
// will
func GenerateProof(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
	stack := make(Stack, 0)
	proofs := make([][]byte, 0)

	for _, k := range keys {
		nk := keyToNibbles(k)

		unwindStack(&stack, proofs, nk)

		lookup := NewLookup(root, db)
		recorder := new(Recorder)
		expectedValue, err := lookup.Find(nk, recorder)
		if err != nil {
			return nil, err
		}

		// Skip over recorded nodes already on the stack
		stackIter := stack.iter()
		for stackIter.HasNext() {
			nxtRec, nxtEntry := recorder.Peek(), stackIter.Peek()
			if !bytes.Equal(nxtRec.Hash, nxtEntry.nodeHash) {
				break
			}

			stackIter.Next()
			recorder.Next()
		}

		for {
			var step Step
			if len(stack) == 0 {
				// as the stack is empty then start from the root node
				step = Step{
					Found:     false,
					NextNode:  root,
					KeyOffset: 0,
				}
			} else {
				entryOnTop := stack.Last()
				step, err = matchKeyToNode(entryOnTop, nk)
				if err != nil {
					return nil, err
				}
			}

			if step.Found {
				if len(step.Value) > 0 && bytes.Equal(step.Value, expectedValue) && recorder.Len() > 0 {
					return nil, errors.New("value found is not expected or there is recNodes to traverse")
				}

				break
			}

			rec := recorder.Next()
			if rec == nil {
				return nil, errors.New("recorder must has nodes to traverse")
			}

			if !bytes.Equal(rec.Hash, step.NextNode) {
				return nil, errors.New("recorded node does not match expected node")
			}

			n, err := decodeBytes(rec.RawData)
			if err != nil {
				return nil, err
			}

			outputIndex := len(proofs)
			proofs = append(proofs, []byte{})

			prefix := make([]byte, len(nk[:step.KeyOffset]))
			copy(prefix, nk[:step.KeyOffset])

			ne, err := NewStackEntry(n, rec.Hash, rec.RawData, prefix, outputIndex, step.KeyOffset)
			if err != nil {
				return nil, err
			}

			stack.Push(ne)
		}
	}

	unwindStack(&stack, proofs, nil)
	return proofs, nil
}

func matchKeyToNode(e *StackEntry, nibbleKey []byte) (Step, error) {
	switch ntype := e.node.(type) {
	case nil:
		return Step{Found: true}, nil
	case *leaf:
		keyToCompare := nibbleKey[e.keyOffset:]
		if bytes.Equal(keyToCompare, ntype.key) && len(nibbleKey) == len(ntype.key)+e.keyOffset {
			return Step{
				Found: true,
				Value: ntype.value,
			}, nil
		}

		return Step{Found: true}, nil
	case *branch:
		return matchKeyToBranchNode(ntype, e, nibbleKey)
	default:
		return Step{}, errors.New("could not be possible to define the node type")
	}
}

func matchKeyToBranchNode(n *branch, e *StackEntry, nibbleKey []byte) (Step, error) {
	keyToCompare := nibbleKey[e.keyOffset:]
	if !bytes.HasPrefix(keyToCompare, n.key) {
		return Step{Found: true}, nil
	}

	if len(nibbleKey) == len(n.key)+e.keyOffset {
		return Step{
			Found: true,
			Value: n.value,
		}, nil
	}

	newIndex := nibbleKey[e.keyOffset+len(n.key)]
	child := n.children[newIndex]
	if child == nil {
		return Step{Found: true}, nil
	}

	_, hash, err := child.encodeAndHash()
	if err != nil {
		return Step{}, err
	}

	return Step{
		Found:     false,
		KeyOffset: e.keyOffset + len(n.key) + 1,
		NextNode:  hash,
	}, nil
}

func unwindStack(stack *Stack, proof [][]byte, key []byte) error {
	for {
		entry := stack.Pop()
		if entry == nil {
			break
		}

		if key != nil && bytes.HasPrefix(key, entry.key) {
			stack.Push(entry)
			break
		}

		proof[entry.outputIndex] = entry.nodeRawData
	}

	return nil
}
