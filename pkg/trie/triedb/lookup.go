// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"fmt"
	"slices"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
)

// Description of what kind of query will be made to the trie.
type Query[Item any] func(data []byte) Item

// Trie lookup helper object.
type TrieLookup[H hash.Hash, Hasher hash.Hasher[H], QueryItem any] struct {
	// db to query from
	db hashdb.HashDB[H]
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

// NewTrieLookup is constructor for [TrieLookup]
func NewTrieLookup[H hash.Hash, Hasher hash.Hasher[H], QueryItem any](
	db hashdb.HashDB[H],
	hash H,
	cache TrieCache[H],
	recorder TrieRecorder,
	query Query[QueryItem],
) TrieLookup[H, Hasher, QueryItem] {
	return TrieLookup[H, Hasher, QueryItem]{
		db:       db,
		hash:     hash,
		cache:    cache,
		recorder: recorder,
		query:    query,
	}
}

func (l *TrieLookup[H, Hasher, QueryItem]) recordAccess(access TrieAccess) {
	if l.recorder != nil {
		l.recorder.Record(access)
	}
}

// Look up the merkle value (hash) of the node that is the closest descendant for the provided
// key.
//
// When the provided key leads to a node, then the merkle value (hash) of that node
// is returned. However, if the key does not lead to a node, then the merkle value
// of the closest descendant is returned. `None` if no such descendant exists.
func (l *TrieLookup[H, Hasher, QueryItem]) LookupFirstDescendant(
	fullKey []byte, nibbleKey nibbles.Nibbles,
) (MerkleValue[H], error) {
	partial := nibbleKey
	hash := l.hash
	var keyNibbles uint

	// this loop iterates through non-inline nodes.
	var depth uint
	for {
		var nodeData []byte

		var getCachedNode = func() (CachedNode[H], error) {
			data := l.db.Get(hash, hashdb.Prefix(nibbleKey.Mid(keyNibbles).Left()))
			if data == nil {
				if depth == 0 {
					return nil, ErrInvalidStateRoot
				} else {
					return nil, ErrIncompleteDB
				}
			}

			reader := bytes.NewReader(data)
			decoded, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			owned, err := NewCachedNodeFromNode[H, Hasher](decoded)
			if err != nil {
				return nil, err
			}
			nodeData = data
			return owned, nil
		}

		var node CachedNode[H]
		if l.cache != nil {
			n, err := l.cache.GetOrInsertNode(hash, getCachedNode)
			if err != nil {
				return nil, err
			}

			l.recordAccess(CachedNodeAccess[H]{Hash: hash, Node: node})
			node = n
		} else {
			n, err := getCachedNode()
			if err != nil {
				return nil, err
			}

			l.recordAccess(EncodedNodeAccess[H]{Hash: hash, EncodedNode: nodeData})
			node = n
		}

		// this loop iterates through all inline children (usually max 1)
		// without incrementing the depth.
		var isInline bool
	inlineLoop:
		for {
			var nextNode CachedNodeHandle
			switch node := node.(type) {
			case LeafCachedNode[H]:
				// The leaf slice can be longer than remainder of the provided key
				// (descendent), but not the other way around.
				if !node.PartialKey.StartsWithNibbles(partial) {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				if partial.Len() != node.PartialKey.Len() {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
				}

				if isInline {
					return NodeMerkleValue(nodeData), nil
				}
				return HashMerkleValue[H]{Hash: hash}, nil
			case BranchCachedNode[H]:
				// Not enough remainder key to continue the search.
				if partial.Len() < node.PartialKey.Len() {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})

					// Branch slice starts with the remainder key, there's nothing to
					// advance.
					if node.PartialKey.StartsWithNibbles(partial) {
						if isInline {
							return NodeMerkleValue(nodeData), nil
						}
						return HashMerkleValue[H]{Hash: hash}, nil
					}
					return nil, nil //nolint:nilnil
				}

				// Partial key is longer or equal than the branch slice.
				// Ensure partial key starts with the branch slice.
				if !partial.StartsWithNibbleSlice(node.PartialKey) {
					l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					return nil, nil //nolint:nilnil
				}

				// Partial key starts with the branch slice.
				if partial.Len() == node.PartialKey.Len() {
					if node.Value != nil {
						l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
					}

					if isInline {
						return NodeMerkleValue(nodeData), nil
					}
					return HashMerkleValue[H]{Hash: hash}, nil
				}

				child := node.Children[partial.At(node.PartialKey.Len())]
				if child != nil {
					partial = partial.Mid(node.PartialKey.Len() + 1)
					keyNibbles += node.PartialKey.Len() + 1
					nextNode = child
				} else {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil //nolint:nilnil
				}
			case EmptyCachedNode[H]:
				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
				return nil, nil //nolint:nilnil
			default:
				panic("unreachable")
			}

			// check if new node data is inline or hash.
			switch nextNode := nextNode.(type) {
			case HashCachedNodeHandle[H]:
				hash = nextNode.Hash
				break inlineLoop
			case InlineCachedNodeHandle[H]:
				node = nextNode.CachedNode
				isInline = true
			default:
				panic("unreachable")
			}
		}

		depth++
	}
}

