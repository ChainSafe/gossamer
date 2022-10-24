// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import "github.com/ChainSafe/gossamer/lib/common"

// Getter gets deleted or inserted node hashes.
type Getter interface {
	Get() (insertedNodeHashes, deletedNodeHashes map[common.Hash]struct{})
}
