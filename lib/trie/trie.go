// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// EmptyHash is the empty trie hash.
var EmptyHash, _ = NewEmptyTrie().Hash()

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use NewTrie to create a trie that sits on top of a database.
type Trie struct {
	generation  uint64
	root        Node
	childTries  map[common.Hash]*Trie // Used to store the child tries.
	deletedKeys []common.Hash
	parallel    bool
}

// NewEmptyTrie creates a trie with a nil root
func NewEmptyTrie() *Trie {
	return NewTrie(nil)
}

// NewTrie creates a trie with an existing root node
func NewTrie(root Node) *Trie {
	return &Trie{
		root:        root,
		childTries:  make(map[common.Hash]*Trie),
		generation:  0, // Initially zero but increases after every snapshot.
		deletedKeys: make([]common.Hash, 0),
		parallel:    true,
	}
}

// Snapshot created a copy of the trie.
func (t *Trie) Snapshot() *Trie {
	children := make(map[common.Hash]*Trie)
	for h, c := range t.childTries {
		children[h] = &Trie{
			generation:  c.generation + 1,
			root:        c.root,
			deletedKeys: make([]common.Hash, 0),
			parallel:    c.parallel,
		}
	}

	newTrie := &Trie{
		generation:  t.generation + 1,
		root:        t.root,
		childTries:  children,
		deletedKeys: make([]common.Hash, 0),
		parallel:    t.parallel,
	}

	return newTrie
}

