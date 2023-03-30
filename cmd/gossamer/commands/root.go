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

	var con *cfg.Config

	switch chain {
	case "polkadot":
		con = polkadot.DefaultConfig()
	case "kusama":
		con = kusama.DefaultConfig()
	case "westend":
		con = westend.DefaultConfig()
	case "westend-dev":
		con = westend_dev.DefaultConfig()
	default:
		return nil, fmt.Errorf("chain %s not supported", chain)
	}

	fmt.Println("babe-lead")
	fmt.Println(viper.GetString("babe-lead"))

	err = viper.Unmarshal(con)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %s", err)
	}

	fmt.Println("babe-lead")
	fmt.Println(con.Core.BABELead)

	var basePath string
	if os.Getenv("GSSMRHOME") != "" {
		basePath = os.Getenv("GSSMRHOME")
	} else {
		basePath, err = cmd.Flags().GetString("base-path")
		if err != nil {
			return nil, err
		}
	}

	fmt.Println(con)
	fmt.Println(basePath)
	if err := con.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("error in config file: %v", err)
	}

	//cfg.EnsureRoot(con.BasePath, con)

	return con, nil
}

var (
	config = westend_dev.DefaultConfig()
	logger = log.NewFromGlobal(log.AddContext("pkg", "cmd"))
)

