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
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"

	log "github.com/ChainSafe/log15"
)

const (
	BitmapLength       = 2
	NibbleLenght       = 16
	NibblePerByte      = 2
	NibbleSizeBound    = ^uint16(0)
	BranchWithMask     = byte(0b11 << 6)
	BranchWithoutMask  = byte(0b10 << 6)
	LeafPrefixWithMask = byte(0b01 << 6)
)

var (
	// ErrEmptyTrieRoot occurs when trying to craft a prove with an empty trie root
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")

	ErrEmptyNibbles = errors.New("empty nibbles provided from key")

	logger = log.New("lib", "trie")
)

type StackEntry struct {
	keyOffset   int
	path        []byte
	nodeHash    []byte
	node        node
	nodeRawData []byte
	children    [][]byte
	childIndex  int
	outputIndex int
	omitValue   bool
}

func (s *StackEntry) encodeNode() ([]byte, error) {
	switch ntype := s.node.(type) {
	case nil:
		return s.nodeRawData, nil
	case *leaf:
		if s.omitValue {
			return leafNodeOmitValue(ntype.key), nil
		}

		return s.nodeRawData, nil
	case *branch:
		// Populate the remaining references in `children`
		for i := s.childIndex; i < NibbleLenght; i++ {
			nodeChild := ntype.children[i]
			if nodeChild != nil {
				var err error
				if _, s.children[i], err = nodeChild.encodeAndHash(); err != nil {
					return nil, err
				}
			}
		}

		v := ntype.value
		if s.omitValue {
			v = []byte{}
		}

		return branchNodeNibbled(ntype.key, s.children, v), nil
	}

	return nil, nil
}

func leafNodeOmitValue(path []byte) []byte {
	split := len(path) % NibblePerByte / NibblePerByte
	nb := byte(len(path) % NibblePerByte)

	var (
		first  byte
		second byte
		data   []byte
	)

	if nb > 0 {
		first, second, data = nb, path[split]&0x0f, path[split+1:]
	} else {
		first, second, data = 0, 0, path[split:]
	}

	nibbleCount := len(data)*NibblePerByte + int(first)
	if nibbleCount > int(NibbleSizeBound) {
		nibbleCount = int(NibbleSizeBound)
	}

	output := make([]byte, 0, 3+len(data))
	sizeIterator(len(path), LeafPrefixWithMask, &output)

	if first > 0 {
		output = append(output, second&0x0f)
	}

	output = append(output, data...)
	return output
}

func branchNodeNibbled(path []byte, children [][]byte, value []byte) []byte {
	nibbleCount := len(path)
	if nibbleCount > int(NibbleSizeBound) {
		nibbleCount = int(NibbleSizeBound)
	}

	output := make([]byte, 0, 3+(nibbleCount/NibblePerByte))

	var prefix byte
	if len(value) == 0 {
		prefix = BranchWithoutMask
	} else {
		prefix = BranchWithMask
	}

	sizeIterator(nibbleCount, prefix, &output)

	output = append(output, path...)
	bitmapIndex := len(output)
	bitmap := make([]byte, BitmapLength)

	for i := 0; i < BitmapLength; i++ {
		output = append(output, 0)
	}

	if len(value) > 0 {
		output = append(output, value...)
	}

	var (
		bitmapValue  uint16 = 0
		bitmapCursor uint16 = 1
	)

	for _, c := range children {
		if len(c) > 0 {
			output = append(output, c...)
			bitmapValue |= bitmapCursor
		}

		bitmapCursor <<= 1
	}

	bitmap[0] = byte(bitmapValue % 256)
	bitmap[1] = byte(bitmapValue / 256)
	copy(output[bitmapIndex:bitmapIndex+BitmapLength], bitmap[:BitmapLength])

	return output
}

func (s *StackEntry) setChild(enc []byte) error {
	switch s.node.(type) {
	case *branch:
		//child := ntype.children[s.childIndex]
		s.children[s.childIndex] = enc
		s.childIndex += 1
		return nil
	default:
		return errors.New("nil, leaf or other nodes does not has children set")
	}
}

