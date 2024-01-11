// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/spf13/cobra"
)

func init() {
	PruneStateCmd.Flags().String("chain", "", "chain id")
	PruneStateCmd.Flags().Uint32("retain-blocks", 512, "number of blocks to retain")
}

// PruneStateCmd is the command to prune the state trie
var PruneStateCmd = &cobra.Command{
	Use:   "prune-state",
	Short: "Prune state will prune the state trie",
	Long: `prune-state <retain-blocks> will prune historical state data.
All trie nodes that do not belong to the specified version state will be deleted from the database.
The default pruning target is the HEAD-256 state`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execPruneState(cmd)
	},
}

// execPruneState executes the prune-state command
func execPruneState(cmd *cobra.Command) error {
	retainBlocks, err := cmd.Flags().GetUint32("retain-blocks")
	if err != nil {
		return fmt.Errorf("failed to get retain-blocks: %s", err)
	}

	dbPath := filepath.Join(config.DataDir, "db")
	const uint32Max = ^uint32(0)
	if uint32Max < retainBlocks {
		return fmt.Errorf("retain blocks value overflows uint32 boundaries, must be less than or equal to: %d", uint32Max)
	}

	pruner, err := state.NewOfflinePruner(dbPath, retainBlocks)
	if err != nil {
		return err
	}

	logger.Info("Offline pruner initialised")

	err = pruner.SetBloomFilter()
	if err != nil {
		return fmt.Errorf("failed to set keys into bloom filter: %w", err)
	}

	err = pruner.Prune()
	if err != nil {
		return fmt.Errorf("failed to prune: %w", err)
	}

	return nil
}
