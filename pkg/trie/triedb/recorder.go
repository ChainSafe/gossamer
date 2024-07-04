package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

type TrieAccess interface {
	isTrieAccess()
}

type (
	EncodedNodeAccess struct {
		hash        common.Hash
		encodedNode []byte
	}
	ValueAccess struct {
		// We are not using common.Hash here since hash size could be > 32 bytes when we use prefixed keys
		hash    []byte
		value   []byte
		fullKey []byte
	}
	InlineValueAccess struct {
		fullKey []byte
	}
	HashAccess struct {
		fullKey []byte
	}
	NonExistingNodeAccess struct {
		fullKey []byte
	}
)

func (EncodedNodeAccess) isTrieAccess()     {}
func (ValueAccess) isTrieAccess()           {}
func (InlineValueAccess) isTrieAccess()     {}
func (HashAccess) isTrieAccess()            {}
func (NonExistingNodeAccess) isTrieAccess() {}

type RecordedForKey int

const (
	RecordedValue RecordedForKey = iota
	RecordedHash
)

type RecordedNodesIterator struct {
	nodes []Record
	index int
}

func NewRecordedNodesIterator(nodes []Record) *RecordedNodesIterator {
	return &RecordedNodesIterator{nodes: nodes, index: -1}
}

func (r *RecordedNodesIterator) Next() (Record, bool) {
	if r.index < len(r.nodes)-1 {
		r.index++
		return r.nodes[r.index], true
	}
	return Record{}, false
}

func (r *RecordedNodesIterator) Peek() (Record, bool) {
	if r.index+1 < len(r.nodes)-1 {
		r.index++
		return r.nodes[r.index+1], true
	}
	return Record{}, false
}

type Record struct {
	// We are not using common.Hash here since Hash size could be > 32 bytes when we use prefixed keys.
	// See ValueAccess.Hash
	Hash []byte
	Data []byte
}

type Recorder struct {
	nodes        []Record
	recordedKeys btree.Map[string, RecordedForKey]
}

func NewRecorder() *Recorder {
	return &Recorder{
		nodes:        []Record{},
		recordedKeys: *btree.NewMap[string, RecordedForKey](0),
	}
}

func (r *Recorder) record(access TrieAccess) {
	switch a := access.(type) {
	case EncodedNodeAccess:
		r.nodes = append(r.nodes, Record{Hash: a.hash.ToBytes(), Data: a.encodedNode})
	case ValueAccess:
		r.nodes = append(r.nodes, Record{Hash: a.hash, Data: a.value})
		r.recordedKeys.Set(string(a.fullKey), RecordedValue)
	case InlineValueAccess:
		r.recordedKeys.Set(string(a.fullKey), RecordedValue)
	case HashAccess:
		if _, ok := r.recordedKeys.Get(string(a.fullKey)); !ok {
			r.recordedKeys.Set(string(a.fullKey), RecordedHash)
		}
	case NonExistingNodeAccess:
		// We handle the non existing value/hash like having recorded the value
		r.recordedKeys.Set(string(a.fullKey), RecordedValue)
	}
}

func (r *Recorder) Drain() []Record {
	r.recordedKeys.Clear()
	nodesToReturn := r.nodes
	r.nodes = []Record{}
	return nodesToReturn
}
