// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"github.com/ChainSafe/gossamer/chain/dev"
	"github.com/urfave/cli"
)

// Node flags
var (
	// UnlockFlag keystore
	UnlockFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Unlock an account. eg. --unlock=0,2 to unlock accounts 0 and 2. Can be used with --password=[password] to avoid prompt. For multiple passwords, do --password=password1,password2",
	}
	// ForceFlag disables all confirm prompts ("Y" to all)
	ForceFlag = cli.BoolFlag{
		Name:  "force",
		Usage: "Disable all confirm prompts (the same as answering \"Y\" to all)",
	}
	// KeyFlag specifies a test keyring account to use
	KeyFlag = cli.StringFlag{
		Name:  "key",
		Usage: "Specify a test keyring account to use: eg --key=alice",
	}
	// RolesFlag role of the node (see Table D.2)
	RolesFlag = cli.StringFlag{
		Name:  "roles",
		Usage: "Roles of the gossamer node",
	}
	// RewindFlag rewinds the head of the chain to the given block number. Useful for development
	RewindFlag = cli.IntFlag{
		Name:  "rewind",
		Usage: "Rewind head of chain to the given block number",
	}
)

// Global node configuration flags
var (
	// LogFlag cli service settings
	LogFlag = cli.StringFlag{
		Name:  "log",
		Usage: "Global log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogCoreLevelFlag = cli.StringFlag{
		Name:  "log-core",
		Usage: "Core package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogSyncLevelFlag = cli.StringFlag{
		Name:  "log-sync",
		Usage: "Sync package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogNetworkLevelFlag = cli.StringFlag{
		Name:  "log-network",
		Usage: "Network package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogRPCLevelFlag = cli.StringFlag{
		Name:  "log-rpc",
		Usage: "RPC package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogStateLevelFlag = cli.StringFlag{
		Name:  "log-state",
		Usage: "State package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogRuntimeLevelFlag = cli.StringFlag{
		Name:  "log-runtime",
		Usage: "Runtime package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogBabeLevelFlag = cli.StringFlag{
		Name:  "log-babe",
		Usage: "BABE package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}
	LogGrandpaLevelFlag = cli.StringFlag{
		Name:  "log-grandpa",
		Usage: "Grandpa package log level. Supports levels crit (silent), eror, warn, info, dbug and trce (trace)",
	}

	// NameFlag node implementation name
	NameFlag = cli.StringFlag{
		Name:  "name",
		Usage: "Node implementation name",
	}
	// ChainFlag is chain id used to load default configuration for specified chain
	ChainFlag = cli.StringFlag{
		Name:  "chain",
		Usage: "Chain id used to load default configuration for specified chain",
	}
	// ConfigFlag TOML configuration file
	ConfigFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
	// BasePathFlag data directory for node
	BasePathFlag = cli.StringFlag{
		Name:  "basepath",
		Usage: "Data directory for the node",
	}
	PprofServerFlag = cli.StringFlag{
		Name:  "pprofserver",
		Usage: "enable or disable the pprof HTTP server",
	}
	PprofAddressFlag = cli.StringFlag{
		Name:  "pprofaddress",
		Usage: "pprof HTTP server listening address, if it is enabled.",
	}
	PprofBlockRateFlag = cli.IntFlag{
		Name:  "pprofblockrate",
		Value: -1,
		Usage: "pprof block rate. See https://pkg.go.dev/runtime#SetBlockProfileRate.",
	}
	PprofMutexRateFlag = cli.IntFlag{
		Name:  "pprofmutexrate",
		Value: -1,
		Usage: "profiling mutex rate. See https://pkg.go.dev/runtime#SetMutexProfileFraction.",
	}

	// PublishMetricsFlag publishes node metrics to prometheus.
	PublishMetricsFlag = cli.BoolFlag{
		Name:  "publish-metrics",
		Usage: "Publish node metrics",
	}

	// MetricsPortFlag set metric listen port
	MetricsPortFlag = cli.StringFlag{
		Name:  "metrics-port",
		Usage: "Set metric listening port ",
	}

	// NoTelemetryFlag stops publishing telemetry to default defined in genesis.json
	NoTelemetryFlag = cli.BoolFlag{
		Name:  "no-telemetry",
		Usage: "Disable connecting to the Substrate telemetry server",
	}

	// TelemetryURLFlag is URL of the telemetry server to connect to.
	// This flag can be passed multiple times as a means to specify multiple
	// telemetry endpoints. Verbosity levels range from 0-9, with 0 denoting the
	// least verbosity.
	// Expected format is 'URL VERBOSITY', e.g. `--telemetry-url 'wss://foo/bar 0'`.
	TelemetryURLFlag = cli.StringSliceFlag{
		Name: "telemetry-url",
		Usage: `The URL of the telemetry server to connect to, this flag can be
		passed multiple times, the verbosity levels range from 0-9, with 0 denoting
		least verbosity.
		Expected format --telemetry-url 'wss://foo/bar 0'`,
	}
)

// Initialization-only flags
var (
	// GenesisFlag is the path to a genesis JSON file
	GenesisFlag = cli.StringFlag{
		Name:  "genesis",
		Usage: "Path to genesis JSON file",
	}
)

// ImportState-only flags
var (
	StateFlag = cli.StringFlag{
		Name:  "state",
		Usage: "Path to JSON file consisting of key-value pairs",
	}
	HeaderFlag = cli.StringFlag{
		Name:  "header",
		Usage: "Path to JSON file of block header corresponding to the given state",
	}
	FirstSlotFlag = cli.IntFlag{
		Name:  "first-slot",
		Usage: "The first BABE slot of the network",
	}
)

// BuildSpec-only flags
var (
	RawFlag = cli.BoolFlag{
		Name:  "raw",
		Usage: "Output as raw genesis JSON",
	}
	GenesisSpecFlag = cli.StringFlag{
		Name:  "genesis-spec",
		Usage: "Path to human-readable genesis JSON file",
	}
	OutputSpecFlag = cli.StringFlag{
		Name:  "output",
		Usage: "Path to output the recently created genesis JSON file",
	}
)

// Network service configuration flags
var (
	// PortFlag Set network listening port
	PortFlag = cli.UintFlag{
		Name:  "port",
		Usage: "Set network listening port",
	}
	// BootnodesFlag Network service settings
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated node URLs for network discovery bootstrap",
	}
	// ProtocolFlag Set protocol id
	ProtocolFlag = cli.StringFlag{
		Name:  "protocol",
		Usage: "Set protocol id",
	}
	// NoBootstrapFlag Disables network bootstrapping
	NoBootstrapFlag = cli.BoolFlag{
		Name:  "nobootstrap",
		Usage: "Disables network bootstrapping (mDNS still enabled)",
	}
	// NoMDNSFlag Disables network mDNS
	NoMDNSFlag = cli.BoolFlag{
		Name:  "nomdns",
		Usage: "Disables network mDNS discovery",
	}
)

