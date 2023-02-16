// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestConfigFromChainFlag tests createDotConfig using the --chain flag
func TestConfigFromChainFlag(t *testing.T) {
	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    *dot.Config
	}{
		{
			"Test gossamer --chain kusama",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"kusama", dot.KusamaConfig().Global.Name, kusama.DefaultPruningMode, kusama.DefaultRetainBlocks},
			dot.KusamaConfig(),
		},
		{
			"Test gossamer --chain polkadot",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"polkadot", dot.PolkadotConfig().Global.Name,
				polkadot.DefaultPruningMode, polkadot.DefaultRetainBlocks},
			dot.PolkadotConfig(),
		},
		{
			"Test gossamer --chain westend-dev",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"westend-dev", dot.WestendDevConfig().Global.Name, "archive", uint32(512)},
			dot.WestendDevConfig(),
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			cfg.System = types.SystemInfo{}
			require.Equal(t, c.expected, cfg)
		})
	}
}

// TestInitConfigFromFlags tests createDotInitConfig using relevant init flags
func TestInitConfigFromFlags(t *testing.T) {
	_, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.InitConfig
	}{
		{
			"Test gossamer --genesis",
			[]string{"config", "genesis", "pruning", "retain-blocks"},
			[]interface{}{testCfgFile, "test_genesis", "archive", uint32(512)},
			dot.InitConfig{
				Genesis: "test_genesis",
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createInitConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, cfg.Init)
		})
	}
}

// TestGlobalConfigFromFlags tests createDotGlobalConfig using relevant global flags
func TestGlobalConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.GlobalConfig
	}{
		{
			"Test gossamer --config",
			[]string{"config", "name"},
			[]interface{}{testCfgFile, testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
			},
		},
		{
			"Test kusama --chain",
			[]string{"config", "chain", "name"},
			[]interface{}{testCfgFile, "kusama", dot.KusamaConfig().Global.Name},
			dot.GlobalConfig{
				Name:           dot.KusamaConfig().Global.Name,
				ID:             "ksmcc3",
				BasePath:       dot.KusamaConfig().Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: false,
				MetricsAddress: "localhost:9876",
			},
		},
		{
			"Test gossamer --name",
			[]string{"config", "name"},
			[]interface{}{testCfgFile, "test_name"},
			dot.GlobalConfig{
				Name:           "test_name",
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
			},
		},
		{
			"Test gossamer --basepath",
			[]string{"config", "basepath", "name"},
			[]interface{}{testCfgFile, "test_basepath", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       "test_basepath",
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
			},
		},
		{
			"Test gossamer --roles",
			[]string{"config", "roles", "name"},
			[]interface{}{testCfgFile, "1", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
			},
		},
		{
			"Test gossamer --publish-metrics",
			[]string{"config", "publish-metrics", "name"},
			[]interface{}{testCfgFile, true, testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: true,
				MetricsAddress: testCfg.Global.MetricsAddress,
			},
		},
		{
			"Test gossamer --metrics-address",
			[]string{"config", "metrics-address", "name"},
			[]interface{}{testCfgFile, ":9871", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: ":9871",
			},
		},
		{
			"Test gossamer --no-telemetry",
			[]string{"config", "no-telemetry", "name"},
			[]interface{}{testCfgFile, true, testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
				NoTelemetry:    true,
			},
		},
		{
			"Test gossamer --telemetry-url",
			[]string{"config", "telemetry-url", "name"},
			[]interface{}{
				testCfgFile,
				[]string{"ws://localhost:8001/submit 0", "ws://foo/bar 0"},
				testCfg.Global.Name,
			},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsAddress: testCfg.Global.MetricsAddress,
				NoTelemetry:    false,
				TelemetryURLs: []genesis.TelemetryEndpoint{
					{Endpoint: "ws://localhost:8001/submit", Verbosity: 0},
					{Endpoint: "ws://foo/bar", Verbosity: 0},
				},
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)

			require.Equal(t, c.expected, cfg.Global)
		})
	}
}

