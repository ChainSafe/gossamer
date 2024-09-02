// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mmr

import (
	"hash"

	"github.com/tidwall/btree"
)

type MemStorage struct {
	storage *btree.Map[uint64, MMRElement]
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		storage: btree.NewMap[uint64, MMRElement](0),
	}
}

func (s *MemStorage) getElement(pos uint64) (*MMRElement, error) {
	if element, ok := s.storage.Get(pos); ok {
		return &element, nil
	}
	return nil, nil
}

func (s *MemStorage) append(pos uint64, elements []MMRElement) error {
	for i, element := range elements {
		s.storage.Set(pos+uint64(i), element)
	}
	return nil
}

func (s *MemStorage) commit() error {
	// Do nothing since all changes are automatically commited
	return nil
}

func NewInMemMMR(hasher hash.Hash) *MMR {
	storage := NewMemStorage()
	return NewMMR(0, storage, hasher)
}
