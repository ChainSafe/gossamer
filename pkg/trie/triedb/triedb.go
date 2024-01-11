package triedb

import (
	"errors"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	node_types "github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
	"github.com/gammazero/deque"
)

type HashOut = node_types.HashOut
type Children = [nibble.NibbleLength]NodeHandle

// Node types in the Trie
type Node[H HashOut] interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	Empty struct{}
	// NodeLeaf represents a leaf node
	Leaf struct {
		encoded nibble.NibbleSlice
		value   TrieValue
	}
	// NodeNibbledBranch represents a branch node
	NibbledBranch struct {
		encoded  nibble.NibbleSlice
		children Children
		value    TrieValue
	}
)

func (n Empty) Type() string         { return "Empty" }
func (n Leaf) Type() string          { return "Leaf" }
func (n NibbledBranch) Type() string { return "NibbledBranch" }

type StorageHandle = uint
type NibbleFullKey = nibble.NibbleSlice

type Stored[H HashOut] interface {
	Type() string
}

type (
	StoredNew[H HashOut] struct {
		Node Node[H]
	}
	StoredCached[H HashOut] struct {
		Hash H
		Node Node[H]
	}
)

func (s StoredNew[Out]) Type() string    { return "New" }
func (s StoredCached[Out]) Type() string { return "Cached" }

type NodeHandle interface {
	Type() string
}
type (
	Hash[H HashOut] struct {
		Value H
	}
	InMemory struct {
		Value StorageHandle
	}
)

func (h Hash[H]) Type() string  { return "Hash" }
func (h InMemory) Type() string { return "InMemory" }

// Compact storage for tree nodes
type NodeStorage[H HashOut] struct {
	nodes       []Stored[H]
	freeIndices deque.Deque[uint]
}

func NewEmptyNodeStorage[H HashOut]() *NodeStorage[H] {
	return &NodeStorage[H]{
		nodes: make([]Stored[H], 0),
	}
}

func (ns *NodeStorage[H]) alloc(stored Stored[H]) StorageHandle {
	if ns.freeIndices.Len() > 0 {
		idx := ns.freeIndices.PopFront()
		ns.nodes[idx] = stored
		return idx
	}

	ns.nodes = append(ns.nodes, stored)
	return uint(len(ns.nodes) - 1)
}

func (ns *NodeStorage[H]) destroy(handle StorageHandle) Stored[H] {
	idx := handle

	ns.freeIndices.PushBack(idx)
	ns.nodes[idx] = StoredNew[H]{Empty{}}
	return ns.nodes[idx]
}

type TrieDB[Out HashOut] struct {
	storage    NodeStorage[Out]
	db         hashdb.HashDB[Out, DBValue]
	root       Out
	rootHandle NodeHandle
	deathRow   map[Out]nibble.Prefix
	cache      TrieCache[Out]
	recorder   TrieRecorder[Out]
	layout     TrieLayout[Out]
}

func NewTrieDB[H HashOut](
	db hashdb.HashDB[H, DBValue],
	root H,
	cache TrieCache[H],
	recorder TrieRecorder[H],
) *TrieDB[H] {
	return &TrieDB[H]{
		db:       db,
		root:     root,
		cache:    cache,
		recorder: recorder,
	}
}

type RemoveAtResult struct {
	handle  StorageHandle
	changed bool
}

func (tdb *TrieDB[H]) record(
	access TrieAccess[H],
) {
	if tdb.recorder != nil {
		tdb.recorder.record(access)
	}
}

func (tdb *TrieDB[H]) lookupAndCache(
	hash H,
	key nibble.Prefix,
) (StorageHandle, error) {
	var node Node[H]

	nodeFromCache := tdb.cache.GetNode(hash)
	if nodeFromCache != nil {
		tdb.record(TrieAccessNodeOwned[H]{hash: hash, nodeOwned: *nodeFromCache})
		node = NodeFromNodeOwned(*nodeFromCache, tdb.storage)
	} else {
		nodeEncoded := tdb.db.Get(hash, key)
		if nodeEncoded == nil {
			return 0, ErrIncompleteDB
		}

		tdb.record(TrieAccessEncodedNode[H]{hash: hash, encodedNode: *nodeEncoded})
	}

	return tdb.storage.alloc(StoredCached[H]{Node: node, Hash: hash}), nil
}

type PostInspectAction interface {
	Type() string
}

type (
	PostInspectActionReplace[H HashOut] struct {
		node Node[H]
	}
	PostInspectActionRestore[H HashOut] struct {
		node Node[H]
	}
	PostInspectActionDelete struct{}
)

