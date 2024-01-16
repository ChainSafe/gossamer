// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/stretchr/testify/require"

	"github.com/spf13/viper"

	"github.com/ChainSafe/gossamer/chain/westend"
)

func TestAddStringFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addStringFlagBindViper function
	err := addStringFlagBindViper(cmd, "testFlag", "default", "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetString("testBindName")
	if viperValue != "default" {
		t.Fatalf("Expected Viper value 'testBindName' to be 'default', got '%s'", viperValue)
	}
}

func TestAddIntFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addIntFlagBindViper function
	err := addIntFlagBindViper(cmd, "testFlag", 42, "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetInt("testBindName")
	if viperValue != 42 {
		t.Fatalf("Expected Viper value 'testBindName' to be 42, got '%d'", viperValue)
	}
}

func TestAddBoolFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addBoolFlagBindViper function
	err := addBoolFlagBindViper(cmd, "testFlag", true, "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetBool("testBindName")
	if viperValue != true {
		t.Fatalf("Expected Viper value 'testBindName' to be true, got '%t'", viperValue)
	}
}

func TestAddUintFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addUintFlagBindViper function
	err := addUintFlagBindViper(cmd, "testFlag", 123, "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetUint("testBindName")
	if viperValue != 123 {
		t.Fatalf("Expected Viper value 'testBindName' to be 123, got '%d'", viperValue)
	}
}

func TestAddUint32FlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addUint32FlagBindViper function
	err := addUint32FlagBindViper(cmd, "testFlag", uint32(123), "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetUint32("testBindName")
	if viperValue != 123 {
		t.Fatalf("Expected Viper value 'testBindName' to be 123, got '%d'", viperValue)
	}
}

func TestAddUint16FlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addUint16FlagBindViper function
	err := addUint16FlagBindViper(cmd, "testFlag", uint16(123), "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetUint("testBindName")
	if viperValue != 123 {
		t.Fatalf("Expected Viper value 'testBindName' to be 123, got '%d'", viperValue)
	}
}

func TestAddDurationFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addDurationFlagBindViper function
	err := addDurationFlagBindViper(cmd, "testFlag", 5*time.Second, "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetDuration("testBindName")
	if viperValue != 5*time.Second {
		t.Fatalf("Expected Viper value 'testBindName' to be 5 seconds, got '%v'", viperValue)
	}
}

func TestAddStringSliceFlagBindViper(t *testing.T) {
	// Create a mock command object
	cmd := &cobra.Command{}

	// Call the addStringSliceFlagBindViper function
	err := addStringSliceFlagBindViper(cmd, "testFlag", []string{"default"}, "usage", "testBindName")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify that the flag was added to the command
	flag := cmd.PersistentFlags().Lookup("testFlag")
	if flag == nil {
		t.Fatalf("Expected flag 'testFlag' to be added to command")
	}

	// Verify that the flag is bound to the Viper configuration value
	viperValue := viper.GetStringSlice("testBindName")
	if len(viperValue) != 1 || viperValue[0] != "default" {
		t.Fatalf("Expected Viper value 'testBindName' to be ['default'], got '%v'", viperValue)
	}

	// Set the flag value and verify that the Viper value is updated
	cmd.SetArgs([]string{"--testFlag", "value1", "--testFlag", "value2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	viperValue = viper.GetStringSlice("testBindName")
	if len(viperValue) != 2 || viperValue[0] != "value1" || viperValue[1] != "value2" {
		t.Fatalf("Expected Viper value 'testBindName' to be ['value1', 'value2'], got '%v'", viperValue)
	}
}

func TestBindToViper(t *testing.T) {
	t.Parallel()
	viper.Reset()

	config = westend.DefaultConfig()
	setViperDefault(config)

	tests := []struct {
		name     string
		expected any
		result   any
	}{
		{
			expected: config.Name,
			result:   viper.Get("name"),
		},
		{
			expected: config.ID,
			result:   viper.Get("id"),
		},
		{
			expected: config.DataDir,
			result:   viper.Get("base-path"),
		},
		{
			expected: config.LogLevel,
			result:   viper.Get("log-level"),
		},
		{
			expected: config.Log.Core,
			result:   viper.Get("log.core"),
		},
		{
			expected: config.Core.Role,
			result:   viper.Get("core.role"),
		},
		{
			expected: config.Network.NoMDNS,
			result:   viper.Get("network.no-mdns"),
		},
		{
			expected: config.RPC.Port,
			result:   viper.Get("rpc.port"),
		},
		{
			expected: config.Account.Key,
			result:   viper.Get("account.key"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.result)
		})
	}
}
