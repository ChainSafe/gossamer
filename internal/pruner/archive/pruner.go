// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package archive

import "github.com/ChainSafe/gossamer/lib/common"

// Pruner is a no-op since we don't prune nodes in archive mode.
type Pruner struct{}

// New returns a new archive mode pruner (no-op).
func New() *Pruner {
	return &Pruner{}
}

// Record is a no-op implementation.
func (*Pruner) Record(_, _ map[common.Hash]struct{}, _ common.Hash, _ uint32) (_ error) {
	return nil
}

// Prune is a no-op implementation.
func (*Pruner) Prune(prunedBlockHashes []common.Hash) (_ error) {
	return nil
}
