// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"

// Used to report the trie access to the [TrieRecorder].
//
// As the trie can use a [TrieCache], there are multiple kinds of accesses.
type TrieAccess interface {
	isTrieAccess()
}

type (
	// The given [CachedNode] was accessed using its hash.
	CachedNodeAccess[H hash.Hash] struct {
		Hash H
		Node CachedNode[H]
	}
	// The given EncodedNode was accessed using its hash.
	EncodedNodeAccess[H hash.Hash] struct {
		Hash        H
		EncodedNode []byte
	}
	// The given Value was accessed using its hash.
	//
	// The given FullKey is the key to access this value in the trie.
	//
	// Should map to [RecordedValue] when checking the recorder.
	ValueAccess[H hash.Hash] struct {
		Hash    H
		Value   []byte
		FullKey []byte
	}
	// A value was accessed that is stored inline a node.
	//
	// As the value is stored inline there is no need to separately record the value as it is part
	// of a node. The given FullKey is the key to access this value in the trie.
	//
	// Should map to [RecordedValue] when checking the recorder.
	InlineValueAccess struct {
		FullKey []byte
	}
	// The hash of the value for the given FullKey was accessed.
	//
	// Should map to [RecordedHash] when checking the recorder.
	HashAccess struct {
		FullKey []byte
	}
	// The value/hash for FullKey was accessed, but it couldn't be found in the trie.
	//
	// Should map to [RecordedValue] when checking the recorder.
	NonExistingNodeAccess struct {
		FullKey []byte
	}
)

func (CachedNodeAccess[H]) isTrieAccess()   {}
func (EncodedNodeAccess[H]) isTrieAccess()  {}
func (ValueAccess[H]) isTrieAccess()        {}
func (InlineValueAccess) isTrieAccess()     {}
func (HashAccess) isTrieAccess()            {}
func (NonExistingNodeAccess) isTrieAccess() {}

// A trie recorder that can be used to record all kind of [TrieAccess].
//
// To build a trie proof a recorder is required that records all trie accesses. These recorded trie
// accesses can then be used to create the proof.
type TrieRecorder interface {
	//  Record the given [TrieAccess].
	//
	//  Depending on the [TrieAccess] a call to [TrieRecorder.TrieNodesRecordedForKey] afterwards
	//  must return the correct recorded state.
	Record(access TrieAccess)

	//  Check if we have recorded any trie nodes for the given key.
	//
	//  Returns [RecordedForKey] to express the state of the recorded trie nodes.
	TrieNodesRecordedForKey(key []byte) RecordedForKey
}

type RecordedForKey int

const (
	// We recorded all trie nodes up to the value for a storage key.
	//
	// This should be returned when the recorder has seen the following [TrieAccess]:
	//
	// - [ValueAccess]: If we see this [TrieAccess], it means we have recorded all the
	//   trie nodes up to the value.
	// - [NonExistingNodeAccess]: If we see this [TrieAccess], it means we have recorded all
	//   the necessary  trie nodes to prove that the value doesn't exist in the trie.
	RecordedValue RecordedForKey = iota
	// We recorded all trie nodes up to the value hash for a storage key.
	//
	// If we have a [RecordedValue], it means that we also have the hash of this value.
	// This also means that if we first have recorded the hash of a value and then also record the
	// value, the access should be upgraded to [RecordedValue].
	//
	// This should be returned when the recorder has seen the following [TrieAccess]:
	//
	// - [HashAccess]: If we see this [TrieAccess], it means we have recorded all trie
	//   nodes to have the hash of the value.
	RecordedHash
	// We haven't recorded any trie nodes yet for a storage key.
	//
	// This means we have not seen any [TrieAccess] referencing the searched key.
	RecordedNone
)

// The record of a visited node.
type Record[H hash.Hash] struct {
	Hash H
	Data []byte
}

// Records trie nodes as they pass it.
type Recorder[H hash.Hash] struct {
	nodes        []Record[H]
	recordedKeys map[string]RecordedForKey
}

// Constructor for [Recorder]
func NewRecorder[H hash.Hash]() *Recorder[H] {
	return &Recorder[H]{
		nodes:        []Record[H]{},
		recordedKeys: make(map[string]RecordedForKey),
	}
}

// Drain all visited records.
func (r *Recorder[H]) Drain() []Record[H] {
	r.recordedKeys = make(map[string]RecordedForKey)
	nodes := r.nodes
	r.nodes = []Record[H]{}
	return nodes
}

// Record the given [TrieAccess].
//
// Depending on the [TrieAccess] a call to [TrieRecorder.TrieNodesRecordedForKey] afterwards
// must return the correct recorded state.
func (r *Recorder[H]) Record(access TrieAccess) {
	switch a := access.(type) {
	case EncodedNodeAccess[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: a.Hash, Data: a.EncodedNode})
	case CachedNodeAccess[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: a.Hash, Data: a.Node.encoded()})
	case ValueAccess[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: a.Hash, Data: a.Value})
		r.recordedKeys[string(a.FullKey)] = RecordedValue
	case InlineValueAccess:
		r.recordedKeys[string(a.FullKey)] = RecordedValue
	case HashAccess:
		if _, ok := r.recordedKeys[string(a.FullKey)]; !ok {
			r.recordedKeys[string(a.FullKey)] = RecordedHash
		}
	case NonExistingNodeAccess:
		// We handle the non existing value/hash like having recorded the value
		r.recordedKeys[string(a.FullKey)] = RecordedValue
	default:
		panic("unreachable")
	}
}

// Check if we have recorded any trie nodes for the given key.
//
// Returns [RecordedForKey] to express the state of the recorded trie nodes.
func (r *Recorder[H]) TrieNodesRecordedForKey(key []byte) RecordedForKey {
	rfk, ok := r.recordedKeys[string(key)]
	if !ok {
		return RecordedNone
	}
	return rfk
}
