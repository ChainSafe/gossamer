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
	"fmt"

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

		lookup := NewLookup(root, db)
		recorder := new(Recorder)

		_, err := lookup.Find(nk, recorder)
		if err != nil {
			return nil, err
		}

		fmt.Println("Len of records ", len(*recorder))

		for recorder.HasNext() {
			recNode := recorder.Next()
			nodeHashHex := common.BytesToHex(recNode.Hash)
			if _, ok := trackedProofs[nodeHashHex]; !ok {
				trackedProofs[nodeHashHex] = recNode.RawData
			}
		}
	}

	proofs := make([][]byte, 0)

	for _, p := range trackedProofs {
		fmt.Printf("tracked proofs: 0x%x\n", p)
		proofs = append(proofs, p)
	}

	return proofs, nil
}