func TestGlobalConfigFromFlagsFails(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		err         string
	}{
		{
			"Test gossamer --telemetry-url invalid format",
			[]string{"config", "telemetry-url", "name"},
			[]interface{}{
				testCfgFile,
				[]string{"ws://localhost:8001/submit"},
				testCfg.Global.Name,
			},
			"could not set global config from flags: telemetry-url must be in the format 'URL VERBOSITY'",
		},
		{
			"Test gossamer invalid --telemetry-url invalid verbosity",
			[]string{"config", "telemetry-url", "name"},
			[]interface{}{
				testCfgFile,
				[]string{"ws://foo/bar k"},
				testCfg.Global.Name,
			},
			"could not set global config from flags: could not parse verbosity from telemetry-url: " +
				`strconv.Atoi: parsing "k": invalid syntax`,
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)

			cfg, err := createDotConfig(ctx)
			require.NotNil(t, err)
			require.Nil(t, cfg)
			require.Equal(t, c.err, err.Error())
		})
	}
}

// TestAccountConfigFromFlags tests createDotAccountConfig using relevant account flags
func TestAccountConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.AccountConfig
	}{
		{
			"Test gossamer --key",
			[]string{"config", "key"},
			[]interface{}{testCfgFile, "alice"},
			dot.AccountConfig{
				Key:    "alice",
				Unlock: testCfg.Account.Unlock,
			},
		},
		{
			"Test gossamer --unlock",
			[]string{"config", "key", "unlock"},
			[]interface{}{testCfgFile, "alice", "0"},
			dot.AccountConfig{
				Key:    "alice",
				Unlock: "0",
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, cfg.Account)
		})
	}
}

// TestCoreConfigFromFlags tests createDotCoreConfig using relevant core flags
func TestCoreConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.CoreConfig
	}{
		{
			"Test gossamer --roles",
			[]string{"config", "roles"},
			[]interface{}{testCfgFile, "4"},
			dot.CoreConfig{
				Roles:            4,
				BabeAuthority:    true,
				GrandpaAuthority: true,
				WasmInterpreter:  wasmer.Name,
				GrandpaInterval:  testCfg.Core.GrandpaInterval,
			},
		},
		{
			"Test gossamer --roles",
			[]string{"config", "roles"},
			[]interface{}{testCfgFile, "0"},
			dot.CoreConfig{
				Roles:            0,
				BabeAuthority:    false,
				GrandpaAuthority: false,
				WasmInterpreter:  wasmer.Name,
				GrandpaInterval:  testCfg.Core.GrandpaInterval,
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, cfg.Core)
		})
	}
}

// TestNetworkConfigFromFlags tests createDotNetworkConfig using relevant network flags
func TestNetworkConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.NetworkConfig
	}{
		{
			"Test gossamer --port",
			[]string{"config", "port"},
			[]interface{}{testCfgFile, "1234"},
			dot.NetworkConfig{
				Port:              1234,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            testCfg.Network.NoMDNS,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
			},
		},
		{
			"Test gossamer --bootnodes",
			[]string{"config", "bootnodes"},
			[]interface{}{testCfgFile, "peer1,peer2"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         []string{"peer1", "peer2"},
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            testCfg.Network.NoMDNS,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
			},
		},
		{
			"Test gossamer --protocol",
			[]string{"config", "protocol"},
			[]interface{}{testCfgFile, "/gossamer/test/0"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        "/gossamer/test/0",
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            testCfg.Network.NoMDNS,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
			},
		},
		{
			"Test gossamer --nobootstrap",
			[]string{"config", "nobootstrap"},
			[]interface{}{testCfgFile, "true"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       true,
				NoMDNS:            testCfg.Network.NoMDNS,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
			},
		},
		{
			"Test gossamer --nomdns",
			[]string{"config", "nomdns"},
			[]interface{}{testCfgFile, "true"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            true,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
			},
		},
		{
			"Test gossamer --pubip",
			[]string{"config", "pubip"},
			[]interface{}{testCfgFile, "10.0.5.2"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            false,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
				PublicIP:          "10.0.5.2",
			},
		},
		{
			"Test gossamer --pubdns",
			[]string{"config", "pubdns"},
			[]interface{}{testCfgFile, "alice"},
			dot.NetworkConfig{
				Port:              testCfg.Network.Port,
				Bootnodes:         testCfg.Network.Bootnodes,
				ProtocolID:        testCfg.Network.ProtocolID,
				NoBootstrap:       testCfg.Network.NoBootstrap,
				NoMDNS:            false,
				DiscoveryInterval: time.Second * 10,
				MinPeers:          testCfg.Network.MinPeers,
				MaxPeers:          testCfg.Network.MaxPeers,
				PublicDNS:         "alice",
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, cfg.Network)
		})
	}
}

