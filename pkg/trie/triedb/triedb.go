package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
	"github.com/gammazero/deque"
)

type StorageHandle = uint
type NibbleFullKey = nibble.NibbleSlice

type Stored[H node.HashOut] interface {
	Type() string
}

type (
	StoredNew[Out node.HashOut] struct {
		Node node.Node[Out]
	}
	StoredCached[Out node.HashOut] struct {
		Node node.Node[Out]
		Hash Out
	}
)

func (s StoredNew[Out]) Type() string    { return "New" }
func (s StoredCached[Out]) Type() string { return "Cached" }

type NodeHandle interface {
	Type() string
}
type (
	Hash[H node.HashOut] struct {
		Value H
	}
	InMemory struct {
		Value StorageHandle
	}
)

func (h Hash[H]) Type() string  { return "Hash" }
func (h InMemory) Type() string { return "InMemory" }

// Compact storage for tree nodes
type NodeStorage[H node.HashOut] struct {
	nodes       []Stored[H]
	freeIndices deque.Deque[uint]
}

func NewEmptyNodeStorage[H node.HashOut]() *NodeStorage[H] {
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
	ns.nodes[idx] = StoredNew[H]{node.Empty{}}
	return ns.nodes[idx]
}

type deathRowValue struct {
	backingByte [40]byte
	b           *[]byte
}

type TrieDB[Out node.HashOut] struct {
	storage    NodeStorage[Out]
	db         hashdb.HashDB[Out, DBValue]
	root       Out
	rootHandle node.NodeHandle
	deathRow   map[Out]struct{}
	hashCount  uint
	cache      TrieCache[Out]
	recorder   TrieRecorder[Out]
	layout     TrieLayout[Out]
}

func NewTrieDB[H node.HashOut](
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

// TODO: implement me
func (tdb *TrieDB[H]) lookupAndCache(
	hash H,
	key hashdb.Prefix,
) (StorageHandle, error) {
	return 0, nil
}

type PostInspectAction interface {
	Type() string
}

type (
	Replace[H node.HashOut] struct {
		node node.Node[H]
	}
	Restore[H node.HashOut] struct {
		node node.Node[H]
	}
	Delete struct{}
)

func (r Replace[H]) Type() string { return "Replace" }
func (r Restore[H]) Type() string { return "Restore" }
func (r Delete) Type() string     { return "Delete" }

type InspectResult[H node.HashOut] struct {
	stored  Stored[H]
	changed bool
}

// TODO: implement me
func (tdb *TrieDB[H]) inspect(
	stored Stored[H],
	key NibbleFullKey,
	inspector func(
		node node.Node[H],
		key NibbleFullKey,
	) (PostInspectAction, error),
) (InspectResult[H], error) {
	panic("implement me")
}

// TODO: implement me
func (tdb *TrieDB[H]) removeInspector(
	node node.Node[H],
	key NibbleFullKey,
	oldVal *TrieValue,
) (PostInspectAction, error) {
	panic("implement me")
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

	res, err := tdb.inspect(stored, key, func(node node.Node[H], key NibbleFullKey) (PostInspectAction, error) {
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

// TODO: implement me
func (tdb *TrieDB[H]) insertAt(
	handle NodeHandle,
	key NibbleFullKey,
	value []byte,
	oldVal *TrieValue,
) (InsertAtResult, error) {
	panic("implement me")
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
