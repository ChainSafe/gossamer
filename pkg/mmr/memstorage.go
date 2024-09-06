// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mmr

import (
	"github.com/tidwall/btree"
)

// MemStorage provides an in-memory storage mechanism for an MMR.
type MemStorage[T any] struct {
	storage *btree.Map[uint64, T]
}

// NewMemStorage initialises a new instance of MemStorage with an empty storage.
func NewMemStorage[T any]() *MemStorage[T] {
	return &MemStorage[T]{
		storage: btree.NewMap[uint64, T](0),
	}
}

//nolint:unparam
func (s *MemStorage[T]) getElement(pos uint64) (*T, error) {
	if element, ok := s.storage.Get(pos); ok {
		return &element, nil
	}
	return nil, nil
}

func (s *MemStorage[T]) append(pos uint64, elements []T) error {
	for i, element := range elements {
		s.storage.Set(pos+uint64(i), element)
	}
	return nil
}

//nolint:unused
func (s *MemStorage[T]) commit() error {
	// Do nothing since all changes are automatically committed
	return nil
}
