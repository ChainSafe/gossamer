// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package recorder

import (
	"fmt"
	"log"
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
)

// Stores all the information per transaction.
type transaction[H comparable] struct {
	// Stores transaction information about recorder keys.
	// For each transaction we only store the storage root and the old states per key.
	// nil map entry means that the key wasn't recorded before.
	recordedKeys map[H]map[string]*triedb.RecordedForKey
	// Stores transaction information about accessed nodes.
	// For each transaction we only store the hashes of added nodes.
	accessedNodes map[H]bool
}

func newTransaction[H comparable]() transaction[H] {
	return transaction[H]{
		recordedKeys:  make(map[H]map[string]*triedb.RecordedForKey),
		accessedNodes: make(map[H]bool),
	}
}

// The internals of [Recorder].
type recorderInner[H comparable] struct {
	// The keys for that we have recorded the trie nodes and if we have recorded up to the value.
	recordedKeys map[H]map[string]triedb.RecordedForKey
	// Currently active transactions.
	transactions []transaction[H]

	// The encoded nodes we accessed while recording.
	accessedNodes map[H][]byte
}

func newRecorderInner[H comparable]() recorderInner[H] {
	return recorderInner[H]{
		recordedKeys:  make(map[H]map[string]triedb.RecordedForKey),
		transactions:  make([]transaction[H], 0),
		accessedNodes: make(map[H][]byte),
	}
}

// Recorder is be used to record accesses to the trie and then to convert them into a [StorageProof].
type Recorder[H runtime.Hash] struct {
	inner    recorderInner[H]
	innerMtx sync.Mutex
	// The estimated encoded size of the storage proof this recorder will produce.
	encodedSizeEstimation    uint
	encodedSizeEstimationMtx sync.Mutex
}

// Constructor for [Recorder].
func NewRecorder[H runtime.Hash]() *Recorder[H] {
	return &Recorder[H]{
		inner: newRecorderInner[H](),
	}
}

// Returns the recorder as an implementation of [triedb.TrieRecorder].
//
// The storage root supplied is of the trie for which accesses are recorded.
// This is important when recording access to different tries at once (like top and child tries).
func (r *Recorder[H]) TrieRecorder(storageRoot H) triedb.TrieRecorder {
	return &trieRecorder[H]{
		inner:                 &r.inner,
		innerMtx:              &r.innerMtx,
		storageRoot:           storageRoot,
		encodedSizeEstimation: &r.encodedSizeEstimation,
	}
}

// Drain the recording into a [StorageProof].
//
// While a recorder can be cloned, all share the same internal state. After calling this
// function, all other instances will have their internal state reset as well.
//
// If you don't want to drain the recorded state, use [Recorder.StorageProof()].
//
// Returns a [StorageProof].
func (r *Recorder[H]) DrainStorageProof() trie.StorageProof {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()

	proof := r.storageProof()
	r.inner.accessedNodes = make(map[H][]byte)
	return proof
}

func (r *Recorder[H]) storageProof() trie.StorageProof {
	values := make([][]byte, len(r.inner.accessedNodes))
	i := 0
	for _, v := range r.inner.accessedNodes {
		values[i] = v
		i++
	}
	return trie.NewStorageProof(values)
}

// Convert the recording to a [StorageProof].
//
// In contrast to [Recorder.DrainStorageProof] this doesn't consume and clear the
// recordings.
//
// Returns a [StorageProof].
func (r *Recorder[H]) StorageProof() trie.StorageProof {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()
	return r.storageProof()
}

// Returns the estimated encoded size of the proof.
//
// The estimation is based on all the nodes that were accessed until now while
// accessing the trie.
func (r *Recorder[H]) EstimateEncodedSize() uint {
	return r.encodedSizeEstimation
}

// Reset the state.
//
// This discards all recorded data.
func (r *Recorder[H]) Reset() {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()
	r.inner = newRecorderInner[H]()
}

// Start a new transaction.
func (r *Recorder[H]) StartTransaction() {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()
	r.inner.transactions = append(r.inner.transactions, newTransaction[H]())
}