// RPC service configuration flags
var (
	// RPCEnabledFlag Enable the HTTP-RPC
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	// RPCExternalFlag Enable the external HTTP-RPC
	RPCExternalFlag = cli.BoolFlag{
		Name:  "rpc-external",
		Usage: "Enable external HTTP-RPC connections",
	}
	// RPCEnabledFlag Enable the HTTP-RPC
	RPCUnsafeEnabledFlag = cli.BoolFlag{
		Name:  "rpc-unsafe",
		Usage: "Enable the HTTP-RPC server to unsafe procedures",
	}
	// RPCExternalFlag Enable the external HTTP-RPC
	RPCUnsafeExternalFlag = cli.BoolFlag{
		Name:  "rpc-unsafe-external",
		Usage: "Enable external HTTP-RPC connections to unsafe procedures",
	}
	// RPCHostFlag HTTP-RPC server listening hostname
	RPCHostFlag = cli.StringFlag{
		Name:  "rpchost",
		Usage: "HTTP-RPC server listening hostname",
	}
	// RPCPortFlag HTTP-RPC server listening port
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
	}
	// RPCModulesFlag API modules to enable via HTTP-RPC
	RPCModulesFlag = cli.StringFlag{
		Name:  "rpcmods",
		Usage: "API modules to enable via HTTP-RPC, comma separated list",
	}
	// WSPortFlag WebSocket server listening port
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "Websockets server listening port",
	}
	// WSFlag Enable the websockets server
	WSFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the websockets server",
	}
	// WSExternalFlag Enable external websocket connections
	WSExternalFlag = cli.BoolFlag{
		Name:  "ws-external",
		Usage: "Enable external websocket connections",
	}
	// WSFlag Enable the websockets server
	WSUnsafeFlag = cli.BoolFlag{
		Name:  "ws-unsafe",
		Usage: "Enable access to websocket unsafe calls",
	}
	// WSExternalFlag Enable external websocket connections
	WSUnsafeExternalFlag = cli.BoolFlag{
		Name:  "ws-unsafe-external",
		Usage: "Enable external access to websocket unsafe calls",
	}
)

