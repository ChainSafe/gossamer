package btree

import (
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
)

// Map is btree.Map with added methods for SCALE marshalling and unmarshalling
type Map[K constraints.Ordered, V any] btree.Map[K, V]

func NewMap[K constraints.Ordered, V any](degree int) *Map[K, V] {
	m := btree.NewMap[K, V](degree)
	mm := Map[K, V](*m)
	return &mm
}

func (tr *Map[K, V]) Copy() *Map[K, V] {
	m := btree.Map[K, V](*tr)
	copied := m.Copy()
	mm := Map[K, V](*copied)
	return &mm
}

func (tr *Map[K, V]) IsoCopy() *Map[K, V] {
	m := btree.Map[K, V](*tr)
	copied := m.IsoCopy()
	mm := Map[K, V](*copied)
	return &mm
}

// Set or replace a value for a key
func (tr *Map[K, V]) Set(key K, value V) (V, bool) {
	m := btree.Map[K, V](*tr)
	v, b := m.Set(key, value)
	*tr = Map[K, V](m)
	return v, b
}

func (tr *Map[K, V]) Scan(iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.Scan(iter)
}

func (tr *Map[K, V]) ScanMut(iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.ScanMut(iter)
}

// Get a value for key.
func (tr *Map[K, V]) Get(key K) (V, bool) {
	m := btree.Map[K, V](*tr)
	return m.Get(key)
}

// GetMut gets a value for key.
// If needed, this may perform a copy the resulting value before returning.
//
// Mut methods are only useful when all of the following are true:
//   - The interior data of the value requires changes.
//   - The value is a pointer type.
//   - The BTree has been copied using `Copy()` or `IsoCopy()`.
//   - The value itself has a `Copy()` or `IsoCopy()` method.
//
// Mut methods may modify the tree structure and should have the same
// considerations as other mutable operations like Set, Delete, Clear, etc.
func (tr *Map[K, V]) GetMut(key K) (V, bool) {
	m := btree.Map[K, V](*tr)
	v, b := m.GetMut(key)
	*tr = Map[K, V](m)
	return v, b
}

// Len returns the number of items in the tree
func (tr *Map[K, V]) Len() int {
	m := btree.Map[K, V](*tr)
	return m.Len()
}

// Delete a value for a key and returns the deleted value.
// Returns false if there was no value by that key found.
func (tr *Map[K, V]) Delete(key K) (V, bool) {
	m := btree.Map[K, V](*tr)
	v, b := m.Delete(key)
	*tr = Map[K, V](m)
	return v, b
}

// Ascend the tree within the range [pivot, last]
// Pass nil for pivot to scan all item in ascending order
// Return false to stop iterating
func (tr *Map[K, V]) Ascend(pivot K, iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.Ascend(pivot, iter)
}

func (tr *Map[K, V]) AscendMut(pivot K, iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.AscendMut(pivot, iter)
	*tr = Map[K, V](m)
}

func (tr *Map[K, V]) Reverse(iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.Reverse(iter)
}

func (tr *Map[K, V]) ReverseMut(iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.Reverse(iter)
	*tr = Map[K, V](m)
}

// Descend the tree within the range [pivot, first]
// Pass nil for pivot to scan all item in descending order
// Return false to stop iterating
func (tr *Map[K, V]) Descend(pivot K, iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.Descend(pivot, iter)
}

func (tr *Map[K, V]) DescendMut(pivot K, iter func(key K, value V) bool) {
	m := btree.Map[K, V](*tr)
	m.DescendMut(pivot, iter)
	*tr = Map[K, V](m)
}

// Load is for bulk loading pre-sorted items
func (tr *Map[K, V]) Load(key K, value V) (V, bool) {
	m := btree.Map[K, V](*tr)
	v, b := m.Load(key, value)
	*tr = Map[K, V](m)
	return v, b
}

// Min returns the minimum item in tree.
// Returns nil if the treex has no items.
func (tr *Map[K, V]) Min() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	return m.Min()
}

func (tr *Map[K, V]) MinMut() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	k, v, b := m.MinMut()
	*tr = Map[K, V](m)
	return k, v, b
}