// TestRPCConfigFromFlags tests createDotRPCConfig using relevant rpc flags
func x(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.RPCConfig
	}{
		{
			"Test gossamer --rpc",
			[]string{"config", "rpc"},
			[]interface{}{testCfgFile, "true"},
			dot.RPCConfig{
				Enabled:    true,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpc false",
			[]string{"config", "rpc"},
			[]interface{}{testCfgFile, "false"},
			dot.RPCConfig{
				Enabled:    false,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpc-external",
			[]string{"config", "rpc-external"},
			[]interface{}{testCfgFile, "true"},
			dot.RPCConfig{
				Enabled:    true,
				External:   true,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpc-external false",
			[]string{"config", "rpc-external"},
			[]interface{}{testCfgFile, "false"},
			dot.RPCConfig{
				Enabled:    true,
				External:   false,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpchost",
			[]string{"config", "rpchost"},
			[]interface{}{testCfgFile, "testhost"}, // rpc must be enabled
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       "testhost",
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpcport",
			[]string{"config", "rpcport"},
			[]interface{}{testCfgFile, "5678"}, // rpc must be enabled
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       5678,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --rpcsmods",
			[]string{"config", "rpcmods"},
			[]interface{}{testCfgFile, "mod1,mod2"}, // rpc must be enabled
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    []string{"mod1", "mod2"},
				WSPort:     testCfg.RPC.WSPort,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --wsport",
			[]string{"config", "wsport"},
			[]interface{}{testCfgFile, "7070"},
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     7070,
				WS:         testCfg.RPC.WS,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --ws",
			[]string{"config", "ws"},
			[]interface{}{testCfgFile, true},
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         true,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --ws false",
			[]string{"config", "w"},
			[]interface{}{testCfgFile, false},
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         false,
				WSExternal: testCfg.RPC.WSExternal,
			},
		},
		{
			"Test gossamer --ws-external",
			[]string{"config", "ws-external"},
			[]interface{}{testCfgFile, true},
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         true,
				WSExternal: true,
			},
		},
		{
			"Test gossamer --ws-external false",
			[]string{"config", "ws-external"},
			[]interface{}{testCfgFile, false},
			dot.RPCConfig{
				Enabled:    testCfg.RPC.Enabled,
				External:   testCfg.RPC.External,
				Port:       testCfg.RPC.Port,
				Host:       testCfg.RPC.Host,
				Modules:    testCfg.RPC.Modules,
				WSPort:     testCfg.RPC.WSPort,
				WS:         true,
				WSExternal: false,
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			cfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, cfg.RPC)
		})
	}
}

