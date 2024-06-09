// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ChainSafe/gossamer/chain/paseo"

	"github.com/spf13/cobra"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westenddev "github.com/ChainSafe/gossamer/chain/westend-dev"
	westendlocal "github.com/ChainSafe/gossamer/chain/westend-local"
	gssmros "github.com/ChainSafe/gossamer/lib/os"

	"github.com/ChainSafe/gossamer/lib/genesis"

	terminal "golang.org/x/term"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"

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
			continue
		}
		fmt.Printf("\n")
		return password
	}
}

// parseIdentity parses the node identity from the command line flags
func parseIdentity() {
	if name != "" {
		config.Name = name
		viper.Set("name", name)
	}

	if id != "" {
		config.ID = id
		viper.Set("id", id)
	}
}

// parseChainSpec parses the chain spec from the given chain
// and sets the default config
func parseChainSpec(chain string) error {
	// check if the chain is a path to a chain spec
	if _, err := os.Stat(chain); err == nil {
		spec, err := genesis.NewGenesisFromJSONRaw(chain)
		if err != nil {
			return fmt.Errorf("failed to load chain spec: %s", err)
		}
		config = cfg.DefaultConfigFromSpec(spec)
		config.ChainSpec = chain
	} else {
		switch cfg.Chain(chain) {
		case cfg.PolkadotChain:
			config = polkadot.DefaultConfig()
		case cfg.KusamaChain:
			config = kusama.DefaultConfig()
		case cfg.WestendChain:
			config = westend.DefaultConfig()
		case cfg.WestendDevChain:
			config = westenddev.DefaultConfig()
		case cfg.PaseoChain:
			config = paseo.DefaultConfig()
		case cfg.WestendLocalChain:
			if alice || key == "alice" {
				config = westendlocal.DefaultAliceConfig()
			} else if bob || key == "bob" {
				config = westendlocal.DefaultBobConfig()
			} else if charlie || key == "charlie" {
				config = westendlocal.DefaultCharlieConfig()
			} else {
				config = westendlocal.DefaultConfig()
			}
		default:
			return nil
		}
	}

	// parse chain spec and set config fields
	spec, err := genesis.NewGenesisFromJSONRaw(config.ChainSpec)
	if err != nil {
		return fmt.Errorf("failed to load chain spec: %s", err)
	}

	config.Network.Bootnodes = spec.Bootnodes
	config.Network.ProtocolID = spec.ProtocolID
	parseIdentity()

	return nil
}

// configureViper sets up viper to read from the config file and command line flags
func configureViper(basePath string) error {
	viper.SetConfigName("config")                          // name of config file (without extension)
	viper.AddConfigPath(basePath)                          // search `base-path`
	viper.AddConfigPath(filepath.Join(basePath, "config")) // search `base-path/config`

	setViperDefault(config)

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

	return nil
}

