// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dev

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

var (
	// GlobalConfig

	// DefaultName is the node name
	DefaultName = string("Gossamer")
	// DefaultID is the chain ID
	DefaultID = string("dev")
	// DefaultConfig is the toml configuration path
	DefaultConfig = string("./chain/dev/config.toml")
	// DefaultBasePath is the node base directory path
	DefaultBasePath = string("~/.gossamer/dev")

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
	DefaultGenesis = string("./chain/dev/genesis-spec.json")

	// AccountConfig

	// DefaultKey is the default account key
	DefaultKey = string("alice")
	// DefaultUnlock is the account to unlock
	DefaultUnlock = string("")

	// CoreConfig

	// DefaultAuthority is true if the node is a block producer and a grandpa authority
	DefaultAuthority = true
	// DefaultRoles Default node roles
	DefaultRoles = byte(4) // authority node (see Table D.2)
	// DefaultBabeAuthority is true if the node is a block producer (overwrites previous settings)
	DefaultBabeAuthority = true
	// DefaultGrandpaAuthority is true if the node is a grandpa authority (overwrites previous settings)
	DefaultGrandpaAuthority = true
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
	// DefaultRPCEnabled enables the RPC server
	DefaultRPCEnabled = true
	// DefaultWSEnabled enables the WS server
	DefaultWSEnabled = true
)
