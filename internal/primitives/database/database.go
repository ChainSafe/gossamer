// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import "github.com/ChainSafe/gossamer/internal/primitives/runtime"

// ColumnID is an identifier for a column.
type ColumnID uint32

// Change is an alteration to the database.
type Change[H any] any

// ChangeTypes is the interface constraint which can be a Change
type ChangeTypes[H any] interface {
	ChangeSet | ChangeRemove | ChangeStore[H] | ChangeReference[H] | ChangeRelease[H]
}

// NewChange is the constructor for Change
func NewChange[H any, CT ChangeTypes[H]](change CT) Change[H] {
	return Change[H](change)
}

// ChangeSet sets a key in column to a value
type ChangeSet struct {
	ColumnID
	Key   []byte
	Value []byte
}

// ChangeRemove removes the value of a key in column
type ChangeRemove struct {
	ColumnID
	Key []byte
}

// ChangeStore will store the preimage of hash
type ChangeStore[H any] struct {
	ColumnID
	Hash     H
	Preimage []byte
}

// ChangeReference will increase the number of references for hash
type ChangeReference[H any] struct {
	ColumnID
	Hash H
}

// ChangeRelease will release the preimage of hash
type ChangeRelease[H any] struct {
	ColumnID
	Hash H
}

// Transaction is a series of changes to the database that can be committed atomically. They do not take effect until
// passed into `Database.Commit`.
type Transaction[H any] []Change[H]

// Set the value of `key` in `col` to `value`, replacing anything that is there currently.
func (t *Transaction[H]) Set(col ColumnID, key []byte, value []byte) {
	*t = append(*t, NewChange[H](ChangeSet{col, key, value}))
}

// Remove the value of `key` in `col`.
func (t *Transaction[H]) Remove(col ColumnID, key []byte) {
	*t = append(*t, NewChange[H](ChangeRemove{col, key}))
}

// Store the `preimage` of `hash` into the database, so that it may be looked up later with
// `Database.Get`. This may be called multiple times, but subsequent
// calls will ignore `preimage` and simply increase the number of references on `hash`.
func (t *Transaction[H]) Store(col ColumnID, hash H, preimage []byte) {
	*t = append(*t, NewChange[H](ChangeStore[H]{col, hash, preimage}))
}

// Reference will increase the number of references for `hash` in the database.
func (t *Transaction[H]) Reference(col ColumnID, hash H) {
	*t = append(*t, NewChange[H](ChangeReference[H]{col, hash}))
}

// Release the preimage of `hash` from the database. An equal number of these to the number of corresponding `store`s
// must have been given before it is legal for `Database::get` to be unable to provide the preimage.
func (t *Transaction[H]) Release(col ColumnID, hash H) {
	*t = append(*t, NewChange[H](ChangeRelease[H]{col, hash}))
}

// Database is the interface to commit transactions as well as retrieve values
type Database[H runtime.Hash] interface {
	// Commit the `transaction` to the database atomically. Any further calls to `get` or `lookup`
	// will reflect the new state.
	Commit(transaction Transaction[H]) error

	// Retrieve the value previously stored against `key` or `nil` if `key` is not currently in the database.
	Get(col ColumnID, key []byte) []byte

	// Check if the value exists in the database without retrieving it.
	Contains(col ColumnID, key []byte) bool
}
