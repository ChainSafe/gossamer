// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/internal/log"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westenddev "github.com/ChainSafe/gossamer/chain/westend-dev"
	westendlocal "github.com/ChainSafe/gossamer/chain/westend-local"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	terminal "golang.org/x/term"
)

// Package level variables
var (
	config = westenddev.DefaultConfig()
	logger = log.NewFromGlobal(log.AddContext("pkg", "cmd"))
)

// Flag values for the root command which needs type conversion
var (
	// Base Config
	logLevelGlobal string
	pruning        string

	// Log Config
	logLevelCore    string
	logLevelDigest  string
	logLevelSync    string
	logLevelNetwork string
	logLevelRPC     string
	logLevelState   string
	logLevelRuntime string
	logLevelBABE    string
	logLevelGRANDPA string
)

// Flag values for persistent flags
var (
	// Default accounts
	alice   bool
	bob     bool
	charlie bool

	// Initialization flags for node
	chain    string
	basePath string
)

// Default values
const (
	// DefaultHomeEnv is the default environment variable for the base path
	DefaultHomeEnv = "GSSMRHOME"
)

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
	if config.Account.Key == "" {
		var key string
		if alice {
			key = "alice"
		} else if bob {
			key = "bob"
		} else if charlie {
			key = "charlie"
		}

		config.Account.Key = key
	}

	// bind it to viper so that it can be used during the config parsing
	viper.Set("account.key", config.Account.Key)
}

