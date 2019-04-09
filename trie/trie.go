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
	if len(key) == 0 {
		return errors.New("cannot put nil key")
	}

	if err := t.tryPut(key, value); err != nil {
		return err
	}

	return nil
}

func (t *Trie) tryPut(key, value []byte) (err error) {
	k := keyToHex(key)
	var n node

	if len(value) > 0 {
		_, n, err = t.insert(t.root, nil, k, &leaf{key: nil, value: value})
	} else {
		_, n, err = t.delete(t.root, nil, k)
	}

	if err != nil {
		return err
	}

	t.root = n
	return nil
}

// TryPut attempts to insert a key with value into the trie
func (t *Trie) insert(parent node, prefix, key []byte, value node) (ok bool, n node, err error) {
	if len(key) == 0 {
		if v, ok := parent.(*leaf); ok {
			return !bytes.Equal(v.value, value.(*leaf).value), value, nil
		}
		return true, value, nil
	}

	switch p := parent.(type) {
	case *branch:
		ok, n, err = t.updateBranch(p, prefix, key, value)
	case nil:
		switch v := value.(type) {
		case *branch:
			n = value
			ok = true
		case *leaf:
			n = &leaf{key, v.value}
		}
	case *leaf:
		br := new(branch)
		length := lenCommonPrefix(key, p.key)
		br.key = key[:length]
		if length == len(p.key) {
			br.value = p.value
			br.children[key[length]] = value
		} else {
			br.children[p.key[length]] = p
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
func (t *Trie) updateBranch(p *branch, prefix, key []byte, value node) (ok bool, n node, err error) {
	length := lenCommonPrefix(key, p.key)

	// whole parent key matches except last nibble
	if length == len(p.key) {
		// if value node has same key as this branch, then update the value at this branch
		if bytes.Equal(key, p.key) {
			value.(*leaf).key = nil
			p.value = value
		} else {
			value.(*leaf).key = key[length+1:]

			// otherwise, add value as child of this branch
			p.children[key[length]] = value
		}
		
		return true, p, nil		
	}

	// otherwise, we need to branch out at the point where the keys diverge
	br := new(branch)
	br.key = key[:length]

	_, br.children[p.key[length]], err = t.insert(nil, append(prefix, p.key[:length+1]...), p.key[length+1:], p)
	if err != nil {
		return false, nil, err
	}

	_, br.children[key[length]], err = t.insert(nil, append(prefix, key[:length+1]...), key[length+1:], value)
	if err != nil {
		return false, nil, err
	}

	return true, br, nil
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
// leaf includes both partial key and value
func (t *Trie) getLeaf(key []byte) (value *leaf, err error) {
	l, err := t.tryGet(key)
	return l, err
}

func (t *Trie) tryGet(key []byte) (value *leaf, err error) {
	k := keyToHex(key)
	value, err = t.retrieve(t.root, k)
	return value, err
}

func (t *Trie) retrieve(parent node, key []byte) (value *leaf, err error) {
	switch p := parent.(type) {
	case *branch:
		// found the value at this node
		if bytes.Equal(p.key, key) {
			if p.value == nil {
				return nil, nil
			}

			switch v := p.value.(type) {
			case *leaf:
				return v, nil
			case []byte:
				return &leaf{key: hexToKey(key), value: v}, nil
			default:
				return nil, errors.New("get error: invalid branch value")
			}
		}

		length := lenCommonPrefix(p.key, key)

		// if branch's child at the key is a leaf, return it
		switch v := p.children[key[length]].(type) {
		case *leaf:
			value = &leaf{key: hexToKey(key[length:]), value: v.value}
		default:
			value, err = t.retrieve(p.children[key[length]], key[length:])
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
	k := keyToHex(key)
	_, n, err := t.delete(t.root, nil, k)
	if err != nil {
		return err
	}
	t.root = n
	return nil
}

func (t *Trie) delete(parent node, prefix, key []byte) (ok bool, n node, err error) {
	return true, nil, nil
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
