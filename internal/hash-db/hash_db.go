package hashdb

import "golang.org/x/exp/constraints"

// Trait describing an object that can hash a slice of bytes. Used to abstract
// other types over the hashing algorithm. Defines a single `hash` method and an
// `Out` associated type with the necessary bounds.
type Hasher[Out constraints.Ordered] interface {
	// Compute the hash of the provided slice of bytes returning the `Out` type of the `Hasher`.
	Hash(x []byte) Out
}
