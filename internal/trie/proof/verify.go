// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var (
	ErrKeyNotFoundInProofTrie = errors.New("key not found in proof trie")
	ErrValueMismatchProofTrie = errors.New("value found in proof trie does not match")
)

// Verify verifies a given key and value belongs to the trie by creating
// a proof trie based on the encoded proof nodes given. The order of proofs is ignored.
func Verify(encodedProofNodes [][]byte, rootHash, key, value []byte) (ok bool, err error) {
	proofTrie, err := buildTrie(encodedProofNodes, rootHash)
	if err != nil {
		return false, fmt.Errorf("cannot build trie from proof encoded nodes: %w", err)
	}

	proofTrieValue := proofTrie.Get(key)
	if proofTrieValue == nil {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFoundInProofTrie, bytesToString(key))
	}

	// compare the value only if the caller pass a non empty value
	if len(value) > 0 && !bytes.Equal(value, proofTrieValue) {
		return false, fmt.Errorf("%w: expected value %s but got value %s from proof trie",
			ErrValueMismatchProofTrie, bytesToString(value), bytesToString(proofTrieValue))
	}

	return true, nil
}

var (
	ErrEmptyProof = errors.New("proof slice empty")
)

// buildTrie sets a partial trie based on the proof slice of encoded nodes.
func buildTrie(encodedProofNodes [][]byte, rootHash []byte) (t *trie.Trie, err error) {
	if len(encodedProofNodes) == 0 {
		return nil, fmt.Errorf("%w: for Merkle root hash %s",
			ErrEmptyProof, bytesToString(rootHash))
	}

	proofHashToNode := make(map[string]*node.Node, len(encodedProofNodes))

	var root *node.Node
	for i, encodedProofNode := range encodedProofNodes {
		decodedNode, err := node.Decode(bytes.NewReader(encodedProofNode))
		if err != nil {
			return nil, fmt.Errorf("cannot decode node at index %d: %w (node encoded is 0x%x)",
				i, err, encodedProofNode)
		}

		const dirty = false
		decodedNode.SetDirty(dirty)
		decodedNode.Encoding = encodedProofNode
		// isRoot is set to true in order to force the hash to be the blake2b
		// digest for every node from the proof and not the smaller than 32 bytes
		// encoded value, since we do not know which node in the proof is the root node.
		const isRoot = false // TODO
		decodedNode.HashDigest, err = node.MerkleValue(encodedProofNode, isRoot)
		if err != nil {
			return nil, fmt.Errorf("cannot calculate Merkle value of node at index %d: %w", i, err)
		}

		proofHash := common.BytesToHex(decodedNode.HashDigest)
		proofHashToNode[proofHash] = decodedNode

		if bytes.Equal(decodedNode.HashDigest, rootHash) {
			// Found root in proof
			root = decodedNode
		}
	}

	loadProof(proofHashToNode, root)

	return trie.NewTrie(root), nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the mapped proofs slice starting at the root
func loadProof(proofHashToNode map[string]*node.Node, n *node.Node) {
	if n.Type() != node.Branch {
		return
	}

	branch := n
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		proofHash := common.BytesToHex(child.HashDigest)
		node, ok := proofHashToNode[proofHash]
		if !ok {
			continue
		}

		branch.Children[i] = node
		loadProof(proofHashToNode, node)
	}
}

func bytesToString(b []byte) (s string) {
	switch {
	case b == nil:
		return "nil"
	case len(b) <= 20:
		return fmt.Sprintf("0x%x", b)
	default:
		return fmt.Sprintf("0x%x...%x", b[:8], b[len(b)-8:])
	}
}