// Max returns the maximum item in tree.
// Returns nil if the tree has no items.
func (tr *Map[K, V]) Max() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	return m.Max()
}

func (tr *Map[K, V]) MaxMut() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	k, v, b := m.MaxMut()
	*tr = Map[K, V](m)
	return k, v, b
}

// PopMin removes the minimum item in tree and returns it.
// Returns nil if the tree has no items.
func (tr *Map[K, V]) PopMin() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	return m.PopMin()
}

// PopMax removes the maximum item in tree and returns it.
// Returns nil if the tree has no items.
func (tr *Map[K, V]) PopMax() (K, V, bool) {
	m := btree.Map[K, V](*tr)
	return m.PopMax()
}

// GetAt returns the value at index.
// Return nil if the tree is empty or the index is out of bounds.
func (tr *Map[K, V]) GetAt(index int) (K, V, bool) {
	m := btree.Map[K, V](*tr)
	return m.GetAt(index)
}

func (tr *Map[K, V]) GetAtMut(index int) (K, V, bool) {
	m := btree.Map[K, V](*tr)
	k, v, b := m.GetAtMut(index)
	*tr = Map[K, V](m)
	return k, v, b
}

// DeleteAt deletes the item at index.
// Return nil if the tree is empty or the index is out of bounds.
func (tr *Map[K, V]) DeleteAt(index int) (K, V, bool) {
	m := btree.Map[K, V](*tr)
	k, v, b := m.DeleteAt(index)
	*tr = Map[K, V](m)
	return k, v, b
}

// Height returns the height of the tree.
// Returns zero if tree has no items.
func (tr *Map[K, V]) Height() int {
	m := btree.Map[K, V](*tr)
	return m.Height()
}

// Iter returns a read-only iterator.
func (tr *Map[K, V]) Iter() btree.MapIter[K, V] {
	m := btree.Map[K, V](*tr)
	return m.Iter()
}

func (tr *Map[K, V]) IterMut() btree.MapIter[K, V] {
	m := btree.Map[K, V](*tr)
	iter := m.IterMut()
	*tr = Map[K, V](m)
	return iter
}

// Values returns all the values in order.
func (tr *Map[K, V]) Values() []V {
	m := btree.Map[K, V](*tr)
	return m.Values()
}

func (tr *Map[K, V]) ValuesMut() []V {
	m := btree.Map[K, V](*tr)
	values := m.ValuesMut()
	*tr = Map[K, V](m)
	return values
}

// Keys returns all the keys in order.
func (tr *Map[K, V]) Keys() []K {
	m := btree.Map[K, V](*tr)
	return m.Keys()
}

// KeyValues returns all the keys and values in order.
func (tr *Map[K, V]) KeyValues() ([]K, []V) {
	m := btree.Map[K, V](*tr)
	return m.KeyValues()
}

func (tr *Map[K, V]) KeyValuesMut() ([]K, []V) {
	m := btree.Map[K, V](*tr)
	keys, values := m.KeyValuesMut()
	*tr = Map[K, V](m)
	return keys, values
}

// Clear will delete all items.
func (tr *Map[K, V]) Clear() {
	m := btree.Map[K, V](*tr)
	m.Clear()
	*tr = Map[K, V](m)
}

// MarshalSCALE is used by scale package for marshalling
func (tr *Map[K, V]) MarshalSCALE() ([]byte, error) {
	m := btree.Map[K, V](*tr)
	// load into map to be marshalled
	mapped := make(map[K]V)
	m.Scan(func(key K, value V) bool {
		mapped[key] = value
		return true
	})
	return scale.Marshal(mapped)
}

// UnmarshalSCALE is used by scale package for unmarshalling
func (tr *Map[K, V]) UnmarshalSCALE(r io.Reader) error {
	m := btree.Map[K, V](*tr)
	decoder := scale.NewDecoder(r)
	mapped := make(map[K]V)
	err := decoder.Decode(&mapped)
	if err != nil {
		return err
	}
	for k, v := range mapped {
		m.Set(k, v)
	}
	*tr = Map[K, V](m)
	return nil
}

var (
	_ scale.Marshaler   = &Map[string, string]{}
	_ scale.Unmarshaler = &Map[string, string]{}
)
