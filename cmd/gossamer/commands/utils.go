// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Chain is a string representing a chain
type Chain string

const (
	// PolkadotChain is the Polkadot chain
	PolkadotChain Chain = "polkadot"
	// KusamaChain is the Kusama chain
	KusamaChain Chain = "kusama"
	// WestendChain is the Westend chain
	WestendChain Chain = "westend"
	// WestendDevChain is the Westend dev chain
	WestendDevChain Chain = "westend-dev"
)

// String returns the string representation of the chain
func (c Chain) String() string {
	return string(c)
}

// addStringFlagBindViper adds a string flag to the given command and binds it to the given viper name
func addStringFlagBindViper(cmd *cobra.Command,
	name,
	defaultValue,
	usage,
	viperBindName string,
) error {
	cmd.PersistentFlags().String(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addIntFlagBindViper adds an int flag to the given command and binds it to the given viper name
func addIntFlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue int,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Int(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addBoolFlagBindViper adds a bool flag to the given command and binds it to the given viper name
func addBoolFlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue bool,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Bool(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addUintFlagBindViper adds a uint flag to the given command and binds it to the given viper name
func addUintFlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue uint,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Uint(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addUint64FlagBindViper adds a uint64 flag to the given command and binds it to the given viper name
func addUint64FlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue uint64,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Uint64(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addUint32FlagBindViper adds a uint32 flag to the given command and binds it to the given viper name
func addUint32FlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue uint32,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Uint32(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addUint16FlagBindViper adds a uint16 flag to the given command and binds it to the given viper name
func addUint16FlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue uint16,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Uint16(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addDurationFlagBindViper adds a duration flag to the given command and binds it to the given viper name
func addDurationFlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue time.Duration,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().Duration(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

// addStringSliceFlagBindViper adds a string slice flag to the given command and binds it to the given viper name
func addStringSliceFlagBindViper(
	cmd *cobra.Command,
	name string,
	defaultValue []string,
	usage string,
	viperBindName string,
) error {
	cmd.PersistentFlags().StringSlice(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}
