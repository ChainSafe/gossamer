// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/db"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// Database defines a key value Get method used
// for proof generation.
type Database interface {
	db.DBGetter
	db.DBPutter
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

	buffer := pools.DigestBuffers.Get().(*bytes.Buffer)
	defer pools.DigestBuffers.Put(buffer)

	nodeHashesSeen := make(map[common.Hash]struct{})
	for _, fullKey := range fullKeys {
		fullKeyNibbles := codec.KeyLEToNibbles(fullKey)
		newEncodedProofNodes, err := walkRoot(rootNode, fullKeyNibbles)
		if err != nil {
			// Note we wrap the full key context here since walk is recursive and
			// may not be aware of the initial full key.
			return nil, fmt.Errorf("walking to node at key 0x%x: %w", fullKey, err)
		}

		for _, encodedProofNode := range newEncodedProofNodes {
			buffer.Reset()
			err := node.MerkleValue(encodedProofNode, buffer)
			if err != nil {
				return nil, fmt.Errorf("blake2b hash: %w", err)
			}
			// Note: all encoded proof nodes are larger than 32B so their
			// merkle value is the encoding hash digest (32B) and never the
			// encoding itself.
			nodeHash := common.NewHash(buffer.Bytes())

			_, seen := nodeHashesSeen[nodeHash]
			if seen {
				continue
			}
			nodeHashesSeen[nodeHash] = struct{}{}

			encodedProofNodes = append(encodedProofNodes, encodedProofNode)
		}
	}

	return encodedProofNodes, nil
}

func walkRoot(root *node.Node, fullKey []byte) (
	encodedProofNodes [][]byte, err error) {
	if root == nil {
		if len(fullKey) == 0 {
			return nil, nil
		}
		return nil, ErrKeyNotFound
	}

	// Note we do not use sync.Pool buffers since we would have
	// to copy it so it persists in encodedProofNodes.
	encodingBuffer := bytes.NewBuffer(nil)
	err = root.Encode(encodingBuffer, trie.NoMaxInlineValueSize)
	if err != nil {
		return nil, fmt.Errorf("encode node: %w", err)
	}
	encodedProofNodes = append(encodedProofNodes, encodingBuffer.Bytes())

	nodeFound := len(fullKey) == 0 || bytes.Equal(root.PartialKey, fullKey)
	if nodeFound {
		return encodedProofNodes, nil
	}

	if root.Kind() == node.Leaf && !nodeFound {
		return nil, ErrKeyNotFound
	}

	nodeIsDeeper := len(fullKey) > len(root.PartialKey)
	if !nodeIsDeeper {
		return nil, ErrKeyNotFound
	}

	commonLength := lenCommonPrefix(root.PartialKey, fullKey)
	childIndex := fullKey[commonLength]
	nextChild := root.Children[childIndex]
	nextFullKey := fullKey[commonLength+1:]
	deeperEncodedProofNodes, err := walk(nextChild, nextFullKey)
	if err != nil {
		return nil, err // note: do not wrap since this is recursive
	}

	encodedProofNodes = append(encodedProofNodes, deeperEncodedProofNodes...)
	return encodedProofNodes, nil
}

func walk(parent *node.Node, fullKey []byte) (
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
	err = parent.Encode(encodingBuffer, trie.NoMaxInlineValueSize)
	if err != nil {
		return nil, fmt.Errorf("encode node: %w", err)
	}

	if encodingBuffer.Len() >= 32 {
		// Only add (non root) node encodings greater or equal to 32 bytes.
		// This is because child node encodings of less than 32 bytes
		// are inlined in the parent node encoding, so there is no need
		// to duplicate them in the proof generated.
		encodedProofNodes = append(encodedProofNodes, encodingBuffer.Bytes())
	}

	nodeFound := len(fullKey) == 0 || bytes.Equal(parent.PartialKey, fullKey)
	if nodeFound {
		return encodedProofNodes, nil
	}

	if parent.Kind() == node.Leaf && !nodeFound {
		return nil, ErrKeyNotFound
	}

	nodeIsDeeper := len(fullKey) > len(parent.PartialKey)
	if !nodeIsDeeper {
		return nil, ErrKeyNotFound
	}

	commonLength := lenCommonPrefix(parent.PartialKey, fullKey)
	childIndex := fullKey[commonLength]
	nextChild := parent.Children[childIndex]
	nextFullKey := fullKey[commonLength+1:]
	deeperEncodedProofNodes, err := walk(nextChild, nextFullKey)
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
