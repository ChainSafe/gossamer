// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westend_dev "github.com/ChainSafe/gossamer/chain/westend-dev"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	terminal "golang.org/x/term"
	"os"
	"strings"
	"syscall"
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

// ParseConfig parses the config from the command line flags
func ParseConfig(cmd *cobra.Command) (*cfg.Config, error) {
	chain, err := cmd.Flags().GetString("chain")
	if err != nil {
		return nil, fmt.Errorf("failed to get --chain: %s", err)
	}
	if chain == "" {
		return nil, fmt.Errorf("--chain cannot be empty")
	}

	var config *cfg.Config

	switch chain {
	case "polkadot":
		config = polkadot.DefaultConfig()
	case "kusama":
		config = kusama.DefaultConfig()
	case "westend":
		config = westend.DefaultConfig()
	case "westend-dev":
		config = westend_dev.DefaultConfig()
	default:
		return nil, fmt.Errorf("chain %s not supported", chain)
	}

	err = viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %s", err)
	}

	var basePath string
	if os.Getenv("GSSMRHOME") != "" {
		basePath = os.Getenv("GSSMRHOME")
	} else {
		basePath, err = cmd.Flags().GetString("base-path")
		if err != nil {
			return nil, err
		}
	}

	config.BasePath = basePath
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("error in config file: %v", err)
	}

	cfg.EnsureRoot(config.BasePath, config)

	return config, nil
}

var (
	config = westend_dev.DefaultConfig()
	logger = log.NewFromGlobal(log.AddContext("pkg", "cmd"))

	// RootCmd is the root command for the gossamer node
	RootCmd = &cobra.Command{
		Use:   "gossamer",
		Short: "Official gossamer command-line interface",
		Long:  `Gossamer is a Golang implementation of the Polkadot Host`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return execRoot(cmd)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if cmd.Name() != "gossamer" {
				return nil
			}

			config, err = ParseConfig(cmd)
			if err != nil {
				return err
			}

			// Create the config.toml file
			if err := cfg.WriteConfigFile(utils.ExpandDir(config.BasePath+"/config.toml"), config); err != nil {
				return fmt.Errorf("failed to write config file: %s", err)
			}

			return nil
		},
	}
)

func init() {
	AddRootFlags(RootCmd)

	RootCmd.AddCommand(accountCmd, buildSpecCmd, importRuntimeCmd, importStateCmd, initCmd, pruneStateCmd)
}

// Execute executes the root command.
func Execute() error {
	return RootCmd.Execute()
}

