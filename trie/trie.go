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
)

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use NewTrie to create a trie that sits on top of a database.
type Trie struct {
	db   *Database
	root node
}

// NewEmptyTrie creates a trie with a nil root and merkleRoot
func NewEmptyTrie(db *Database) *Trie {
	return &Trie{
		db:   db,
		root: nil,
	}
}

// NewTrie creates a trie with an existing root node from db
func NewTrie(db *Database, root node) *Trie {
	return &Trie{
		db:   db,
		root: root,
	}
}

// Put inserts a key with value into the trie
func (t *Trie) Put(key, value []byte) error {
	if err := t.tryPut(key, value); err != nil {
		return err
	}

	return nil
}

func (t *Trie) tryPut(key, value []byte) (err error) {
	k := keyToNibbles(key)
	var n node

	if len(value) > 0 {
		_, n, err = t.insert(t.root, k, &leaf{key: nil, value: value, dirty: true})
	} else {
		_, n, err = t.delete(t.root, k)
	}

	if err != nil {
		return err
	}

	t.root = n
	return nil
}

// TryPut attempts to insert a key with value into the trie
func (t *Trie) insert(parent node, key []byte, value node) (ok bool, n node, err error) {
	switch p := parent.(type) {
	case *branch:
		ok, n, err = t.updateBranch(p, key, value)
	case nil:
		switch v := value.(type) {
		case *branch:
			v.key = key
			n = v
			ok = true
		case *leaf:
			v.key = key
			n = v
			ok = true
		}
	case *leaf:
		// need to convert this into a branch
		br := &branch{dirty: true}
		length := lenCommonPrefix(key, p.key)

		if len(key) <= length {
			br.key = key[:length]
			br.value = value.(*leaf).value
			parentKey := p.key
			//if len(p.key) > 0 {
			//	p.key = p.key[1:]
			//}
			if len(parentKey) > length {
				p.key = p.key[length+1:]
				br.children[parentKey[length]] = p
			}

			return true, br, nil
		}

		br.key = key[:length]

		switch v := value.(type) {
		case *leaf:
			v.key = key[length+1:]
		case *branch:
			v.key = key[length+1:]
		}

		if length == len(p.key) {
			// if leaf's key is covered by this branch, then make the leaf's
			// value the value at this branch
			br.value = p.value
			br.children[key[length]] = value
		} else {
			// otherwise, make the leaf a child of the branch and update its partial key
			parentKey := p.key
			p.key = p.key[length+1:]
			br.children[parentKey[length]] = p
			br.children[key[length]] = value
		}

		return ok, br, nil
	default:
		err = errors.New("put error: invalid node")
	}

	return ok, n, err
}

// updateBranch attempts to add the value node to a branch
// inserts the value node as the branch's child at the index that's
// the first nibble of the key
func (t *Trie) updateBranch(p *branch, key []byte, value node) (ok bool, n node, err error) {
	length := lenCommonPrefix(key, p.key)

	// whole parent key matches
	if length == len(p.key) {
		// if node has same key as this branch, then update the value at this branch
		if bytes.Equal(key, p.key) {
			switch v := value.(type) {
			case *branch:
				p.value = v.value
			case *leaf:
				p.value = v.value
			}
			return true, p, nil
		}

		switch c := p.children[key[length]].(type) {
		case *branch, *leaf:
			_, n, err = t.insert(c, key[length+1:], value)
			p.children[key[length]] = n
			n = p
		default: // nil or leaf
			// otherwise, add node as child of this branch
			value.(*leaf).key = key[length+1:]
			p.children[key[length]] = value
			n = p
		}

		return true, n, err
	}

	// we need to branch out at the point where the keys diverge
	// update partial keys, new branch has key up to matching length
	br := &branch{key: key[:length], dirty: true}

	parentIndex := p.key[length]
	_, br.children[parentIndex], err = t.insert(nil, p.key[length+1:], p)
	if err != nil {
		return false, nil, err
	}

	if len(key) <= length {
		br.value = value.(*leaf).value
	} else {
		nodeIndex := key[length]
		_, br.children[nodeIndex], err = t.insert(nil, key[length+1:], value)
		if err == nil {
			ok = true
		}
	}

	return ok, br, err
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
			return &leaf{key: p.key, value: p.value, dirty: true}, nil
		}

		// did not find value
		//if len(key) <= length && len(key) < len(p.key) {
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
		//if len(key) <= length {
			return nil, nil
		}

		// if branch's child at the key is a leaf, return it if the key matches
		switch v := p.children[key[length]].(type) {
		case *leaf:
			if bytes.Equal(v.key, key[length+1:]) {
				value = v
			} else {
				value = nil
			}
		default:
			value, err = t.retrieve(p.children[key[length]], key[length+1:])
		}
	case *leaf:
		value = p
	case nil:
		return nil, nil
	default:
		err = errors.New("get error: invalid node")
	}
	return value, err
}


// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) error {
	k := keyToNibbles(key)
	val, err := t.Get(key)
	if val == nil {
		return errors.New("delete error: node not found")
	}

	_, n, err := t.delete(t.root, k)
	if err != nil {
		return err
	}
	t.root = n
	return nil
}

func (t *Trie) delete(parent node, key []byte) (ok bool, n node, err error) {
	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			p.value = nil
			n = p
		} else {
			switch p.children[key[length]].(type) {
			case *branch:
				_, n, err = t.delete(p.children[key[length]], key[length+1:])
				p.children[key[length]] = n
				n = p
				return true, n, nil
			case *leaf:
				p.children[key[length]] = nil
				ok = true
				n = p
				//return true, n, nil
			default:
				return false, p, nil
			}
		}

		bitmap := p.childrenBitmap()
		// if branch has no children, just a value, turn it into a leaf
		if bitmap == 0 && p.value != nil {
			n = &leaf{key: key[:length], value: p.value}
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

			ok = true
		}
	case *leaf:
		ok = true
	case nil:
		// do nothing
	}
	return ok, n, err
}

// lenCommonPrefix returns the length of the common prefix between two keys
func lenCommonPrefix(a, b []byte) int {
	var length, max = 0, len(a)

	if len(a) > len(b) {
		max = len(b)
	}

	for ; length < max; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
