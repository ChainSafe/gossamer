// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	// ErrEmptyTrieRoot occurs when trying to craft a prove with an empty trie root
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")

	// ErrValueNotFound indicates that a returned verify proof value doesnt match with the expected value on items array
	ErrValueNotFound = errors.New("expected value not found in the trie")

	// ErrDuplicateKeys not allowed to verify proof with duplicate keys
	ErrDuplicateKeys = errors.New("duplicate keys on verify proof")

	// ErrLoadFromProof occurs when there are problems with the proof slice while building the partial proof trie
	ErrLoadFromProof = errors.New("failed to build the proof trie")
)

// GenerateProof receive the keys to proof, the trie root and a reference to database
// will
func GenerateProof(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
	trackedProofs := make(map[string][]byte)

	proofTrie := NewEmptyTrie()
	if err := proofTrie.Load(db, common.BytesToHash(root)); err != nil {
		return nil, err
	}

	for _, k := range keys {
		nk := keyToNibbles(k)

		recorder := new(recorder)
		err := findAndRecord(proofTrie, nk, recorder)
		if err != nil {
			return nil, err
		}

		for !recorder.isEmpty() {
			recNode := recorder.next()
			nodeHashHex := common.BytesToHex(recNode.hash)
			if _, ok := trackedProofs[nodeHashHex]; !ok {
				trackedProofs[nodeHashHex] = recNode.rawData
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

		// here we need to compare value only if the caller pass the value
		if item.Value != nil && !bytes.Equal(item.Value, recValue) {
			return false, ErrValueNotFound
		}
	}

	return true, nil
}
