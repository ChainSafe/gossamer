// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/spf13/cobra"
	"path/filepath"
)

func init() {
	pruneStateCmd.Flags().String("base-path", "", "base path")
	pruneStateCmd.Flags().String("chain", "", "chain id")
	pruneStateCmd.Flags().Uint32("retain-blocks", 512, "number of blocks to retain")
}

var pruneStateCmd = &cobra.Command{
	Use:   "prune-state",
	Short: "Prune state will prune the state trie",
	Long: `prune-state <retain-blocks> will prune historical state data.
All trie nodes that do not belong to the specified version state will be deleted from the database.
The default pruning target is the HEAD-256 state`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := execPruneState(cmd); err != nil {
			return err
		}
		return nil
	},
}

func execPruneState(cmd *cobra.Command) error {
	retainBlocks, err := cmd.Flags().GetUint32("retain-blocks")
	if err != nil {
		return fmt.Errorf("failed to get retain-blocks: %s", err)
	}

	chainID, err := cmd.Flags().GetString("chain")
	if err != nil {
		return fmt.Errorf("failed to get chain: %s", err)
	}

	basePath, err := cmd.Flags().GetString("base-path")
	if err != nil {
		return fmt.Errorf("failed to get base-path: %s", err)
	}

	if chainID == "" && basePath == "" {
		return fmt.Errorf("one of chain or base-path must be specified")
	}

	if basePath == "" {
		switch chainID {
		case "polkadot":
			basePath = polkadot.DefaultBasePath
		case "kusama":
			basePath = kusama.DefaultBasePath
		case "westend":
			basePath = westend.DefaultBasePath
		case "westend-dev":
			basePath = "~/.gossamer/westend-dev"
		default:
			return fmt.Errorf("chain %s not supported", chainID)
		}
	}

	dbPath := filepath.Join(basePath, "db")

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
