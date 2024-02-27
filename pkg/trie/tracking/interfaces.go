// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import "github.com/ChainSafe/gossamer/lib/common"

// DeletedGetter gets deleted node hashes.
type DeletedGetter interface {
	Deleted() (nodeHashes map[common.Hash]struct{})
}
