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

	"github.com/ChainSafe/gossamer/lib/common"
)

//nolint
var EmptyHash, _ = NewEmptyTrie().Hash()

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use NewTrie to create a trie that sits on top of a database.
type Trie struct {
	generation uint64
	root       node
	children   map[common.Hash]*Trie // Used to store the child tries.
}

// NewEmptyTrie creates a trie with a nil root
func NewEmptyTrie() *Trie {
	return NewTrie(nil)
}

// NewTrie creates a trie with an existing root node
func NewTrie(root node) *Trie {
	return &Trie{
		root:       root,
		children:   make(map[common.Hash]*Trie),
		generation: 0, // Initially zero but increases after every snapshot.
	}
}

// Snapshot created a copy of the trie.
func (t *Trie) Snapshot() *Trie {
	oldTrie := &Trie{
		generation: t.generation,
		root:       t.root,
		children:   t.children,
	}
	t.generation++
	return oldTrie
}

func (t *Trie) maybeUpdateLeafGeneration(n *leaf) *leaf {
	// Make a copy if the generation is updated.
	if n.getGeneration() < t.generation {
		// Insert a new leaf node in the current generation.
		newLeaf := n.copy()
		newLeaf.generation = t.generation
		return newLeaf
	}
	return n
}

func (t *Trie) maybeUpdateBranchGeneration(n *branch) *branch {
	// Make a copy if the generation is updated.
	if n.getGeneration() < t.generation {
		// Insert a new branch node in the current generation.
		newBranch := n.copy()
		newBranch.generation = t.generation
		return newBranch
	}
	return n
}

// DeepCopy makes a new trie and copies over the existing trie into the new trie
func (t *Trie) DeepCopy() (*Trie, error) {
	cp := NewEmptyTrie()
	for k, v := range t.Entries() {
		keyCp := make([]byte, len(k))
		copy(keyCp, k)
		valCp := make([]byte, len(v))
		copy(valCp, v)

		cp.Put(keyCp, valCp)
	}

	return cp, nil
}

// RootNode returns the root of the trie
func (t *Trie) RootNode() node { //nolint
	return t.root
}

// EncodeRoot returns the encoded root of the trie
func (t *Trie) EncodeRoot() ([]byte, error) {
	return encode(t.RootNode())
}

// MustHash returns the hashed root of the trie. It panics if it fails to hash the root node.
func (t *Trie) MustHash() common.Hash {
	h, err := t.Hash()
	if err != nil {
		panic(err)
	}

	return h
}

// Hash returns the hashed root of the trie
func (t *Trie) Hash() (common.Hash, error) {
	encRoot, err := t.EncodeRoot()
	if err != nil {
		return [32]byte{}, err
	}

	return common.Blake2bHash(encRoot)
}

// Entries returns all the key-value pairs in the trie as a map of keys to values
func (t *Trie) Entries() map[string][]byte {
	return t.entries(t.root, nil, make(map[string][]byte))
}

func (t *Trie) entries(current node, prefix []byte, kv map[string][]byte) map[string][]byte {
	switch c := current.(type) {
	case *branch:
		if c.value != nil {
			kv[string(nibblesToKeyLE(append(prefix, c.key...)))] = c.value
		}
		for i, child := range c.children {
			t.entries(child, append(prefix, append(c.key, byte(i))...), kv)
		}
	case *leaf:
		kv[string(nibblesToKeyLE(append(prefix, c.key...)))] = c.value
		return kv
	}

	return kv
}

// NextKey returns the next key in the trie in lexicographic order. It returns nil if there is no next key
func (t *Trie) NextKey(key []byte) []byte {
	k := keyToNibbles(key)

	next := t.nextKey([]node{}, t.root, nil, k)
	if next == nil {
		return nil
	}

	return nibblesToKeyLE(next)
}

