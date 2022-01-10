// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
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
	deletedKeys map[common.Hash]struct{}
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
		deletedKeys: make(map[common.Hash]struct{}),
	}
}

// Snapshot created a copy of the trie.
func (t *Trie) Snapshot() *Trie {
	children := make(map[common.Hash]*Trie)
	for h, c := range t.childTries {
		children[h] = &Trie{
			generation:  c.generation + 1,
			root:        c.root,
			deletedKeys: make(map[common.Hash]struct{}),
		}
	}

	newTrie := &Trie{
		generation:  t.generation + 1,
		root:        t.root,
		childTries:  children,
		deletedKeys: make(map[common.Hash]struct{}),
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
			t.deletedKeys[hash] = struct{}{}
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
	if t.root == nil {
		_, err = buffer.Write([]byte{0})
		if err != nil {
			return fmt.Errorf("cannot write nil root node to buffer: %w", err)
		}
		return nil
	}
	return t.root.Encode(buffer)
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
	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

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
	case *node.Branch:
		if c.Value != nil {
			kv[string(codec.NibblesToKeyLE(append(prefix, c.Key...)))] = c.Value
		}
		for i, child := range c.Children {
			t.entries(child, append(prefix, append(c.Key, byte(i))...), kv)
		}
	case *node.Leaf:
		kv[string(codec.NibblesToKeyLE(append(prefix, c.Key...)))] = c.Value
		return kv
	}

	return kv
}

// NextKey returns the next key in the trie in lexicographic order. It returns nil if there is no next key
func (t *Trie) NextKey(key []byte) []byte {
	k := codec.KeyLEToNibbles(key)

	next := t.nextKey(t.root, nil, k)
	if next == nil {
		return nil
	}

	return codec.NibblesToKeyLE(next)
}

