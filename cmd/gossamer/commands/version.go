// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VersionCmd returns the Gossamer version
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "gossamer version",
	Long:  `gossamer version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("%s version %s\n", config.System.SystemName, config.System.SystemVersion)
		return nil
	},
}
