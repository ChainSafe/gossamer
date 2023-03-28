// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"github.com/ChainSafe/gossamer/cmd/gossamer/commands"
)

func main() {
	rootCmd := commands.RootCmd
	configureCobraCmd(rootCmd, "GSSMR", "gossamer")
	if err := commands.Execute(); err != nil {
		panic(err)
	}
}
