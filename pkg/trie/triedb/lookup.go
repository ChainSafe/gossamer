// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

// Description of what kind of query will be made to the trie.
type Query[Item any] func(data []byte) Item

type TrieLookup[H hash.Hash, Hasher hash.Hasher[H], QueryItem any] struct {
	// db to query from
	db db.DBGetter
	// hash to start at
	hash H
	// optional cache to speed up the db lookups
	cache TrieCache[H]
	// optional recorder for recording trie accesses
	recorder TrieRecorder
	// layout for the trie
	layout trie.TrieLayout
	// query function
	query Query[QueryItem]
}

func NewTrieLookup[H hash.Hash, Hasher hash.Hasher[H], QueryItem any](
	db db.DBGetter,
	hash H,
	cache TrieCache[H],
	recorder TrieRecorder,
) TrieLookup[H, Hasher, QueryItem] {
	return TrieLookup[H, Hasher, QueryItem]{
		db:       db,
		hash:     hash,
		cache:    cache,
		recorder: recorder,
	}
}

func (l *TrieLookup[H, Hasher, QueryItem]) lookupNode(
	nibbleKey nibbles.Nibbles, fullKey []byte,
) (codec.EncodedNode, error) {
	// Start from root node and going downwards
	partialKey := nibbleKey.Clone()
	hash := l.hash
	var keyNibbles uint

	// Iterates through non inlined nodes
	for {
		// Get node from DB
		prefixedKey := append(nibbleKey.Mid(keyNibbles).Left().JoinedBytes(), hash.Bytes()...)
		nodeData, err := l.db.Get(prefixedKey)
		if err != nil {
			return nil, ErrIncompleteDB
		}

		l.recordAccess(EncodedNodeAccess[H]{Hash: hash, EncodedNode: nodeData})

	InlinedChildrenIterator:
		for {
			// Decode node
			reader := bytes.NewReader(nodeData)
			decodedNode, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			var nextNode codec.MerkleValue

			switch n := decodedNode.(type) {
			case codec.Empty:
				return nil, nil //nolint:nilnil
			case codec.Leaf:
				// We are in the node we were looking for
				if partialKey.Equal(n.PartialKey) {
					return n, nil
				}

				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})

				return nil, nil //nolint:nilnil
			case codec.Branch:
				nodePartialKey := n.PartialKey

				// This is unusual but could happen if for some reason one
				// branch has a hashed child node that points to a node that
				// doesn't share the prefix we are expecting
				if !partialKey.StartsWith(nodePartialKey) {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// We are in the node we were looking for
				if partialKey.Equal(n.PartialKey) {
					if n.Value != nil {
						return n, nil
					}

					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// This is not the node we were looking for but it might be in
				// one of its children
				childIdx := int(partialKey.At(nodePartialKey.Len()))
				nextNode = n.Children[childIdx]
				if nextNode == nil {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// Advance the partial key consuming the part we already checked
				partialKey = partialKey.Mid(nodePartialKey.Len() + 1)
				keyNibbles += nodePartialKey.Len() + 1
			}

			// Next node could be inlined or hashed (pointer to a node)
			// https://spec.polkadot.network/chap-state#defn-merkle-value
			switch merkleValue := nextNode.(type) {
			case codec.HashedNode[H]:
				// If it's hashed we set the hash to look for it in next loop
				hash = merkleValue.Hash
				break InlinedChildrenIterator
			case codec.InlineNode:
				// If it is inlined we just need to decode it in the next loop
				nodeData = merkleValue
			}
		}
	}
}

func (l *TrieLookup[H, Hasher, QueryItem]) lookupValue(
	fullKey []byte, keyNibbles nibbles.Nibbles,
) (value []byte, err error) {
	node, err := l.lookupNode(keyNibbles, fullKey)
	if err != nil {
		return nil, err
	}

	// node not found so we return nil
	if node == nil {
		return nil, nil
	}

	if nodeValue := node.GetValue(); nodeValue != nil {
		value, err = l.fetchValue(keyNibbles.OriginalDataPrefix(), fullKey, nodeValue)
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	return nil, nil
}

func (l *TrieLookup[H, Hasher, QueryItem]) lookupValueWithCache(fullKey []byte, keyNibbles nibbles.Nibbles, cache TrieCache[H]) (*QueryItem, error) {
	var trieNodesRecorded *RecordedForKey
	if l.recorder != nil {
		recorded := l.recorder.TrieNodesRecordedForKey(fullKey)
		trieNodesRecorded = &recorded
	}

	var (
		valueCacheAllowed      bool = true
		valueRecordingRequired bool
	)

	if trieNodesRecorded != nil {
		switch *trieNodesRecorded {
		// If we already have the trie nodes recorded up to the value, we are allowed
		// to use the value cache.
		case RecordedValue:
			valueCacheAllowed = true
			valueRecordingRequired = false
		// If we only have recorded the hash, we are allowed to use the value cache, but
		// we may need to have the value recorded.
		case RecordedHash:
			valueCacheAllowed = true
			valueRecordingRequired = true
		// As we don't allow the value cache, the second value can be actually anything.
		case RecordedNone:
			valueCacheAllowed = false
			valueRecordingRequired = true
		}
	}

	var lookupData = func() ([]byte, error) {
		data, err := lookupValueWithCacheInternal[H, Hasher](l, fullKey, keyNibbles, cache, loadValueOwned[H])
		if err != nil {
			return nil, err
		}

		cache.SetValue(fullKey, data.CachedValue())
		return data.Value, nil
	}

	var res []byte
	if valueCacheAllowed {
		cachedVal := cache.GetValue(fullKey)
		switch cachedVal := cachedVal.(type) {
		case NonExistingCachedValue:
			res = nil
		case ExistingHashCachedValue[H]:
			data, err := loadValueOwned[H](
				// If we only have the hash cached, this can only be a value node.
				// For inline nodes we cache them directly as `CachedValue::Existing`.
				codec.ValueOwned(codec.ValueOwnedNode[H]{Hash: cachedVal.Hash}),
				keyNibbles, // nibble_key.original_data_as_prefix(),
				fullKey,
				cache,
				l.db,
				l.recorder,
			)
			if err != nil {
				break
			}
			cache.SetValue(fullKey, data.CachedValue())
			res = data.Value
		case ExistingCachedValue[H]:
			data := cachedVal.Data
			hash := cachedVal.Hash
			if data != nil {
				// inline is either when no limit defined or when content
				// is less than the limit.
				isInline := l.layout.MaxInlineValue() > len(data)
				if valueRecordingRequired && !isInline {
					// As a value is only raw data, we can directly record it.
					l.recordAccess(ValueAccess[H]{
						Hash:    hash,
						Value:   data,
						FullKey: fullKey,
					})
				}
				res = data
			} else {
				var err error
				res, err = lookupData()
				if err != nil {
					return nil, err
				}
			}
		case nil:
			var err error
			res, err = lookupData()
			if err != nil {
				return nil, err
			}
		}
	}

	if res != nil {
		item := l.query(res)
		return &item, nil
	}
	return nil, nil
}

type loadValueFunc[H hash.Hash, R any] func(
	v codec.ValueOwned,
	prefix nibbles.Nibbles,
	fullKey []byte,
	cache TrieCache[H],
	db db.DBGetter,
	recorder TrieRecorder,
) (R, error)

func lookupValueWithCacheInternal[H hash.Hash, Hasher hash.Hasher[H], R, QueryItem any](
	l *TrieLookup[H, Hasher, QueryItem],
	fullKey []byte,
	nibbleKey nibbles.Nibbles,
	cache TrieCache[H],
	loadValue loadValueFunc[H, R],
) (*R, error) {
	partial := nibbleKey
	hash := l.hash
	var keyNibbles uint

	var depth uint
	for {
		node, err := cache.GetOrInsertNode(hash, func() (codec.NodeOwned, error) {
			prefixedKey := append(nibbleKey.Mid(keyNibbles).Left().JoinedBytes(), hash.Bytes()...)
			nodeData, err := l.db.Get(prefixedKey)
			if err != nil {
				if depth == 0 {
					return nil, fmt.Errorf("invalid state root")
				} else {
					return nil, fmt.Errorf("incomplete database")
				}
			}
			reader := bytes.NewReader(nodeData)
			decoded, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			return codec.NodeOwnedFromNode[H, Hasher](decoded)
		})
		if err != nil {
			return nil, err
		}

		l.recordAccess(NodeOwnedAccess[H]{Hash: hash, Node: node})

	inlineLoop:
		// this loop iterates through all inline children (usually max 1)
		// without incrementing the depth.
		for {
			var nextNode codec.NodeHandleOwned
			switch node := node.(type) {
			case codec.NodeOwnedLeaf[H]:
				if partial.Equal(node.PartialKey) {
					value := node.Value
					r, err := loadValue(value, nibbleKey, fullKey, cache, l.db, l.recorder)
					if err != nil {
						return nil, err
					}
					return &r, nil
				} else {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}
			case codec.NodeOwnedBranch[H]:
				if partial.Len() == 0 {
					value := node.Value
					r, err := loadValue(value, nibbleKey, fullKey, cache, l.db, l.recorder)
					if err != nil {
						return nil, err
					}
					return &r, nil
				} else {
					child := node.Children[partial.At(0)]
					if child != nil {
						partial = partial.Mid(1)
						keyNibbles += 1
						nextNode = child
					} else {
						l.recordAccess(NonExistingNodeAccess{fullKey})
						return nil, nil
					}
				}
			case codec.NodeOwnedEmpty:
				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
			default:
				panic("unreachable")
			}

			// check if new node data is inline or hash.
			switch nextNode := nextNode.(type) {
			case codec.NodeHandleOwnedHash[H]:
				hash = nextNode.Hash
				break inlineLoop
			case codec.NodeHandleOwnedInline[H]:
				node = nextNode.NodeOwned
			default:
				panic("unreachable")
			}
		}
		depth++
	}
}

type valueHash[H any] struct {
	Value []byte
	Hash  H
}

func (vh *valueHash[H]) CachedValue() CachedValue {
	// valid case since this is supposed to be optional
	if vh == nil {
		return NonExistingCachedValue{}
	}
	return ExistingCachedValue[H]{
		Hash: vh.Hash,
		Data: vh.Value,
	}
}

// Load the given value.
//
// This will access the `db` if the value is not already in memory, but then it will put it
// into the given `cache` as `NodeOwned::Value`.
//
// Returns the bytes representing the value and its hash.
func loadValueOwned[H hash.Hash](
	v codec.ValueOwned,
	prefix nibbles.Nibbles,
	fullKey []byte,
	cache TrieCache[H],
	db db.DBGetter,
	recorder TrieRecorder) (valueHash[H], error) {

	switch v := v.(type) {
	case codec.ValueOwnedInline[H]:
		if recorder != nil {
			recorder.Record(InlineValueAccess{fullKey})
		}
		return valueHash[H]{
			Value: v.Value,
			Hash:  v.Hash,
		}, nil
	case codec.ValueOwnedNode[H]:
		node, err := cache.GetOrInsertNode(v.Hash, func() (codec.NodeOwned, error) {
			prefixedKey := append(prefix.Left().JoinedBytes(), v.Hash.Bytes()...)
			val, err := db.Get(prefixedKey)
			if err != nil {
				return nil, err
			}
			return codec.NodeOwnedValue[H]{Value: val, Hash: v.Hash}, nil
		})
		if err != nil {
			return valueHash[H]{}, err
		}

		var value []byte
		switch node := node.(type) {
		case codec.NodeOwnedValue[H]:
			value = node.Value
		default:
			panic("we are caching a `NodeOwnedValue` for a value node hash and this cached node has always data attached")
		}

		if recorder != nil {
			recorder.Record(ValueAccess[H]{
				Hash:    v.Hash,
				Value:   value,
				FullKey: fullKey,
			})
		}

		return valueHash[H]{
			Value: value,
			Hash:  v.Hash,
		}, nil

	default:
		panic("unreachable")
	}
}

// fetchValue gets the value from the node, if it is inlined we can return it
// directly. But if it is hashed (V1) we have to look up for its value in the DB
func (l *TrieLookup[H, Hasher, QueryItem]) fetchValue(
	prefix nibbles.Prefix, fullKey []byte, value codec.EncodedValue,
) ([]byte, error) {
	switch v := value.(type) {
	case codec.InlineValue:
		l.recordAccess(InlineValueAccess{FullKey: fullKey})
		return v, nil
	case codec.HashedValue[H]:
		prefixedKey := bytes.Join([][]byte{prefix.JoinedBytes(), v.Hash.Bytes()}, nil)

		nodeData, err := l.db.Get(prefixedKey)
		if err != nil {
			return nil, ErrIncompleteDB
		}

		l.recordAccess(ValueAccess[H]{Hash: v.Hash, FullKey: fullKey, Value: nodeData})

		return nodeData, nil
	default:
		panic("unreachable")
	}
}

func (l *TrieLookup[H, Hasher, QueryItem]) recordAccess(access TrieAccess) {
	if l.recorder != nil {
		l.recorder.Record(access)
	}
}
