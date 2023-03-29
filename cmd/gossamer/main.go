// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"github.com/ChainSafe/gossamer/cmd/gossamer/commands"
)

func main() {
	rootCmd, err := commands.NewRootCommand()
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(
		commands.InitCmd,
		commands.AccountCmd,
		commands.ImportRuntimeCmd,
		commands.BuildSpecCmd,
		commands.PruneStateCmd,
		commands.ImportStateCmd,
	)
	configureCobraCmd(rootCmd, "GSSMR", "gossamer")
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
