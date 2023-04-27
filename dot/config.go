// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"fmt"
	"strings"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/pprof"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

// TODO: update config to have toml rules and perhaps un-export some fields,
// also type conversions might be needed from toml -> internal types (#1848)

// Config is a collection of configurations throughout the system
type Config struct {
	Global  GlobalConfig
	Log     LogConfig
	Init    InitConfig
	Account AccountConfig
	Core    CoreConfig
	Network NetworkConfig
	RPC     RPCConfig
	System  types.SystemInfo
	State   StateConfig
	Pprof   PprofConfig
}

// GlobalConfig is used for every node command
type GlobalConfig struct {
	Name               string
	ID                 string
	BasePath           string
	LogLvl             log.Level
	PrometheusExternal bool
	PrometheusPort     uint32
	NoTelemetry        bool
	TelemetryURLs      []genesis.TelemetryEndpoint
	RetainBlocks       uint32
	Pruning            pruner.Mode
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
	CoreLvl           log.Level
	DigestLvl         log.Level
	SyncLvl           log.Level
	NetworkLvl        log.Level
	RPCLvl            log.Level
	StateLvl          log.Level
	RuntimeLvl        log.Level
	BlockProducerLvl  log.Level
	FinalityGadgetLvl log.Level
}

func (l LogConfig) String() string {
	entries := []string{
		fmt.Sprintf("core: %s", l.CoreLvl),
		fmt.Sprintf("digest: %s", l.DigestLvl),
		fmt.Sprintf("sync: %s", l.SyncLvl),
		fmt.Sprintf("network: %s", l.NetworkLvl),
		fmt.Sprintf("rpc: %s", l.RPCLvl),
		fmt.Sprintf("state: %s", l.StateLvl),
		fmt.Sprintf("runtime: %s", l.RuntimeLvl),
		fmt.Sprintf("block producer: %s", l.BlockProducerLvl),
		fmt.Sprintf("finality gadget: %s", l.FinalityGadgetLvl),
	}
	return strings.Join(entries, ", ")
}

// InitConfig is the configuration for the node initialization
type InitConfig struct {
	Genesis string
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string
	Unlock string // TODO: change to []int (#1849)
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port              uint16
	Bootnodes         []string
	ProtocolID        string
	NoBootstrap       bool
	NoMDNS            bool
	MinPeers          int
	MaxPeers          int
	PersistentPeers   []string
	DiscoveryInterval time.Duration
	PublicIP          string
	PublicDNS         string
	NodeKey           string
	ListenAddress     string
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Roles            common.NetworkRole
	BabeAuthority    bool
	GrandpaAuthority bool
	WasmInterpreter  string
	GrandpaInterval  time.Duration
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled          bool
	External         bool
	Unsafe           bool
	UnsafeExternal   bool
	Port             uint32
	Host             string
	Modules          []string
	WSPort           uint32
	WS               bool
	WSExternal       bool
	WSUnsafe         bool
	WSUnsafeExternal bool
}

func (r *RPCConfig) isRPCEnabled() bool {
	return r.Enabled || r.External || r.Unsafe || r.UnsafeExternal
}

func (r *RPCConfig) isWSEnabled() bool {
	return r.WS || r.WSExternal || r.WSUnsafe || r.WSUnsafeExternal
}

// Strings returns the configuration in the format
// field1=value1 field2=value2.
func (r *RPCConfig) String() string {
	return "" +
		"enabled=" + fmt.Sprint(r.Enabled) + " " +
		"external=" + fmt.Sprint(r.External) + " " +
		"unsafe=" + fmt.Sprint(r.Unsafe) + " " +
		"unsafeexternal=" + fmt.Sprint(r.UnsafeExternal) + " " +
		"port=" + fmt.Sprint(r.Port) + " " +
		"host=" + r.Host + " " +
		"modules=" + strings.Join(r.Modules, ",") + " " +
		"wsport=" + fmt.Sprint(r.WSPort) + " " +
		"ws=" + fmt.Sprint(r.WS) + " " +
		"wsexternal=" + fmt.Sprint(r.WSExternal) + " " +
		"wsunsafe=" + fmt.Sprint(r.WSUnsafe) + " " +
		"wsunsafeexternal=" + fmt.Sprint(r.WSUnsafeExternal)
}

// StateConfig is the config for the State service
type StateConfig struct {
	Rewind uint
}

func (s *StateConfig) String() string {
	return "rewind " + fmt.Sprint(s.Rewind)
}

// networkServiceEnabled returns true if the network service is enabled
func networkServiceEnabled(config *cfg.Config) bool {
	return config.Core.Role != common.NoNetworkRole
}

// PprofConfig is the configuration for the pprof HTTP server.
type PprofConfig struct {
	Enabled  bool
	Settings pprof.Settings
}

func (p PprofConfig) String() string {
	if !p.Enabled {
		return "disabled"
	}

	return p.Settings.String()
}
