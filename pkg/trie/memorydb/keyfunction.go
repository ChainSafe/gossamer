// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memorydb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type KeyFunction[H hashdb.HashOut] func(key H, prefix nibble.Prefix, hasher hashdb.Hasher[H]) H

func HashKey[H hashdb.HashOut](key H, _ nibble.Prefix, _ hashdb.Hasher[H]) H {
	return key
}

func PrefixKey[H hashdb.HashOut](key H, prefix nibble.Prefix, hasher hashdb.Hasher[H]) H {
	newLen := len(key.Bytes()) + len(prefix.PartialKey) + 1
	prefixedKey := make([]byte, 0, newLen)

	prefixedKey = append(prefixedKey, prefix.PartialKey...)
	if prefix.PaddedByte != nil {
		prefixedKey = append(prefixedKey, *prefix.PaddedByte)
	}
	prefixedKey = append(prefixedKey, key.Bytes()...)

	return hasher.FromBytes(prefixedKey)
}