// Look up the given fullKey.
// If the value is found, it will be passed to the [Query] associated to [TrieLookup].
//
// The given fullKey should be the full key to the data that is requested. This will
// be used when there is a cache to potentially speed up the lookup.
func (l *TrieLookup[H, Hasher, QueryItem]) Lookup(fullKey []byte) (*QueryItem, error) {
	nibbleKey := nibbles.NewNibbles(slices.Clone(fullKey))
	if l.cache != nil {
		return l.lookupWithCache(fullKey, nibbleKey)
	}
	return lookupWithoutCache(l, nibbleKey, fullKey, loadValue[H, QueryItem])
}

// Look up the given key. If the value is found, it will be passed to the [Query] associated to [TrieLookup].
// It uses the given cache to speed-up lookups.
func (l *TrieLookup[H, Hasher, QueryItem]) lookupWithCache(
	fullKey []byte, nibbleKey nibbles.Nibbles,
) (*QueryItem, error) {
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
		data, err := lookupWithCacheInternal[H, Hasher](l, fullKey, nibbleKey, l.cache, loadCachedNodeValue[H])
		if err != nil {
			return nil, err
		}

		l.cache.SetValue(fullKey, data.CachedValue())
		if data != nil {
			return data.Value, nil
		}
		return nil, nil
	}

	var res []byte
	if valueCacheAllowed {
		cachedVal := l.cache.GetValue(fullKey)
		switch cachedVal := cachedVal.(type) {
		case NonExistingCachedValue[H]:
			res = nil
		case ExistingHashCachedValue[H]:
			data, err := loadCachedNodeValue[H](
				// If we only have the hash cached, this can only be a value node.
				// For inline nodes we cache them directly as [ExistingCachedValue].
				NodeCachedNodeValue[H](cachedVal),
				nibbleKey.OriginalDataPrefix(),
				fullKey,
				l.cache,
				l.db,
				l.recorder,
			)
			if err != nil {
				break
			}
			l.cache.SetValue(fullKey, data.CachedValue())
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
		default:
			panic("unreachable")
		}
	} else {
		var err error
		res, err = lookupData()
		if err != nil {
			return nil, err
		}
	}

	if res != nil {
		item := l.query(res)
		return &item, nil
	}
	return nil, nil
}

type loadCachedNodeValueFunc[H hash.Hash, R any] func(
	v CachedNodeValue[H],
	prefix nibbles.Prefix,
	fullKey []byte,
	cache TrieCache[H],
	db hashdb.HashDB[H],
	recorder TrieRecorder,
) (R, error)

