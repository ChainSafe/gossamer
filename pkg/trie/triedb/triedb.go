package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	n "github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
	"github.com/gammazero/deque"
)

type HashOut = n.HashOut

// Value
type Value interface {
	Type() string
}

type (
	InlineValue struct {
		Bytes []byte
	}
	NodeValue[H HashOut] struct {
		Hash H
	}

	NewNode[H HashOut] struct {
		Hash  *H
		Bytes []byte
	}
)

// Node types in the Trie
type Node[H HashOut] interface {
	Type() string
}

type (
	// NodeEmptyNode represents an empty node
	Empty struct{}
	// NodeLeaf represents a leaf node
	Leaf struct {
		PartialKey nibble.NodeKey
		Value      Value
	}
	// NodeNibbledBranch represents a branch node
	NibbledBranch struct {
		PartialKey nibble.NodeKey
		Children   [nibble.NibbleLength]NodeHandle
		Value      Value
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
	StoredNew[Out HashOut] struct {
		Node Node[Out]
	}
	StoredCached[Out HashOut] struct {
		Node Node[Out]
		Hash Out
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

type deathRowValue struct {
	backingByte [40]byte
	b           *[]byte
}

type TrieDB[Out HashOut] struct {
	storage    NodeStorage[Out]
	db         hashdb.HashDB[Out, DBValue]
	root       Out
	rootHandle NodeHandle
	deathRow   map[Out]struct{}
	hashCount  uint
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
	key hashdb.Prefix,
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

// TODO: implement me
func (tdb *TrieDB[H]) inspect(
	stored Stored[H],
	key NibbleFullKey,
	inspector func(
		node Node[H],
		key NibbleFullKey,
	) (PostInspectAction, error),
) (InspectResult[H], error) {
	panic("implement me")
}

// TODO: implement me
func (tdb *TrieDB[H]) removeInspector(
	node Node[H],
	key NibbleFullKey,
	oldVal *TrieValue,
) (PostInspectAction, error) {
	panic("implement me")
}

// TODO: implement me
func (tdb *TrieDB[H]) insertInspector(
	node Node[H],
	key NibbleFullKey,
	value []byte,
	oldVal *TrieValue,
) (PostInspectAction, error) {
	panic("Implement me")
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

func NodeFromNodeOwned[H HashOut](nodeOwned n.NodeOwned[H], storage NodeStorage[H]) Node[H] {
	switch node := nodeOwned.(type) {
	case Empty:
		return Empty{}
	case Leaf:
		return Leaf{
			PartialKey: node.PartialKey,
			Value:      node.Value,
		}
	case NibbledBranch:
		return NibbledBranch{
			PartialKey: node.PartialKey,
			Children:   node.Children,
			Value:      node.Value,
		}
	default:
		panic("Invalid node")
	}
}
