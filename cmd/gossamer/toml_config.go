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

package main

// Config is a collection of configurations throughout the system
type Config struct {
	Global  GlobalConfig  `toml:"global"`
	Log     LogConfig     `toml:"log"`
	Init    InitConfig    `toml:"init"`
	Account AccountConfig `toml:"account"`
	Core    CoreConfig    `toml:"core"`
	Network NetworkConfig `toml:"network"`
	RPC     RPCConfig     `toml:"rpc"`
}

// GlobalConfig is to marshal/unmarshal toml global config vars
type GlobalConfig struct {
	Name     string `toml:"name"`
	ID       string `toml:"id"`
	BasePath string `toml:"basepath"`
	LogLevel string `toml:"log"`
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
	CoreLvl           string `toml:"core"`
	SyncLvl           string `toml:"sync"`
	NetworkLvl        string `toml:"network"`
	RPCLvl            string `toml:"rpc"`
	StateLvl          string `toml:"state"`
	RuntimeLvl        string `toml:"runtime"`
	BlockProducerLvl  string `toml:"babe"`
	FinalityGadgetLvl string `toml:"grandpa"`
}

// InitConfig is the configuration for the node initialization
type InitConfig struct {
	GenesisRaw string `toml:"genesis-raw"`
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string `toml:"key"`
	Unlock string `toml:"unlock"`
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port        uint32   `toml:"port"`
	Bootnodes   []string `toml:"bootnodes"`
	ProtocolID  string   `toml:"protocol"`
	NoBootstrap bool     `toml:"nobootstrap"`
	NoMDNS      bool     `toml:"nomdns"`
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Authority        bool   `toml:"authority"`
	BabeAuthority    bool   `toml:"babe-authority"`
	GrandpaAuthority bool   `toml:"grandpa-authority"`
	BabeThreshold    string `toml:"babe-threshold"`
	SlotDuration     uint64 `toml:"slot-duration"`
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled   bool     `toml:"enabled"`
	Port      uint32   `toml:"port"`
	Host      string   `toml:"host"`
	Modules   []string `toml:"modules"`
	WSPort    uint32   `toml:"ws-port"`
	WSEnabled bool     `toml:"ws-enabled"`
}