func (r PostInspectActionReplace[H]) Type() string { return "Replace" }
func (r PostInspectActionRestore[H]) Type() string { return "Restore" }
func (r PostInspectActionDelete) Type() string     { return "Delete" }

type InspectResult[H HashOut] struct {
	stored  Stored[H]
	changed bool
}

func (tdb *TrieDB[H]) inspect(
	stored Stored[H],
	key NibbleFullKey,
	inspector func(
		node Node[H],
		key NibbleFullKey,
	) (PostInspectAction, error),
) (*InspectResult[H], error) {
	var result InspectResult[H]

	switch s := stored.(type) {
	case StoredNew[H]:
		execution, err := inspector(s.Node, key)
		if err != nil {
			return nil, err
		}
		switch action := execution.(type) {
		case PostInspectActionRestore[H]:
			result = InspectResult[H]{StoredNew[H]{action.node}, false}
		case PostInspectActionReplace[H]:
			result = InspectResult[H]{StoredNew[H]{action.node}, true}
		case PostInspectActionDelete:
			return nil, nil
		}
	case StoredCached[H]:
		execution, err := inspector(s.Node, key)
		if err != nil {
			return nil, err
		}

		switch action := execution.(type) {
		case PostInspectActionRestore[H]:
			result = InspectResult[H]{StoredCached[H]{s.Hash, action.node}, false}
		case PostInspectActionReplace[H]:
			tdb.deathRow[s.Hash] = key.Left()
			result = InspectResult[H]{StoredNew[H]{action.node}, true}
		case PostInspectActionDelete:
			tdb.deathRow[s.Hash] = key.Left()
			return nil, nil
		}
	}

	return &result, nil
}

// Given a node which may be in an invalid state, fix it such that it is then in a valid
// state.
//
// invalid state means:
// - Branch node where there is only a single entry;
func (tdb *TrieDB[H]) fix(node Node[H], key nibble.NibbleSlice) (Node[H], error) {
	switch n := node.(type) {
	case NibbledBranch:
		var ix byte
		usedIndexes := 0
		for i := 0; i < 16; i++ {
			if n.children[i] != nil {
				ix = byte(i)
				usedIndexes += 1
			}
		}
		if n.value == nil {
			if usedIndexes == 0 {
				panic("Branch with no subvalues. Something went wrong.")
			}
			if usedIndexes == 1 {
				// only one onward node. use child instead
				child := n.children[ix]
				if child == nil {
					return nil, errors.New("used_index only set if occupied")
				}
				key2 := key.Clone()
				offset := uint(len(n.encoded.Data()))*nibble.NibblePerByte - n.encoded.Offset()
				key2.Advance(offset)
				prefix := key2.Left()

				start := prefix.PartialKey
				var allocStart *[]byte
				var prefixEnd *byte
				if prefix.PaddedByte == nil {
					allocStart = nil
					pEnd := nibble.PushAtLeft(0, ix, 0)
					prefixEnd = &pEnd
				} else {
					so := prefix.PartialKey
					so = append(so, nibble.PadLeft(*prefix.PaddedByte)|ix)
					allocStart = &so
					prefixEnd = nil
				}
				childPrefix := nibble.Prefix{}
				if allocStart != nil {
					childPrefix.PartialKey = *allocStart
				} else {
					childPrefix.PartialKey = start
				}
				childPrefix.PaddedByte = prefixEnd
				var stored Stored[H]

				switch c := child.(type) {
				case InMemory:
					stored = tdb.storage.destroy(c.Value)
				case Hash[H]:
					handle, err := tdb.lookupAndCache(c.Value, childPrefix)
					if err != nil {
						return nil, err
					}
					stored = tdb.storage.destroy(handle)
				}

				var childNode Node[H]

				switch s := stored.(type) {
				case StoredNew[H]:
					childNode = s.Node
				case StoredCached[H]:
					tdb.deathRow[s.Hash] = childPrefix
					childNode = s.Node
				}

				switch cn := childNode.(type) {
				case Leaf:
					encNibble := n.encoded.Clone()
					end := nibble.NewNibbleSliceWithPadding([]byte{ix}, nibble.NibblePerByte-1)
					nibble.CombineKeys(encNibble, *end)

					end = nibble.NewNibbleSliceWithPadding(cn.encoded.Data(), cn.encoded.Offset())
					nibble.CombineKeys(encNibble, *end)
					return Leaf{*encNibble, cn.value}, nil
				case NibbledBranch:
					encNibble := n.encoded.Clone()
					end := nibble.NewNibbleSliceWithPadding([]byte{ix}, nibble.NibblePerByte-1)
					nibble.CombineKeys(encNibble, *end)
					end = nibble.NewNibbleSliceWithPadding(cn.encoded.Data(), cn.encoded.Offset())
					nibble.CombineKeys(encNibble, *end)
					return NibbledBranch{*encNibble, cn.children, cn.value}, nil
				default:
					panic("Unreachable")
				}
			}
		}
		if n.value != nil {
			if usedIndexes == 0 {
				// Make a lift
				// Fixing branch -> Leaf
				return Leaf{n.encoded, n.value}, nil
			}
		}
		// All is well, restoring branch
		return NibbledBranch{n.encoded, n.children, n.value}, nil
	default:
		return node, nil
	}
}