// Account management flags
var (
	// GenerateFlag Generate a new keypair
	GenerateFlag = cli.BoolFlag{
		Name:  "generate",
		Usage: "Generate a new keypair. If type is not specified, defaults to sr25519",
	}
	// PasswordFlag Password used to encrypt the keystore.
	PasswordFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password used to encrypt the keystore. Used with --generate or --unlock",
	}
	// ImportFlag Import encrypted keystore
	ImportFlag = cli.StringFlag{
		Name:  "import",
		Usage: "Import encrypted keystore file generated with gossamer",
	}
	// ImportRawFlag imports a raw private key
	ImportRawFlag = cli.StringFlag{
		Name:  "import-raw",
		Usage: "Import  a raw private key",
	}
	// ListFlag List node keys
	ListFlag = cli.BoolFlag{
		Name:  "list",
		Usage: "List node keys",
	}
	// Ed25519Flag Specify account type ed25519
	Ed25519Flag = cli.BoolFlag{
		Name:  "ed25519",
		Usage: "Specify account type as ed25519",
	}
	// Sr25519Flag Specify account type sr25519
	Sr25519Flag = cli.BoolFlag{
		Name:  "sr25519",
		Usage: "Specify account type as sr25519",
	}
	// Secp256k1Flag Specify account type secp256k1
	Secp256k1Flag = cli.BoolFlag{
		Name:  "secp256k1",
		Usage: "Specify account type as secp256k1",
	}
)

// State Prune flags
var (
	// BloomFilterSizeFlag size for bloom filter, valid for the use with prune-state subcommand
	BloomFilterSizeFlag = cli.IntFlag{
		Name:  "bloom-size",
		Usage: "Megabytes of memory allocated to bloom-filter for pruning",
		Value: 2048,
	}

	// DBPathFlag data directory for pruned DB, valid for the use with prune-state subcommand
	DBPathFlag = cli.StringFlag{
		Name:  "pruned-db-path",
		Usage: "Data directory for the output DB",
	}

	// RetainBlockNumberFlag retain number of block from latest block while pruning, valid for the use with prune-state subcommand
	RetainBlockNumberFlag = cli.Int64Flag{
		Name:  "retain-blocks",
		Usage: "Retain number of block from latest block while pruning",
		Value: dev.DefaultRetainBlocks,
	}

	// PruningFlag triggers the online pruning of historical state tries. It's either full or archive. To enable pruning the value
	// should be set to `full`.
	PruningFlag = cli.StringFlag{
		Name:  "pruning",
		Usage: `State trie online pruning ("full", "archive")`,
		Value: dev.DefaultPruningMode,
	}
)

// BABE flags
var (
	BABELeadFlag = cli.BoolFlag{
		Name:  "babe-lead",
		Usage: `specify whether node should build block 1 of the network. only used when starting a new network`,
	}
)