// TestUpdateConfigFromGenesisJSON tests updateDotConfigFromGenesisJSON
func TestUpdateConfigFromGenesisJSON(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile, genFile, testCfg.Global.Name},
	)
	require.NoError(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsAddress: testCfg.Global.MetricsAddress,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			DigestLvl:         log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init: dot.InitConfig{
			Genesis: genFile,
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: testCfg.Network,
		RPC:     testCfg.RPC,
		System:  testCfg.System,
		Pprof:   testCfg.Pprof,
	}

	cfg, err := createDotConfig(ctx)
	require.NoError(t, err)

	cfg.Init.Genesis = genFile
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

// TestUpdateConfigFromGenesisJSON_Default tests updateDotConfigFromGenesisJSON
// using the default genesis path if no genesis path is provided (ie, an empty
// genesis value provided in the toml configuration file or with --genesis "")
func TestUpdateConfigFromGenesisJSON_Default(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile, "", testCfg.Global.Name},
	)
	require.NoError(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsAddress: testCfg.Global.MetricsAddress,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			DigestLvl:         log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init: dot.InitConfig{
			Genesis: DefaultCfg().Init.Genesis,
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: testCfg.Network,
		RPC:     testCfg.RPC,
		System:  testCfg.System,
		Pprof:   testCfg.Pprof,
	}

	cfg, err := createDotConfig(ctx)
	require.NoError(t, err)
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

func TestUpdateConfigFromGenesisData(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile, genFile, testCfg.Global.Name},
	)
	require.NoError(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsAddress: testCfg.Global.MetricsAddress,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			DigestLvl:         log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init: dot.InitConfig{
			Genesis: genFile,
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: dot.NetworkConfig{
			Port:              testCfg.Network.Port,
			Bootnodes:         []string{},
			ProtocolID:        testCfg.Network.ProtocolID,
			NoBootstrap:       testCfg.Network.NoBootstrap,
			NoMDNS:            testCfg.Network.NoMDNS,
			DiscoveryInterval: testCfg.Network.DiscoveryInterval,
			MinPeers:          testCfg.Network.MinPeers,
			MaxPeers:          testCfg.Network.MaxPeers,
		},
		RPC:    testCfg.RPC,
		System: testCfg.System,
		Pprof:  testCfg.Pprof,
	}

	cfg, err := createDotConfig(ctx)
	require.NoError(t, err)

	cfg.Init.Genesis = genFile

	db, err := utils.SetupDatabase(cfg.Global.BasePath, false)
	require.NoError(t, err)

	gen, err := genesis.NewGenesisFromJSONRaw(genFile)
	require.NoError(t, err)

	err = state.NewBaseState(db).StoreGenesisData(gen.GenesisData())
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	err = updateDotConfigFromGenesisData(ctx, cfg) // name should not be updated if provided as flag value
	require.NoError(t, err)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

func TestGlobalNodeName_WhenNodeAlreadyHasStoredName(t *testing.T) {
	// Initialise a node with a random name
	globalName := dot.RandomNodeName()

	cfg := newTestConfig(t)
	cfg.Global.Name = globalName

	runtimeFilePath, err := runtime.GetRuntime(context.Background(), runtime.NODE_RUNTIME)
	require.NoError(t, err)
	runtimeData, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	fp := utils.GetWestendDevRawGenesisPath(t)

	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       "test",
		ID:         "test",
		Bootnodes:  []string(nil),
		ProtocolID: "/gossamer/test/0",
		Genesis:    westendDevGenesis.GenesisFields(),
	}

	gen.Genesis.Raw = map[string]map[string]string{
		"top": {
			"0x3a636f6465": "0x" + hex.EncodeToString(runtimeData),
			"0xcf722c0832b5231d35e29f319ff27389f5032bfc7bfc3ba5ed7839f2042fb99f": "0x0000000000000001",
		},
	}

	genData, err := json.Marshal(gen)
	require.NoError(t, err)

	genPath := filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(genPath, genData, os.ModePerm)
	require.NoError(t, err)

	cfg.Core.Roles = common.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genPath

	err = dot.InitNode(cfg)
	require.NoError(t, err)

	// call another command and test the name
	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    string
	}{
		{
			"Test gossamer --roles --basepath",
			[]string{"basepath", "roles"},
			[]interface{}{cfg.Global.BasePath, "4"},
			globalName,
		},
		{
			"Test gossamer --roles",
			[]string{"basepath", "roles"},
			[]interface{}{cfg.Global.BasePath, "0"},
			globalName,
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)
			createdCfg, err := createDotConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, c.expected, createdCfg.Global.Name)
		})
	}
}

