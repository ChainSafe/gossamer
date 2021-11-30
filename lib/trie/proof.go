package trie

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/lib/trie/record"
)

var (
	// ErrEmptyTrieRoot ...
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")

	// ErrValueNotFound ...
	ErrValueNotFound = errors.New("expected value not found in the trie")

	// ErrKeyNotFound ...
	ErrKeyNotFound = errors.New("expected key not found in the trie")

	// ErrDuplicateKeys ...
	ErrDuplicateKeys = errors.New("duplicate keys on verify proof")

	// ErrLoadFromProof ...
	ErrLoadFromProof = errors.New("failed to build the proof trie")
)

// GenerateProof receive the keys to proof, the trie root and a reference to database
func GenerateProof(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
	trackedProofs := make(map[string][]byte)

	proofTrie := NewEmptyTrie()
	if err := proofTrie.Load(db, common.BytesToHash(root)); err != nil {
		return nil, err
	}

	for _, k := range keys {
		nk := decode.KeyLEToNibbles(k)

		recorder := record.NewRecorder()
		err := findAndRecord(proofTrie, nk, recorder)
		if err != nil {
			return nil, err
		}

		for _, recNode := range recorder.GetNodes() {
			nodeHashHex := common.BytesToHex(recNode.Hash)
			if _, ok := trackedProofs[nodeHashHex]; !ok {
				trackedProofs[nodeHashHex] = recNode.RawData
			}
		}
	}

	proofs := make([][]byte, 0)
	for _, p := range trackedProofs {
		proofs = append(proofs, p)
	}

	return proofs, nil
}

// Pair holds the key and value to check while verifying the proof
type Pair struct{ Key, Value []byte }

// VerifyProof ensure a given key is inside a proof by creating a proof trie based on the proof slice
// this function ignores the order of proofs
func VerifyProof(proof [][]byte, root []byte, items []Pair) (bool, error) {
	set := make(map[string]struct{}, len(items))

	// check for duplicate keys
	for _, item := range items {
		hexKey := hex.EncodeToString(item.Key)
		if _, ok := set[hexKey]; ok {
			return false, ErrDuplicateKeys
		}
		set[hexKey] = struct{}{}
	}

	proofTrie := NewEmptyTrie()
	if err := proofTrie.LoadFromProof(proof, root); err != nil {
		return false, fmt.Errorf("%w: %s", ErrLoadFromProof, err)
	}

	for _, item := range items {
		recValue := proofTrie.Get(item.Key)
		if recValue == nil {
			return false, ErrKeyNotFound
		}
		// here we need to compare value only if the caller pass the value
		if len(item.Value) > 0 && !bytes.Equal(item.Value, recValue) {
			return false, ErrValueNotFound
		}
	}

	return true, nil
}