// When modifying any logic inside this function, you also need to do the same in
// lookupWithoutCache.
func lookupWithCacheInternal[H hash.Hash, Hasher hash.Hasher[H], R, QueryItem any](
	l *TrieLookup[H, Hasher, QueryItem],
	fullKey []byte,
	nibbleKey nibbles.Nibbles,
	cache TrieCache[H],
	loadValue loadCachedNodeValueFunc[H, R],
) (*R, error) {
	partial := nibbleKey
	hash := l.hash
	var keyNibbles uint

	var depth uint
	for {
		node, err := cache.GetOrInsertNode(hash, func() (CachedNode[H], error) {
			nodeData := l.db.Get(hash, hashdb.Prefix(nibbleKey.Mid(keyNibbles).Left()))
			if nodeData == nil {
				if depth == 0 {
					return nil, ErrInvalidStateRoot
				} else {
					return nil, ErrIncompleteDB
				}
			}
			reader := bytes.NewReader(nodeData)
			decoded, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			return NewCachedNodeFromNode[H, Hasher](decoded)
		})
		if err != nil {
			return nil, err
		}

		l.recordAccess(CachedNodeAccess[H]{Hash: hash, Node: node})

	inlineLoop:
		// this loop iterates through all inline children (usually max 1)
		// without incrementing the depth.
		for {
			var nextNode CachedNodeHandle
			switch node := node.(type) {
			case LeafCachedNode[H]:
				if partial.EqualNibbleSlice(node.PartialKey) {
					value := node.Value
					r, err := loadValue(value, nibbleKey.OriginalDataPrefix(), fullKey, cache, l.db, l.recorder)
					if err != nil {
						return nil, err
					}
					return &r, nil
				} else {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}
			case BranchCachedNode[H]:
				if !partial.StartsWithNibbleSlice(node.PartialKey) {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}

				if partial.Len() == node.PartialKey.Len() {
					if node.Value == nil {
						l.recordAccess(NonExistingNodeAccess{fullKey})
						return nil, nil
					}
					value := node.Value
					r, err := loadValue(value, nibbleKey.OriginalDataPrefix(), fullKey, cache, l.db, l.recorder)
					if err != nil {
						return nil, err
					}
					return &r, nil
				}

				child := node.Children[partial.At(node.PartialKey.Len())]
				if child != nil {
					partial = partial.Mid(node.PartialKey.Len() + 1)
					keyNibbles += node.PartialKey.Len() + 1
					nextNode = child
				} else {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}
			case EmptyCachedNode[H]:
				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
				return nil, nil
			default:
				panic("unreachable")
			}

			// check if new node data is inline or hash.
			switch nextNode := nextNode.(type) {
			case HashCachedNodeHandle[H]:
				hash = nextNode.Hash
				break inlineLoop
			case InlineCachedNodeHandle[H]:
				node = nextNode.CachedNode
			default:
				panic("unreachable")
			}
		}
		depth++
	}
}

type loadValueFunc[H hash.Hash, QueryItem, R any] func(
	v codec.EncodedValue,
	prefix nibbles.Prefix,
	fullKey []byte,
	db hashdb.HashDB[H],
	recorder TrieRecorder,
	query Query[QueryItem],
) (R, error)

