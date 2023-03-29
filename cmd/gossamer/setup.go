// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const (
	BasePathFlag = "base-path"
)

// configureCobraCmd configures the cobra command with the given environment prefix and default base path.
func configureCobraCmd(cmd *cobra.Command, envPrefix, defaultBasePath string) {
	cobra.OnInitialize(func() { initEnv(envPrefix) })
	cmd.PersistentPreRunE = concatCobraCmdFuncs(configureViper, cmd.PersistentPreRunE)
}

// initEnv sets to use ENV variables if set.
func initEnv(prefix string) {
	copyEnvVars(prefix)

	// env variables with GSSMR prefix (eg. GSSMR_ROOT)
	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

// copyEnvVars copies all envs like GSSMRROOT to GSSMR_ROOT,
// so we can support both formats.
func copyEnvVars(prefix string) {
	prefix = strings.ToUpper(prefix)
	ps := prefix + "_"
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			k, v := kv[0], kv[1]
			if strings.HasPrefix(k, prefix) && !strings.HasPrefix(k, ps) {
				k2 := strings.Replace(k, prefix, ps, 1)
				os.Setenv(k2, v)
			}
		}
	}
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

// configureViper sets up viper to read from the config file and command line flags
func configureViper(cmd *cobra.Command, args []string) error {
	basePath := viper.GetString(BasePathFlag)
	viper.Set(BasePathFlag, basePath)
	viper.SetConfigName("config")                          // name of config file (without extension)
	viper.AddConfigPath(basePath)                          // search `root-directory`
	viper.AddConfigPath(filepath.Join(basePath, "config")) // search `root-directory/config`

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// ignore not found error, return other errors
			return err
		}
	}

	return nil
}