func (tdb *TrieDB[H]) removeInspector(
	node Node[H],
	key NibbleFullKey,
	oldVal *TrieValue,
) (PostInspectAction, error) {
	switch n := node.(type) {
	case Empty:
		return PostInspectActionDelete{}, nil
	case Leaf:
		existingKey := n.encoded
		if key.Eq(existingKey) {
			// We found the node we want to delete, so we are going to remove it
			keyVal := key.Clone()
			keyVal.Advance(existingKey.Len())
			tdb.replaceOldValue(oldVal, n.value, keyVal.Left())
			return PostInspectActionDelete{}, nil
		} else {
			// Leaf the node alone, restoring leaf wrong partial
			return PostInspectActionRestore[H]{
				Leaf{n.encoded, n.value},
			}, nil
		}
	case NibbledBranch:
		if key.IsEmpty() {
			if n.value == nil {
				return PostInspectActionRestore[H]{NibbledBranch{n.encoded, n.children, nil}}, nil
			}
			tdb.replaceOldValue(oldVal, n.value, key.Left())
			fixedNode, err := tdb.fix(NibbledBranch{n.encoded, n.children, nil}, key)
			if err != nil {
				return nil, err
			}
			return PostInspectActionReplace[H]{fixedNode}, nil
		}
		common := n.encoded.CommonPrefix(key)
		existingLength := n.encoded.Len()

		if common == existingLength && common == key.Len() {
			// Replace val
			if n.value != nil {
				keyVal := key.Clone()
				keyVal.Advance(existingLength)
				tdb.replaceOldValue(oldVal, n.value, keyVal.Left())
				fixedNode, err := tdb.fix(NibbledBranch{n.encoded, n.children, nil}, key)
				if err != nil {
					return nil, err
				}
				return PostInspectActionReplace[H]{fixedNode}, nil
			}
			return PostInspectActionRestore[H]{NibbledBranch{n.encoded, n.children, nil}}, nil
		} else if common < existingLength {
			// Nothing to do here
			return PostInspectActionRestore[H]{NibbledBranch{n.encoded, n.children, n.value}}, nil
		} else {
			// common == existing_length && common < partial.len() : check children
			idx := key.At(common)

			child := n.children[idx]
			if child != nil {
				key.Advance(common + 1)
				res, err := tdb.removeAt(child, key, oldVal)
				if err != nil {
					return nil, err
				}

				if res != nil {
					n.children[idx] = InMemory{res.handle}
					branch := NibbledBranch{n.encoded, n.children, n.value}
					if res.changed {
						return PostInspectActionReplace[H]{branch}, nil
					} else {
						return PostInspectActionRestore[H]{branch}, nil
					}
				}
				fixedNode, err := tdb.fix(NibbledBranch{n.encoded, n.children, n.value}, key)
				if err != nil {
					return nil, err
				}
				return PostInspectActionReplace[H]{fixedNode}, nil
			}
			return PostInspectActionRestore[H]{NibbledBranch{n.encoded, n.children, n.value}}, nil
		}
	default:
		panic("Invalid node type")
	}
}