// Rollback the latest transaction.
//
// Returns an error if there wasn't any active transaction.
func (r *Recorder[H]) RollBackTransaction() error {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()
	r.encodedSizeEstimationMtx.Lock()
	defer r.encodedSizeEstimationMtx.Unlock()
	// We locked everything and can just update the encoded size locally and then store it back
	newEncodedSizeEstimation := r.encodedSizeEstimation
	if len(r.inner.transactions) == 0 {
		return fmt.Errorf("no transactions to roll back")
	}
	tx := r.inner.transactions[len(r.inner.transactions)-1]
	r.inner.transactions = r.inner.transactions[:len(r.inner.transactions)-1]

	for n := range tx.accessedNodes {
		old, ok := r.inner.accessedNodes[n]
		if ok {
			delete(r.inner.accessedNodes, n)
			oldEncodedSize := uint(len(scale.MustMarshal(old)))
			// saturating sub
			if newEncodedSizeEstimation >= oldEncodedSize {
				newEncodedSizeEstimation = newEncodedSizeEstimation - oldEncodedSize
			} else {
				newEncodedSizeEstimation = 0
			}
		}
	}

	for storageRoot, keys := range tx.recordedKeys {
		for k, oldState := range keys {
			if oldState != nil {
				state := *oldState
				_, ok := r.inner.recordedKeys[storageRoot]
				if !ok {
					r.inner.recordedKeys[storageRoot] = make(map[string]triedb.RecordedForKey)
				}
				r.inner.recordedKeys[storageRoot][k] = state
			} else {
				_, ok := r.inner.recordedKeys[storageRoot]
				if !ok {
					r.inner.recordedKeys[storageRoot] = make(map[string]triedb.RecordedForKey)
				}
				delete(r.inner.recordedKeys[storageRoot], k)
			}
		}
	}

	r.encodedSizeEstimation = newEncodedSizeEstimation
	return nil
}

// Commit the latest transaction.
//
// Returns an error if there wasn't any active transaction.
func (r *Recorder[H]) CommitTransaction() error {
	r.innerMtx.Lock()
	defer r.innerMtx.Unlock()

	if len(r.inner.transactions) == 0 {
		return fmt.Errorf("no transactions to roll back")
	}
	tx := r.inner.transactions[len(r.inner.transactions)-1]
	r.inner.transactions = r.inner.transactions[:len(r.inner.transactions)-1]

	if len(r.inner.transactions) != 0 {
		parentTx := r.inner.transactions[len(r.inner.transactions)-1]
		for h, v := range tx.accessedNodes {
			parentTx.accessedNodes[h] = v
		}

		for storageRoot, keys := range tx.recordedKeys {
			for k, oldState := range keys {
				_, ok := parentTx.recordedKeys[storageRoot]
				if !ok {
					parentTx.recordedKeys[storageRoot] = make(map[string]*triedb.RecordedForKey)
				}
				_, ok = parentTx.recordedKeys[storageRoot][k]
				if !ok {
					parentTx.recordedKeys[storageRoot][k] = oldState
				}
			}
		}
	}
	return nil
}

// The [triedb.TrieRecorder] implementation.
type trieRecorder[H runtime.Hash] struct {
	inner                 *recorderInner[H]
	innerMtx              *sync.Mutex
	storageRoot           H
	encodedSizeEstimation *uint
}

// Update the recorded keys entry for the given fullKey.
func (tr *trieRecorder[H]) updateRecordedKeys(fullKey []byte, access triedb.RecordedForKey) {
	_, ok := tr.inner.recordedKeys[tr.storageRoot]
	if !ok {
		tr.inner.recordedKeys[tr.storageRoot] = make(map[string]triedb.RecordedForKey)
	}
	key := string(fullKey)
	old, ok := tr.inner.recordedKeys[tr.storageRoot][key]

	// We don't need to update the record if we only accessed the hash for the given
	// fullKey.
	switch access {
	case triedb.RecordedValue:
		if ok {
			if len(tr.inner.transactions) > 0 {
				tx := &tr.inner.transactions[len(tr.inner.transactions)-1]
				// Store the previous state only once per transaction.
				_, ok := tx.recordedKeys[tr.storageRoot]
				if !ok {
					tx.recordedKeys[tr.storageRoot] = make(map[string]*triedb.RecordedForKey)
				}
				_, ok = tx.recordedKeys[tr.storageRoot][key]
				if !ok {
					tx.recordedKeys[tr.storageRoot][key] = &old
				}
			}
			tr.inner.recordedKeys[tr.storageRoot][key] = access
		}
	default:
	}

	if !ok {
		if len(tr.inner.transactions) > 0 {
			tx := &tr.inner.transactions[len(tr.inner.transactions)-1]
			// The key wasn't yet recorded, so there isn't any old state.
			_, ok := tx.recordedKeys[tr.storageRoot]
			if !ok {
				tx.recordedKeys[tr.storageRoot] = make(map[string]*triedb.RecordedForKey)
			}
			_, ok = tx.recordedKeys[tr.storageRoot][key]
			if !ok {
				tx.recordedKeys[tr.storageRoot][key] = nil
			}
		}
		tr.inner.recordedKeys[tr.storageRoot][key] = access
	}
}