// flag sets that are shared by multiple commands
var (
	// GlobalFlags are flags that are valid for use with the root command and all subcommands
	GlobalFlags = []cli.Flag{
		LogFlag,
		LogCoreLevelFlag,
		LogSyncLevelFlag,
		LogNetworkLevelFlag,
		LogRPCLevelFlag,
		LogStateLevelFlag,
		LogRuntimeLevelFlag,
		LogBabeLevelFlag,
		LogGrandpaLevelFlag,
		NameFlag,
		ChainFlag,
		ConfigFlag,
		BasePathFlag,
		PprofServerFlag,
		PprofAddressFlag,
		PprofBlockRateFlag,
		PprofMutexRateFlag,
		RewindFlag,
		DBPathFlag,
		BloomFilterSizeFlag,
	}

	// StartupFlags are flags that are valid for use with the root command and the export subcommand
	StartupFlags = []cli.Flag{
		// keystore flags
		KeyFlag,
		UnlockFlag,

		// network flags
		PortFlag,
		BootnodesFlag,
		ProtocolFlag,
		RolesFlag,
		NoBootstrapFlag,
		NoMDNSFlag,

		// rpc flags
		RPCEnabledFlag,
		RPCExternalFlag,
		RPCUnsafeEnabledFlag,
		RPCUnsafeExternalFlag,
		RPCHostFlag,
		RPCPortFlag,
		RPCModulesFlag,
		WSFlag,
		WSExternalFlag,
		WSUnsafeFlag,
		WSUnsafeExternalFlag,
		WSPortFlag,

		// metrics flag
		PublishMetricsFlag,
		MetricsPortFlag,

		// telemetry flags
		NoTelemetryFlag,
		TelemetryURLFlag,

		// BABE flags
		BABELeadFlag,
	}
)

// local flag sets for the root gossamer command and all subcommands
var (
	// RootFlags are the flags that are valid for use with the root gossamer command
	RootFlags = append(GlobalFlags, StartupFlags...)

	// InitFlags are flags that are valid for use with the init subcommand
	InitFlags = append([]cli.Flag{
		ForceFlag,
		GenesisFlag,
		PruningFlag,
		RetainBlockNumberFlag,
	}, GlobalFlags...)

	BuildSpecFlags = append([]cli.Flag{
		RawFlag,
		GenesisSpecFlag,
		OutputSpecFlag,
	}, GlobalFlags...)

	// ExportFlags are the flags that are valid for use with the export subcommand
	ExportFlags = append([]cli.Flag{
		ForceFlag,
		GenesisFlag,
	}, append(GlobalFlags, StartupFlags...)...)

	// AccountFlags are flags that are valid for use with the account subcommand
	AccountFlags = append([]cli.Flag{
		GenerateFlag,
		PasswordFlag,
		ImportFlag,
		ImportRawFlag,
		ListFlag,
		Ed25519Flag,
		Sr25519Flag,
		Secp256k1Flag,
	}, GlobalFlags...)

	ImportStateFlags = []cli.Flag{
		BasePathFlag,
		ChainFlag,
		ConfigFlag,
		StateFlag,
		HeaderFlag,
		FirstSlotFlag,
	}

	PruningFlags = []cli.Flag{
		ChainFlag,
		ConfigFlag,
		DBPathFlag,
		BloomFilterSizeFlag,
		RetainBlockNumberFlag,
	}
)

// FixFlagOrder allow us to use various flag order formats (ie, `gossamer init
// --config config.toml` and `gossamer --config config.toml init`). FixFlagOrder
// only fixes global flags, all local flags must come after the subcommand (ie,
// `gossamer --force --config config.toml init` will not recognise `--force` but
// `gossamer init --force --config config.toml` will work as expected).
func FixFlagOrder(f func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		const trace = "trace"

		// loop through all flags (global and local)
		for _, flagName := range ctx.FlagNames() {

			// check if flag is set as global or local flag
			if ctx.GlobalIsSet(flagName) {
				// log global flag if log equals trace
				if ctx.String(LogFlag.Name) == trace {
					logger.Trace("[cmd] global flag set with name: " + flagName)
				}
			} else if ctx.IsSet(flagName) {
				// check if global flag using set as global flag
				err := ctx.GlobalSet(flagName, ctx.String(flagName))
				if err == nil {
					// log fixed global flag if log equals trace
					if ctx.String(LogFlag.Name) == trace {
						logger.Trace("[cmd] global flag fixed with name: " + flagName)
					}
				} else {
					// if not global flag, log local flag if log equals trace
					if ctx.String(LogFlag.Name) == trace {
						logger.Trace("[cmd] local flag set with name: " + flagName)
					}
				}
			}
		}

		return f(ctx)
	}
}
