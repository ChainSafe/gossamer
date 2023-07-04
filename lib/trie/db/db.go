// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package db

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Database defines a key value Get method used
// for proof generation.
type Database interface {
	Get(key []byte) (value []byte, err error)
}

type MemoryDB struct {
	data map[common.Hash][]byte
}

func NewMemoryDBFromProof(encodedNodes [][]byte) (*MemoryDB, error) {
	data := make(map[common.Hash][]byte, len(encodedNodes))

	for _, encodedProofNode := range encodedNodes {
		nodeHash, err := common.Blake2bHash(encodedProofNode)
		if err != nil {
			return nil, err
		}

		data[nodeHash] = encodedProofNode
	}

	return &MemoryDB{
		data: data,
	}, nil

}

func (mdb *MemoryDB) Get(key []byte) (value []byte, err error) {
	if len(key) < common.HashLength {
		return nil, fmt.Errorf("expected %d bytes length key, given %d (%x)", common.HashLength, len(key), value)
	}

	if value, found := mdb.data[common.Hash(key)]; found {
		return value, nil
	}

	return nil, nil
}
