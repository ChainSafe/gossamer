// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"fmt"
)

func (t *TrieDB[H, Hasher]) String() string {
	if t.rootHash == (*new(H)) {
		return "empty"
	}

	return fmt.Sprintf("TrieDB: %v", t.rootHash)
}