// Look up the given key. If the value is found, it will be passed to the given
// function to decode or copy.
//
// When modifying any logic inside this function, you also need to do the same in
// lookupWithCacheInternal.
func lookupWithoutCache[H hash.Hash, Hasher hash.Hasher[H], QueryItem, R any](
	l *TrieLookup[H, Hasher, QueryItem],
	nibbleKey nibbles.Nibbles,
	fullKey []byte,
	loadValue loadValueFunc[H, QueryItem, R],
) (*R, error) {
	partial := nibbleKey
	hash := l.hash
	var keyNibbles uint

	var depth uint
	for {
		nodeData := l.db.Get(hash, hashdb.Prefix(nibbleKey.Mid(keyNibbles).Left()))
		if nodeData == nil {
			if depth == 0 {
				return nil, ErrInvalidStateRoot
			} else {
				return nil, ErrIncompleteDB
			}
		}

		l.recordAccess(EncodedNodeAccess[H]{Hash: hash, EncodedNode: nodeData})

	inlineLoop:
		// this loop iterates through all inline children (usually max 1)
		// without incrementing the depth.
		for {
			reader := bytes.NewReader(nodeData)
			decoded, err := codec.Decode[H](reader)
			if err != nil {
				return nil, err
			}

			var nextNode codec.MerkleValue
			switch decoded := decoded.(type) {
			case codec.Leaf:
				leaf := decoded
				if partial.Equal(leaf.PartialKey) {
					r, err := loadValue(
						leaf.Value,
						nibbleKey.OriginalDataPrefix(),
						fullKey,
						l.db,
						l.recorder,
						l.query,
					)
					if err != nil {
						return nil, err
					}
					return &r, nil
				}
				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
				return nil, nil
			case codec.Branch:
				branch := decoded
				if !partial.StartsWith(branch.PartialKey) {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}

				if partial.Len() == branch.PartialKey.Len() {
					if branch.Value != nil {
						r, err := loadValue(
							branch.Value,
							nibbleKey.OriginalDataPrefix(),
							fullKey,
							l.db,
							l.recorder,
							l.query,
						)
						if err != nil {
							return nil, err
						}
						return &r, nil
					}
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}

				child := branch.Children[partial.At(branch.PartialKey.Len())]
				if child != nil {
					partial = partial.Mid(branch.PartialKey.Len() + 1)
					keyNibbles += branch.PartialKey.Len() + 1
					nextNode = child
				} else {
					l.recordAccess(NonExistingNodeAccess{fullKey})
					return nil, nil
				}
			case codec.Empty:
				l.recordAccess(NonExistingNodeAccess{FullKey: fullKey})
				return nil, nil
			default:
				panic("unreachable")
			}

			// check if new node data is inline or hash.
			switch nextNode := nextNode.(type) {
			case codec.HashedNode[H]:
				hash = nextNode.Hash
				break inlineLoop
			case codec.InlineNode:
				nodeData = nextNode
			default:
				panic("unreachable")
			}
		}
		depth++
	}
}

type valueHash[H hash.Hash] struct {
	Value []byte
	Hash  H
}

func (vh *valueHash[H]) CachedValue() CachedValue[H] {
	// valid case since this is supposed to be optional
	if vh == nil {
		return NonExistingCachedValue[H]{}
	}
	return ExistingCachedValue[H]{
		Hash: vh.Hash,
		Data: vh.Value,
	}
}

