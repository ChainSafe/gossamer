// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package cache

type TrieCache interface {
	GetValue(key []byte) []byte
	SetValue(key []byte, value []byte)
	GetNode(key []byte) []byte
	SetNode(key []byte, value []byte)
}