func (tdb *TrieDB[H]) insertInspector(
	node Node[H],
	key NibbleFullKey,
	value []byte,
	oldVal *TrieValue,
) (PostInspectAction, error) {
	partial := key

	switch n := node.(type) {
	case Empty:
		value := NewTrieValueFromBytes[H](value, tdb.layout.MaxInlineValue())
		return PostInspectActionReplace[H]{Leaf{partial, value}}, nil
	case Leaf:
		existingKey := n.encoded
		common := partial.CommonPrefix(existingKey)
		if common == existingKey.Len() && common == partial.Len() {
			// Equivalent leaf replace
			value := NewTrieValueFromBytes[H](value, tdb.layout.MaxInlineValue())
			unchanged := n.value == value
			keyVal := key.Clone()
			keyVal.Advance(existingKey.Len())
			tdb.replaceOldValue(oldVal, n.value, keyVal.Left())
			if unchanged {
				return PostInspectActionRestore[H]{Leaf{n.encoded, n.value}}, nil
			}
			return PostInspectActionReplace[H]{Leaf{n.encoded, n.value}}, nil
		} else if common < existingKey.Len() {
			// one of us isn't empty: transmute to branch here
			children := Children{}
			idx := existingKey.At(common)
			newLeaf := Leaf{*existingKey.Mid(common + 1), n.value}
			children[idx] = InMemory{tdb.storage.alloc(StoredNew[H]{newLeaf})}
			branch := NibbledBranch{partial.ToStoredRange(common), children, nil}

			branchAction, err := tdb.insertInspector(branch, key, value, oldVal)
			if err != nil {
				return nil, err
			}
			return PostInspectActionReplace[H]{branchAction}, nil
		} else {
			branch := NibbledBranch{
				existingKey.ToStored(),
				Children{},
				n.value,
			}

			branchAction, err := tdb.insertInspector(branch, key, value, oldVal)
			if err != nil {
				return nil, err
			}
			return PostInspectActionReplace[H]{branchAction}, nil
		}
	case NibbledBranch:
		existingKey := n.encoded
		common := partial.CommonPrefix(existingKey)
		if common == existingKey.Len() && common == partial.Len() {
			value := NewTrieValueFromBytes[H](value, tdb.layout.MaxInlineValue())
			unchanged := n.value == value
			branch := NibbledBranch{n.encoded, n.children, n.value}

			keyVal := key.Clone()
			keyVal.Advance(existingKey.Len())
			tdb.replaceOldValue(oldVal, n.value, keyVal.Left())

			if unchanged {
				return PostInspectActionRestore[H]{branch}, nil
			}
			return PostInspectActionReplace[H]{branch}, nil
		} else if common < existingKey.Len() {
			nbranchPartial := existingKey.Mid(common + 1).ToStored()
			low := NibbledBranch{nbranchPartial, n.children, n.value}
			ix := existingKey.At(common)
			children := Children{}
			allocStorage := tdb.storage.alloc(StoredNew[H]{low})

			children[ix] = InMemory{allocStorage}
			value := NewTrieValueFromBytes[H](value, tdb.layout.MaxInlineValue())

			if partial.Len()-common == 0 {
				return PostInspectActionReplace[H]{
					NibbledBranch{
						existingKey.ToStoredRange(common),
						children,
						value,
					},
				}, nil
			} else {
				ix := partial.At(common)
				storedLeaf := Leaf{partial.Mid(common + 1).ToStored(), value}
				leaf := tdb.storage.alloc(StoredNew[H]{storedLeaf})

				children[ix] = InMemory{leaf}
				return PostInspectActionReplace[H]{
					NibbledBranch{
						existingKey.ToStoredRange(common),
						children,
						nil,
					},
				}, nil
			}
		} else {
			// Append after common == existing_key and partial > common
			idx := partial.At(common)
			key.Advance(common + 1)
			child := n.children[idx]
			if child != nil {
				// Original had something there. recurse down into it.
				res, err := tdb.insertAt(child, key, value, oldVal)
				if err != nil {
					return nil, err
				}
				n.children[idx] = InMemory{res.handle}
				if !res.changed {
					// The new node we composed didn't change.
					// It means our branch is untouched too.
					nBranch := NibbledBranch{n.encoded.ToStored(), n.children, n.value}
					return PostInspectActionRestore[H]{nBranch}, nil
				}
			} else {
				// Original had nothing there. compose a leaf.
				value := NewTrieValueFromBytes[H](value, tdb.layout.MaxInlineValue())
				leaf := tdb.storage.alloc(StoredNew[H]{Leaf{key.ToStored(), value}})
				n.children[idx] = InMemory{leaf}
			}
			return PostInspectActionReplace[H]{NibbledBranch{existingKey.ToStored(), n.children, n.value}}, nil
		}
	default:
		panic("Invalid node type")
	}
}

func (tdb *TrieDB[H]) replaceOldValue(
	oldVal *TrieValue,
	storedValue TrieValue,
	prefix nibble.Prefix,
) {
	switch n := storedValue.(type) {
	case NewNodeTrie[H]:
		if n.Hash != nil {
			tdb.deathRow[*n.Hash] = prefix
		}
	case NodeTrieValue[H]:
		tdb.deathRow[n.Hash] = prefix
	}
	*oldVal = storedValue
}