// NewRootCommand creates the root command
func NewRootCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "gossamer",
		Short: "Official gossamer command-line interface",
		Long: `Gossamer is a Golang implementation of the Polkadot Host.
Usage:
	gossamer --chain westend-local --alice --babe-lead
	gossamer --chain westend-dev --key alice --port 7002
	gossamer --chain westend --key bob --port 7003
	gossamer --chain kusama --key charlie --port 7004
	gossamer --chain polkadot --key dave --port 7005`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return execRoot(cmd)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := setDefaultConfig(Chain(chain)); err != nil {
				return fmt.Errorf("failed to set default config: %s", err)
			}

			if cmd.Name() == "gossamer" || cmd.Name() == "init" {
				if err := parseBasePath(); err != nil {
					return fmt.Errorf("failed to parse base path: %s", err)
				}

				parseAccount()

				if cmd.Name() == "gossamer" {
					if err := configureViper(config.BasePath); err != nil {
						return fmt.Errorf("failed to configure viper: %s", err)
					}

					if err := ParseConfig(); err != nil {
						return fmt.Errorf("failed to parse config: %s", err)
					}
				}
			}

			return nil
		},
	}

	if err := addRootFlags(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// addRootFlags adds the root flags to the command
func addRootFlags(cmd *cobra.Command) error {
	// global flags
	cmd.PersistentFlags().StringVar(&basePath,
		"base-path",
		"",
		"The base path for the node. Defaults to $GSSMRHOME if set")
	cmd.PersistentFlags().StringVar(&chain,
		"chain",
		WestendLocalChain.String(),
		"The default chain configuration to load. Example: --chain kusama")

	// Base Config
	if err := addBaseConfigFlags(cmd); err != nil {
		return fmt.Errorf("failed to add base config flags: %s", err)
	}

	// Log Config
	addLogFlags(cmd)

	// Account Config
	if err := addAccountFlags(cmd); err != nil {
		return fmt.Errorf("failed to add account flags: %s", err)
	}

	// Network Config
	if err := addNetworkFlags(cmd); err != nil {
		return fmt.Errorf("failed to add network flags: %s", err)
	}

	// Core Config
	if err := addCoreFlags(cmd); err != nil {
		return fmt.Errorf("failed to add core flags: %s", err)
	}

	// State Config
	if err := addStateFlags(cmd); err != nil {
		return fmt.Errorf("failed to add state flags: %s", err)
	}

	// RPC Config
	if err := addRPCFlags(cmd); err != nil {
		return fmt.Errorf("failed to add rpc flags: %s", err)
	}

	// pprof Config
	addPprofFlags(cmd)

	return nil
}

// addBaseConfigFlags adds the base config flags to the command
func addBaseConfigFlags(cmd *cobra.Command) error {
	if err := addStringFlagBindViper(cmd,
		"name",
		config.BaseConfig.Name,
		"Name of the node",
		"Name of the node"); err != nil {
		return fmt.Errorf("failed to add --name flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"id", config.BaseConfig.ID,
		"Identifier for the node",
		"id"); err != nil {
		return fmt.Errorf("failed to add --id flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"genesis", config.BaseConfig.Genesis,
		"path to the genesis file",
		"genesis"); err != nil {
		return fmt.Errorf("failed to add --genesis flag: %s", err)
	}
	if err := addBoolFlagBindViper(cmd,
		"no-telemetry",
		config.BaseConfig.NoTelemetry,
		"Disable connecting to the Substrate telemetry server",
		"no-telemetry"); err != nil {
		return fmt.Errorf("failed to add --no-telemetry flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"metrics-address",
		config.BaseConfig.MetricsAddress,
		"Listen address of the metric server",
		"metrics-address"); err != nil {
		return fmt.Errorf("failed to add --metrics-address flag: %s", err)
	}
	if err := addUint32FlagBindViper(cmd,
		"retain-blocks",
		config.BaseConfig.RetainBlocks,
		"Retain number of block from latest block while pruning",
		"retain-blocks"); err != nil {
		return fmt.Errorf("failed to add --retain-blocks flag: %s", err)
	}
	cmd.Flags().StringVar(&logLevelGlobal,
		"log", config.BaseConfig.LogLevel,
		"Global log level. Supports levels critical (silent), error, warn, info, debug and trace")
	cmd.Flags().StringVar(&pruning,
		"state-pruning",
		string(config.BaseConfig.Pruning),
		"State trie online pruning")
	cmd.Flags().BoolVar(&config.PublishMetrics,
		"publish-metrics",
		config.BaseConfig.PublishMetrics,
		"Publish metrics to prometheus")

	// TODO: telemetry-url

	return nil
}

// addLogFlags adds the log flags to the command
func addLogFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&logLevelCore, "lcore", config.Log.Core, "Core module log level")
	cmd.Flags().StringVar(&logLevelDigest, "ldigest", config.Log.Digest, "Digest module log level")
	cmd.Flags().StringVar(&logLevelSync, "lsync", config.Log.Sync, "Sync module log level")
	cmd.Flags().StringVar(&logLevelNetwork, "lnetwork", config.Log.Network, "Network module log level")
	cmd.Flags().StringVar(&logLevelRPC, "lrpc", config.Log.RPC, "RPC module log level")
	cmd.Flags().StringVar(&logLevelState, "lstate", config.Log.State, "State module log level")
	cmd.Flags().StringVar(&logLevelRuntime, "lruntime", config.Log.Runtime, "Runtime module log level")
	cmd.Flags().StringVar(&logLevelBABE, "lbabe", config.Log.Babe, "BABE module log level")
	cmd.Flags().StringVar(&logLevelGRANDPA, "lgrandpa", config.Log.Grandpa, "GRANDPA module log level")
}

// addAccountFlags adds account flags and binds to viper
func addAccountFlags(cmd *cobra.Command) error {
	if err := addStringFlagBindViper(cmd,
		"key",
		config.Account.Key,
		"Keyring to use for the node",
		"account.key"); err != nil {
		return fmt.Errorf("failed to add --key flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"unlock",
		config.Account.Unlock,
		"Unlock an account. eg. --unlock=0 to unlock account 0.",
		"account.unlock"); err != nil {
		return fmt.Errorf("failed to add --unlock flag: %s", err)
	}

	// Default Account flags
	cmd.PersistentFlags().BoolVar(&alice,
		"alice",
		false,
		"use Alice's key")
	cmd.PersistentFlags().BoolVar(&bob,
		"bob",
		false,
		"use Bob's key")
	cmd.PersistentFlags().BoolVar(&charlie,
		"charlie",
		false,
		"use Charlie's key")

	cmd.Flags().String(
		"password",
		"",
		"Password used to encrypt the keystore")

	return nil
}

// addNetworkFlags adds network flags and binds to viper
func addNetworkFlags(cmd *cobra.Command) error {
	if err := addUint16FlagBindViper(cmd,
		"port",
		config.Network.Port,
		"Network port to use",
		"network.port"); err != nil {
		return fmt.Errorf("failed to add --port flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"bootnodes",
		config.Network.Bootnodes,
		"Comma separated node URLs for network discovery bootstrap",
		"network.bootnodes"); err != nil {
		return fmt.Errorf("failed to add --bootnodes flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"protocol-id",
		config.Network.ProtocolID,
		"Protocol ID to use",
		"network.protocol-id"); err != nil {
		return fmt.Errorf("failed to add --protocol-id flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"no-bootstrap",
		config.Network.NoBootstrap,
		"Disables network bootstrapping (mDNS still enabled)",
		"network.no-bootstrap"); err != nil {
		return fmt.Errorf("failed to add --no-bootstrap flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"no-mdns", config.Network.NoMDNS,
		"Disables network mDNS discovery",
		"network.no-mdns"); err != nil {
		return fmt.Errorf("failed to add --no-mdns flag: %s", err)
	}

	if err := addIntFlagBindViper(cmd,
		"min-peers",
		config.Network.MinPeers,
		"Minimum number of peers to connect to",
		"network.min-peers"); err != nil {
		return fmt.Errorf("failed to add --min-peers flag: %s", err)
	}

	if err := addIntFlagBindViper(cmd,
		"max-peers",
		config.Network.MaxPeers,
		"Maximum number of peers to connect to",
		"network.max-peers"); err != nil {
		return fmt.Errorf("failed to add --max-peers flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"persistent-peers",
		config.Network.PersistentPeers,
		"Comma separated list of peers to always keep connected to",
		"network.persistent-peers"); err != nil {
		return fmt.Errorf("failed to add --persistent-peers flag: %s", err)
	}

	if err := addDurationFlagBindViper(cmd,
		"discovery-interval",
		config.Network.DiscoveryInterval,
		"Interval to perform peer discovery",
		"network.discovery-interval"); err != nil {
		return fmt.Errorf("failed to add --discovery-interval flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"public-ip",
		config.Network.PublicIP,
		"Overrides the public IP address used for peer to peer networking",
		"network.public-ip"); err != nil {
		return fmt.Errorf("failed to add --public-ip flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"public-dns",
		config.Network.PublicDNS,
		"Overrides public DNS used for peer to peer networking",
		"network.public-dns"); err != nil {
		return fmt.Errorf("failed to add --public-dns flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"node-key",
		config.Network.NodeKey,
		"Overrides the secret Ed25519 key to use for libp2p networking",
		"network.node-key"); err != nil {
		return fmt.Errorf("failed to add --node-key flag: %s", err)
	}

	return nil
}

// addRPCFlags adds rpc flags and binds to viper
func addRPCFlags(cmd *cobra.Command) error {
	if err := addBoolFlagBindViper(cmd,
		"rpc-enabled",
		config.RPC.Enabled,
		"Enable the HTTP-RPC server",
		"rpc.enabled"); err != nil {
		return fmt.Errorf("failed to add --rpc-enabled flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"rpc-unsafe",
		config.RPC.Unsafe,
		"Enable the HTTP-RPC server to unsafe procedures",
		"rpc.unsafe"); err != nil {
		return fmt.Errorf("failed to add --rpc-unsafe flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"unsafe-rpc-external",
		config.RPC.UnsafeExternal,
		"Enable external HTTP-RPC connections to unsafe procedures",
		"rpc.unsafe-external"); err != nil {
		return fmt.Errorf("failed to add --unsafe-rpc-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"rpc-external",
		config.RPC.External,
		"Enable external HTTP-RPC connections",
		"rpc.external"); err != nil {
		return fmt.Errorf("failed to add --rpc-external flag: %s", err)
	}

	if err := addUint32FlagBindViper(cmd,
		"rpc-port",
		config.RPC.Port,
		"HTTP-RPC server listening port",
		"rpc.port"); err != nil {
		return fmt.Errorf("failed to add --rpc-port flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"rpc-host",
		config.RPC.Host,
		"HTTP-RPC server listening hostname",
		"rpc.host"); err != nil {
		return fmt.Errorf("failed to add --rpc-host flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"rpc-methods",
		config.RPC.Modules,
		"API modules to enable via HTTP-RPC, comma separated list",
		"rpc.modules"); err != nil {
		return fmt.Errorf("failed to add --rpc-methods flag: %s", err)
	}

	if err := addUint32FlagBindViper(cmd,
		"ws-port",
		config.RPC.WSPort,
		"Websockets server listening port",
		"rpc.ws-port"); err != nil {
		return fmt.Errorf("failed to add --ws-port flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws", config.RPC.WS,
		"Enable the websockets server",
		"rpc.ws"); err != nil {
		return fmt.Errorf("failed to add --ws flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-external",
		config.RPC.WSExternal,
		"Enable external websocket connections",
		"rpc.ws-external"); err != nil {
		return fmt.Errorf("failed to add --ws-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-unsafe",
		config.RPC.WSUnsafe,
		"Enable access to websocket unsafe calls",
		"rpc.ws-unsafe"); err != nil {
		return fmt.Errorf("failed to add --ws-unsafe flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-unsafe-external",
		config.RPC.WSUnsafeExternal,
		"Enable external access to websocket unsafe calls",
		"rpc.ws-unsafe-external"); err != nil {
		return fmt.Errorf("failed to add --ws-unsafe-external flag: %s", err)
	}

	return nil
}

// addCoreFlags adds core flags and binds to viper
func addCoreFlags(cmd *cobra.Command) error {
	// TODO: role

	if err := addBoolFlagBindViper(cmd,
		"babe-authority",
		config.Core.BabeAuthority,
		"Run as a BABE authority",
		"core.babe-authority"); err != nil {
		return fmt.Errorf("failed to add --babe-authority flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"grandpa-authority",
		config.Core.GrandpaAuthority,
		"Run as a GRANDPA authority",
		"core.grandpa-authority"); err != nil {
		return fmt.Errorf("failed to add --grandpa-authority flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"wasm-interpreter",
		config.Core.WasmInterpreter,
		"WASM interpreter",
		"core.wasm-interpreter"); err != nil {
		return fmt.Errorf("failed to add --wasm-interpreter flag: %s", err)
	}

	if err := addDurationFlagBindViper(cmd,
		"grandpa-interval",
		config.Core.GrandpaInterval,
		"GRANDPA voting period in seconds",
		"core.grandpa-interval"); err != nil {
		return fmt.Errorf("failed to add --grandpa-interval flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"babe-lead",
		config.Core.BABELead,
		"Run as a BABE authority and produce blocks",
		"core.babe-lead"); err != nil {
		return fmt.Errorf("failed to add --babe-lead flag: %s", err)
	}

	return nil
}

// addStateFlags adds state flags and binds to viper
func addStateFlags(cmd *cobra.Command) error {
	if err := addUintFlagBindViper(cmd,
		"rewind", config.State.Rewind,
		"Rewind head of chain to the given block number",
		"state.rewind"); err != nil {
		return fmt.Errorf("failed to add --rewind flag: %s", err)
	}

	return nil
}

// addPprofFlags adds pprof flags and binds to viper
func addPprofFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("pprof.enabled",
		config.Pprof.Enabled,
		"enabled")
	cmd.Flags().String("pprof.listening-address",
		config.Pprof.ListeningAddress,
		"Address to listen on for pprof")
	cmd.Flags().Int("pprof.block-profile-rate",
		config.Pprof.BlockProfileRate,
		"The frequency at which the Go runtime samples the state of goroutines to generate block profile information.")
	cmd.Flags().Int("pprof.mutex-profile-rate",
		config.Pprof.MutexProfileRate,
		"The frequency at which the Go runtime samples the state of mutexes to generate mutex profile information.")
}

// execRoot executes the root command
func execRoot(cmd *cobra.Command) error {
	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return fmt.Errorf("failed to get password: %s", err)
	}

	// if the node is not initialised, initialise it
	if !dot.IsNodeInitialised(config.BasePath) {
		if err := dot.InitNode(config); err != nil {
			logger.Errorf("failed to initialise node: %s", err)
			return err
		}
	}

	// ensure configuration matches genesis data stored during node initialization
	// but do not overwrite configuration if the corresponding flag value is set
	if err := updateDotConfigFromGenesisData(); err != nil {
		logger.Errorf("failed to update config from genesis data: %s", err)
		return err
	}

	// Ensure that the base path exists and is accessible
	// Create the folders(config, data) in the base path if they don't exist
	// Write the config to the base path
	if err := cfg.EnsureRoot(config.BasePath, config); err != nil {
		return fmt.Errorf("failed to ensure root: %s", err)
	}

	ks := keystore.NewGlobalKeystore()
	if config.Account.Key != "" {
		if err := loadBuiltInTestKeys(config.Account.Key, *ks); err != nil {
			return fmt.Errorf("error loading built-in test keys: %s", err)
		}
	}

	// load user keys if specified
	if err := unlockKeystore(ks.Acco, config.BasePath, config.Account.Unlock, password); err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	if err := unlockKeystore(ks.Babe, config.BasePath, config.Account.Unlock, password); err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	if err := unlockKeystore(ks.Gran, config.BasePath, config.Account.Unlock, password); err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	node, err := dot.NewNode(config, ks)
	if err != nil {
		logger.Errorf("failed to create node services: %s", err)
		return err
	}

	logger.Info("starting node " + node.Name + "...")

	// start node
	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %s", err)
	}

	return nil
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

	// check genesis id and use genesis id if --chain flag not set
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