func TestGlobalNodeNamePriorityOrder(t *testing.T) {
	cfg, testCfgFile := newTestConfigWithFile(t)

	// call another command and test the name
	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	// when name flag is defined
	whenNameFlagIsDefined := struct {
		description string
		flags       []string
		values      []interface{}
		expected    string
	}{
		"Test gossamer --basepath --name --config",
		[]string{"basepath", "name", "config"},
		[]interface{}{cfg.Global.BasePath, "mydefinedname", testCfgFile},
		"mydefinedname",
	}

	c := whenNameFlagIsDefined
	t.Run(c.description, func(t *testing.T) {
		ctx, err := newTestContext(c.description, c.flags, c.values)
		require.NoError(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, c.expected, createdCfg.Global.Name)
	})

	// when name flag is not defined
	// then should load name from toml if it exists
	whenNameIsDefinedOnTomlConfig := struct {
		description string
		flags       []string
		values      []interface{}
		expected    string
	}{
		"Test gossamer --basepath --config",
		[]string{"basepath", "config"},
		[]interface{}{cfg.Global.BasePath, testCfgFile},
		cfg.Global.Name,
	}

	c = whenNameIsDefinedOnTomlConfig
	t.Run(c.description, func(t *testing.T) {
		ctx, err := newTestContext(c.description, c.flags, c.values)
		require.NoError(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, c.expected, createdCfg.Global.Name)
	})

	// when there is no name flag and no name in config
	// should check the load is initialised or generate a new random name
	cfg.Global.Name = ""

	whenThereIsNoName := struct {
		description string
		flags       []string
		values      []interface{}
	}{
		"Test gossamer --basepath",
		[]string{"basepath"},
		[]interface{}{cfg.Global.BasePath},
	}

	t.Run(c.description, func(t *testing.T) {
		ctx, err := newTestContext(whenThereIsNoName.description, whenThereIsNoName.flags, whenThereIsNoName.values)
		require.NoError(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, createdCfg.Global.Name)
		require.NotEqual(t, cfg.Global.Name, createdCfg.Global.Name)
	})
}

type mockGetStringer struct {
	kv map[string]string
}

func (m *mockGetStringer) String(key string) (value string) {
	return m.kv[key]
}

func newMockGetStringer(keyValue map[string]string) *mockGetStringer {
	kv := make(map[string]string, len(keyValue))
	for k, v := range keyValue {
		kv[k] = v
	}
	return &mockGetStringer{kv: kv}
}

func Test_getLogLevel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		flagsKVStore stringKVStore
		flagName     string
		tomlValue    string
		defaultLevel log.Level
		level        log.Level
		err          error
	}{
		"no_value_with_default": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			defaultLevel: log.Error,
			level:        log.Error,
		},
		"flag_integer_value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "1"}),
			flagName:     "x",
			level:        log.Error,
		},
		"flag_string_value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "error"}),
			flagName:     "x",
			level:        log.Error,
		},
		"flag_bad_string_value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "garbage"}),
			flagName:     "x",
			err:          errors.New("cannot parse log level string: level is not recognised: garbage"),
		},
		"toml_integer_value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "1",
			level:        log.Error,
		},
		"toml_string_value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "error",
			level:        log.Error,
		},
		"toml_bad_string_value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "garbage",
			err:          errors.New("cannot parse log level string: level is not recognised: garbage"),
		},
		"flag_takes_precedence": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "error"}),
			flagName:     "x",
			tomlValue:    "warn",
			level:        log.Error,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			level, err := getLogLevel(testCase.flagsKVStore, testCase.flagName,
				testCase.tomlValue, testCase.defaultLevel)

			if testCase.err != nil {
				assert.EqualError(t, err, testCase.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.level, level)
		})
	}
}

