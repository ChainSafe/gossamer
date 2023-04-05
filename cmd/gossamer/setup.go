// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// BasePathFlag is the base path flag
	BasePathFlag = "base-path"
)

// configureCobraCmd configures the cobra command with the given environment prefix and default base path.
func configureCobraCmd(cmd *cobra.Command, envPrefix string) {
	cobra.OnInitialize(func() {
		if err := initEnv(envPrefix); err != nil {
			return
		}
	})
	cmd.PersistentPreRunE = concatCobraCmdFuncs(cmd.PersistentPreRunE)
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

type cobraCmdFunc func(cmd *cobra.Command, args []string) error

// concatCobraCmdFuncs concatenates the given cobra command functions into a single function.
func concatCobraCmdFuncs(fs ...cobraCmdFunc) cobraCmdFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range fs {
			if f != nil {
				if err := f(cmd, args); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