// parseBasePath parses the base path from the command line flags
func parseBasePath() error {
	home := basePath
	// For the base path, prefer the environment variable over the flag
	// If neither are set, use the default base path from the config
	if os.Getenv(DefaultHomeEnv) != "" {
		home = os.Getenv(DefaultHomeEnv)
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

	// Ensure that the base path exists and is accessible
	// Create the folders(config, data) in the base path if they don't exist
	if err := cfg.EnsureRoot(config.BasePath); err != nil {
		return fmt.Errorf("failed to ensure root: %s", err)
	}

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

// parseRPC parses the rpc config from the command line flags
func parseRPC() {
	// if rpc modules is not set, set it to the default
	// if rpc modules is set to unsafe, set it to all modules
	//TODO: refactor this to follow the same pattern as substrate
	// Substrate accepts `unsafe`,`safe` and `auto` for --rpc-methods
	config.RPC.Modules = strings.Split(rpcModules, ",")
	if rpcModules == "unsafe" || rpcModules == "" {
		config.RPC.Modules = cfg.DefaultRPCModules
	}

	// bind it to viper so that it can be used during the config parsing
	viper.Set("rpc.modules", config.RPC.Modules)
}

// copyChainSpec copies the chain-spec file to the base path
func copyChainSpec(source, destination string) error {
	if err := gssmros.CopyFile(source, destination); err != nil {
		return fmt.Errorf("failed to copy genesis file: %s", err)
	}
	config.ChainSpec = destination
	// bind it to viper so that it can be used during the config parsing
	viper.Set("chain-spec", destination)
	return nil
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

		config.TelemetryURLs = append(config.TelemetryURLs, genesis.TelemetryEndpoint{
			Endpoint:  url,
			Verbosity: verbosity,
		})
	}

	viper.Set("telemetry-url", config.TelemetryURLs)
	return nil
}

// setViperDefault sets the default values for the config
// The method goes through the config struct and binds each field to viper
// in the format <parent-name>.<field-name> = <field-value>
// The name of the field is taken from the mapstructure tag
func setViperDefault(config *cfg.Config) {
	configType := reflect.TypeOf(*config)
	configValue := reflect.ValueOf(*config)

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag == "" {
			continue
		}

		tagParts := strings.Split(mapstructureTag, ",")
		if len(tagParts) > 0 && tagParts[0] == "-" {
			continue
		}

		parentPrefix := tagParts[0]
		if field.Type.Kind() == reflect.Struct || field.Type.Kind() == reflect.Ptr {
			subType := field.Type
			if subType.Kind() == reflect.Ptr {
				subType = subType.Elem()
			}

			for j := 0; j < subType.NumField(); j++ {
				subField := subType.Field(j)
				subMapstructureTag := subField.Tag.Get("mapstructure")
				if subMapstructureTag == "" {
					continue
				}

				subTagParts := strings.Split(subMapstructureTag, ",")
				if len(subTagParts) > 0 && subTagParts[0] == "-" {
					continue
				}

				prefix := subTagParts[0]
				if parentPrefix != "" {
					prefix = parentPrefix + "." + subTagParts[0]
				}

				var value interface{}
				if configValue.Field(i).Kind() == reflect.Ptr {
					value = configValue.Field(i).Elem().Field(j).Interface()
				} else {
					value = configValue.Field(i).Field(j).Interface()
				}

				if !viper.IsSet(prefix) {
					viper.SetDefault(prefix, value)
				}
			}
		}
	}
}

func parseLogLevel() error {
	// set default log level from config
	moduleToLogLevel := map[string]string{
		"global":  config.LogLevel,
		"core":    config.Log.Core,
		"digest":  config.Log.Digest,
		"sync":    config.Log.Sync,
		"network": config.Log.Network,
		"rpc":     config.Log.RPC,
		"state":   config.Log.State,
		"runtime": config.Log.Runtime,
		"babe":    config.Log.Babe,
		"grandpa": config.Log.Grandpa,
		"wasmer":  config.Log.Wasmer,
	}

	if logLevel != "" {
		logConfigurations := strings.Split(logLevel, ",")
		for _, config := range logConfigurations {
			parts := strings.SplitN(config, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid log configuration: %s", config)
			}

			module := strings.TrimSpace(parts[0])
			logLevel := strings.TrimSpace(parts[1])

			if _, ok := moduleToLogLevel[module]; !ok {
				return fmt.Errorf("invalid module: %s", module)
			}
			moduleToLogLevel[module] = logLevel
		}
	}

	// set global log level
	config.LogLevel = moduleToLogLevel["global"]
	viper.Set("log-level", config.LogLevel)

	// set config.Log
	jsonData, err := json.Marshal(moduleToLogLevel)
	if err != nil {
		return fmt.Errorf("error marshalling logs: %s", err)
	}
	err = json.Unmarshal(jsonData, &config.Log)
	if err != nil {
		return fmt.Errorf("error unmarshalling logs: %s", err)
	}
	viper.Set("log", config.Log)

	return nil
}