func Test_parseLogLevelString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		logLevelString string
		logLevel       log.Level
		err            error
	}{
		"empty_string": {
			err: errors.New("cannot parse log level string: level is not recognised: "),
		},
		"valid_integer": {
			logLevelString: "1",
			logLevel:       log.Error,
		},
		"minus_one": {
			logLevelString: "-1",
			err:            errors.New("log level integer can only be between 0 and 5 included: log level given: -1"),
		},
		"over_5": {
			logLevelString: "6",
			err:            errors.New("log level integer can only be between 0 and 5 included: log level given: 6"),
		},
		"valid_string": {
			logLevelString: "error",
			logLevel:       log.Error,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logLevel, err := parseLogLevelString(testCase.logLevelString)

			if testCase.err != nil {
				assert.EqualError(t, err, testCase.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.logLevel, logLevel)
		})
	}
}

func Test_setLogConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		ctx               stringKVStore
		initialCfg        ctoml.Config
		initialGlobalCfg  dot.GlobalConfig
		initialLogCfg     dot.LogConfig
		expectedCfg       ctoml.Config
		expectedGlobalCfg dot.GlobalConfig
		expectedLogCfg    dot.LogConfig
		err               error
	}{
		"no_value": {
			ctx: newMockGetStringer(map[string]string{}),
			expectedCfg: ctoml.Config{
				Global: ctoml.GlobalConfig{
					LogLvl: log.Info.String(),
				},
			},
			expectedGlobalCfg: dot.GlobalConfig{
				LogLvl: log.Info,
			},
			expectedLogCfg: dot.LogConfig{
				CoreLvl:           log.Info,
				DigestLvl:         log.Info,
				SyncLvl:           log.Info,
				NetworkLvl:        log.Info,
				RPCLvl:            log.Info,
				StateLvl:          log.Info,
				RuntimeLvl:        log.Info,
				BlockProducerLvl:  log.Info,
				FinalityGadgetLvl: log.Info,
			},
		},
		"some_values": {
			ctx: newMockGetStringer(map[string]string{}),
			initialCfg: ctoml.Config{
				Log: ctoml.LogConfig{
					CoreLvl:  log.Error.String(),
					SyncLvl:  log.Debug.String(),
					StateLvl: log.Warn.String(),
				},
			},
			expectedCfg: ctoml.Config{
				Global: ctoml.GlobalConfig{
					LogLvl: log.Info.String(),
				},
				Log: ctoml.LogConfig{
					CoreLvl:  log.Error.String(),
					SyncLvl:  log.Debug.String(),
					StateLvl: log.Warn.String(),
				},
			},
			expectedGlobalCfg: dot.GlobalConfig{
				LogLvl: log.Info,
			},
			expectedLogCfg: dot.LogConfig{
				CoreLvl:           log.Error,
				DigestLvl:         log.Info,
				SyncLvl:           log.Debug,
				NetworkLvl:        log.Info,
				RPCLvl:            log.Info,
				StateLvl:          log.Warn,
				RuntimeLvl:        log.Info,
				BlockProducerLvl:  log.Info,
				FinalityGadgetLvl: log.Info,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := setLogConfig(testCase.ctx, &testCase.initialCfg,
				&testCase.initialGlobalCfg, &testCase.initialLogCfg)

			if testCase.err != nil {
				assert.EqualError(t, err, testCase.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.expectedCfg, testCase.initialCfg)
			assert.Equal(t, testCase.expectedGlobalCfg, testCase.initialGlobalCfg)
			assert.Equal(t, testCase.expectedLogCfg, testCase.initialLogCfg)
		})
	}
}
