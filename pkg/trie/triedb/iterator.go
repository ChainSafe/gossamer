// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"fmt"
	"iter"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

type (
	status interface {
		isStatus()
	}
	statusEntering   struct{}
	statusAt         struct{}
	statusAtChild    uint
	statusExiting    struct{}
	statusAftExiting struct{}
)

func (statusEntering) isStatus()   {}
func (statusAt) isStatus()         {}
func (statusAtChild) isStatus()    {}
func (statusExiting) isStatus()    {}
func (statusAftExiting) isStatus() {}

type crumb[H hash.Hash] struct {
	hash *H
	node codec.EncodedNode
	status
}

func (c *crumb[H]) step(fwd bool) {
	switch status := c.status.(type) {
	case statusEntering:
		switch c.node.(type) {
		case codec.Branch:
			c.status = statusAt{}
		default:
			c.status = statusExiting{}
		}
	case statusAt:
		switch c.node.(type) {
		case codec.Branch:
			if fwd {
				c.status = statusAtChild(0)
			} else {
				c.status = statusAtChild(15)
			}
		default:
			c.status = statusExiting{}
		}
	case statusAtChild:
		switch c.node.(type) {
		case codec.Branch:
			if fwd && status < 15 {
				c.status = status + 1
			} else if !fwd && status > 15 {
				c.status = status - 1
			} else {
				c.status = statusExiting{}
			}
		}
	case statusExiting:
		c.status = statusAftExiting{}
	default:
		c.status = statusExiting{}
	}
}

type extractedKey struct {
	Key     []byte
	Padding *byte
	Value   codec.EncodedValue
}

type rawItem[H hash.Hash] struct {
	nibbles.NibbleSlice
	hash *H
	codec.EncodedNode
}

// Extracts the key from the result of a raw item retrieval.
//
// Given a raw item, it extracts the key information, including the key bytes, an optional
// extra nibble (prefix padding), and the node value.
func (ri rawItem[H]) extractKey() *extractedKey {
	prefix := ri.NibbleSlice
	node := ri.EncodedNode

	var value codec.EncodedValue
	switch node := node.(type) {
	case codec.Leaf:
		prefix.AppendPartial(node.PartialKey.RightPartial())
		value = node.Value
	case codec.Branch:
		prefix.AppendPartial(node.PartialKey.RightPartial())
		if node.Value == nil {
			return nil
		}
		value = node.Value
	default:
		return nil
	}

	p := prefix.Prefix()
	return &extractedKey{
		Key:     p.Key,
		Padding: p.Padded,
		Value:   value,
	}
}

type TrieDBRawIterator[H hash.Hash, Hasher hash.Hasher[H]] struct {
	// Forward trail of nodes to visit.
	trail []crumb[H]
	// Forward iteration key nibbles of the current node.
	keyNibbles nibbles.NibbleSlice
	db         *TrieDB[H, Hasher]
}

// Create a new iterator.
func NewTrieDBRawIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher],
) (*TrieDBRawIterator[H, Hasher], error) {
	rootNode, rootHash, err := db.getNodeOrLookup(
		codec.HashedNode[H]{Hash: db.rootHash},
		nibbles.Prefix{},
		true,
	)
	if err != nil {
		return nil, err
	}

	r := TrieDBRawIterator[H, Hasher]{
		db: db,
	}
	r.descend(rootNode, rootHash)
	return &r, nil
}

