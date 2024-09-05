// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mmr

import (
	"github.com/tidwall/btree"
)

// MemStorage provides an in-memory storage mechanism for an MMR.
type MemStorage struct {
	storage *btree.Map[uint64, MMRElement]
}

// NewMemStorage initializes a new instance of MemStorage with an empty storage.
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
	// Do nothing since all changes are automatically committed
	return nil
}
