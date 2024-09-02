// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

type Iterator[T any] struct {
	items []T
	index int
}

func NewIterator[T any](items []T) *Iterator[T] {
	return &Iterator[T]{items: items, index: -1}
}

func (it *Iterator[T]) Next() *T {
	if it.index < len(it.items)-1 {
		it.index++
		return &it.items[it.index]
	}
	return nil
}

func (it *Iterator[T]) Peek() *T {
	if it.index+1 < len(it.items) {
		return &it.items[it.index+1]
	}
	return nil
}
