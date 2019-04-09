package trie

import (
	"bytes"
	"errors"
	//"fmt"
	//"github.com/ChainSafe/gossamer/common"
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
			return true, p, nil
		} 

		//fmt.Printf("NEW VALUE KEY %x\n", key[length+1:])
		// NOTE: do we need to switch for leaf/branch ?
		value.(*leaf).key = key[length+1:]

		// otherwise, add value as child of this branch
		p.children[key[length]] = value
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
		// if len(key)-i < len(p.key) || !bytes.Equal(p.key, key[i:i+len(p.key)]) {
		// 	return nil, nil
		// }
		//fmt.Printf("PARENTKEY %x\n", p.key)
		//fmt.Printf("KEY %x\n", key)

		// found the value at this node
		if bytes.Equal(p.key, key) {
			if p.value == nil {
				return nil, nil
			}

			return p.value.(*leaf), nil
		}

		length := lenCommonPrefix(p.key, key)

		//fmt.Printf("KEY AT LEN %x: %x\n", length, key[length])

		// if branch's child at the key is a leaf, return it
		switch v := p.children[key[length]].(type) {
		case *leaf:
			value = &leaf{key: key[length:], value: v.value}
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
	// switch p := parent.(type) {
	// // case *extension:
	// // 	ok, n, err = t.deleteFromExtension(p, prefix, key)
	// case *branch:
	// 	ok, n, err = t.deleteFromBranch(p, prefix, key)
	// case *leaf:
	// 	ok = true
	// case nil:
	// 	// do nothing
	// default:
	// 	err = errors.New("delete error: invalid node")
	// }

	// return ok, n, err

	return true, nil, nil
}

// func (t *Trie) deleteFromExtension(p *extension, prefix, key []byte) (ok bool, n node, err error) {
// 	length := lenCommonPrefix(key, p.key)

// 	// matching key is shorter than parent key, don't replace
// 	if length < len(p.key) {
// 		return false, p, nil
// 	}

// 	// key matches, delete this node
// 	if length == len(key) {
// 		return true, nil, nil
// 	}

// 	// the matching key is longer than the parent's key, so the node to delete
// 	// is somewhere in the extension's subtrie
// 	// try to delete the child from the subtrie
// 	var child node
// 	ok, child, err = t.delete(p.value, append(prefix, key[:len(p.key)]...), key[len(p.key):])
// 	if !ok || err != nil {
// 		return false, p, err
// 	}

// 	// if child is also an extension node, we can combine these two extension nodes into one
// 	switch child := child.(type) {
// 	case *extension:
// 		ok = true
// 		n = &extension{common.Concat(p.key, child.key...), child.value}
// 	default:
// 		ok = true
// 		n = &extension{p.key, child}
// 	}
// 	return ok, n, nil
// }

// func (t *Trie) deleteFromBranch(p *branch, prefix, key []byte) (ok bool, n node, err error) {
// 	ok, n, err = t.delete(p.children[key[0]], append(prefix, key[0]), key[1:])
// 	if !ok || err != nil {
// 		return false, p, err
// 	}

// 	p.children[key[0]] = n

// 	// check how many children are in this branch
// 	// if there are only two children, and we're deleting one, we can turn this branch into an extension
// 	// otherwise, leave it as a branch
// 	// when the loop exits, pos will be the index of the other child (if only 2 children) or -2 if there
// 	// multiple children
// 	pos := -1
// 	for i, child := range &p.children {
// 		if child != nil && pos == -1 {
// 			pos = i
// 		} else if child != nil {
// 			pos = -2
// 			break
// 		}
// 	}

// 	// if there is only one other child, and it's not the branch's value, replace it with an extension
// 	// and attach the branch's key nibble onto the front of the extension key
// 	if pos >= 0 {
// 		if pos != 16 {
// 			child := p.children[pos]
// 			if child, ok := child.(*extension); ok {
// 				k := append([]byte{byte(pos)}, child.key...)
// 				return true, &extension{k, child.value}, nil
// 			}
// 		}
// 		ok = true
// 		n = &extension{[]byte{byte(pos)}, p.children[pos]}
// 		return ok, n, nil
// 	}

// 	return true, p, nil
// }

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