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
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	// ErrEmptyTrieRoot occurs when trying to craft a prove with an empty trie root
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")

	// ErrEmptyNibbles occurs when trying to prove or valid a proof to an empty key
	ErrEmptyNibbles = errors.New("empty nibbles provided from key")
)

// GenerateProof constructs the merkle-proof for key. The result contains all encoded nodes
// on the path to the key. Returns the amount of nodes of the path and error if could not found the key
func (t *Trie) GenerateProof(keys [][]byte) (map[string][]byte, error) {
	var nodes []node

	for _, k := range keys {
		currNode := t.root

		nk := keyToNibbles(k)
		if len(nk) == 0 {
			return nil, ErrEmptyNibbles
		}

	proveLoop:
		for {
			switch n := currNode.(type) {
			case nil:
				return nil, errors.New("no more paths to follow")

			case *leaf:
				nodes = append(nodes, n)

				if bytes.Equal(n.key, nk) {
					break proveLoop
				}

				return nil, errors.New("leaf node doest not match the key")

			case *branch:
				nodes = append(nodes, n)
				if bytes.Equal(n.key, nk) || len(nk) == 0 {
					break proveLoop
				}

				length := lenCommonPrefix(n.key, nk)
				currNode = n.children[nk[length]]
				nk = nk[length+1:]
			}
		}
	}

	proof := make(map[string][]byte)
	for _, n := range nodes {
		var (
			hashNode    []byte
			encHashNode []byte
			err         error
		)

		if encHashNode, hashNode, err = n.encodeAndHash(); err != nil {
			return nil, fmt.Errorf("problems while encoding and hashing the node: %w", err)
		}

		// avoid duplicate hashes
		proof[common.BytesToHex(hashNode)] = encHashNode
	}

	return proof, nil
}

// VerifyProof checks merkle proofs given an proof
func VerifyProof(rootHash common.Hash, key []byte, proof map[string][]byte) (bool, error) {
	key = keyToNibbles(key)
	if len(key) == 0 {
		return false, ErrEmptyNibbles
	}

	var wantedHash string
	wantedHash = common.BytesToHex(rootHash.ToBytes())

	for {
		enc, ok := proof[wantedHash]
		if !ok {
			return false, nil
		}

		currNode, err := decodeBytes(enc)
		if err != nil {
			return false, fmt.Errorf("could not decode node bytes: %w", err)
		}

		switch n := currNode.(type) {
		case nil:
			return false, nil
		case *leaf:
			if bytes.Equal(n.key, key) {
				return true, nil
			}

			return false, nil
		case *branch:
			if bytes.Equal(n.key, key) {
				return true, nil
			}

			if len(key) == 0 {
				return false, nil
			}

			length := lenCommonPrefix(n.key, key)
			next := n.children[key[length]]
			if next == nil {
				return false, nil
			}

			key = key[length+1:]
			wantedHash = common.BytesToHex(next.getHash())
		}
	}
}
