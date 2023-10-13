// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"reflect"

	"github.com/tidwall/btree"
)

// BTree is a wrapper around tidwall/btree.BTree that also stores the comparator function and the type of the items
// stored in the BTree. This is needed during decoding because the BTree is a generic type, and we need to know the
// type of the items stored in the BTree in order to decode them.
type BTree struct {
	*btree.BTree
	Comparator func(a, b interface{}) bool
	ItemType   reflect.Type
}

// NewBTree creates a new BTree with the given comparator function.
func NewBTree[T any](comparator func(a, b any) bool) BTree {
	// There's no instantiation overhead of the actual type T because we're only creating a slice type and
	// getting the element type from it.
	var dummySlice []T
	elementType := reflect.TypeOf(dummySlice).Elem()

	return BTree{
		BTree:      btree.New(comparator),
		Comparator: comparator,
		ItemType:   elementType,
	}
}
