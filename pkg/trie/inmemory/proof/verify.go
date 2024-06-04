// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
)

var (
	ErrKeyNotFoundInProofTrie = errors.New("key not found in proof trie")
	ErrValueMismatchProofTrie = errors.New("value found in proof trie does not match")
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "proof"))

// Verify verifies a given key and value belongs to the trie by creating
// a proof trie based on the encoded proof nodes given. The order of proofs is ignored.
// A nil error is returned on success.
// Note this is exported because it is imported and used by:
// https://github.com/ComposableFi/ibc-go/blob/6d62edaa1a3cb0768c430dab81bb195e0b0c72db/modules/light-clients/11-beefy/types/client_state.go#L78
func Verify(encodedProofNodes [][]byte, rootHash, key, value []byte) (err error) {
	if len(encodedProofNodes) == 0 {
		return fmt.Errorf("%w: for Merkle root hash 0x%x",
			ErrEmptyProof, rootHash)
	}

	proofDB, err := db.NewMemoryDBFromProof(encodedProofNodes)

	if err != nil {
		return err
	}

	proofTrie, err := buildTrie(proofDB, rootHash)
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
func buildTrie(db db.Database, rootHash []byte) (t trie.Trie, err error) {
	if _, err := db.Get(rootHash); err != nil {
		return nil, fmt.Errorf("%w: for root hash 0x%x",
			ErrRootNodeNotFound, rootHash)
	}

	tr := inmemory.NewEmptyTrie()
	err = tr.Load(db, common.BytesToHash(rootHash))

	if err != nil {
		return nil, err
	}

	return tr, nil
}

// loadProof is a recursive function that will create all the trie paths based
// on the map from node hash digest to node encoding, starting from the node `n`.
func loadProof(digestToEncoding map[string][]byte, n *node.Node) (err error) {
	if n.Kind() != node.Branch {
		return nil
	}

	branch := n
	for i, child := range branch.Children {
		if child == nil {
			continue
		}

		merkleValue := child.MerkleValue
		encoding, ok := digestToEncoding[string(merkleValue)]

		logger.Infof("Node: %x", encoding)

		if !ok {
			inlinedChild := len(child.StorageValue) > 0 || child.HasChild()
			if inlinedChild {
				// The built proof trie is not used with a database, but just in case
				// it becomes used with a database in the future, we set the dirty flag
				// to true.
				child.Dirty = true
			} else {
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

		logger.Info("loading proof DECODING...")
		child, err := node.Decode(bytes.NewReader(encoding))
		if err != nil {
			return fmt.Errorf("decoding child node for hash digest 0x%x: %w",
				merkleValue, err)
		}

		// The built proof trie is not used with a database, but just in case
		// it becomes used with a database in the future, we set the dirty flag
		// to true.
		child.Dirty = true

		branch.Children[i] = child
		branch.Descendants += child.Descendants
		err = loadProof(digestToEncoding, child)
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