// Create a new iterator, but limited to a given prefix.
func NewPrefixedTrieDBRawIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte,
) (*TrieDBRawIterator[H, Hasher], error) {
	iter, err := NewTrieDBRawIterator(db)
	if err != nil {
		return nil, err
	}
	err = iter.prefix(prefix, true)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// Create a new iterator, but limited to a given prefix.
// It then do a seek operation from prefixed context (using seek lose
// prefix context by default).
func NewPrefixedTrieDBRawIteratorThenSeek[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte, seek []byte,
) (*TrieDBRawIterator[H, Hasher], error) {
	iter, err := NewTrieDBRawIterator(db)
	if err != nil {
		return nil, err
	}
	err = iter.prefixThenSeek(prefix, seek)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// Descend into a node.
func (ri *TrieDBRawIterator[H, Hasher]) descend(node codec.EncodedNode, nodeHash *H) {
	ri.trail = append(ri.trail, crumb[H]{
		hash:   nodeHash,
		status: statusEntering{},
		node:   node,
	})
}

// Seek a node position at key for iterator.
// Returns true if the cursor is at or after the key, but still shares
// a common prefix with the key, return false if the key do not
// share its prefix with the node.
// This indicates if there is still nodes to iterate over in the case
// where we limit iteration to key as a prefix.
func (ri *TrieDBRawIterator[H, Hasher]) seek(keyBytes []byte, fwd bool) (bool, error) {
	ri.trail = nil
	ri.keyNibbles.Clear()
	key := nibbles.NewNibbles(keyBytes)

	node, nodeHash, err := ri.db.getNodeOrLookup(
		codec.HashedNode[H]{Hash: ri.db.rootHash}, nibbles.Prefix{}, true,
	)
	if err != nil {
		return false, err
	}
	partial := key
	var fullKeyNibbles uint
	for {
		var (
			nextNode     codec.EncodedNode
			nextNodeHash *H
		)

		ri.descend(node, nodeHash)
		crumb := &ri.trail[len(ri.trail)-1]

		switch node := crumb.node.(type) {
		case codec.Leaf:
			if (fwd && node.PartialKey.Compare(partial) == -1) ||
				(!fwd && node.PartialKey.Compare(partial) == 1) {
				crumb.status = statusAftExiting{}
				return false, nil
			}
			return node.PartialKey.StartsWith(partial), nil
		case codec.Branch:
			pk := node.PartialKey
			if !partial.StartsWith(pk) {
				if (fwd && pk.Compare(partial) == -1) ||
					(!fwd && pk.Compare(partial) == 1) {
					crumb.status = statusAftExiting{}
					return false, nil
				}
				return pk.StartsWith(partial), nil
			}

			fullKeyNibbles += pk.Len()
			partial = partial.Mid(pk.Len())

			if partial.Len() == 0 {
				return true, nil
			}

			i := partial.At(0)
			crumb.status = statusAtChild(i)
			ri.keyNibbles.AppendPartial(pk.RightPartial())
			ri.keyNibbles.Push(i)

			if child := node.Children[i]; child != nil {
				fullKeyNibbles += 1
				partial = partial.Mid(1)

				prefix := key.Back(fullKeyNibbles)
				var err error
				nextNode, nextNodeHash, err = ri.db.getNodeOrLookup(child, prefix.Left(), true)
				if err != nil {
					return false, err
				}
			} else {
				return false, nil
			}
		case codec.Empty:
			if !(partial.Len() == 0) {
				crumb.status = statusExiting{}
				return false, nil
			}
			return true, nil
		}

		node = nextNode
		nodeHash = nextNodeHash
	}
}

// Advance the iterator into a prefix, no value out of the prefix will be accessed
// or returned after this operation.
func (ri *TrieDBRawIterator[H, Hasher]) prefix(prefix []byte, fwd bool) error {
	found, err := ri.seek(prefix, fwd)
	if err != nil {
		return err
	}
	if found {
		if len(ri.trail) > 0 {
			popped := ri.trail[len(ri.trail)-1]
			ri.trail = nil
			ri.trail = append(ri.trail, popped)
		}
	} else {
		ri.trail = nil
	}
	return nil
}

// Advance the iterator into a prefix, no value out of the prefix will be accessed
// or returned after this operation.
func (ri *TrieDBRawIterator[H, Hasher]) prefixThenSeek(prefix []byte, seek []byte) error {
	if len(prefix) == 0 {
		// Theres no prefix, so just seek.
		_, err := ri.seek(seek, true)
		if err != nil {
			return err
		}
	}

	if len(seek) == 0 || bytes.Compare(seek, prefix) <= 0 {
		// Either were not supposed to seek anywhere,
		// or were supposed to seek *before* the prefix,
		// so just directly go to the prefix.
		return ri.prefix(prefix, true)
	}

	if !bytes.HasPrefix(seek, prefix) {
		// Were supposed to seek *after* the prefix,
		// so just return an empty iterator.
		ri.trail = nil
		return nil
	}

	found, err := ri.seek(prefix, true)
	if err != nil {
		return err
	}
	if !found {
		// The database doesnt have a key with such a prefix.
		ri.trail = nil
		return nil
	}

	// Now seek forward again
	_, err = ri.seek(seek, true)
	if err != nil {
		return err
	}

	prefixLen := uint(len(prefix)) * nibbles.NibblesPerByte
	var length uint
	// look first prefix in trail
	for i := 0; i < len(ri.trail); i++ {
		switch node := ri.trail[i].node.(type) {
		case codec.Empty:
		case codec.Leaf:
			length += node.PartialKey.Len()
		case codec.Branch:
			length++
			length += node.PartialKey.Len()
		}
		if length > prefixLen {
			ri.trail = ri.trail[i:]
			return nil
		}
	}

	ri.trail = nil
	return nil
}

// Fetches the next raw item.
//
// Must be called with the same db as when the iterator was created.
//
// Specify fwd to indicate the direction of the iteration (true for forward).
func (ri *TrieDBRawIterator[H, Hasher]) nextRawItem(fwd bool) (*rawItem[H], error) {
	for {
		if len(ri.trail) == 0 {
			return nil, nil
		}
		crumb := &ri.trail[len(ri.trail)-1]
		switch status := crumb.status.(type) {
		case statusEntering:
			crumb.step(fwd)
			if fwd {
				return &rawItem[H]{ri.keyNibbles, crumb.hash, crumb.node}, nil
			}
		case statusAftExiting:
			ri.trail = ri.trail[:len(ri.trail)-1]
			if len(ri.trail) > 0 {
				crumb := &ri.trail[len(ri.trail)-1]
				crumb.step(fwd)
			}
		case statusExiting:
			switch node := crumb.node.(type) {
			case codec.Empty, codec.Leaf:
			case codec.Branch:
				ri.keyNibbles.DropLasts(node.PartialKey.Len() + 1)
			default:
				panic("unreachable")
			}
			crumb := &ri.trail[len(ri.trail)-1]
			crumb.step(fwd)
			if !fwd {
				return &rawItem[H]{ri.keyNibbles, crumb.hash, crumb.node}, nil
			}
		case statusAt:
			branch, ok := crumb.node.(codec.Branch)
			if !ok {
				panic("unsupported")
			}
			partial := branch.PartialKey
			ri.keyNibbles.AppendPartial(partial.RightPartial())
			if fwd {
				ri.keyNibbles.Push(0)
			} else {
				ri.keyNibbles.Push(15)
			}
			crumb.step(fwd)
		case statusAtChild:
			i := status
			branch, ok := crumb.node.(codec.Branch)
			if !ok {
				panic("unsupported")
			}
			children := branch.Children
			child := children[i]
			if child != nil {
				ri.keyNibbles.Pop()
				ri.keyNibbles.Push(uint8(i)) //nolint:gosec

				node, nodeHash, err := ri.db.getNodeOrLookup(children[i], ri.keyNibbles.Prefix(), true)
				if err != nil {
					crumb.step(fwd)
					return nil, err
				}
				ri.descend(node, nodeHash)
			} else {
				crumb.step(fwd)
			}
		default:
			panic(fmt.Errorf("unreachable: %T", status))
		}
	}
}

// Fetches the next trie item.
//
// Must be called with the same db as when the iterator was created.
func (ri *TrieDBRawIterator[H, Hasher]) NextItem() (*TrieItem, error) {
	for {
		rawItem, err := ri.nextRawItem(true)
		if err != nil {
			return nil, err
		}
		if rawItem == nil {
			return nil, nil
		}
		extracted := rawItem.extractKey()
		if extracted == nil {
			continue
		}
		key := extracted.Key
		maybeExtraNibble := extracted.Padding
		value := extracted.Value

		if maybeExtraNibble != nil {
			return nil, fmt.Errorf("ValueAtIncompleteKey: %v %v", key, *maybeExtraNibble)
		}

		switch value := value.(type) {
		case codec.HashedValue[H]:
			val, err := ri.db.fetchValue(value.Hash, nibbles.Prefix{Key: key})
			if err != nil {
				return nil, err
			}
			return &TrieItem{key, val}, nil
		case codec.InlineValue:
			return &TrieItem{key, value}, nil
		default:
			panic("unreachable")
		}
	}
}

// Fetches the next key.
//
// Must be called with the same `db` as when the iterator was created.
func (ri *TrieDBRawIterator[H, Hasher]) NextKey() ([]byte, error) {
	for {
		rawItem, err := ri.nextRawItem(true)
		if err != nil {
			return nil, err
		}
		if rawItem == nil {
			return nil, nil
		}
		extracted := rawItem.extractKey()
		if extracted == nil {
			continue
		}
		key := extracted.Key
		maybeExtraNibble := extracted.Padding

		if maybeExtraNibble != nil {
			return nil, fmt.Errorf("ValueAtIncompleteKey: %v %v", key, *maybeExtraNibble)
		}
		return key, nil
	}
}

type TrieItem struct {
	Key   []byte
	Value []byte
}

// A trie iterator that also supports random access (`seek()`).
type TrieIterator[H hash.Hash, Item any] interface {
	Seek(key []byte) error
	Next() (Item, error)
	// Items() iter.Seq2[Item, error]
}

// Iterator for going through all values in the trie in pre-order traversal order.
type TrieDBIterator[H hash.Hash, Hasher hash.Hasher[H]] struct {
	db      *TrieDB[H, Hasher]
	rawIter TrieDBRawIterator[H, Hasher]
}

// Create a new iterator.
func NewTrieDBIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher],
) (*TrieDBIterator[H, Hasher], error) {
	rawIter, err := NewTrieDBRawIterator(db)
	if err != nil {
		return nil, err
	}
	return &TrieDBIterator[H, Hasher]{
		db:      db,
		rawIter: *rawIter,
	}, nil
}

