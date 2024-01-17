// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/node"
)

type TrieLayout[H hashdb.HashOut] interface {
	AllowEmpty() bool
	MaxInlineValue() *uint
	Codec() node.NodeCodec[H]
}
