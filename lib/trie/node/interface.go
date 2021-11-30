// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"github.com/ChainSafe/gossamer/lib/trie/encode"
)

// Node is node in the trie and can be a leaf or a branch.
type Node interface {
	Encode(buffer encode.Buffer) (err error) // TODO change to io.Writer
	EncodeAndHash() ([]byte, []byte, error)
	ScaleEncodeHash() (b []byte, err error)
	IsDirty() bool
	SetDirty(dirty bool)
	SetKey(key []byte)
	String() string
	SetEncodingAndHash([]byte, []byte)
	GetHash() []byte
	GetGeneration() uint64
	SetGeneration(uint64)
	Copy() Node
}
