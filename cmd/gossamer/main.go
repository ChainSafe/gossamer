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