func (t *Trie) nextKey(ancestors []node, current node, prefix, target []byte) []byte {
	switch c := current.(type) {
	case *branch:
		fullKey := append(prefix, c.key...)

		if bytes.Equal(target, fullKey) {
			for i, child := range c.children {
				if child == nil {
					continue
				}

				// descend and return first key
				return returnFirstKey(append(fullKey, byte(i)), child)
			}
		}

		if len(target) >= len(fullKey) && bytes.Equal(target[:len(fullKey)], fullKey) {
			for i, child := range c.children {
				if child == nil || byte(i) != target[len(fullKey)] {
					continue
				}

				return t.nextKey(append([]node{c}, ancestors...), child, append(fullKey, byte(i)), target)
			}
		}
	case *leaf:
		fullKey := append(prefix, c.key...)

		if bytes.Equal(target, fullKey) {
			// ancestors are all branches, find one with another child w/ index greater than ours
			for _, anc := range ancestors {
				// index of the current node in its parent branch
				myIdx := prefix[len(prefix)-1]

				br, ok := anc.(*branch)
				if !ok {
					return nil
				}

				prefix = prefix[:len(prefix)-len(br.key)-1]

				if br.childrenBitmap()>>(myIdx+1) == 0 {
					continue
				}

				// descend into ancestor's other children
				for i, child := range br.children[myIdx+1:] {
					idx := byte(i) + myIdx + 1

					if child == nil {
						continue
					}

					return returnFirstKey(append(prefix, append(br.key, idx)...), child)
				}
			}
		}
	}

	return nil
}

// returnFirstKey descends into a node and returns the first key with an associated value
func returnFirstKey(prefix []byte, n node) []byte {
	switch c := n.(type) {
	case *branch:
		if c.value != nil {
			return append(prefix, c.key...)
		}

		for i, child := range c.children {
			if child == nil {
				continue
			}

			return returnFirstKey(append(prefix, append(c.key, byte(i))...), child)
		}
	case *leaf:
		return append(prefix, c.key...)
	}

	return nil
}

// Put inserts a key with value into the trie
func (t *Trie) Put(key, value []byte) {
	t.tryPut(key, value)
}

func (t *Trie) tryPut(key, value []byte) {
	k := keyToNibbles(key)

	t.root = t.insert(t.root, k, &leaf{key: nil, value: value, dirty: true, generation: t.generation})
}

// TryPut attempts to insert a key with value into the trie
func (t *Trie) insert(parent node, key []byte, value node) node {
	switch p := parent.(type) {
	case *branch:
		p = t.maybeUpdateBranchGeneration(p)
		n := t.updateBranch(p, key, value)

		if p != nil && n != nil && n.isDirty() {
			p.setDirty(true)
		}
		return n
	case nil:
		// We are creating new node so it will always have the latest generation.
		switch v := value.(type) {
		case *branch:
			v.key = key
			return v
		case *leaf:
			v.key = key
			return v
		}
	case *leaf:
		p = t.maybeUpdateLeafGeneration(p)

		// if a value already exists in the trie at this key, overwrite it with the new value
		// if the values are the same, don't mark node dirty
		if p.value != nil && bytes.Equal(p.key, key) {
			if !bytes.Equal(value.(*leaf).value, p.value) {
				p.value = value.(*leaf).value
				p.dirty = true
			}
			return p
		}
		length := lenCommonPrefix(key, p.key)

		// need to convert this leaf into a branch
		br := &branch{key: key[:length], dirty: true, generation: t.generation}
		parentKey := p.key

		// value goes at this branch
		if len(key) == length {
			br.value = value.(*leaf).value
			br.setDirty(true)

			// if we are not replacing previous leaf, then add it as a child to the new branch
			if len(parentKey) > len(key) {
				p.key = p.key[length+1:]
				br.children[parentKey[length]] = p
				p.setDirty(true)
			}

			return br
		}

		value.setKey(key[length+1:])

		if length == len(p.key) {
			// if leaf's key is covered by this branch, then make the leaf's
			// value the value at this branch
			br.value = p.value
			br.children[key[length]] = value
		} else {
			// otherwise, make the leaf a child of the branch and update its partial key
			p.key = p.key[length+1:]
			p.setDirty(true)
			br.children[parentKey[length]] = p
			br.children[key[length]] = value
		}

		return br
	}
	// This will never happen.
	return nil
}

