// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

type trieAccess interface {
	isTrieAccess()
}

type (
	encodedNodeAccess struct {
		hash        common.Hash
		encodedNode []byte
	}
	valueAccess struct {
		// We are not using common.Hash here since hash size could be > 32 bytes when we use prefixed keys
		hash    []byte
		value   []byte
		fullKey []byte
	}
	inlineValueAccess struct {
		fullKey []byte
	}
	hashAccess struct {
		fullKey []byte
	}
	nonExistingNodeAccess struct {
		fullKey []byte
	}
)

func (encodedNodeAccess) isTrieAccess()     {}
func (valueAccess) isTrieAccess()           {}
func (inlineValueAccess) isTrieAccess()     {}
func (hashAccess) isTrieAccess()            {}
func (nonExistingNodeAccess) isTrieAccess() {}

type recordedForKey int

const (
	recordedValue recordedForKey = iota
	recordedHash
)

type Record struct {
	// We are not using common.Hash here since hash size could be > 32 bytes when we use prefixed keys.
	// See ValueAccess.hash
	hash []byte
	data []byte
}

type Recorder struct {
	nodes        []Record
	recordedKeys btree.Map[string, recordedForKey]
}

func NewRecorder() *Recorder {
	return &Recorder{
		nodes:        []Record{},
		recordedKeys: *btree.NewMap[string, recordedForKey](0),
	}
}

func (r *Recorder) record(access trieAccess) {
	switch a := access.(type) {
	case encodedNodeAccess:
		r.nodes = append(r.nodes, Record{hash: a.hash.ToBytes(), data: a.encodedNode})
	case valueAccess:
		r.nodes = append(r.nodes, Record{hash: a.hash, data: a.value})
		r.recordedKeys.Set(string(a.fullKey), recordedValue)
	case inlineValueAccess:
		r.recordedKeys.Set(string(a.fullKey), recordedValue)
	case hashAccess:
		if _, ok := r.recordedKeys.Get(string(a.fullKey)); !ok {
			r.recordedKeys.Set(string(a.fullKey), recordedHash)
		}
	case nonExistingNodeAccess:
		// We handle the non existing value/hash like having recorded the value
		r.recordedKeys.Set(string(a.fullKey), recordedValue)
	}
}

func (r *Recorder) Drain() []Record {
	r.recordedKeys.Clear()
	nodes := r.nodes
	r.nodes = []Record{}
	return nodes
}
