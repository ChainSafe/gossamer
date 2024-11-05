// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashdb

import "golang.org/x/exp/constraints"

// A trie node prefix, it is the nibble path from the trie root
// to the trie node.
// For a node containing no partial key value it is the full key.
// For a value node or node containing a partial key, it is the full key minus its node partial
// nibbles (the node key can be split into prefix and node partial).
// Therefore it is always the leftmost portion of the node key, so its internal representation
// is a non expanded byte slice followed by a last padded byte representation.
// The padded byte is an optional padded value.
type Prefix struct {
	Key    []byte
	Padded *byte
}

// An empty prefix constant.
// Can be use when the prefix is not use dinternally or for root nodes.
var EmptyPrefix = Prefix{}

// Hasher is an interface describing an object that can hash a slice of bytes. Used to abstract
// other types over the hashing algorithm. Defines a single hash method and an
// Out associated type with the necessary bounds.
type Hasher[Out constraints.Ordered] interface {
	// Compute the hash of the provided slice of bytes returning the Out type of the Hasher.
	Hash(x []byte) Out
}

// HashDB is an interface modelling datastore keyed by a hash defined by the Hasher.
type HashDB[Hash comparable] interface {
	// Look up a given hash into the bytes that hash to it, returning None if the
	// hash is not known.
	Get(key Hash, prefix Prefix) []byte

	// Check for the existence of a hash-key.
	Contains(key Hash, prefix Prefix) bool

	// Insert a datum item into the DB and return the datum's hash for a later lookup. Insertions
	// are counted and the equivalent number of remove()s must be performed before the data
	// is considered dead.
	Insert(prefix Prefix, value []byte) Hash

	// Like Insert(), except you provide the key and the data is all moved.
	Emplace(key Hash, prefix Prefix, value []byte)

	// Remove a datum previously inserted. Insertions can be "owed" such that the same number of
	// inserts may happen without the data being eventually being inserted into the DB.
	// It can be "owed" more than once.
	Remove(key Hash, prefix Prefix)
}
