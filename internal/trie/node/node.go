// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Node is a node in the trie and can be a leaf or a branch.
type Node interface {
	Encode(buffer Buffer) (err error) // TODO change to io.Writer
	EncodeAndHash() (encoding []byte, hash []byte, err error)
	ScaleEncodeHash() (encoding []byte, err error)
	IsDirty() bool
	SetDirty(dirty bool)
	SetKey(key []byte)
	String() string
	SetEncodingAndHash(encoding []byte, hash []byte)
	GetHash() (hash []byte)
	GetKey() (key []byte)
	GetValue() (value []byte)
	GetGeneration() (generation uint64)
	SetGeneration(generation uint64)
	Copy() Node
}