func (tr *trieRecorder[H]) Record(access triedb.TrieAccess) {
	tr.innerMtx.Lock()
	defer tr.innerMtx.Unlock()

	var encodedSizeUpdate uint
	switch access := access.(type) {
	case triedb.CachedNodeAccess[H]:
		log.Printf("TRACE: Recording node: %v", access.Hash)
		_, ok := tr.inner.accessedNodes[access.Hash]
		if !ok {
			node := access.Node.Encoded()
			encodedSizeUpdate += uint(len(scale.MustMarshal(node)))

			if len(tr.inner.transactions) > 0 {
				tx := tr.inner.transactions[len(tr.inner.transactions)-1]
				tx.accessedNodes[access.Hash] = true
				tr.inner.transactions[len(tr.inner.transactions)-1] = tx
			}
			tr.inner.accessedNodes[access.Hash] = node
		}
	case triedb.EncodedNodeAccess[H]:
		log.Printf("TRACE: Recording node: %v", access.Hash)
		_, ok := tr.inner.accessedNodes[access.Hash]
		if !ok {
			node := access.EncodedNode
			encodedSizeUpdate += uint(len(scale.MustMarshal(node)))

			if len(tr.inner.transactions) > 0 {
				tx := tr.inner.transactions[len(tr.inner.transactions)-1]
				tx.accessedNodes[access.Hash] = true
				tr.inner.transactions[len(tr.inner.transactions)-1] = tx
			}
			tr.inner.accessedNodes[access.Hash] = node
		}
	case triedb.ValueAccess[H]:
		log.Printf("TRACE: Recording value {hash:%v value:%v}", access.Hash, access.FullKey)
		_, ok := tr.inner.accessedNodes[access.Hash]
		if !ok {
			value := access.Value
			encodedSizeUpdate += uint(len(scale.MustMarshal(value)))
			if len(tr.inner.transactions) > 0 {
				tx := tr.inner.transactions[len(tr.inner.transactions)-1]
				tx.accessedNodes[access.Hash] = true
				tr.inner.transactions[len(tr.inner.transactions)-1] = tx
			}
			tr.inner.accessedNodes[access.Hash] = value
		}
		tr.updateRecordedKeys(access.FullKey, triedb.RecordedValue)
	case triedb.HashAccess:
		log.Printf("TRACE: Recorded hash access for key: %s", access.FullKey)
		// We don't need to update the encodedSizeUpdate as the hash was already
		// accounted for by the recorded node that holds the hash.
		tr.updateRecordedKeys(access.FullKey, triedb.RecordedHash)
	case triedb.NonExistingNodeAccess:
		log.Printf("TRACE: Recorded non-existing value access for key for key: %s", access.FullKey)
		// Non-existing access means we recorded all trie nodes up to the value.
		// Not the actual value, as it doesn't exist, but all trie nodes to know
		// that the value doesn't exist in the trie.
		tr.updateRecordedKeys(access.FullKey, triedb.RecordedValue)
	case triedb.InlineValueAccess:
		log.Printf("TRACE: Recorded inline value access for key: %s", access.FullKey)
		// A value was accessed that is stored inline a node and we recorded all trie nodes
		// to access this value.
		tr.updateRecordedKeys(access.FullKey, triedb.RecordedValue)
	default:
		panic("unreachable")
	}

	*tr.encodedSizeEstimation = *tr.encodedSizeEstimation + encodedSizeUpdate
}

func (tr *trieRecorder[H]) TrieNodesRecordedForKey(key []byte) triedb.RecordedForKey {
	tr.innerMtx.Lock()
	defer tr.innerMtx.Unlock()

	_, ok := tr.inner.recordedKeys[tr.storageRoot]
	if !ok {
		return triedb.RecordedNone
	}
	recorded, ok := tr.inner.recordedKeys[tr.storageRoot][string(key)]
	if !ok {
		return triedb.RecordedNone
	}
	return recorded
}
