// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// Database defines a key value Get method used
// for proof generation.
type Database interface {
	Get(key []byte) (value []byte, err error)
}

// Generate generates and deduplicates the encoded proof nodes
// for the trie corresponding to the root hash given, and for
// the slice of (Little Endian) full keys given. The database given
// is used to load the trie using the root hash given.
func Generate(rootHash []byte, fullKeys [][]byte, database Database) (
	encodedProofNodes [][]byte, err error) {
	trie := trie.NewEmptyTrie()
	if err := trie.Load(database, common.BytesToHash(rootHash)); err != nil {
		return nil, fmt.Errorf("loading trie: %w", err)
	}
	rootNode := trie.RootNode()

	hashesSeen := make(map[string]struct{})
	for _, fullKey := range fullKeys {
		fullKeyNibbles := codec.KeyLEToNibbles(fullKey)
		const isRoot = true
		newEncodedProofNodes, err := walk(rootNode, fullKeyNibbles, isRoot)
		if err != nil {
			// Note we wrap the full key context here since walk is recursive and
			// may not be aware of the initial full key.
			return nil, fmt.Errorf("walking to node at key 0x%x: %w", fullKey, err)
		}

		for _, encodedProofNode := range newEncodedProofNodes {
			digest, err := common.Blake2bHash(encodedProofNode)
			if err != nil {
				return nil, fmt.Errorf("blake2b hash: %w", err)
			}
			hashString := string(digest.ToBytes())

			_, seen := hashesSeen[hashString]
			if seen {
				continue
			}
			hashesSeen[hashString] = struct{}{}

			encodedProofNodes = append(encodedProofNodes, encodedProofNode)
		}
	}

	return encodedProofNodes, nil
}

func walk(parent *node.Node, fullKey []byte, isRoot bool) (
	encodedProofNodes [][]byte, err error) {
	if parent == nil {
		if len(fullKey) == 0 {
			return nil, nil
		}
		return nil, ErrKeyNotFound
	}

	// Note we do not use sync.Pool buffers since we would have
	// to copy it so it persists in encodedProofNodes.
	encodingBuffer := bytes.NewBuffer(nil)
	err = parent.Encode(encodingBuffer)
	if err != nil {
		return nil, fmt.Errorf("encode node: %w", err)
	}

	if isRoot || encodingBuffer.Len() >= 32 {
		// Only add the root node encoding (whatever its length)
		// and child node encodings greater or equal to 32 bytes.
		// This is because child node encodings of less than 32 bytes
		// are inlined in the parent node encoding, so there is no need
		// to duplicate them in the proof generated.
		encodedProofNodes = append(encodedProofNodes, encodingBuffer.Bytes())
	}

	nodeFound := len(fullKey) == 0 || bytes.Equal(parent.Key, fullKey)
	if nodeFound {
		return encodedProofNodes, nil
	}

	if parent.Type() == node.Leaf && !nodeFound {
		return nil, ErrKeyNotFound
	}

	nodeIsDeeper := len(fullKey) > len(parent.Key)
	if !nodeIsDeeper {
		return nil, ErrKeyNotFound
	}

	commonLength := lenCommonPrefix(parent.Key, fullKey)
	childIndex := fullKey[commonLength]
	nextChild := parent.Children[childIndex]
	nextFullKey := fullKey[commonLength+1:]
	isRoot = false
	deeperEncodedProofNodes, err := walk(nextChild, nextFullKey, isRoot)
	if err != nil {
		return nil, err // note: do not wrap since this is recursive
	}

	encodedProofNodes = append(encodedProofNodes, deeperEncodedProofNodes...)
	return encodedProofNodes, nil
}

// lenCommonPrefix returns the length of the
// common prefix between two byte slices.
func lenCommonPrefix(a, b []byte) (length int) {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}

	for length = 0; length < min; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
