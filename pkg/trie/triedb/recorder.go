package triedb

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

type RecordedForKey uint8

const (
	RecordedForKeyValue RecordedForKey = iota
	RecordedForKeyHash
	RecordedForKeyNone
)

type TrieRecorder[Out node.HashOut] interface {
	record(access TrieAccess[Out])
	trieNodesRecordedForKey(key []byte) RecordedForKey
}

// TrieAccess is used to report the trie access to the TrieRecorder
type TrieAccess[Out node.HashOut] interface {
	Type() string
}

type (
	// TrieAccessNode means that the given node was accessed using its hash
	TrieAccessNode[H node.HashOut] struct {
		hash H
		node node.Node[H]
	}

	// TrieAccessEncodedNode means that the given encodedNode was accessed using its hash
	TrieAccessEncodedNode[H node.HashOut] struct {
		hash        H
		encodedNode []byte
	}

	// TrieAccessValue means that the given value was accessed using its hash
	// fullKey is the key to access this value in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessValue[H node.HashOut] struct {
		hash    H
		value   []byte
		fullKey []byte
	}

	// TrieAccessInlineValue means that a value stored in an inlined node was accessed
	// The given fullKey is the key to access this value in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessInlineValue[H node.HashOut] struct {
		fullKey []byte
	}

	// TrieAccessHash means that the hash of the value for a given fullKey was accessed
	// Should map to RecordedForKeyHash when checking the recorder
	TrieAccessHash[H node.HashOut] struct {
		fullKey []byte
	}

	// TrieAccessNonExisting means that the value/hash for fullKey was accessed, but it couldn't be found in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessNonExisting struct {
		fullKey []byte
	}
)

func (a TrieAccessNode[H]) Type() string        { return "Node" }
func (a TrieAccessEncodedNode[H]) Type() string { return "EncodedNode" }
func (a TrieAccessValue[H]) Type() string       { return "Value" }
func (a TrieAccessInlineValue[H]) Type() string { return "InlineValue" }
func (a TrieAccessHash[H]) Type() string        { return "Hash" }
func (a TrieAccessNonExisting) Type() string    { return "NotExisting" }

// Recorder implementation

type Record[H node.HashOut] struct {
	/// The hash of the node.
	Hash H
	/// The data representing the node.
	Data []byte
}

type Recorder[H node.HashOut] struct {
	nodes        []Record[H]
	recorderKeys map[string]RecordedForKey // TODO: revisit this later, it should be a BTreeMap
	layout       TrieLayout[H]
}

// NewRecorder creates a new Recorder which records all given nodes
func NewRecorder[H node.HashOut]() *Recorder[H] {
	return &Recorder[H]{
		nodes:        make([]Record[H], 0),
		recorderKeys: make(map[string]RecordedForKey),
	}
}

// Drain drains all visited records
func (r *Recorder[H]) Drain() []Record[H] {
	// Store temporal nodes
	nodes := make([]Record[H], len(r.nodes))
	copy(nodes, r.nodes)

	// Clean up internal data and return the nodes
	clear(r.nodes)
	clear(r.recorderKeys)

	return nodes
}

// Impl of TrieRecorder for Recorder
func (r *Recorder[H]) record(access TrieAccess[H]) {
	switch access := access.(type) {
	case TrieAccessEncodedNode[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: access.hash, Data: access.encodedNode})
	case TrieAccessNode[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: access.hash, Data: node.EncodeNode(access.node, r.layout.Codec())})
	case TrieAccessValue[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: access.hash, Data: access.value})
		r.recorderKeys[string(access.fullKey)] = RecordedForKeyValue
	case TrieAccessHash[H]:
		if _, inserted := r.recorderKeys[string(access.fullKey)]; !inserted {
			r.recorderKeys[string(access.fullKey)] = RecordedForKeyHash
		}
	case TrieAccessNonExisting:
		// We handle the non existing value/hash like having recorded the value.
		r.recorderKeys[string(access.fullKey)] = RecordedForKeyValue
	case TrieAccessInlineValue[H]:
		r.recorderKeys[string(access.fullKey)] = RecordedForKeyValue
	default:
		panic(fmt.Sprintf("unknown access type %s", access.Type()))
	}
}
