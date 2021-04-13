// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
}

// GlobalConfig is to marshal/unmarshal toml global config vars
type GlobalConfig struct {
	Name        string `toml:"name,omitempty"`
	ID          string `toml:"id,omitempty"`
	BasePath    string `toml:"basepath,omitempty"`
	LogLvl      string `toml:"log,omitempty"`
	MetricsPort uint32 `toml:"metrics-port,omitempty"`
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
	Port            uint32   `toml:"port,omitempty"`
	Bootnodes       []string `toml:"bootnodes,omitempty"`
	ProtocolID      string   `toml:"protocol,omitempty"`
	NoBootstrap     bool     `toml:"nobootstrap,omitempty"`
	NoMDNS          bool     `toml:"nomdns,omitempty"`
	MinPeers        int      `toml:"min-peers,omitempty"`
	MaxPeers        int      `toml:"max-peers,omitempty"`
	PersistentPeers []string `toml:"persistent-peers,omitempty"`
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Roles                    byte   `toml:"roles,omitempty"`
	BabeAuthority            bool   `toml:"babe-authority"`
	GrandpaAuthority         bool   `toml:"grandpa-authority"`
	BabeThresholdNumerator   uint64 `toml:"babe-threshold-numerator,omitempty"`
	BabeThresholdDenominator uint64 `toml:"babe-threshold-denominator,omitempty"`
	SlotDuration             uint64 `toml:"slot-duration,omitempty"`
	EpochLength              uint64 `toml:"epoch-length,omitempty"`
	WasmInterpreter          string `toml:"wasm-interpreter,omitempty"`
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled    bool     `toml:"enabled,omitempty"`
	External   bool     `toml:"external,omitempty"`
	Port       uint32   `toml:"port,omitempty"`
	Host       string   `toml:"host,omitempty"`
	Modules    []string `toml:"modules,omitempty"`
	WSPort     uint32   `toml:"ws-port,omitempty"`
	WS         bool     `toml:"ws,omitempty"`
	WSExternal bool     `toml:"ws-external,omitempty"`
}
