// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pruner

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	// Archive pruner mode.
	Archive = Mode("archive")
)

// Mode online pruning mode of historical state tries
type Mode string

// IsValid checks whether the pruning mode is valid
func (p Mode) IsValid() bool {
	switch p {
	case Archive:
		return true
	default:
		return false
	}
}

// Config holds state trie pruning mode and retained blocks
type Config struct {
	Mode           Mode
	RetainedBlocks uint32
}

// Pruner is implemented by FullNode and ArchiveNode.
type Pruner interface {
	StoreJournalRecord(deletedNodeHashes, insertedNodeHashes map[common.Hash]struct{},
		blockHash common.Hash, blockNum int64) error
}

// ArchiveNode is a no-op since we don't prune nodes in archive mode.
type ArchiveNode struct{}

// StoreJournalRecord for archive node doesn't do anything.
func (*ArchiveNode) StoreJournalRecord(_, _ map[common.Hash]struct{},
	_ common.Hash, _ int64) error {
	return nil
}
