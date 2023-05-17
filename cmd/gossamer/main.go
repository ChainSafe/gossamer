// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"os"
	"strings"

	"github.com/ChainSafe/gossamer/cmd/gossamer/commands"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		commands.VersionCmd,
	)
	configureCobraCmd("GSSMR")
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("failed to execute root command: %s", err)
		panic(err)
	}
}

// configureCobraCmd configures the cobra command with the given environment prefix and default base path.
func configureCobraCmd(envPrefix string) {
	cobra.OnInitialize(func() {
		if err := initEnv(envPrefix); err != nil {
			return
		}
	})
}

// initEnv sets to use ENV variables if set.
func initEnv(prefix string) error {
	if err := copyEnvVars(prefix); err != nil {
		return err
	}

	// env variables with GSSMR prefix (eg. GSSMR_ROOT)
	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	return nil
}

// copyEnvVars copies all envs like GSSMRROOT to GSSMR_ROOT,
// so we can support both formats.
func copyEnvVars(prefix string) error {
	prefix = strings.ToUpper(prefix)
	ps := prefix + "_"
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			k, v := kv[0], kv[1]
			if strings.HasPrefix(k, prefix) && !strings.HasPrefix(k, ps) {
				k2 := strings.Replace(k, prefix, ps, 1)
				if err := os.Setenv(k2, v); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