// Removes a node from the trie based on key
func (tdb *TrieDB[H]) removeAt(
	handle NodeHandle,
	key NibbleFullKey,
	oldVal *TrieValue,
) (*RemoveAtResult, error) {
	var stored Stored[H]

	switch h := handle.(type) {
	case InMemory:
		stored = tdb.storage.destroy(h.Value)
	case Hash[H]:
		fromCache, err := tdb.lookupAndCache(h.Value, key.Left())
		if err != nil {
			return nil, err
		}
		stored = tdb.storage.destroy(fromCache)
	}

	res, err := tdb.inspect(stored, key, func(node Node[H], key NibbleFullKey) (PostInspectAction, error) {
		return tdb.removeInspector(node, key, oldVal)
	})

	if err != nil {
		return nil, err
	}

	return &RemoveAtResult{
		tdb.storage.alloc(res.stored),
		res.changed,
	}, nil
}

func (tdb *TrieDB[H]) Remove(key []byte) (*TrieValue, error) {
	rootHandle := tdb.rootHandle
	keySlice := nibble.NewNibbleSlice(key)
	var oldVal *TrieValue

	res, err := tdb.removeAt(rootHandle, *keySlice, oldVal)

	if err != nil {
		return nil, err
	}

	if res != nil {
		tdb.rootHandle = InMemory{res.handle}
	} else {
		tdb.rootHandle = Hash[H]{tdb.layout.Codec().HashedNullNode()}
		tdb.root = tdb.layout.Codec().HashedNullNode()
	}

	return oldVal, nil
}

type InsertAtResult struct {
	handle  StorageHandle
	changed bool
}

// / Insert a key-value pair into the trie, creating new nodes if necessary.
func (tdb *TrieDB[H]) insertAt(
	handle NodeHandle,
	key NibbleFullKey,
	value []byte,
	oldVal *TrieValue,
) (InsertAtResult, error) {
	var storageHandle StorageHandle
	var err error

	switch h := handle.(type) {
	case InMemory:
		storageHandle = h.Value
	case Hash[H]:
		storageHandle, err = tdb.lookupAndCache(h.Value, key.Left())
		if err != nil {
			return InsertAtResult{}, err
		}
	}

	stored := tdb.storage.destroy(storageHandle)

	res, err := tdb.inspect(stored, key, func(node Node[H], key NibbleFullKey) (PostInspectAction, error) {
		return tdb.insertInspector(node, key, value, oldVal)
	})

	if err != nil {
		return InsertAtResult{}, err
	}

	return InsertAtResult{
		tdb.storage.alloc(res.stored),
		res.changed,
	}, nil
}

func (tdb *TrieDB[H]) Insert(key []byte, value []byte) (*TrieValue, error) {
	if !tdb.layout.AllowEmpty() && len(value) == 0 {
		return tdb.Remove(key)
	}

	var oldVal *TrieValue

	insertRes, err := tdb.insertAt(tdb.rootHandle, *nibble.NewNibbleSlice(key), value, oldVal)

	if err != nil {
		return nil, err
	}

	tdb.rootHandle = InMemory{insertRes.handle}

	return oldVal, nil
}

func inlineOrHashOwned[H HashOut](child node_types.NodeHandleOwned[H], storage NodeStorage[H]) NodeHandle {
	switch n := child.(type) {
	case node_types.NodeHandleOwnedHash[H]:
		return Hash[H]{n.Hash}
	case node_types.NodeHandleOwnedInline[H]:
		child := NodeFromNodeOwned[H](n.Node, storage)
		return InMemory{storage.alloc(StoredNew[H]{child})}
	default:
		panic("Invalid child")
	}
}

func NodeFromNodeOwned[H HashOut](nodeOwned node_types.NodeOwned[H], storage NodeStorage[H]) Node[H] {
	switch node := nodeOwned.(type) {
	case node_types.NodeOwnedEmpty:
		return Empty{}
	case node_types.NodeOwnedLeaf[H]:
		return Leaf{
			encoded: node.PartialKey,
			value:   node.Value,
		}
	case node_types.NodeOwnedNibbledBranch[H]:
		child := func(i uint) NodeHandle {
			if node.EncodedChildren[i] != nil {
				return inlineOrHashOwned(node.EncodedChildren[i], storage)
			}
			return nil
		}

		var children [16]NodeHandle
		for i := uint(0); i < 16; i++ {
			children[i] = child(i)
		}

		return NibbledBranch{
			encoded:  node.PartialKey,
			children: children,
			value:    node.Value,
		}
	default:
		panic("Invalid node")
	}
}