// updateBranch attempts to add the value node to a branch
// inserts the value node as the branch's child at the index that's
// the first nibble of the key
func (t *Trie) updateBranch(p *branch, key []byte, value node) (n node) {
	length := lenCommonPrefix(key, p.key)

	// whole parent key matches
	if length == len(p.key) {
		// if node has same key as this branch, then update the value at this branch
		if bytes.Equal(key, p.key) {
			p.setDirty(true)
			switch v := value.(type) {
			case *branch:
				p.value = v.value
			case *leaf:
				p.value = v.value
			}
			return p
		}

		switch c := p.children[key[length]].(type) {
		case *branch, *leaf:
			n = t.insert(c, key[length+1:], value)
			p.children[key[length]] = n
			n.setDirty(true)
			p.setDirty(true)
			return p
		case nil:
			// otherwise, add node as child of this branch
			value.(*leaf).key = key[length+1:]
			p.children[key[length]] = value
			p.setDirty(true)
			return p
		}

		return n
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	br := &branch{key: key[:length], dirty: true, generation: t.generation}

	parentIndex := p.key[length]
	br.children[parentIndex] = t.insert(nil, p.key[length+1:], p)

	if len(key) <= length {
		br.value = value.(*leaf).value
	} else {
		br.children[key[length]] = t.insert(nil, key[length+1:], value)
	}

	br.setDirty(true)
	return br
}

// LoadFromMap loads the given data into trie
func (t *Trie) LoadFromMap(data map[string]string) error {
	for key, value := range data {
		keyBytes, err := common.HexToBytes(key)
		if err != nil {
			return err
		}
		valueBytes, err := common.HexToBytes(value)
		if err != nil {
			return err
		}
		t.Put(keyBytes, valueBytes)
	}

	return nil
}

// GetKeysWithPrefix returns all keys in the trie that have the given prefix
func (t *Trie) GetKeysWithPrefix(prefix []byte) [][]byte {
	p := []byte{}
	if len(prefix) != 0 {
		p = keyToNibbles(prefix)
		if p[len(p)-1] == 0 {
			p = p[:len(p)-1]
		}
	}

	return t.getKeysWithPrefix(t.root, []byte{}, p, [][]byte{})
}

func (t *Trie) getKeysWithPrefix(parent node, prefix, key []byte, keys [][]byte) [][]byte {
	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		if bytes.Equal(p.key[:length], key) || len(key) == 0 {
			// node has prefix, add to list and add all descendant nodes to list
			keys = t.addAllKeys(p, prefix, keys)
			return keys
		}

		if len(key) <= len(p.key) {
			// no prefixed keys to be found here, return
			return keys
		}

		key = key[len(p.key):]
		keys = t.getKeysWithPrefix(p.children[key[0]], append(append(prefix, p.key...), key[0]), key[1:], keys)
	case *leaf:
		length := lenCommonPrefix(p.key, key)
		if bytes.Equal(p.key[:length], key) || len(key) == 0 {
			keys = append(keys, nibblesToKeyLE(append(prefix, p.key...)))
		}
	case nil:
		return keys
	}
	return keys
}

// addAllKeys appends all keys that are descendants of the parent node to a slice of keys
// it uses the prefix to determine the entire key
func (t *Trie) addAllKeys(parent node, prefix []byte, keys [][]byte) [][]byte {
	switch p := parent.(type) {
	case *branch:
		if p.value != nil {
			keys = append(keys, nibblesToKeyLE(append(prefix, p.key...)))
		}

		for i, child := range p.children {
			keys = t.addAllKeys(child, append(append(prefix, p.key...), byte(i)), keys)
		}
	case *leaf:
		keys = append(keys, nibblesToKeyLE(append(prefix, p.key...)))
	case nil:
		return keys
	}

	return keys
}

// Get returns the value for key stored in the trie at the corresponding key
func (t *Trie) Get(key []byte) (value []byte, err error) {
	l, err := t.tryGet(key)
	if l != nil {
		return l.value, err
	}
	return nil, err
}

// getLeaf returns the leaf node stored in the trie at the corresponding key
// leaf includes both partial key and value, need the partial key for encoding
func (t *Trie) getLeaf(key []byte) (value *leaf, err error) {
	l, err := t.tryGet(key)
	return l, err
}

func (t *Trie) tryGet(key []byte) (value *leaf, err error) {
	k := keyToNibbles(key)

	value, err = t.retrieve(t.root, k)
	return value, err
}

func (t *Trie) retrieve(parent node, key []byte) (value *leaf, err error) {
	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			return &leaf{key: p.key, value: p.value, dirty: false}, nil
		}

		// did not find value
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
			return nil, nil
		}

		value, err = t.retrieve(p.children[key[length]], key[length+1:])
	case *leaf:
		if bytes.Equal(p.key, key) {
			value = p
		}
	case nil:
		return nil, nil
	default:
		err = errors.New("get error: invalid node")
	}
	return value, err
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *Trie) ClearPrefix(prefix []byte) {
	if len(prefix) == 0 {
		t.root = nil
		return
	}

	p := keyToNibbles(prefix)
	if p[len(p)-1] == 0 {
		p = p[:len(p)-1]
	}

	t.root, _ = t.clearPrefix(t.root, p)
}

