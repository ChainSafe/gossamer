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
	"errors"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	// ErrEmptyTrieRoot occurs when trying to craft a prove with an empty trie root
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")
)

// GenerateProof receive the keys to proof, the trie root and a reference to database
// will
func GenerateProof(root []byte, keys [][]byte, db chaindb.Database) ([][]byte, error) {
	trackedProofs := make(map[string][]byte)

	for _, k := range keys {
		nk := keyToNibbles(k)

		lookup := newLookup(root, db)
		recorder := new(recorder)

		_, err := lookup.find(nk, recorder)
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
