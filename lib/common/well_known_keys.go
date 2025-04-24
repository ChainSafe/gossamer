// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import "bytes"

var (
	// CodeKey is the key where runtime code is stored in the trie
	CodeKey = []byte(":code")

	// UpgradedToDualRefKey is set to true (0x01) if the account format has been upgraded to v0.9
	// it's set to empty or false (0x00) otherwise
	UpgradedToDualRefKey = MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7c21aab032aaa6e946ca50ad39ab66603")

	// Prefix of child storage keys
	ChildStorageKeyPrefix = []byte(":child_storage:")
)

func IsChildStorageKey(key []byte) bool {
	return bytes.HasPrefix(key, ChildStorageKeyPrefix)
}