func AddRootFlags(cmd *cobra.Command) {
	// Persistent flags
	cmd.Flags().String("chain", "westend_dev", "the default chain configuration to load. Example: --chain kusama")
	cmd.Flags().StringP("base-path", "d", "", "base-path")
	cmd.Flags().String("password", "", "base-path")

	// Base Config
	cmd.Flags().String("name", config.BaseConfig.Name, "node name")
	cmd.Flags().String("id", config.BaseConfig.ID, "node ID")
	cmd.Flags().String("genesis", config.BaseConfig.Genesis, "path to the genesis file")
	cmd.Flags().StringP("log", "l", config.BaseConfig.LogLevel, "log-level")
	cmd.Flags().Bool("no-telemetry", config.BaseConfig.NoTelemetry, "no-telemetry")
	cmd.Flags().String("metrics-address", config.BaseConfig.MetricsAddress, "metrics-address")
	cmd.Flags().Uint32("retain-blocks", config.BaseConfig.RetainBlocks, "retain-blocks")
	cmd.Flags().StringVar(&pruning, "state-pruning", string(config.BaseConfig.Pruning), "state-pruning")
	// TODO: telemetry-url

	// Log Config
	cmd.Flags().StringVar(&logLevelCore, "lcore", config.Log.Core, "lcore")
	cmd.Flags().StringVar(&logLevelDigest, "ldigest", config.Log.Digest, "ldigest")
	cmd.Flags().StringVar(&logLevelSync, "lsync", config.Log.Sync, "log-sync")
	cmd.Flags().StringVar(&logLevelNetwork, "lnetwork", config.Log.Network, "lnetwork")
	cmd.Flags().StringVar(&logLevelRPC, "lrpc", config.Log.RPC, "lrpc")
	cmd.Flags().StringVar(&logLevelState, "lstate", config.Log.State, "lstate")
	cmd.Flags().StringVar(&logLevelRuntime, "lruntime", config.Log.Runtime, "lruntime")
	cmd.Flags().StringVar(&logLevelBABE, "lbabe", config.Log.Babe, "lbabe")
	cmd.Flags().StringVar(&logLevelGRANDPA, "lgrandpa", config.Log.Grandpa, "lgrandpa")

	// Account Config
	cmd.Flags().String("account.key", config.Account.Key, "key")
	cmd.Flags().String("account.unlock", config.Account.Unlock, "unlock")

	// Network Config
	cmd.Flags().Uint16("port", config.Network.Port, "port")
	cmd.Flags().StringArray("bootnodes", config.Network.Bootnodes, "bootnodes")
	cmd.Flags().String("protocol-id", config.Network.ProtocolID, "protocol-id")
	cmd.Flags().Bool("no-bootstrap", config.Network.NoBootstrap, "no-bootstrap")
	cmd.Flags().Bool("no-mdns", config.Network.NoMDNS, "no-mdns")
	cmd.Flags().Int("min-peers", config.Network.MinPeers, "min-peers")
	cmd.Flags().Int("max-peers", config.Network.MaxPeers, "max-peers")
	cmd.Flags().StringArray("persistent-peers", config.Network.PersistentPeers, "persistent-peers")
	cmd.Flags().Duration("discovery-interval", config.Network.DiscoveryInterval, "discovery-interval")
	cmd.Flags().String("public-ip", config.Network.PublicIP, "public-ip")
	cmd.Flags().String("public-dns", config.Network.PublicDNS, "public-dns")

	// Core Config
	// TODO: role
	cmd.Flags().Bool("babe-authority", config.Core.BabeAuthority, "babe-authority")
	cmd.Flags().Bool("grandpa-authority", config.Core.GrandpaAuthority, "grandpa-authority")
	cmd.Flags().Uint64("slot-duration", config.Core.SlotDuration, "slot-duration")
	cmd.Flags().Uint64("epoch-length", config.Core.EpochLength, "epoch-length")
	cmd.Flags().String("wasm-interpreter", config.Core.WasmInterpreter, "wasm-interpreter")
	cmd.Flags().Duration("grandpa-interval", config.Core.GrandpaInterval, "grandpa-interval")
	cmd.Flags().Bool("babe-lead", config.Core.BABELead, "babe-lead")

	// State Config
	cmd.Flags().Uint("rewind", config.State.Rewind, "rewind")

	// RPC Config
	cmd.Flags().Bool("rpc-enabled", config.RPC.Enabled, "enabled")
	cmd.Flags().Bool("rpc-unsafe", config.RPC.Unsafe, "unsafe")
	cmd.Flags().Bool("unsafe-rpc-external", config.RPC.UnsafeExternal, "unsafe-external")
	cmd.Flags().Bool("rpc-external", config.RPC.External, "external")
	cmd.Flags().Uint32("rpc-port", config.RPC.Port, "port")
	cmd.Flags().StringArray("rpc-methods", config.RPC.Modules, "modules")
	cmd.Flags().Uint32("ws-port", config.RPC.WSPort, "ws-port")
	cmd.Flags().Bool("ws", config.RPC.WS, "ws")
	cmd.Flags().Bool("ws-external", config.RPC.WSExternal, "ws-external")
	cmd.Flags().Bool("ws-unsafe", config.RPC.WSUnsafe, "ws-unsafe")
	cmd.Flags().Bool("unsafe-ws-external", config.RPC.WSUnsafeExternal, "ws-unsafe-external")

	// pprof Config
	cmd.Flags().Bool("pprof.enabled", config.Pprof.Enabled, "enabled")
	cmd.Flags().String("pprof.listening-address", config.Pprof.ListeningAddress, "listening-address")
	cmd.Flags().Int("pprof.block-profile-rate", config.Pprof.BlockProfileRate, "block-profile-rate")
	cmd.Flags().Int("pprof.mutex-profile-rate", config.Pprof.MutexProfileRate, "mutex-profile-rate")

	// Misc Config
	cmd.Flags().Bool("dev", false, "dev")
}

func execRoot(cmd *cobra.Command) error {
	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return fmt.Errorf("failed to get password: %s", err)
	}

	config.BasePath = utils.ExpandDir(config.BasePath)

	if !dot.IsNodeInitialised(config.BasePath) {
		// initialise node (initialise state database and load genesis data)
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

	ks := keystore.NewGlobalKeystore()

	if config.Account.Key != "" {
		if err := loadBuiltInTestKeys(config.Account.Key, *ks); err != nil {
			return fmt.Errorf("loading built-in test keys: %s", err)
		}
	}

	// load user keys if specified
	if err := unlockKeystore(ks.Acco, config.BasePath, config.Account.Unlock, password); err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	err = unlockKeystore(ks.Babe, config.BasePath, config.Account.Unlock, password)
	if err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	err = unlockKeystore(ks.Gran, config.BasePath, config.Account.Unlock, password)
	if err != nil {
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
	err = node.Start()
	if err != nil {
		return err
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
	//if chain == "" {
	//	config.ID = gen.ID
	//}

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
