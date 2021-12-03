// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package toml

// Config is a collection of configurations throughout the system
type Config struct {
	Global  GlobalConfig  `toml:"global,omitempty"`
	Log     LogConfig     `toml:"log,omitempty"`
	Init    InitConfig    `toml:"init,omitempty"`
	Account AccountConfig `toml:"account,omitempty"`
	Core    CoreConfig    `toml:"core,omitempty"`
	Network NetworkConfig `toml:"network,omitempty"`
	RPC     RPCConfig     `toml:"rpc,omitempty"`
	Pprof   PprofConfig   `toml:"pprof,omitempty"`
}

// GlobalConfig is to marshal/unmarshal toml global config vars
type GlobalConfig struct {
	Name         string `toml:"name,omitempty"`
	ID           string `toml:"id,omitempty"`
	BasePath     string `toml:"basepath,omitempty"`
	LogLvl       string `toml:"log,omitempty"`
	MetricsPort  uint32 `toml:"metrics-port,omitempty"`
	RetainBlocks int64  `toml:"retain-blocks,omitempty"`
	Pruning      string `toml:"pruning,omitempty"`
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
	CoreLvl           string `toml:"core,omitempty"`
	SyncLvl           string `toml:"sync,omitempty"`
	NetworkLvl        string `toml:"network,omitempty"`
	RPCLvl            string `toml:"rpc,omitempty"`
	StateLvl          string `toml:"state,omitempty"`
	RuntimeLvl        string `toml:"runtime,omitempty"`
	BlockProducerLvl  string `toml:"babe,omitempty"`
	FinalityGadgetLvl string `toml:"grandpa,omitempty"`
}

// InitConfig is the configuration for the node initialization
type InitConfig struct {
	Genesis string `toml:"genesis,omitempty"`
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string `toml:"key,omitempty"`
	Unlock string `toml:"unlock,omitempty"`
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port              uint16   `toml:"port,omitempty"`
	Bootnodes         []string `toml:"bootnodes,omitempty"`
	ProtocolID        string   `toml:"protocol,omitempty"`
	NoBootstrap       bool     `toml:"nobootstrap,omitempty"`
	NoMDNS            bool     `toml:"nomdns,omitempty"`
	MinPeers          int      `toml:"min-peers,omitempty"`
	MaxPeers          int      `toml:"max-peers,omitempty"`
	PersistentPeers   []string `toml:"persistent-peers,omitempty"`
	DiscoveryInterval int      `toml:"discovery-interval,omitempty"`
	PublicIP          string   `toml:"public-ip,omitempty"`
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Roles            byte   `toml:"roles,omitempty"`
	BabeAuthority    bool   `toml:"babe-authority"`
	GrandpaAuthority bool   `toml:"grandpa-authority"`
	SlotDuration     uint64 `toml:"slot-duration,omitempty"`
	EpochLength      uint64 `toml:"epoch-length,omitempty"`
	WasmInterpreter  string `toml:"wasm-interpreter,omitempty"`
	GrandpaInterval  uint32 `toml:"grandpa-interval,omitempty"`
	BABELead         bool   `toml:"babe-lead,omitempty"`
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled          bool     `toml:"enabled,omitempty"`
	Unsafe           bool     `toml:"unsafe,omitempty"`
	UnsafeExternal   bool     `toml:"unsafe-external,omitempty"`
	External         bool     `toml:"external,omitempty"`
	Port             uint32   `toml:"port,omitempty"`
	Host             string   `toml:"host,omitempty"`
	Modules          []string `toml:"modules,omitempty"`
	WSPort           uint32   `toml:"ws-port,omitempty"`
	WS               bool     `toml:"ws,omitempty"`
	WSExternal       bool     `toml:"ws-external,omitempty"`
	WSUnsafe         bool     `toml:"ws-unsafe,omitempty"`
	WSUnsafeExternal bool     `toml:"ws-unsafe-external,omitempty"`
}

// PprofConfig contains the configuration for Pprof.
type PprofConfig struct {
	Enabled          bool   `toml:"enabled,omitempty"`
	ListeningAddress string `toml:"listening-address,omitempty"`
	BlockRate        int    `toml:"block-rate,omitempty"`
	MutexRate        int    `toml:"mutex-rate,omitempty"`
}
