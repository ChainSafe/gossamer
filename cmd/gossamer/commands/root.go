// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/keystore"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/internal/log"

	westenddev "github.com/ChainSafe/gossamer/chain/westend-dev"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/spf13/cobra"
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
	telemetryURLs  string

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

	// Core Config
	// role of the node. one of: full, light or authority
	role string
	// validator when set, the node will be an authority
	validator bool

	// Account Config
	// key to use for the node
	key string
)

// Flag values for persistent flags
var (
	// Default accounts
	alice   bool
	bob     bool
	charlie bool

	// Initialization flags for node
	chain       string
	basePath    string
	genesisPath string
)

// Default values
const (
	// DefaultHomeEnv is the default environment variable for the base path
	DefaultHomeEnv = "GSSMRHOME"
)

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
			if err := setDefaultConfig(cfg.Chain(chain)); err != nil {
				return fmt.Errorf("failed to set default config: %s", err)
			}

			if err := parseBasePath(); err != nil {
				return fmt.Errorf("failed to parse base path: %s", err)
			}

			if !(cmd.Name() == "gossamer" || cmd.Name() == "init") {
				return nil
			}

			if err := parseGenesis(); err != nil {
				return fmt.Errorf("failed to parse genesis: %s", err)
			}

			parseAccount()

			if cmd.Name() == "gossamer" {
				if err := parseRole(); err != nil {
					return fmt.Errorf("failed to parse role: %s", err)
				}

				if err := parseTelemetryURL(); err != nil {
					return fmt.Errorf("failed to parse telemetry-url: %s", err.Error())
				}

				if err := configureViper(config.BasePath); err != nil {
					return fmt.Errorf("failed to configure viper: %s", err)
				}

				if err := ParseConfig(); err != nil {
					return fmt.Errorf("failed to parse config: %s", err)
				}
			}

			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
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
		cfg.WestendLocalChain.String(),
		"The default chain configuration to load. Example: --chain kusama")
	cmd.Flags().StringVar(&genesisPath,
		"genesis",
		"",
		"the path to the genesis configuration to load. Example: --genesis genesis.json")

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
	cmd.Flags().StringVar(&telemetryURLs,
		"telemetry-url",
		"",
		`The URL of the telemetry server to connect to.
This flag can be passed multiple times as a means to specify multiple telemetry endpoints.
Verbosity levels range from 0-9, with 0 denoting the least verbosity.
Expected format is 'URL VERBOSITY', e.g. ''--telemetry-url wss://foo/bar:0, wss://baz/quz:1
`)

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
	cmd.PersistentFlags().StringVar(&key,
		"key",
		"",
		"Keyring to use for the node")

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
		"unsafe-rpc-external",
		config.RPC.UnsafeRPCExternal,
		"Enable external HTTP-RPC connections to unsafe procedures",
		"rpc.unsafe-rpc-external"); err != nil {
		return fmt.Errorf("failed to add --unsafe-rpc-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"rpc-external",
		config.RPC.RPCExternal,
		"Enable external HTTP-RPC connections",
		"rpc.rpc-external"); err != nil {
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
		"ws-external",
		config.RPC.WSExternal,
		"Enable external websocket connections",
		"rpc.ws-external"); err != nil {
		return fmt.Errorf("failed to add --ws-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"unsafe-ws-external",
		config.RPC.UnsafeWSExternal,
		"Enable external access to websocket unsafe calls",
		"rpc.unsafe-ws-external"); err != nil {
		return fmt.Errorf("failed to add --ws-unsafe-external flag: %s", err)
	}

	// dummy flag to conform with the substrate cli
	cmd.Flags().String("rpc-cors",
		"",
		"dummy flag to conform with the substrate cli")

	return nil
}

// addCoreFlags adds core flags and binds to viper
func addCoreFlags(cmd *cobra.Command) error {
	cmd.Flags().StringVar(&role,
		"role",
		cfg.AuthorityNode.String(),
		"Role of the node. One of 'full', 'light', or 'authority'.")

	cmd.Flags().BoolVar(&validator,
		"validator",
		true,
		"Run as a validator node")

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

	ks := keystore.NewGlobalKeystore()
	if config.Account.Key != "" {
		if err := loadBuiltInTestKeys(config.Account.Key, *ks); err != nil {
			return fmt.Errorf("error loading built-in test keys: %s", err)
		}
	}

	// load user keys if specified
	if err := unlockKeystore(ks.Acco, config.BasePath, config.Account.Unlock, password); err != nil {
		return fmt.Errorf("failed to unlock keystore: %s", err)
	}

	if err := unlockKeystore(ks.Babe, config.BasePath, config.Account.Unlock, password); err != nil {
		return fmt.Errorf("failed to unlock keystore: %s", err)
	}

	if err := unlockKeystore(ks.Gran, config.BasePath, config.Account.Unlock, password); err != nil {
		return fmt.Errorf("failed to unlock keystore: %s", err)
	}

	if err := config.ValidateBasic(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	// Write the config to the base path
	if err := cfg.WriteConfigFile(config.BasePath, config); err != nil {
		return fmt.Errorf("failed to ensure root: %s", err)
	}

	// if the node is not initialised, initialise it
	if !dot.IsNodeInitialised(config.BasePath) {
		if err := dot.InitNode(config); err != nil {
			return fmt.Errorf("failed to initialise node: %s", err)
		}
	}

	node, err := dot.NewNode(config, ks)
	if err != nil {
		return fmt.Errorf("failed to create node services: %s", err)
	}

	logger.Info("starting node " + node.Name + "...")

	// start node
	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %s", err)
	}

	return nil
}
