// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/tidwall/btree"
)

type TrieAccess interface {
	isTrieAccess()
}

type (
	NodeOwnedAccess[H any] struct {
		Hash H
		Node NodeOwned
	}
	EncodedNodeAccess[H any] struct {
		Hash        H
		EncodedNode []byte
	}
	ValueAccess[H any] struct {
		Hash    H
		Value   []byte
		FullKey []byte
	}
	InlineValueAccess struct {
		FullKey []byte
	}
	HashAccess struct {
		FullKey []byte
	}
	NonExistingNodeAccess struct {
		FullKey []byte
	}
)

func (NodeOwnedAccess[H]) isTrieAccess()    {}
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

type RecordedNodesIterator[H any] struct {
	nodes []Record[H]
	index int
}

func NewRecordedNodesIterator[H any](nodes []Record[H]) *RecordedNodesIterator[H] {
	return &RecordedNodesIterator[H]{nodes: nodes, index: -1}
}

func (r *RecordedNodesIterator[H]) Next() *Record[H] {
	if r.index < len(r.nodes)-1 {
		r.index++
		return &r.nodes[r.index]
	}
	return nil
}

func (r *RecordedNodesIterator[H]) Peek() *Record[H] {
	if r.index+1 < len(r.nodes)-1 {
		return &r.nodes[r.index+1]
	}
	return nil
}

type Record[H any] struct {
	Hash H
	Data []byte
}

type Recorder[H any] struct {
	nodes        []Record[H]
	recordedKeys btree.Map[string, RecordedForKey]
}

func NewRecorder[H any]() *Recorder[H] {
	return &Recorder[H]{
		nodes:        []Record[H]{},
		recordedKeys: *btree.NewMap[string, RecordedForKey](0),
	}
}

func (r *Recorder[H]) Record(access TrieAccess) {
	switch a := access.(type) {
	case EncodedNodeAccess[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: a.Hash, Data: a.EncodedNode})
	case ValueAccess[H]:
		r.nodes = append(r.nodes, Record[H]{Hash: a.Hash, Data: a.Value})
		r.recordedKeys.Set(string(a.FullKey), RecordedValue)
	case InlineValueAccess:
		r.recordedKeys.Set(string(a.FullKey), RecordedValue)
	case HashAccess:
		if _, ok := r.recordedKeys.Get(string(a.FullKey)); !ok {
			r.recordedKeys.Set(string(a.FullKey), RecordedHash)
		}
	case NonExistingNodeAccess:
		// We handle the non existing value/hash like having recorded the value
		r.recordedKeys.Set(string(a.FullKey), RecordedValue)
	}
}

func (r *Recorder[H]) Drain() []Record[H] {
	r.recordedKeys.Clear()
	nodes := r.nodes
	r.nodes = []Record[H]{}
	return nodes
}

func (r *Recorder[H]) TrieNodesRecordedForKey(key []byte) RecordedForKey {
	panic("unimpl")
}

var _ TrieRecorder = &Recorder[string]{}
