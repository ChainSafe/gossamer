// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"reflect"

	"github.com/tidwall/btree"
)

type BTreeCodec interface {
	Encode(es *encodeState) error
	Decode(ds *decodeState, dstv reflect.Value) error
}

// BTree is a wrapper around tidwall/btree.BTree that also stores the comparator function and the type of the items
// stored in the BTree. This is needed during decoding because the BTree is a generic type, and we need to know the
// type of the items stored in the BTree in order to decode them.
type BTree struct {
	*btree.BTree
	Comparator func(a, b interface{}) bool
	ItemType   reflect.Type
}

// Encode encodes the BTree using the given encodeState.
func (bt *BTree) Encode(es *encodeState) error {
	// write the number of items in the tree
	err := es.encodeLength(bt.Len())
	if err != nil {
		return err
	}

	bt.Ascend(nil, func(item interface{}) bool {
		err = es.marshal(item)
		return err == nil
	})

	return err
}

// Decode decodes the BTree using the given decodeState.
func (bt *BTree) Decode(ds *decodeState, dstv reflect.Value) error {
	// Decode the number of items in the tree
	length, err := ds.decodeLength()
	if err != nil {
		return fmt.Errorf("decoding BTree length: %w", err)
	}

	if bt.Comparator == nil {
		return fmt.Errorf("no Comparator function provided for BTree")
	}

	if bt.BTree == nil {
		bt.BTree = btree.New(bt.Comparator)
	}

	// Decode each item in the tree
	for i := uint(0); i < length; i++ {
		// Decode the value
		value := reflect.New(bt.ItemType).Elem()
		err = ds.unmarshal(value)
		if err != nil {
			return fmt.Errorf("decoding BTree item: %w", err)
		}

		// convert the value to the correct type for the BTree
		bt.Set(value.Interface())
	}

	dstv.Set(reflect.ValueOf(*bt))
	return nil
}

// Copy returns a copy of the BTree.
func (bt *BTree) Copy() *BTree {
	return &BTree{
		BTree:      bt.BTree.Copy(),
		Comparator: bt.Comparator,
		ItemType:   bt.ItemType,
	}
}

// NewBTree creates a new BTree with the given comparator function.
func NewBTree[T any](comparator func(a, b any) bool) *BTree {
	elementType := reflect.TypeOf((*T)(nil)).Elem()
	return &BTree{
		BTree:      btree.New(comparator),
		Comparator: comparator,
		ItemType:   elementType,
	}
}

// BTreeMap is a wrapper around tidwall/btree.Map
type BTreeMap[K constraints.Ordered, V any] struct {
	*btree.Map[K, V]
	Degree int
}

// Encode encodes the BTreeMap using the given encodeState.
func (btm *BTreeMap[K, V]) Encode(es *encodeState) error {
	// write the number of items in the tree
	err := es.encodeLength(btm.Len())
	if err != nil {
		return err
	}

	// write each item in the tree
	var pivot K
	btm.Ascend(pivot, func(key K, value V) bool {
		if err = es.marshal(key); err != nil {
			return false
		}

		if err = es.marshal(value); err != nil {
			return false
		}

		return true
	})

	return err
}

// Decode decodes the BTreeMap using the given decodeState.
func (btm *BTreeMap[K, V]) Decode(ds *decodeState, dstv reflect.Value) error {
	// Decode the number of items in the tree
	length, err := ds.decodeLength()
	if err != nil {
		return fmt.Errorf("decoding BTreeMap length: %w", err)
	}

	if btm.Map == nil {
		btm.Map = btree.NewMap[K, V](btm.Degree)
	}

	// Decode each item in the tree
	for i := uint(0); i < length; i++ {
		// Decode the key
		keyType := reflect.TypeOf((*K)(nil)).Elem()
		keyInstance := reflect.New(keyType).Elem()
		err = ds.unmarshal(keyInstance)
		if err != nil {
			return fmt.Errorf("decoding BTreeMap key: %w", err)
		}
		key := keyInstance.Interface().(K)

		// Decode the value
		valueType := reflect.TypeOf((*V)(nil)).Elem()
		valueInstance := reflect.New(valueType).Elem()
		err = ds.unmarshal(valueInstance)
		if err != nil {
			return fmt.Errorf("decoding BTreeMap value: %w", err)
		}
		value := valueInstance.Interface().(V)

		btm.Map.Set(key, value)
	}

	dstv.Set(reflect.ValueOf(*btm))
	return nil
}

// Copy returns a copy of the BTreeMap.
func (btm *BTreeMap[K, V]) Copy() BTreeMap[K, V] {
	return BTreeMap[K, V]{
		Map: btm.Map.Copy(),
	}
}

// NewBTreeMap creates a new BTreeMap with the given degree.
func NewBTreeMap[K constraints.Ordered, V any](degree int) *BTreeMap[K, V] {
	return &BTreeMap[K, V]{
		Map: btree.NewMap[K, V](degree),
	}
}