func NewStackEntry(n node, rd []byte, outputIndex, keyOffset int) (*StackEntry, error) {
	var (
		children [][]byte
		path     []byte
	)

	switch nt := n.(type) {
	case *leaf:
		children = make([][]byte, 0)
		path = nt.key
	case *branch:
		children = make([][]byte, NibbleLenght)
		path = nt.key
	default:
		return nil, fmt.Errorf("could not define a stack entry for node: %s", nt)
	}

	_, h, err := n.encodeAndHash()
	if err != nil {
		return nil, err
	}

	return &StackEntry{
		keyOffset:   keyOffset,
		nodeHash:    h,
		path:        path,
		node:        n,
		children:    children,
		childIndex:  0,
		outputIndex: outputIndex,
		nodeRawData: rd,
		omitValue:   false,
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

type Step struct {
	Found     bool
	Value     []byte
	NextNode  []byte
	KeyOffset int
}

func GenerateProofWithRecorder(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
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

		for _, recNodes := range *recorder {
			fmt.Printf("recorded node ==> 0x%x\n", recNodes.Hash)
		}

		fmt.Printf("Found at database\n\tkey:0x%x\n\tvalue:0x%x\n", k, expectedValue)
		fmt.Println("Recorded nodes", recorder.Len())

		stackIter := stack.iter()
		// Skip over recorded nodes already on the stack
		for stackIter.HasNext() {
			nxtRec, nxtEntry := recorder.Peek(), stackIter.Peek()
			if !bytes.Equal(nxtRec.Hash[:], nxtEntry.nodeHash) {
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
				logger.Warn("generate proof", "has last on stack", entryOnTop != nil)

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

			logger.Warn("generate proof", "expected", fmt.Sprintf("Recorded node: 0x%x\n", rec.Hash))
			logger.Warn("generate proof", "got", fmt.Sprintf("Step node: 0x%x\n", step.NextNode))

			if !bytes.Equal(rec.Hash, step.NextNode) {
				return nil, errors.New("recorded node does not match expected node")
			}

			n, err := decodeBytes(rec.RawData)
			if err != nil {
				return nil, err
			}
			logger.Warn("generate proof", "has decoded node", n != nil)

			outputIndex := len(proofs)
			proofs = append(proofs, []byte{})

			ne, err := NewStackEntry(n, rec.RawData, outputIndex, step.KeyOffset)
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
			e.omitValue = true
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

	logger.Warn("matchKeyToBranchNode", "keyToCompare", fmt.Sprintf("0x%x", keyToCompare), "len nibbles", len(nibbleKey))
	logger.Warn("matchKeyToBranchNode", "node key", fmt.Sprintf("0x%x", n.key), "key offset", e.keyOffset)

	if !bytes.HasPrefix(keyToCompare, n.key) {
		return Step{Found: true}, nil
	}

	if len(nibbleKey) == len(n.key)+e.keyOffset {
		e.omitValue = true
		return Step{
			Found: true,
			Value: n.value,
		}, nil
	}

	newIndex := nibbleKey[e.keyOffset+len(n.key)]
	for e.childIndex < int(newIndex) {
		nodeChild := n.children[e.childIndex]
		if nodeChild != nil {
			var err error
			if _, e.children[e.childIndex], err = nodeChild.encodeAndHash(); err != nil {
				return Step{}, err
			}
		}
		e.childIndex++
	}

	child := n.children[e.childIndex]
	logger.Warn("matchKeyToBranchNode", "has child", child != nil, "childIndex", e.childIndex)

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

		if key != nil && bytes.HasPrefix(key, entry.path) {
			stack.Push(entry)
			break
		}

		index := entry.outputIndex

		enc, err := entry.encodeNode()
		if err != nil {
			return err
		}

		parent := stack.Last()
		if parent != nil {
			parent.setChild(enc)
		}

		proof[index] = enc
	}

	return nil
}

func sizeIterator(size int, prefix byte, output *[]byte) {
	l := make([]byte, 0)
	l1, rem := 62, 0

	if size < 62 {
		l1 = size
		l = append(l, prefix+byte(l1))
	} else {
		l = append(l, prefix+byte(63))
		rem = size - l1
	}

	for {
		if rem > 0 {
			if rem < 256 {
				result := rem - 1
				rem = 0
				l = append(l, byte(result))
			} else {
				op := rem - 255
				if op < 0 {
					rem = 0
				} else {
					rem = op
				}

				l = append(l, byte(255))
			}
		} else {
			break
		}
	}

	*output = append(*output, l...)
	fmt.Printf("\n\non encode\noutput == 0x%x\nl == 0x%x\n", output, l)
}

// GenerateProof constructs the merkle-proof for key. The result contains all encoded nodes
// on the path to the key. Returns the amount of nodes of the path and error if could not found the key
func (t *Trie) GenerateProof(keys [][]byte) (map[string][]byte, error) {
	var nodes []node

	for _, k := range keys {
		currNode := t.root

		nk := keyToNibbles(k)
		if len(nk) == 0 {
			return nil, ErrEmptyNibbles
		}

	proveLoop:
		for {
			switch n := currNode.(type) {
			case nil:
				return nil, errors.New("no more paths to follow")

			case *leaf:
				nodes = append(nodes, n)

				if bytes.Equal(n.key, nk) {
					break proveLoop
				}

				return nil, errors.New("leaf node doest not match the key")

			case *branch:
				nodes = append(nodes, n)
				if bytes.Equal(n.key, nk) || len(nk) == 0 {
					break proveLoop
				}

				length := lenCommonPrefix(n.key, nk)
				currNode = n.children[nk[length]]
				nk = nk[length+1:]
			}
		}
	}

	proof := make(map[string][]byte)
	for _, n := range nodes {
		var (
			hashNode    []byte
			encHashNode []byte
			err         error
		)

		if encHashNode, hashNode, err = n.encodeAndHash(); err != nil {
			return nil, fmt.Errorf("problems while encoding and hashing the node: %w", err)
		}

		// avoid duplicate hashes
		proof[common.BytesToHex(hashNode)] = encHashNode
	}

	return proof, nil
}

// VerifyProof checks merkle proofs given an proof
func VerifyProof(rootHash common.Hash, key []byte, proof map[string][]byte) (bool, error) {
	key = keyToNibbles(key)
	if len(key) == 0 {
		return false, ErrEmptyNibbles
	}

	var wantedHash string
	wantedHash = common.BytesToHex(rootHash.ToBytes())

	for {
		enc, ok := proof[wantedHash]
		if !ok {
			return false, nil
		}

		currNode, err := decodeBytes(enc)
		if err != nil {
			return false, fmt.Errorf("could not decode node bytes: %w", err)
		}

		switch n := currNode.(type) {
		case nil:
			return false, nil
		case *leaf:
			if bytes.Equal(n.key, key) {
				return true, nil
			}

			return false, nil
		case *branch:
			if bytes.Equal(n.key, key) {
				return true, nil
			}

			if len(key) == 0 {
				return false, nil
			}

			length := lenCommonPrefix(n.key, key)
			next := n.children[key[length]]
			if next == nil {
				return false, nil
			}

			key = key[length+1:]
			wantedHash = common.BytesToHex(next.getHash())
		}
	}
}