// Create a new iterator, but limited to a given prefix.
func NewPrefixedTrieDBIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte,
) (*TrieDBIterator[H, Hasher], error) {
	rawIter, err := NewPrefixedTrieDBRawIterator(db, prefix)
	if err != nil {
		return nil, err
	}
	return &TrieDBIterator[H, Hasher]{
		db:      db,
		rawIter: *rawIter,
	}, nil
}

// Create a new iterator, but limited to a given prefix.
// It then do a seek operation from prefixed context (using `seek` lose
// prefix context by default).
func NewPrefixedTrieDBIteratorThenSeek[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte, startAt []byte,
) (*TrieDBIterator[H, Hasher], error) {
	rawIter, err := NewPrefixedTrieDBRawIteratorThenSeek(db, prefix, startAt)
	if err != nil {
		return nil, err
	}
	return &TrieDBIterator[H, Hasher]{
		db:      db,
		rawIter: *rawIter,
	}, nil
}

func (tdbi *TrieDBIterator[H, Hasher]) Seek(key []byte) error {
	_, err := tdbi.rawIter.seek(key, true)
	return err
}

func (tdbi *TrieDBIterator[H, Hasher]) Next() (*TrieItem, error) {
	return tdbi.rawIter.NextItem()
}

func (tdbi *TrieDBIterator[H, Hasher]) Items() iter.Seq2[TrieItem, error] {
	return func(yield func(TrieItem, error) bool) {
		for {
			item, err := tdbi.Next()
			if err != nil {
				return
			}
			if item == nil {
				return
			}
			if !yield(*item, err) {
				return
			}
		}
	}
}

