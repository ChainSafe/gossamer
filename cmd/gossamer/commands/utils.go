// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	terminal "golang.org/x/term"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westenddev "github.com/ChainSafe/gossamer/chain/westend-dev"
	westendlocal "github.com/ChainSafe/gossamer/chain/westend-local"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
func setDefaultConfig(chain cfg.Chain) error {
	switch chain {
	case cfg.PolkadotChain:
		config = polkadot.DefaultConfig()
	case cfg.KusamaChain:
		config = kusama.DefaultConfig()
	case cfg.WestendChain:
		config = westend.DefaultConfig()
	case cfg.WestendDevChain:
		config = westenddev.DefaultConfig()
	case cfg.WestendLocalChain:
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

// ParseConfig parses the config from the command line flags
func ParseConfig() error {
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %s", err)
	}

	if err := config.ValidateBasic(); err != nil {
		return fmt.Errorf("error in config file: %v", err)
	}

	return nil
}

// parseBasePath parses the base path from the command line flags
func parseBasePath() error {
	var home string
	// For the base path, prefer the environment variable over the flag
	// If neither are set, use the default base path from the config
	if os.Getenv(DefaultHomeEnv) != "" {
		home = os.Getenv(DefaultHomeEnv)
	} else {
		home = basePath
	}
	if config.BasePath == "" && home == "" {
		return fmt.Errorf("--base-path cannot be empty")
	}
	// If the base path is set, use it
	if home != "" {
		config.BasePath = home
	}
	config.BasePath = utils.ExpandDir(config.BasePath)
	// bind it to viper so that it can be used during the config parsing
	viper.Set("base-path", config.BasePath)

	return nil
}

// parseAccount parses the account key from the command line flags
func parseAccount() {
	// if key is not set, check if alice, bob, or charlie are set
	// return error if none are set
	if key == "" {
		if alice {
			key = "alice"
		} else if bob {
			key = "bob"
		} else if charlie {
			key = "charlie"
		}
	}

	// if key is available, set it in the config
	if key != "" {
		config.Account.Key = key
		// bind it to viper so that it can be used during the config parsing
		viper.Set("account.key", key)
	}
}

// parseRole parses the role from the command line flags
func parseRole() error {
	var selectedRole common.NetworkRole
	if validator {
		selectedRole = common.AuthorityRole
	} else {
		switch role {
		case cfg.FullNode.String():
			selectedRole = common.FullNodeRole
		case cfg.LightNode.String():
			selectedRole = common.LightClientRole
		case cfg.AuthorityNode.String():
			selectedRole = common.AuthorityRole
		default:
			return fmt.Errorf("invalid role: %s", role)
		}
	}

	config.Core.Role = selectedRole
	viper.Set("core.role", config.Core.Role)
	return nil
}

// parseTelemetryURL parses the telemetry-url from the command line flag
func parseTelemetryURL() error {
	if telemetryURLs == "" {
		return nil
	}

	var telemetry []genesis.TelemetryEndpoint
	urlVerbosityPairs := strings.Split(telemetryURLs, ",")
	for _, pair := range urlVerbosityPairs {
		urlVerbosity := strings.Split(pair, ":")
		if len(urlVerbosity) != 2 {
			return fmt.Errorf(
				"invalid --telemetry-url. " +
					"URL and verbosity should be specified as a colon-separated list of key-value pairs",
			)
		}

		url := urlVerbosity[0]
		verbosityString := urlVerbosity[1]
		verbosity, err := strconv.Atoi(verbosityString)
		if err != nil {
			return fmt.Errorf("invalid --telemetry-url. Failed to parse verbosity: %v", err.Error())
		}

		telemetry = append(config.TelemetryURLs, genesis.TelemetryEndpoint{
			Endpoint:  url,
			Verbosity: verbosity,
		})
	}

	viper.Set("telemetry-url", telemetry)

	return nil
}
