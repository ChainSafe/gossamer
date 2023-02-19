// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westend

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

var (
	// GlobalConfig

	// DefaultName Default node name
	DefaultName = string("Westend")
	// DefaultID Default chain ID
	DefaultID = string("westend2")
	// DefaultConfig Default toml configuration path
	DefaultConfig = string("./chain/westend/config.toml")
	// DefaultBasePath Default node base directory path
	DefaultBasePath = string("~/.gossamer/westend")

	// DefaultLvl is the default log level
	DefaultLvl = log.Info

	// DefaultPruningMode is the default pruning mode
	DefaultPruningMode = "archive"
	// DefaultRetainBlocks is the default pruning mode
	DefaultRetainBlocks = uint32(512)

	// DefaultTelemetryURLs is the default URL of the telemetry server to connect to.
	DefaultTelemetryURLs []genesis.TelemetryEndpoint

	// InitConfig

	// DefaultGenesis is the default genesis configuration path
	DefaultGenesis = string("./chain/westend/genesis.json")

	// AccountConfig

	// DefaultKey Default account key
	DefaultKey = string("")
	// DefaultUnlock Default account unlock
	DefaultUnlock = string("")

	// CoreConfig

	// DefaultAuthority is true if the node is a block producer and a grandpa authority
	DefaultAuthority = true
	// DefaultRoles Default node roles
	DefaultRoles = common.FullNodeRole // authority node (see Table D.2)
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
	DefaultRPCModules = []string{
		"system", "author", "chain", "state", "rpc",
		"grandpa", "offchain", "childstate", "syncstate", "payment"}
	// DefaultRPCWSPort rpc websocket port
	DefaultRPCWSPort = uint32(8546)
)

const (
	// PprofConfig

	// DefaultPprofListeningAddress default pprof HTTP server listening address.
	DefaultPprofListeningAddress = "localhost:6060"

	// DefaultPprofBlockRate default block profile rate.
	// Set to 0 to disable profiling.
	DefaultPprofBlockRate = 0

	// DefaultPprofMutexRate default mutex profile rate.
	// Set to 0 to disable profiling.
	DefaultPprofMutexRate = 0
)
