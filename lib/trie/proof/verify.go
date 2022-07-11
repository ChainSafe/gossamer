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

	merkleValueToEncoding := make(map[string][]byte, len(encodedProofNodes))

	// This loop finds the root node and decodes it.
	// The other nodes have their Merkle value (blake2b digest or the encoding itself)
	// inserted into a map from merkle value to encoding. They are only decoded
	// later if the root or one of its descendant node reference their Merkle value.
	var root *node.Node
	for _, encodedProofNode := range encodedProofNodes {
		var digest []byte
		if root == nil {
			// root node not found yet
			digestHash, err := common.Blake2bHash(encodedProofNode)
			if err != nil {
				return nil, fmt.Errorf("blake2b hash: %w", err)
			}
			digest = digestHash[:]

			if bytes.Equal(digest, rootHash) {
				root, err = node.Decode(bytes.NewReader(encodedProofNode))
				if err != nil {
					return nil, fmt.Errorf("decoding root node: %w", err)
				}
				continue // no need to add root to map of hash to encoding
			}
		}

		var merkleValue []byte
		if len(encodedProofNode) <= 32 {
			merkleValue = encodedProofNode
		} else {
			if digest == nil {
				digestHash, err := common.Blake2bHash(encodedProofNode)
				if err != nil {
					return nil, fmt.Errorf("blake2b hash: %w", err)
				}
				digest = digestHash[:]
			}
			merkleValue = digest
		}

		merkleValueToEncoding[string(merkleValue)] = encodedProofNode
	}

	if root == nil {
		proofMerkleValues := make([]string, 0, len(merkleValueToEncoding))
		for merkleValueString := range merkleValueToEncoding {
			merkleValueHex := common.BytesToHex([]byte(merkleValueString))
			proofMerkleValues = append(proofMerkleValues, merkleValueHex)
		}
		return nil, fmt.Errorf("%w: for Merkle root hash 0x%x in proof Merkle value(s) %s",
			ErrRootNodeNotFound, rootHash, strings.Join(proofMerkleValues, ", "))
	}

	err = loadProof(merkleValueToEncoding, root)
	if err != nil {
		return nil, fmt.Errorf("loading proof: %w", err)
	}

	return trie.NewTrie(root), nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the map from node hash to node starting at the root.
func loadProof(merkleValueToEncoding map[string][]byte, n *node.Node) (err error) {
	if n.Type() != node.Branch {
		return nil
	}

	branch := n
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		merkleValue := child.HashDigest
		encoding, ok := merkleValueToEncoding[string(merkleValue)]
		if !ok {
			inlinedChild := len(child.Value) > 0 || child.HasChild()
			if !inlinedChild {
				// hash not found and the child is not inlined,
				// so clear the child from the branch.
				branch.Descendants -= 1 + child.Descendants
				branch.Children[i] = nil
				if !branch.HasChild() {
					// Convert branch to a leaf if all its children are nil.
					branch.Children = nil
				}
			}
			continue
		}

		child, err := node.Decode(bytes.NewReader(encoding))
		if err != nil {
			return fmt.Errorf("decoding child node for Merkle value 0x%x: %w",
				merkleValue, err)
		}

		branch.Children[i] = child
		branch.Descendants += child.Descendants
		err = loadProof(merkleValueToEncoding, child)
		if err != nil {
			return err // do not wrap error since this is recursive
		}
	}

	return nil
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
