package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

func AddStringFlagBindViper(cmd *cobra.Command, name, defaultValue, usage, viperBindName string) error {
	cmd.PersistentFlags().String(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddIntFlagBindViper(cmd *cobra.Command, name string, defaultValue int, usage string, viperBindName string) error {
	cmd.PersistentFlags().Int(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddBoolFlagBindViper(cmd *cobra.Command, name string, defaultValue bool, usage string, viperBindName string) error {
	cmd.PersistentFlags().Bool(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddUintFlagBindViper(cmd *cobra.Command, name string, defaultValue uint, usage string, viperBindName string) error {
	cmd.PersistentFlags().Uint(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddUint64FlagBindViper(cmd *cobra.Command, name string, defaultValue uint64, usage string, viperBindName string) error {
	cmd.PersistentFlags().Uint64(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddUint32FlagBindViper(cmd *cobra.Command, name string, defaultValue uint32, usage string, viperBindName string) error {
	cmd.PersistentFlags().Uint32(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddUint16FlagBindViper(cmd *cobra.Command, name string, defaultValue uint16, usage string, viperBindName string) error {
	cmd.PersistentFlags().Uint16(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddDurationFlagBindViper(cmd *cobra.Command, name string, defaultValue time.Duration, usage string, viperBindName string) error {
	cmd.PersistentFlags().Duration(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}

func AddStringSliceFlagBindViper(cmd *cobra.Command, name string, defaultValue []string, usage string, viperBindName string) error {
	cmd.PersistentFlags().StringSlice(name, defaultValue, usage)
	return viper.BindPFlag(viperBindName, cmd.PersistentFlags().Lookup(name))
}