func (t *Trie) maybeUpdateGeneration(n Node) Node {
	if n == nil {
		return nil
	}

	// Make a copy if the generation is updated.
	if n.GetGeneration() < t.generation {
		// Insert a new node in the current generation.
		newNode := n.Copy()
		newNode.SetGeneration(t.generation)

		// Hash of old nodes should already be computed since it belongs to older generation.
		oldNodeHash := n.GetHash()
		if len(oldNodeHash) > 0 {
			hash := common.BytesToHash(oldNodeHash)
			t.deletedKeys = append(t.deletedKeys, hash)
		}
		return newNode
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
func (t *Trie) RootNode() Node {
	return t.root
}

// encodeRoot returns the encoded root of the trie
func (t *Trie) encodeRoot(buffer *bytes.Buffer) (err error) {
	return encodeNode(t.RootNode(), buffer, t.parallel)
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
	buffer := encodingBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer encodingBufferPool.Put(buffer)

	err := t.encodeRoot(buffer)
	if err != nil {
		return [32]byte{}, err
	}

	return common.Blake2bHash(buffer.Bytes())
}

// Entries returns all the key-value pairs in the trie as a map of keys to values
func (t *Trie) Entries() map[string][]byte {
	return t.entries(t.root, nil, make(map[string][]byte))
}

func (t *Trie) entries(current Node, prefix []byte, kv map[string][]byte) map[string][]byte {
	switch c := current.(type) {
	case *Branch:
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

	next := t.nextKey(t.root, nil, k)
	if next == nil {
		return nil
	}

	return nibblesToKeyLE(next)
}

func (t *Trie) nextKey(curr Node, prefix, key []byte) []byte {
	switch c := curr.(type) {
	case *Branch:
		fullKey := append(prefix, c.key...)
		var cmp int
		if len(key) < len(fullKey) {
			if bytes.Compare(key, fullKey[:len(key)]) == 1 { // arg key is greater than full, return nil
				return nil
			}

			// the key is lexicographically less than the current node key. return first key available
			cmp = 1
		} else {
			// if cmp == 1, then node key is lexicographically greater than the key arg
			cmp = bytes.Compare(fullKey, key[:len(fullKey)])
		}

		// if length of key arg is less than branch key,
		// return key of first child, or key of this branch,
		// if it's a branch with value.
		if (cmp == 0 && len(key) == len(fullKey)) || cmp == 1 {
			if c.value != nil && bytes.Compare(fullKey, key) > 0 {
				return fullKey
			}

			for i, child := range c.children {
				if child == nil {
					continue
				}

				next := t.nextKey(child, append(fullKey, byte(i)), key)
				if len(next) != 0 {
					return next
				}
			}
		}

		// node key isn't greater than the arg key, continue to iterate
		if cmp < 1 && len(key) > len(fullKey) {
			idx := key[len(fullKey)]
			for i, child := range c.children[idx:] {
				if child == nil {
					continue
				}

				next := t.nextKey(child, append(fullKey, byte(i)+idx), key)
				if len(next) != 0 {
					return next
				}
			}
		}
	case *leaf:
		fullKey := append(prefix, c.key...)
		var cmp int
		if len(key) < len(fullKey) {
			if bytes.Compare(key, fullKey[:len(key)]) == 1 { // arg key is greater than full, return nil
				return nil
			}

			// the key is lexicographically less than the current node key. return first key available
			cmp = 1
		} else {
			// if cmp == 1, then node key is lexicographically greater than the key arg
			cmp = bytes.Compare(fullKey, key[:len(fullKey)])
		}

		if cmp == 1 {
			return append(prefix, c.key...)
		}
	case nil:
		return nil
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

// insert attempts to insert a key with value into the trie
func (t *Trie) insert(parent Node, key []byte, value Node) Node {
	switch p := t.maybeUpdateGeneration(parent).(type) {
	case *Branch:
		n := t.updateBranch(p, key, value)

		if p != nil && n != nil && n.IsDirty() {
			p.SetDirty(true)
		}
		return n
	case nil:
		value.SetKey(key)
		return value
	case *leaf:
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
		br := &Branch{key: key[:length], dirty: true, generation: t.generation}
		parentKey := p.key

		// value goes at this branch
		if len(key) == length {
			br.value = value.(*leaf).value
			br.SetDirty(true)

			// if we are not replacing previous leaf, then add it as a child to the new branch
			if len(parentKey) > len(key) {
				p.key = p.key[length+1:]
				br.children[parentKey[length]] = p
				p.SetDirty(true)
			}

			return br
		}

		value.SetKey(key[length+1:])

		if length == len(p.key) {
			// if leaf's key is covered by this branch, then make the leaf's
			// value the value at this branch
			br.value = p.value
			br.children[key[length]] = value
		} else {
			// otherwise, make the leaf a child of the branch and update its partial key
			p.key = p.key[length+1:]
			p.SetDirty(true)
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
func (t *Trie) updateBranch(p *Branch, key []byte, value Node) (n Node) {
	length := lenCommonPrefix(key, p.key)

	// whole parent key matches
	if length == len(p.key) {
		// if node has same key as this branch, then update the value at this branch
		if bytes.Equal(key, p.key) {
			p.SetDirty(true)
			switch v := value.(type) {
			case *Branch:
				p.value = v.value
			case *leaf:
				p.value = v.value
			}
			return p
		}

		switch c := p.children[key[length]].(type) {
		case *Branch, *leaf:
			n = t.insert(c, key[length+1:], value)
			p.children[key[length]] = n
			n.SetDirty(true)
			p.SetDirty(true)
			return p
		case nil:
			// otherwise, add node as child of this branch
			value.(*leaf).key = key[length+1:]
			p.children[key[length]] = value
			p.SetDirty(true)
			return p
		}

		return n
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	br := &Branch{key: key[:length], dirty: true, generation: t.generation}

	parentIndex := p.key[length]
	br.children[parentIndex] = t.insert(nil, p.key[length+1:], p)

	if len(key) <= length {
		br.value = value.(*leaf).value
	} else {
		br.children[key[length]] = t.insert(nil, key[length+1:], value)
	}

	br.SetDirty(true)
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
	var p []byte
	if len(prefix) != 0 {
		p = keyToNibbles(prefix)
		if p[len(p)-1] == 0 {
			p = p[:len(p)-1]
		}
	}

	return t.getKeysWithPrefix(t.root, []byte{}, p, [][]byte{})
}

func (t *Trie) getKeysWithPrefix(parent Node, prefix, key []byte, keys [][]byte) [][]byte {
	switch p := parent.(type) {
	case *Branch:
		length := lenCommonPrefix(p.key, key)

		if bytes.Equal(p.key[:length], key) || len(key) == 0 {
			// node has prefix, add to list and add all descendant nodes to list
			keys = t.addAllKeys(p, prefix, keys)
			return keys
		}

		if len(key) <= len(p.key) || length < len(p.key) {
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
func (t *Trie) addAllKeys(parent Node, prefix []byte, keys [][]byte) [][]byte {
	switch p := parent.(type) {
	case *Branch:
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
func (t *Trie) Get(key []byte) []byte {
	l := t.tryGet(key)
	if l == nil {
		return nil
	}

	return l.value
}

func (t *Trie) tryGet(key []byte) *leaf {
	k := keyToNibbles(key)
	return t.retrieve(t.root, k)
}

func (t *Trie) retrieve(parent Node, key []byte) *leaf {
	var (
		value *leaf
	)

	switch p := parent.(type) {
	case *Branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			return &leaf{key: p.key, value: p.value, dirty: false}
		}

		// did not find value
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
			return nil
		}

		value = t.retrieve(p.children[key[length]], key[length+1:])
	case *leaf:
		if bytes.Equal(p.key, key) {
			value = p
		}
	case nil:
		return nil
	}
	return value
}

// ClearPrefixLimit deletes the keys having the prefix till limit reached
func (t *Trie) ClearPrefixLimit(prefix []byte, limit uint32) (uint32, bool) {
	if limit == 0 {
		return 0, false
	}

	p := keyToNibbles(prefix)
	if len(p) > 0 && p[len(p)-1] == 0 {
		p = p[:len(p)-1]
	}

	l := limit
	var allDeleted bool
	t.root, _, allDeleted = t.clearPrefixLimit(t.root, p, &limit)
	return l - limit, allDeleted
}

// clearPrefixLimit deletes the keys having the prefix till limit reached and returns updated trie root node,
// true if any node in the trie got updated, and next bool returns true if there is no keys left with prefix.
func (t *Trie) clearPrefixLimit(cn Node, prefix []byte, limit *uint32) (Node, bool, bool) {
	curr := t.maybeUpdateGeneration(cn)

	switch c := curr.(type) {
	case *Branch:
		length := lenCommonPrefix(c.key, prefix)
		if length == len(prefix) {
			n, _ := t.deleteNodes(c, []byte{}, limit)
			if n == nil {
				return nil, true, true
			}
			return n, true, false
		}

		if len(prefix) == len(c.key)+1 && length == len(prefix)-1 {
			i := prefix[len(c.key)]
			c.children[i], _ = t.deleteNodes(c.children[i], []byte{}, limit)

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)

			if c.children[i] == nil {
				return curr, true, true
			}
			return c, true, false
		}

		if len(prefix) <= len(c.key) || length < len(c.key) {
			// this node doesn't have the prefix, return
			return c, false, true
		}

		i := prefix[len(c.key)]

		var wasUpdated, allDeleted bool
		c.children[i], wasUpdated, allDeleted = t.clearPrefixLimit(c.children[i], prefix[len(c.key)+1:], limit)
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty(), allDeleted
	case *leaf:
		length := lenCommonPrefix(c.key, prefix)
		if length == len(prefix) {
			*limit--
			return nil, true, true
		}
		// Prefix not found might be all deleted
		return curr, false, true

	case nil:
		return nil, false, true
	}

	return nil, false, true
}

func (t *Trie) deleteNodes(cn Node, prefix []byte, limit *uint32) (Node, bool) {
	curr := t.maybeUpdateGeneration(cn)

	switch c := curr.(type) {
	case *leaf:
		if *limit == 0 {
			return c, false
		}
		*limit--
		return nil, true
	case *Branch:
		if len(c.key) != 0 {
			prefix = append(prefix, c.key...)
		}

		for i, child := range c.children {
			if child == nil {
				continue
			}

			var isDel bool
			if c.children[i], isDel = t.deleteNodes(child, prefix, limit); !isDel {
				continue
			}

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			isAllNil := c.numChildren() == 0
			if isAllNil && c.value == nil {
				curr = nil
			}

			if *limit == 0 {
				return curr, true
			}
		}

		if *limit == 0 {
			return c, true
		}

		// Delete the current node as well
		if c.value != nil {
			*limit--
		}
		return nil, true
	}

	return curr, true
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *Trie) ClearPrefix(prefix []byte) {
	if len(prefix) == 0 {
		t.root = nil
		return
	}

	p := keyToNibbles(prefix)
	if len(p) > 0 && p[len(p)-1] == 0 {
		p = p[:len(p)-1]
	}

	t.root, _ = t.clearPrefix(t.root, p)
}

func (t *Trie) clearPrefix(cn Node, prefix []byte) (Node, bool) {
	curr := t.maybeUpdateGeneration(cn)
	switch c := curr.(type) {
	case *Branch:
		length := lenCommonPrefix(c.key, prefix)

		if length == len(prefix) {
			// found prefix at this branch, delete it
			return nil, true
		}

		// Store the current node and return it, if the trie is not updated.

		if len(prefix) == len(c.key)+1 && length == len(prefix)-1 {
			// found prefix at child index, delete child
			i := prefix[len(c.key)]
			c.children[i] = nil
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			return curr, true
		}

		if len(prefix) <= len(c.key) || length < len(c.key) {
			// this node doesn't have the prefix, return
			return c, false
		}

		var wasUpdated bool
		i := prefix[len(c.key)]

		c.children[i], wasUpdated = t.clearPrefix(c.children[i], prefix[len(c.key)+1:])
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty()
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
func (t *Trie) Delete(key []byte) {
	k := keyToNibbles(key)
	t.root, _ = t.delete(t.root, k)
}

func (t *Trie) delete(parent Node, key []byte) (Node, bool) {
	// Store the current node and return it, if the trie is not updated.
	switch p := t.maybeUpdateGeneration(parent).(type) {
	case *Branch:

		length := lenCommonPrefix(p.key, key)
		if bytes.Equal(p.key, key) || len(key) == 0 {
			// found the value at this node
			p.value = nil
			p.SetDirty(true)
			return handleDeletion(p, key), true
		}

		n, del := t.delete(p.children[key[length]], key[length+1:])
		if !del {
			// If nothing was deleted then don't copy the path.
			return p, false
		}

		p.children[key[length]] = n
		p.SetDirty(true)
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
		return nil, false
	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", p, p, key))
	}
}

// handleDeletion is called when a value is deleted from a branch
// if the updated branch only has 1 child, it should be combined with that child
// if the updated branch only has a value, it should be turned into a leaf
func handleDeletion(p *Branch, key []byte) Node {
	var n Node = p
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
		case *Branch:
			br := new(Branch)
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
		n.SetDirty(true)

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
