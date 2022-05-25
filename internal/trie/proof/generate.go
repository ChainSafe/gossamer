// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"errors"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
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

// Generate receive the keys to proof, the trie root and a reference to database
func Generate(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
	trackedProofs := make(map[string][]byte)

	proofTrie := trie.NewEmptyTrie()
	if err := proofTrie.Load(db, common.BytesToHash(root)); err != nil {
		return nil, err
	}

	for _, k := range keys {
		nk := codec.KeyLEToNibbles(k)

		recorder := newRecorder()
		err := findAndRecord(proofTrie, nk, recorder)
		if err != nil {
			return nil, err
		}

		for _, recNode := range recorder.getNodes() {
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
