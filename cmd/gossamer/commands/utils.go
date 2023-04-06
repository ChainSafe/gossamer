// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westenddev "github.com/ChainSafe/gossamer/chain/westend-dev"
	westendlocal "github.com/ChainSafe/gossamer/chain/westend-local"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"
	"golang.org/x/crypto/ssh/terminal"

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
	// WestendLocalChain is the Westend local chain
	WestendLocalChain Chain = "westend-local"
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

// updateDotConfigFromGenesisData updates the configuration from genesis data of an initialised node
func updateDotConfigFromGenesisData() error {
	// initialise database using data directory
	db, err := utils.SetupDatabase(config.BasePath, false)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	// load genesis data from initialised node database
	gen, err := state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		return fmt.Errorf("failed to load genesis data: %s", err)
	}

	// check genesis id and use genesis id if --id flag not set
	if config.ID == "" {
		config.ID = gen.ID
	}

	// check genesis bootnodes and use genesis --bootnodes if name flag not set
	if len(config.Network.Bootnodes) == 0 {
		config.Network.Bootnodes = common.BytesToStringArray(gen.Bootnodes)
	}

	// check genesis protocol and use genesis --protocol if name flag not set
	if config.Network.ProtocolID == "" {
		config.Network.ProtocolID = gen.ProtocolID
	}

	// close database
	err = db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %s", err)
	}

	logger.Debugf(
		"configuration after genesis data:" +
			" name=" + config.Name +
			" id=" + config.ID +
			" bootnodes=" + strings.Join(config.Network.Bootnodes, ",") +
			" protocol=" + config.Network.ProtocolID,
	)

	return nil
}

// loadBuiltInTestKeys loads the built-in test keys into the keystore
func loadBuiltInTestKeys(accountKey string, ks keystore.GlobalKeystore) (err error) {
	sr25519keyRing, err := keystore.NewSr25519Keyring()
	if err != nil {
		return fmt.Errorf("error creating sr22519 keyring: %s", err)
	}

	ed25519keyRing, err := keystore.NewEd25519Keyring()
	if err != nil {
		return fmt.Errorf("error creating ed25519 keyring: %s", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Acco, sr25519keyRing)
	if err != nil {
		return fmt.Errorf("error loading account keystore: %w", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Babe, sr25519keyRing)
	if err != nil {
		return fmt.Errorf("error loading babe keystore: %w", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Gran, ed25519keyRing)
	if err != nil {
		return fmt.Errorf("error loading grandpa keystore: %w", err)
	}

	return nil
}

// KeypairInserter inserts a keypair.
type KeypairInserter interface {
	Insert(kp keystore.KeyPair) error
}

// unlockKeystore compares the length of passwords to the length of accounts,
// prompts the user for a password if no password is provided, and then unlocks
// the accounts within the provided keystore
func unlockKeystore(ks KeypairInserter, basepath, unlock, password string) error {
	var passwords []string

	if password != "" {
		passwords = strings.Split(password, ",")

		// compare length of passwords to length of accounts to unlock (if password provided)
		if len(passwords) != len(unlock) {
			return fmt.Errorf("passwords length does not match unlock length")
		}

	} else {
		// compare length of passwords to length of accounts to unlock (if password not provided)
		if len(passwords) != len(unlock) {
			bytes := getPassword("Enter password to unlock keystore:")
			password = string(bytes)
		}

		err := keystore.UnlockKeys(ks, basepath, unlock, password)
		if err != nil {
			return fmt.Errorf("failed to unlock keys: %s", err)
		}
	}

	return nil
}

// getPassword prompts user to enter password
func getPassword(msg string) []byte {
	for {
		fmt.Println(msg)
		fmt.Print("> ")
		password, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("invalid input: %s\n", err)
		} else {
			fmt.Printf("\n")
			return password
		}
	}
}

// setDefaultConfig sets the default configuration
func setDefaultConfig(chain Chain) error {
	switch chain {
	case PolkadotChain:
		config = polkadot.DefaultConfig()
	case KusamaChain:
		config = kusama.DefaultConfig()
	case WestendChain:
		config = westend.DefaultConfig()
	case WestendDevChain:
		config = westenddev.DefaultConfig()
	case WestendLocalChain:
		if alice {
			config = westendlocal.DefaultAliceConfig()
		} else if bob {
			config = westendlocal.DefaultBobConfig()
		} else if charlie {
			config = westendlocal.DefaultCharlieConfig()
		} else {
			config = westendlocal.DefaultConfig()
		}
	default:
		return fmt.Errorf("chain %s not supported", chain)
	}

	return nil
}

// configureViper sets up viper to read from the config file and command line flags
func configureViper(basePath string) error {
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