func (t *Trie) clearPrefix(curr node, prefix []byte) (node, bool) {
	switch c := curr.(type) {
	case *branch:
		length := lenCommonPrefix(c.key, prefix)

		if length == len(prefix) {
			// found prefix at this branch, delete it
			return nil, true
		}

		// Store the current node and return it, if the trie is not updated.
		currN := c
		c = t.maybeUpdateBranchGeneration(c)

		if len(prefix) == len(c.key)+1 {
			// found prefix at child index, delete child
			i := prefix[len(c.key)]
			c.children[i] = nil
			c.setDirty(true)
			curr = handleDeletion(c, prefix)
			return curr, true
		}

		if len(prefix) <= len(c.key) {
			// this node doesn't have the prefix, return
			return currN, false
		}

		var wasUpdated bool
		i := prefix[len(c.key)]

		c.children[i], wasUpdated = t.clearPrefix(c.children[i], prefix[len(c.key)+1:])
		if wasUpdated {
			c.setDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.isDirty()
	case *leaf:
		length := lenCommonPrefix(c.key, prefix)
		if length == len(prefix) {
			return nil, true
		}
		return c, false
	case nil:
		return nil, false
	}
	// This should never happen.
	return nil, false
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) error {
	k := keyToNibbles(key)
	t.root, _ = t.delete(t.root, k)
	// TODO: Remove error from result.
	return nil
}

func (t *Trie) delete(parent node, key []byte) (node, bool) {
	switch p := parent.(type) {
	case *branch:
		// Store the current node and return it, if the trie is not updated.
		currP := p
		p = t.maybeUpdateBranchGeneration(p)

		length := lenCommonPrefix(p.key, key)
		if bytes.Equal(p.key, key) || len(key) == 0 {
			// found the value at this node
			p.value = nil
			p.setDirty(true)
			return handleDeletion(p, key), true
		}

		n, del := t.delete(p.children[key[length]], key[length+1:])
		if !del {
			// If nothing was deleted then don't copy the path.
			return currP, false
		}

		p.children[key[length]] = n
		p.setDirty(true)
		n = handleDeletion(p, key)
		return n, true
	case *leaf:
		if bytes.Equal(key, p.key) || len(key) == 0 {
			// Key exists. Delete it.
			return nil, true
		}
		// Key doesn't exist.
		return p, false
	case nil:
		// do nothing
	}
	// This should never happen.
	return nil, false
}

// handleDeletion is called when a value is deleted from a branch
// if the updated branch only has 1 child, it should be combined with that child
// if the updated branch only has a value, it should be turned into a leaf
func handleDeletion(p *branch, key []byte) node {
	var n node = p
	length := lenCommonPrefix(p.key, key)
	bitmap := p.childrenBitmap()

	// if branch has no children, just a value, turn it into a leaf
	if bitmap == 0 && p.value != nil {
		n = &leaf{key: key[:length], value: p.value, dirty: true}
	} else if p.numChildren() == 1 && p.value == nil {
		// there is only 1 child and no value, combine the child branch with this branch
		// find index of child
		var i int
		for i = 0; i < 16; i++ {
			bitmap = bitmap >> 1
			if bitmap == 0 {
				break
			}
		}

		child := p.children[i]
		switch c := child.(type) {
		case *leaf:
			n = &leaf{key: append(append(p.key, []byte{byte(i)}...), c.key...), value: c.value}
		case *branch:
			br := new(branch)
			br.key = append(p.key, append([]byte{byte(i)}, c.key...)...)

			// adopt the grandchildren
			for i, grandchild := range c.children {
				if grandchild != nil {
					br.children[i] = grandchild
				}
			}

			br.value = c.value
			n = br
		default:
			// do nothing
		}
		n.setDirty(true)
	}
	return n
}

// lenCommonPrefix returns the length of the common prefix between two keys
func lenCommonPrefix(a, b []byte) int {
	var length, min = 0, len(a)

	if len(a) > len(b) {
		min = len(b)
	}

	for ; length < min; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