// Load the given value.
//
// This will access the db if the value is not already in memory, but then it will put it
// into the given cache as [ValueCachedNode].
//
// Returns the bytes representing the value and its hash.
func loadCachedNodeValue[H hash.Hash](
	v CachedNodeValue[H],
	prefix nibbles.Prefix,
	fullKey []byte,
	cache TrieCache[H],
	db hashdb.HashDB[H],
	recorder TrieRecorder,
) (valueHash[H], error) {
	switch v := v.(type) {
	case InlineCachedNodeValue[H]:
		if recorder != nil {
			recorder.Record(InlineValueAccess{fullKey})
		}
		return valueHash[H](v), nil
	case NodeCachedNodeValue[H]:
		node, err := cache.GetOrInsertNode(v.Hash, func() (CachedNode[H], error) {
			val := db.Get(v.Hash, hashdb.Prefix(prefix))
			if val == nil {
				return nil, ErrIncompleteDB
			}
			return ValueCachedNode[H]{Value: val, Hash: v.Hash}, nil
		})
		if err != nil {
			return valueHash[H]{}, err
		}

		var value []byte
		switch node := node.(type) {
		case ValueCachedNode[H]:
			value = node.Value
		default:
			panic("we are caching a `ValueCachedNode` for a value node hash and this " +
				"cached node has always data attached")
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

// Load the given value.
//
// This will access the db if the value is not already in memory, but then it will put it
// into the given cache as [ValueCachedNode].
func loadValue[H hash.Hash, QueryItem any](
	v codec.EncodedValue,
	prefix nibbles.Prefix,
	fullKey []byte,
	db hashdb.HashDB[H],
	recorder TrieRecorder,
	query Query[QueryItem],
) (qi QueryItem, err error) {
	switch v := v.(type) {
	case codec.InlineValue:
		if recorder != nil {
			recorder.Record(InlineValueAccess{FullKey: fullKey})
		}
		return query(v), nil
	case codec.HashedValue[H]:
		val := db.Get(v.Hash, hashdb.Prefix(prefix))
		if val == nil {
			return qi, fmt.Errorf("%w: %s", ErrIncompleteDB, fullKey)
		}

		if recorder != nil {
			recorder.Record(ValueAccess[H]{
				Hash:    v.Hash,
				Value:   val,
				FullKey: fullKey,
			})
		}
		return query(val), nil
	default:
		panic("unreachable")
	}
}

// Look up the value hash for the given fullKey.
//
// The given fullKey should be the full key to the data that is requested. This will
// be used when there is a cache to potentially speed up the lookup.
func (l *TrieLookup[H, Hasher, QueryItem]) LookupHash(fullKey []byte) (*H, error) {
	nibbleKey := nibbles.NewNibbles(slices.Clone(fullKey))
	if l.cache != nil {
		return l.lookupHashWithCache(fullKey, nibbleKey)
	}
	return lookupWithoutCache(
		l, nibbleKey, fullKey,
		func(
			v codec.EncodedValue,
			_ nibbles.Prefix,
			fullKey []byte,
			_ hashdb.HashDB[H],
			recorder TrieRecorder,
			_ Query[QueryItem],
		) (H, error) {
			switch v := v.(type) {
			case codec.InlineValue:
				if recorder != nil {
					// We can record this as [InlineValueAccess], eventhough we are just
					// returning the hash. This is done to prevent requiring to re-record
					// this key.
					recorder.Record(InlineValueAccess{FullKey: fullKey})
				}
				return (*new(Hasher)).Hash(v), nil
			case codec.HashedValue[H]:
				if recorder != nil {
					recorder.Record(HashAccess{FullKey: fullKey})
				}
				return v.Hash, nil
			default:
				panic("unreachable")
			}
		},
	)

}

// Look up the value hash for the given key.
//
// It uses the given cache to speed-up lookups.
func (l *TrieLookup[H, Hasher, QueryItem]) lookupHashWithCache(
	fullKey []byte,
	nibbleKey nibbles.Nibbles,
) (*H, error) {
	// If there is no recorder, we can always use the value cache.
	var valueCacheAllowed bool = true
	if l.recorder != nil {
		// Check if the recorder has the trie nodes already recorded for this key.
		valueCacheAllowed = l.recorder.TrieNodesRecordedForKey(fullKey) != RecordedNone
	}

	var res *H
	if valueCacheAllowed {
		val := l.cache.GetValue(fullKey)
		if val != nil {
			switch val := val.(type) {
			case ExistingHashCachedValue[H]:
				res = &val.Hash
				return res, nil
			case ExistingCachedValue[H]:
				res = &val.Hash
				return res, nil
			}
		}
	}

	vh, err := lookupWithCacheInternal(l, fullKey, nibbleKey, l.cache, func(
		value CachedNodeValue[H],
		_ nibbles.Prefix,
		fullKey []byte,
		_ TrieCache[H],
		_ hashdb.HashDB[H],
		recorder TrieRecorder,
	) (valueHash[H], error) {
		switch value := value.(type) {
		case InlineCachedNodeValue[H]:
			if recorder != nil {
				// We can record this as [InlineValueAccess], even we are just returning
				// the hash. This is done to prevent requiring to re-record this key.
				recorder.Record(InlineValueAccess{FullKey: fullKey})
			}
			return valueHash[H](value), nil
		case NodeCachedNodeValue[H]:
			if recorder != nil {
				recorder.Record(HashAccess{FullKey: fullKey})
			}
			return valueHash[H]{
				Hash: value.Hash,
			}, nil
		default:
			panic("unreachable")
		}
	})
	if err != nil {
		return nil, err
	}

	if vh != nil {
		if vh.Value != nil {
			l.cache.SetValue(fullKey, vh.CachedValue())
		} else {
			l.cache.SetValue(fullKey, ExistingHashCachedValue[H]{Hash: vh.Hash})
		}
		res = &vh.Hash
	} else {
		l.cache.SetValue(fullKey, NonExistingCachedValue[H]{})
	}

	return res, nil
}