// Iterator for going through all of key with values in the trie in pre-order traversal order.
type TrieDBKeyIterator[H hash.Hash, Hasher hash.Hasher[H]] struct {
	rawIter TrieDBRawIterator[H, Hasher]
}

// Create a new iterator.
func NewTrieDBKeyIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher],
) (*TrieDBKeyIterator[H, Hasher], error) {
	rawIter, err := NewTrieDBRawIterator(db)
	if err != nil {
		return nil, err
	}
	return &TrieDBKeyIterator[H, Hasher]{
		rawIter: *rawIter,
	}, nil
}

// Create a new iterator, but limited to a given prefix.
func NewPrefixedTrieDBKeyIterator[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte,
) (*TrieDBKeyIterator[H, Hasher], error) {
	rawIter, err := NewPrefixedTrieDBRawIterator(db, prefix)
	if err != nil {
		return nil, err
	}
	return &TrieDBKeyIterator[H, Hasher]{
		rawIter: *rawIter,
	}, nil
}

// Create a new iterator, but limited to a given prefix.
// It then do a seek operation from prefixed context (using `seek` lose
// prefix context by default).
func NewPrefixedTrieDBKeyIteratorThenSeek[H hash.Hash, Hasher hash.Hasher[H]](
	db *TrieDB[H, Hasher], prefix []byte, startAt []byte,
) (*TrieDBKeyIterator[H, Hasher], error) {
	rawIter, err := NewPrefixedTrieDBRawIteratorThenSeek(db, prefix, startAt)
	if err != nil {
		return nil, err
	}
	return &TrieDBKeyIterator[H, Hasher]{
		rawIter: *rawIter,
	}, nil
}

func (tdbki *TrieDBKeyIterator[H, Hasher]) Seek(key []byte) error {
	_, err := tdbki.rawIter.seek(key, true)
	return err
}

func (tdbki *TrieDBKeyIterator[H, Hasher]) Next() ([]byte, error) {
	return tdbki.rawIter.NextKey()
}

func (tdbki *TrieDBKeyIterator[H, Hasher]) Items() iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		for {
			key, err := tdbki.Next()
			if err != nil {
				return
			}
			if key == nil {
				return
			}
			if !yield(key, err) {
				return
			}
		}
	}
}