// NewRootCommand creates the root command
func NewRootCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
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

			if err := cfg.EnsureRoot(config.BasePath, config); err != nil {
				return err
			}

			return nil
		},
	}

	if err := AddRootFlags(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// AddRootFlags adds the root flags to the command
func AddRootFlags(cmd *cobra.Command) error {
	// helper flags
	cmd.Flags().String("chain",
		"westend_dev",
		"the default chain configuration to load. Example: --chain kusama")
	cmd.Flags().String(
		"password",
		"",
		"password")

	// Base Config
	if err := addStringFlagBindViper(cmd,
		"name",
		config.BaseConfig.Name,
		"node name",
		"name"); err != nil {
		return fmt.Errorf("failed to add --name flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"id", config.BaseConfig.ID,
		"node ID",
		"id"); err != nil {
		return fmt.Errorf("failed to add --id flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"base-path",
		config.BaseConfig.BasePath,
		"base path", "base-path"); err != nil {
		return fmt.Errorf("failed to add --base-path flag: %s", err)
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
		"disable telemetry",
		"no-telemetry"); err != nil {
		return fmt.Errorf("failed to add --no-telemetry flag: %s", err)
	}
	if err := addStringFlagBindViper(cmd,
		"metrics-address",
		config.BaseConfig.MetricsAddress,
		"metrics-address", "metrics-address"); err != nil {
		return fmt.Errorf("failed to add --metrics-address flag: %s", err)
	}
	if err := addUint32FlagBindViper(cmd,
		"retain-blocks",
		config.BaseConfig.RetainBlocks,
		"retain-blocks",
		"retain-blocks"); err != nil {
		return fmt.Errorf("failed to add --retain-blocks flag: %s", err)
	}
	cmd.Flags().StringVar(&logLevelGlobal,
		"log", config.BaseConfig.LogLevel,
		"log-level")
	cmd.Flags().StringVar(&pruning,
		"state-pruning",
		string(config.BaseConfig.Pruning),
		"state-pruning")

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
	if err := AddNetworkFlags(cmd); err != nil {
		return fmt.Errorf("failed to add network flags: %s", err)
	}

	// Core Config
	if err := AddCoreFlags(cmd); err != nil {
		return fmt.Errorf("failed to add core flags: %s", err)
	}

	// State Config
	if err := addUintFlagBindViper(cmd,
		"rewind", config.State.Rewind,
		"rewind",
		"state.rewind"); err != nil {
		return fmt.Errorf("failed to add --rewind flag: %s", err)
	}

	// RPC Config
	if err := AddRPCFlags(cmd); err != nil {
		return fmt.Errorf("failed to add rpc flags: %s", err)
	}

	// pprof Config
	cmd.Flags().Bool("pprof.enabled",
		config.Pprof.Enabled,
		"enabled")
	cmd.Flags().String("pprof.listening-address",
		config.Pprof.ListeningAddress,
		"listening-address")
	cmd.Flags().Int("pprof.block-profile-rate",
		config.Pprof.BlockProfileRate,
		"block-profile-rate")
	cmd.Flags().Int("pprof.mutex-profile-rate",
		config.Pprof.MutexProfileRate,
		"mutex-profile-rate")

	// Misc Config
	cmd.Flags().Bool("dev", false, "dev")

	return nil
}

func AddNetworkFlags(cmd *cobra.Command) error {
	if err := addUint16FlagBindViper(cmd,
		"port",
		config.Network.Port,
		"port",
		"network.port"); err != nil {
		return fmt.Errorf("failed to add --port flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"bootnodes",
		config.Network.Bootnodes,
		"bootnodes",
		"network.bootnodes"); err != nil {
		return fmt.Errorf("failed to add --bootnodes flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"protocol-id",
		config.Network.ProtocolID,
		"protocol-id",
		"network.protocol-id"); err != nil {
		return fmt.Errorf("failed to add --protocol-id flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"no-bootstrap",
		config.Network.NoBootstrap,
		"no-bootstrap",
		"network.no-bootstrap"); err != nil {
		return fmt.Errorf("failed to add --no-bootstrap flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"no-mdns", config.Network.NoMDNS,
		"no-mdns",
		"network.no-mdns"); err != nil {
		return fmt.Errorf("failed to add --no-mdns flag: %s", err)
	}

	if err := addIntFlagBindViper(cmd,
		"min-peers",
		config.Network.MinPeers,
		"min-peers",
		"network.min-peers"); err != nil {
		return fmt.Errorf("failed to add --min-peers flag: %s", err)
	}

	if err := addIntFlagBindViper(cmd,
		"max-peers",
		config.Network.MaxPeers,
		"max-peers",
		"network.max-peers"); err != nil {
		return fmt.Errorf("failed to add --max-peers flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"persistent-peers",
		config.Network.PersistentPeers,
		"persistent-peers",
		"network.persistent-peers"); err != nil {
		return fmt.Errorf("failed to add --persistent-peers flag: %s", err)
	}

	if err := addDurationFlagBindViper(cmd,
		"discovery-interval",
		config.Network.DiscoveryInterval,
		"discovery-interval",
		"network.discovery-interval"); err != nil {
		return fmt.Errorf("failed to add --discovery-interval flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"public-ip", config.Network.PublicIP,
		"public-ip",
		"network.public-ip"); err != nil {
		return fmt.Errorf("failed to add --public-ip flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"public-dns", config.Network.PublicDNS,
		"public-dns",
		"network.public-dns"); err != nil {
		return fmt.Errorf("failed to add --public-dns flag: %s", err)
	}

	return nil
}

func AddRPCFlags(cmd *cobra.Command) error {
	if err := addBoolFlagBindViper(cmd,
		"rpc-enabled",
		config.RPC.Enabled,
		"rpc-enabled",
		"rpc.enabled"); err != nil {
		return fmt.Errorf("failed to add --rpc-enabled flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"rpc-unsafe",
		config.RPC.Unsafe,
		"rpc-unsafe",
		"rpc.unsafe"); err != nil {
		return fmt.Errorf("failed to add --rpc-unsafe flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"unsafe-rpc-external",
		config.RPC.UnsafeExternal,
		"unsafe-rpc-external",
		"rpc.unsafe-external"); err != nil {
		return fmt.Errorf("failed to add --unsafe-rpc-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"rpc-external",
		config.RPC.External,
		"rpc-external",
		"rpc.external"); err != nil {
		return fmt.Errorf("failed to add --rpc-external flag: %s", err)
	}

	if err := addUint32FlagBindViper(cmd,
		"rpc-port",
		config.RPC.Port,
		"rpc-port",
		"rpc.port"); err != nil {
		return fmt.Errorf("failed to add --rpc-port flag: %s", err)
	}

	if err := addStringSliceFlagBindViper(cmd,
		"rpc-methods",
		config.RPC.Modules,
		"rpc-methods",
		"rpc.modules"); err != nil {
		return fmt.Errorf("failed to add --rpc-methods flag: %s", err)
	}

	if err := addUint32FlagBindViper(cmd,
		"ws-port",
		config.RPC.WSPort,
		"ws-port",
		"rpc.ws-port"); err != nil {
		return fmt.Errorf("failed to add --ws-port flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws", config.RPC.WS,
		"ws",
		"rpc.ws"); err != nil {
		return fmt.Errorf("failed to add --ws flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-external",
		config.RPC.WSExternal,
		"ws-external",
		"rpc.ws-external"); err != nil {
		return fmt.Errorf("failed to add --ws-external flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-unsafe",
		config.RPC.WSUnsafe,
		"ws-unsafe",
		"rpc.ws-unsafe"); err != nil {
		return fmt.Errorf("failed to add --ws-unsafe flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"ws-unsafe-external",
		config.RPC.WSUnsafeExternal,
		"ws-unsafe-external",
		"rpc.ws-unsafe-external"); err != nil {
		return fmt.Errorf("failed to add --ws-unsafe-external flag: %s", err)
	}

	return nil
}

func AddCoreFlags(cmd *cobra.Command) error {
	// TODO: role

	if err := addBoolFlagBindViper(cmd,
		"babe-authority",
		config.Core.BabeAuthority,
		"babe-authority",
		"core.babe-authority"); err != nil {
		return fmt.Errorf("failed to add --babe-authority flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"grandpa-authority",
		config.Core.GrandpaAuthority,
		"grandpa-authority",
		"core.grandpa-authority"); err != nil {
		return fmt.Errorf("failed to add --grandpa-authority flag: %s", err)
	}

	if err := addUint64FlagBindViper(cmd,
		"slot-duration",
		config.Core.SlotDuration,
		"slot-duration",
		"core.slot-duration"); err != nil {
		return fmt.Errorf("failed to add --slot-duration flag: %s", err)
	}

	if err := addUint64FlagBindViper(cmd,
		"epoch-length",
		config.Core.EpochLength,
		"epoch-length",
		"core.epoch-length"); err != nil {
		return fmt.Errorf("failed to add --epoch-length flag: %s", err)
	}

	if err := addStringFlagBindViper(cmd,
		"wasm-interpreter",
		config.Core.WasmInterpreter,
		"wasm-interpreter",
		"core.wasm-interpreter"); err != nil {
		return fmt.Errorf("failed to add --wasm-interpreter flag: %s", err)
	}

	if err := addDurationFlagBindViper(cmd,
		"grandpa-interval",
		config.Core.GrandpaInterval,
		"grandpa-interval",
		"core.grandpa-interval"); err != nil {
		return fmt.Errorf("failed to add --grandpa-interval flag: %s", err)
	}

	if err := addBoolFlagBindViper(cmd,
		"babe-lead",
		config.Core.BABELead,
		"babe-lead",
		"core.babe-lead"); err != nil {
		return fmt.Errorf("failed to add --babe-lead flag: %s", err)
	}

	return nil
}

// execRoot executes the root command
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