func (t *Trie) nextKey(curr Node, prefix, key []byte) []byte {
	switch c := curr.(type) {
	case *node.Branch:
		fullKey := append(prefix, c.Key...)
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
			if c.Value != nil && bytes.Compare(fullKey, key) > 0 {
				return fullKey
			}

			for i, child := range c.Children {
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
			for i, child := range c.Children[idx:] {
				if child == nil {
					continue
				}

				next := t.nextKey(child, append(fullKey, byte(i)+idx), key)
				if len(next) != 0 {
					return next
				}
			}
		}
	case *node.Leaf:
		fullKey := append(prefix, c.Key...)
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
			return append(prefix, c.Key...)
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
	k := codec.KeyLEToNibbles(key)

	t.root = t.insert(t.root, k, node.NewLeaf(nil, value, true, t.generation))
}

// insert attempts to insert a key with value into the trie
func (t *Trie) insert(parent Node, key []byte, value Node) Node {
	newParent := t.maybeUpdateGeneration(parent)
	if newParent == nil {
		value.SetKey(key)
		return value
	}

	switch newParent.Type() {
	case node.BranchType, node.BranchWithValueType:
		p := newParent.(*node.Branch)
		n := t.updateBranch(p, key, value)

		if p != nil && n != nil && n.IsDirty() {
			p.SetDirty(true)
		}
		return n
	case node.LeafType:
		p := newParent.(*node.Leaf)
		// if a value already exists in the trie at this key, overwrite it with the new value
		// if the values are the same, don't mark node dirty
		if p.Value != nil && bytes.Equal(p.Key, key) {
			if !bytes.Equal(value.(*node.Leaf).Value, p.Value) {
				p.Value = value.(*node.Leaf).Value
				p.SetDirty(true)
			}
			return p
		}

		length := lenCommonPrefix(key, p.Key)

		// need to convert this leaf into a branch
		var newBranchValue []byte
		const newBranchDirty = true
		br := node.NewBranch(key[:length], newBranchValue, newBranchDirty, t.generation)
		parentKey := p.Key

		// value goes at this branch
		if len(key) == length {
			br.Value = value.(*node.Leaf).Value
			br.SetDirty(true)

			// if we are not replacing previous leaf, then add it as a child to the new branch
			if len(parentKey) > len(key) {
				p.Key = p.Key[length+1:]
				br.Children[parentKey[length]] = p
				p.SetDirty(true)
			}

			return br
		}

		value.SetKey(key[length+1:])

		if length == len(p.Key) {
			// if leaf's key is covered by this branch, then make the leaf's
			// value the value at this branch
			br.Value = p.Value
			br.Children[key[length]] = value
		} else {
			// otherwise, make the leaf a child of the branch and update its partial key
			p.Key = p.Key[length+1:]
			p.SetDirty(true)
			br.Children[parentKey[length]] = p
			br.Children[key[length]] = value
		}

		return br
	default:
		panic("unknown node type: " + fmt.Sprint(newParent.Type()))
	}
}

// updateBranch attempts to add the value node to a branch
// inserts the value node as the branch's child at the index that's
// the first nibble of the key
func (t *Trie) updateBranch(p *node.Branch, key []byte, value Node) (n Node) {
	length := lenCommonPrefix(key, p.Key)

	// whole parent key matches
	if length == len(p.Key) {
		// if node has same key as this branch, then update the value at this branch
		if bytes.Equal(key, p.Key) {
			p.SetDirty(true)
			switch v := value.(type) {
			case *node.Branch:
				p.Value = v.Value
			case *node.Leaf:
				p.Value = v.Value
			}
			return p
		}

		switch c := p.Children[key[length]].(type) {
		case *node.Branch, *node.Leaf:
			n = t.insert(c, key[length+1:], value)
			p.Children[key[length]] = n
			n.SetDirty(true)
			p.SetDirty(true)
			return p
		case nil:
			// otherwise, add node as child of this branch
			value.(*node.Leaf).Key = key[length+1:]
			p.Children[key[length]] = value
			p.SetDirty(true)
			return p
		}

		return n
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	var newBranchValue []byte
	const newBranchDirty = true
	br := node.NewBranch(key[:length], newBranchValue, newBranchDirty, t.generation)

	parentIndex := p.Key[length]
	br.Children[parentIndex] = t.insert(nil, p.Key[length+1:], p)

	if len(key) <= length {
		br.Value = value.(*node.Leaf).Value
	} else {
		br.Children[key[length]] = t.insert(nil, key[length+1:], value)
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
		p = codec.KeyLEToNibbles(prefix)
		if p[len(p)-1] == 0 {
			p = p[:len(p)-1]
		}
	}

	return t.getKeysWithPrefix(t.root, []byte{}, p, [][]byte{})
}

func (t *Trie) getKeysWithPrefix(parent Node, prefix, key []byte, keys [][]byte) [][]byte {
	switch p := parent.(type) {
	case *node.Branch:
		length := lenCommonPrefix(p.Key, key)

		if bytes.Equal(p.Key[:length], key) || len(key) == 0 {
			// node has prefix, add to list and add all descendant nodes to list
			keys = t.addAllKeys(p, prefix, keys)
			return keys
		}

		if len(key) <= len(p.Key) || length < len(p.Key) {
			// no prefixed keys to be found here, return
			return keys
		}

		key = key[len(p.Key):]
		keys = t.getKeysWithPrefix(p.Children[key[0]], append(append(prefix, p.Key...), key[0]), key[1:], keys)
	case *node.Leaf:
		length := lenCommonPrefix(p.Key, key)
		if bytes.Equal(p.Key[:length], key) || len(key) == 0 {
			keys = append(keys, codec.NibblesToKeyLE(append(prefix, p.Key...)))
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
	case *node.Branch:
		if p.Value != nil {
			keys = append(keys, codec.NibblesToKeyLE(append(prefix, p.Key...)))
		}

		for i, child := range p.Children {
			keys = t.addAllKeys(child, append(append(prefix, p.Key...), byte(i)), keys)
		}
	case *node.Leaf:
		keys = append(keys, codec.NibblesToKeyLE(append(prefix, p.Key...)))
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

	return l.Value
}

func (t *Trie) tryGet(key []byte) *node.Leaf {
	k := codec.KeyLEToNibbles(key)
	return t.retrieve(t.root, k)
}

func (t *Trie) retrieve(parent Node, key []byte) *node.Leaf {
	var (
		value *node.Leaf
	)

	switch p := parent.(type) {
	case *node.Branch:
		length := lenCommonPrefix(p.Key, key)

		// found the value at this node
		if bytes.Equal(p.Key, key) || len(key) == 0 {
			return node.NewLeaf(p.Key, p.Value, false, 0)
		}

		// did not find value
		if bytes.Equal(p.Key[:length], key) && len(key) < len(p.Key) {
			return nil
		}

		value = t.retrieve(p.Children[key[length]], key[length+1:])
	case *node.Leaf:
		if bytes.Equal(p.Key, key) {
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

	p := codec.KeyLEToNibbles(prefix)
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
	case *node.Branch:
		length := lenCommonPrefix(c.Key, prefix)
		if length == len(prefix) {
			n, _ := t.deleteNodes(c, []byte{}, limit)
			if n == nil {
				return nil, true, true
			}
			return n, true, false
		}

		if len(prefix) == len(c.Key)+1 && length == len(prefix)-1 {
			i := prefix[len(c.Key)]
			c.Children[i], _ = t.deleteNodes(c.Children[i], []byte{}, limit)

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)

			if c.Children[i] == nil {
				return curr, true, true
			}
			return c, true, false
		}

		if len(prefix) <= len(c.Key) || length < len(c.Key) {
			// this node doesn't have the prefix, return
			return c, false, true
		}

		i := prefix[len(c.Key)]

		var wasUpdated, allDeleted bool
		c.Children[i], wasUpdated, allDeleted = t.clearPrefixLimit(c.Children[i], prefix[len(c.Key)+1:], limit)
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty(), allDeleted
	case *node.Leaf:
		length := lenCommonPrefix(c.Key, prefix)
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
	case *node.Leaf:
		if *limit == 0 {
			return c, false
		}
		*limit--
		return nil, true
	case *node.Branch:
		if len(c.Key) != 0 {
			prefix = append(prefix, c.Key...)
		}

		for i, child := range c.Children {
			if child == nil {
				continue
			}

			var isDel bool
			if c.Children[i], isDel = t.deleteNodes(child, prefix, limit); !isDel {
				continue
			}

			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			isAllNil := c.NumChildren() == 0
			if isAllNil && c.Value == nil {
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
		if c.Value != nil {
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

	p := codec.KeyLEToNibbles(prefix)
	if len(p) > 0 && p[len(p)-1] == 0 {
		p = p[:len(p)-1]
	}

	t.root, _ = t.clearPrefix(t.root, p)
}

func (t *Trie) clearPrefix(cn Node, prefix []byte) (Node, bool) {
	curr := t.maybeUpdateGeneration(cn)
	switch c := curr.(type) {
	case *node.Branch:
		length := lenCommonPrefix(c.Key, prefix)

		if length == len(prefix) {
			// found prefix at this branch, delete it
			return nil, true
		}

		// Store the current node and return it, if the trie is not updated.

		if len(prefix) == len(c.Key)+1 && length == len(prefix)-1 {
			// found prefix at child index, delete child
			i := prefix[len(c.Key)]
			c.Children[i] = nil
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
			return curr, true
		}

		if len(prefix) <= len(c.Key) || length < len(c.Key) {
			// this node doesn't have the prefix, return
			return c, false
		}

		var wasUpdated bool
		i := prefix[len(c.Key)]

		c.Children[i], wasUpdated = t.clearPrefix(c.Children[i], prefix[len(c.Key)+1:])
		if wasUpdated {
			c.SetDirty(true)
			curr = handleDeletion(c, prefix)
		}

		return curr, curr.IsDirty()
	case *node.Leaf:
		length := lenCommonPrefix(c.Key, prefix)
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
	k := codec.KeyLEToNibbles(key)
	t.root, _ = t.delete(t.root, k)
}

func (t *Trie) delete(parent Node, key []byte) (Node, bool) {
	// Store the current node and return it, if the trie is not updated.
	switch p := t.maybeUpdateGeneration(parent).(type) {
	case *node.Branch:

		length := lenCommonPrefix(p.Key, key)
		if bytes.Equal(p.Key, key) || len(key) == 0 {
			// found the value at this node
			p.Value = nil
			p.SetDirty(true)
			return handleDeletion(p, key), true
		}

		n, del := t.delete(p.Children[key[length]], key[length+1:])
		if !del {
			// If nothing was deleted then don't copy the path.
			return p, false
		}

		p.Children[key[length]] = n
		p.SetDirty(true)
		n = handleDeletion(p, key)
		return n, true
	case *node.Leaf:
		if bytes.Equal(key, p.Key) || len(key) == 0 {
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
func handleDeletion(p *node.Branch, key []byte) Node {
	var n Node = p
	length := lenCommonPrefix(p.Key, key)
	bitmap := p.ChildrenBitmap()

	// if branch has no children, just a value, turn it into a leaf
	if bitmap == 0 && p.Value != nil {
		n = node.NewLeaf(key[:length], p.Value, true, 0)
	} else if p.NumChildren() == 1 && p.Value == nil {
		// there is only 1 child and no value, combine the child branch with this branch
		// find index of child
		var i int
		for i = 0; i < 16; i++ {
			bitmap = bitmap >> 1
			if bitmap == 0 {
				break
			}
		}

		child := p.Children[i]
		switch c := child.(type) {
		case *node.Leaf:
			n = &node.Leaf{Key: append(append(p.Key, []byte{byte(i)}...), c.Key...), Value: c.Value}
		case *node.Branch:
			br := new(node.Branch)
			br.Key = append(p.Key, append([]byte{byte(i)}, c.Key...)...)

			// adopt the grandchildren
			for i, grandchild := range c.Children {
				if grandchild != nil {
					br.Children[i] = grandchild
				}
			}

			br.Value = c.Value
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
