// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

type TrieLayout[Out node.HashOut] interface {
	UseExtension() bool
	AllowEmpty() bool
	MaxInlineValue() *uint
	Hasher() hashdb.Hasher[Out]
	Codec() node.NodeCodec[Out]
}
