// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

var (
	// GlobalConfig

	// DefaultName Default node name
	DefaultName = string("Kusama")
	// DefaultID Default chain ID
	DefaultID = string("ksmcc3")
	// DefaultConfig Default toml configuration path
	DefaultConfig = string("./chain/kusama/config.toml")
	// DefaultBasePath Default node base directory path
	DefaultBasePath = string("~/.gossamer/kusama")

	// DefaultMetricsPort is the metrics server port
	DefaultMetricsPort = uint32(9876)

	// DefaultLvl is the default log level
	DefaultLvl = log.Info

	// DefaultPruningMode is the default pruning mode
	DefaultPruningMode = "archive"
	// DefaultRetainBlocks is the default retained blocks
	DefaultRetainBlocks = int64(512)

	// DefaultTelemetryURLs is the default URL of the telemetry server to connect to.
	DefaultTelemetryURLs []genesis.TelemetryEndpoint

	// InitConfig

	// DefaultGenesis is the default genesis configuration path
	DefaultGenesis = string("./chain/kusama/genesis.json")

	// AccountConfig

	// DefaultKey Default account key
	DefaultKey = string("")
	// DefaultUnlock Default account unlock
	DefaultUnlock = string("")

	// CoreConfig

	// DefaultAuthority true if BABE block producer
	DefaultAuthority = false
	// DefaultRoles Default node roles
	DefaultRoles = byte(1) // full node (see Table D.2)
	// DefaultWasmInterpreter is the name of the wasm interpreter to use by default
	DefaultWasmInterpreter = wasmer.Name

	// NetworkConfig

	// DefaultNetworkPort network port
	DefaultNetworkPort = uint16(7001)
	// DefaultNetworkBootnodes network bootnodes
	DefaultNetworkBootnodes = []string(nil)
	// DefaultNoBootstrap disables bootstrap
	DefaultNoBootstrap = false
	// DefaultNoMDNS disables mDNS discovery
	DefaultNoMDNS = false

	// RPCConfig

	// DefaultRPCHTTPHost rpc host
	DefaultRPCHTTPHost = string("localhost")
	// DefaultRPCHTTPPort rpc port
	DefaultRPCHTTPPort = uint32(8545)
	// DefaultRPCModules rpc modules
	DefaultRPCModules = []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain", "childstate", "syncstate", "payment"}
	// DefaultRPCWSPort rpc websocket port
	DefaultRPCWSPort = uint32(8546)
)
