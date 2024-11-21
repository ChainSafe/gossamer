// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
	"github.com/gammazero/deque"
)

type MerkleProof[H hash.Hash, Hasher hash.Hasher[H]] [][]byte

func NewMerkleProof[H hash.Hash, Hasher hash.Hasher[H]](
	db db.RWDatabase, trieVersion trie.TrieLayout, rootHash H, keys []string) (
	proof MerkleProof[H, Hasher], err error) {
	// Sort and deduplicate keys
	keys = sortAndDeduplicateKeys(keys)

	// The stack of nodes through a path in the trie.
	// Each entry is a child node of the preceding entry.
	stack := new(deque.Deque[*genProofStackEntry[H]])

	// final proof nodes
	var proofNodes MerkleProof[H, Hasher]

	// Iterate over the keys and build the proof nodes
	for i := 0; i < len(keys); i = i + 1 {
		var key = []byte(keys[i])
		var keyNibbles = nibbles.NewLeftNibbles(key)

		err := unwindStack(stack, proofNodes, &keyNibbles)
		if err != nil {
			return nil, err
		}

		// Traverse the trie recording the visited nodes
		recorder := triedb.NewRecorder[H]()
		trie := triedb.NewTrieDB[H, Hasher](rootHash, db, triedb.WithRecorder[H, Hasher](recorder))
		trie.SetVersion(trieVersion)
		trie.Get(key)

		recordedNodes := NewIterator(recorder.Drain())

		// Skip over recorded nodes already on the stack.
		for i := 0; i < stack.Len(); i++ {
			nextEntry := stack.At(i)
			nextRecord := recordedNodes.Peek()

			if nextRecord == nil || !bytes.Equal(nextEntry.nodeHash[:], nextRecord.Hash.Bytes()) {
				break
			}

			recordedNodes.Next()
		}

		// Descend in trie collecting nodes until find the value or the end of the path
	loop:
		for {
			var nextStep genProofStep
			var entry *genProofStackEntry[H]
			if stack.Len() > 0 {
				entry = stack.Back()
			}
			if entry == nil {
				nextStep = genProofStepDescend{childPrefixLen: 0, child: nodeHandleHash[H]{rootHash}}
			} else {
				var err error
				nextStep, err = genProofMatchKeyToNode(
					entry.node,
					&entry.omitValue,
					&entry.childIndex,
					keyNibbles,
					entry.prefix.Len(),
					recordedNodes,
				)

				if err != nil {
					return nil, err
				}
			}

			switch s := nextStep.(type) {
			case genProofStepDescend:
				childPrefix := keyNibbles.Truncate(s.childPrefixLen)
				var childEntry *genProofStackEntry[H]
				switch child := s.child.(type) {
				case nodeHandleHash[H]:
					childRecord := recordedNodes.Next()

					// if !bytes.Equal(childRecord.Hash[:], child[:]) {
					if childRecord.Hash != child.hash {
						panic("hash mismatch")
					}

					outputIndex := uint(len(proofNodes))

					// Insert a placeholder into output which will be replaced when this
					// new entry is popped from the stack.
					proofNodes = append(proofNodes, []byte{})
					childEntry, err = newGenProofStackEntry[H](
						childPrefix,
						childRecord.Data,
						childRecord.Hash.Bytes(),
						&outputIndex,
					)

					if err != nil {
						return nil, err
					}
				case nodeHandleInline:
					if len(child) > (*new(H)).Length() {
						return nil, errors.New("invalid hash length")
					}
					childEntry, err = newGenProofStackEntry[H](
						childPrefix,
						child,
						nil,
						nil,
					)
					if err != nil {
						return nil, err
					}
				}
				stack.PushBack(childEntry)
			default:
				recordedNodes.Next()
				break loop
			}
		}
	}

	err = unwindStack(stack, proofNodes, nil)
	if err != nil {
		return nil, err
	}
	return proofNodes, nil
}
