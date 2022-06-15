// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

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
// A nil error is returned on success.
// Note this is exported because it is imported and used by:
// https://github.com/ComposableFi/ibc-go/blob/6d62edaa1a3cb0768c430dab81bb195e0b0c72db/modules/light-clients/11-beefy/types/client_state.go#L78
func Verify(encodedProofNodes [][]byte, rootHash, key, value []byte) (err error) {
	proofTrie, err := buildTrie(encodedProofNodes, rootHash)
	if err != nil {
		return fmt.Errorf("building trie from proof encoded nodes: %w", err)
	}

	proofTrieValue := proofTrie.Get(key)
	if proofTrieValue == nil {
		return fmt.Errorf("%w: %s in proof trie for root hash 0x%x",
			ErrKeyNotFoundInProofTrie, bytesToString(key), rootHash)
	}

	// compare the value only if the caller pass a non empty value
	if len(value) > 0 && !bytes.Equal(value, proofTrieValue) {
		return fmt.Errorf("%w: expected value %s but got value %s from proof trie",
			ErrValueMismatchProofTrie, bytesToString(value), bytesToString(proofTrieValue))
	}

	return nil
}

var (
	ErrEmptyProof       = errors.New("proof slice empty")
	ErrRootNodeNotFound = errors.New("root node not found in proof")
)

// buildTrie sets a partial trie based on the proof slice of encoded nodes.
func buildTrie(encodedProofNodes [][]byte, rootHash []byte) (t *trie.Trie, err error) {
	if len(encodedProofNodes) == 0 {
		return nil, fmt.Errorf("%w: for Merkle root hash 0x%x",
			ErrEmptyProof, rootHash)
	}

	proofHashToNode := make(map[string]*node.Node, len(encodedProofNodes))

	var root *node.Node
	for i, encodedProofNode := range encodedProofNodes {
		decodedNode, err := node.Decode(bytes.NewReader(encodedProofNode))
		if err != nil {
			return nil, fmt.Errorf("decoding node at index %d: %w (node encoded is 0x%x)",
				i, err, encodedProofNode)
		}

		decodedNode.Encoding = encodedProofNode
		// We compute the Merkle value of nodes treating them all
		// as non-root nodes, meaning nodes with encoding smaller
		// than 33 bytes will have their Merkle value set as their
		// encoding. The Blake2b hash of the encoding is computed
		// later if needed to compare with the root hash given to find
		// which node is the root node.
		const isRoot = false
		decodedNode.HashDigest, err = node.MerkleValue(encodedProofNode, isRoot)
		if err != nil {
			return nil, fmt.Errorf("merkle value of node at index %d: %w", i, err)
		}

		proofHash := common.BytesToHex(decodedNode.HashDigest)
		proofHashToNode[proofHash] = decodedNode

		if root != nil {
			// Root node already found in proof
			continue
		}

		possibleRootMerkleValue := decodedNode.HashDigest
		if len(possibleRootMerkleValue) <= 32 {
			// If the root merkle value is smaller than 33 bytes, it means
			// it is the encoding of the node. However, the root node merkle
			// value is always the blake2b digest of the node, and not its own
			// encoding. Therefore, in this case we force the computation of the
			// blake2b digest of the node to check if it matches the root hash given.
			const isRoot = true
			possibleRootMerkleValue, err = node.MerkleValue(encodedProofNode, isRoot)
			if err != nil {
				return nil, fmt.Errorf("merkle value of possible root node: %w", err)
			}
		}

		if bytes.Equal(rootHash, possibleRootMerkleValue) {
			decodedNode.HashDigest = rootHash
			root = decodedNode
		}
	}

	if root == nil {
		proofHashes := make([]string, 0, len(proofHashToNode))
		for proofHash := range proofHashToNode {
			proofHashes = append(proofHashes, proofHash)
		}
		return nil, fmt.Errorf("%w: for Merkle root hash 0x%x in proof Merkle value(s) %s",
			ErrRootNodeNotFound, rootHash, strings.Join(proofHashes, ", "))
	}

	loadProof(proofHashToNode, root)

	return trie.NewTrie(root), nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the map from node hash to node starting at the root.
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
