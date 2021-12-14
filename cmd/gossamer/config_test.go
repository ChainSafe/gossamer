// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/chain/dev"
	"github.com/ChainSafe/gossamer/chain/gssmr"
	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
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
			"Test gossamer --chain gssmr",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"gssmr", dot.GssmrConfig().Global.Name, gssmr.DefaultPruningMode, gssmr.DefaultRetainBlocks},
			dot.GssmrConfig(),
		},
		{
			"Test gossamer --chain kusama",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"kusama", dot.KusamaConfig().Global.Name, gssmr.DefaultPruningMode, gssmr.DefaultRetainBlocks},
			dot.KusamaConfig(),
		},
		{
			"Test gossamer --chain polkadot",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"polkadot", dot.PolkadotConfig().Global.Name, gssmr.DefaultPruningMode, gssmr.DefaultRetainBlocks},
			dot.PolkadotConfig(),
		},
		{
			"Test gossamer --chain dev",
			[]string{"chain", "name", "pruning", "retain-blocks"},
			[]interface{}{"dev", dot.DevConfig().Global.Name, dev.DefaultPruningMode, dev.DefaultRetainBlocks},
			dot.DevConfig(),
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			cfg.System = types.SystemInfo{}
			require.Equal(t, c.expected, cfg)
		})
	}
}

// TestInitConfigFromFlags tests createDotInitConfig using relevant init flags
func TestInitConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), "test_genesis", dev.DefaultPruningMode, dev.DefaultRetainBlocks},
			dot.InitConfig{
				Genesis: "test_genesis",
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)
			cfg, err := createInitConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, cfg.Init)
		})
	}
}

// TestGlobalConfigFromFlags tests createDotGlobalConfig using relevant global flags
func TestGlobalConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test kusama --chain",
			[]string{"config", "chain", "name"},
			[]interface{}{testCfgFile.Name(), "kusama", dot.KusamaConfig().Global.Name},
			dot.GlobalConfig{
				Name:           dot.KusamaConfig().Global.Name,
				ID:             "ksmcc3",
				BasePath:       dot.KusamaConfig().Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test gossamer --name",
			[]string{"config", "name"},
			[]interface{}{testCfgFile.Name(), "test_name"},
			dot.GlobalConfig{
				Name:           "test_name",
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test gossamer --basepath",
			[]string{"config", "basepath", "name"},
			[]interface{}{testCfgFile.Name(), "test_basepath", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       "test_basepath",
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test gossamer --roles",
			[]string{"config", "roles", "name"},
			[]interface{}{testCfgFile.Name(), "1", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test gossamer --publish-metrics",
			[]string{"config", "publish-metrics", "name"},
			[]interface{}{testCfgFile.Name(), true, testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: true,
				MetricsPort:    testCfg.Global.MetricsPort,
			},
		},
		{
			"Test gossamer --metrics-port",
			[]string{"config", "metrics-port", "name"},
			[]interface{}{testCfgFile.Name(), "9871", testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    uint32(9871),
			},
		},
		{
			"Test gossamer --no-telemetry",
			[]string{"config", "no-telemetry", "name"},
			[]interface{}{testCfgFile.Name(), true, testCfg.Global.Name},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
				NoTelemetry:    true,
			},
		},
		{
			"Test gossamer --telemetry-url",
			[]string{"config", "telemetry-url", "name"},
			[]interface{}{
				testCfgFile.Name(),
				[]string{"ws://localhost:8001/submit 0", "ws://foo/bar 0"},
				testCfg.Global.Name,
			},
			dot.GlobalConfig{
				Name:           testCfg.Global.Name,
				ID:             testCfg.Global.ID,
				BasePath:       testCfg.Global.BasePath,
				LogLvl:         log.Info,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
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
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)

			require.Equal(t, c.expected, cfg.Global)
		})
	}
}

func TestGlobalConfigFromFlagsFails(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
				testCfgFile.Name(),
				[]string{"ws://localhost:8001/submit"},
				testCfg.Global.Name,
			},
			"could not set global config from flags: telemetry-url must be in the format 'URL VERBOSITY'",
		},
		{
			"Test gossamer invalid --telemetry-url invalid verbosity",
			[]string{"config", "telemetry-url", "name"},
			[]interface{}{
				testCfgFile.Name(),
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
			require.Nil(t, err)

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
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), "alice"},
			dot.AccountConfig{
				Key:    "alice",
				Unlock: testCfg.Account.Unlock,
			},
		},
		{
			"Test gossamer --unlock",
			[]string{"config", "key", "unlock"},
			[]interface{}{testCfgFile.Name(), "alice", "0"},
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
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, cfg.Account)
		})
	}
}

// TestCoreConfigFromFlags tests createDotCoreConfig using relevant core flags
func TestCoreConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), "4"},
			dot.CoreConfig{
				Roles:            4,
				BabeAuthority:    true,
				GrandpaAuthority: true,
				WasmInterpreter:  gssmr.DefaultWasmInterpreter,
				GrandpaInterval:  testCfg.Core.GrandpaInterval,
			},
		},
		{
			"Test gossamer --roles",
			[]string{"config", "roles"},
			[]interface{}{testCfgFile.Name(), "0"},
			dot.CoreConfig{
				Roles:            0,
				BabeAuthority:    false,
				GrandpaAuthority: false,
				WasmInterpreter:  gssmr.DefaultWasmInterpreter,
				GrandpaInterval:  testCfg.Core.GrandpaInterval,
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, cfg.Core)
		})
	}
}

// TestNetworkConfigFromFlags tests createDotNetworkConfig using relevant network flags
func TestNetworkConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), "1234"},
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
			[]interface{}{testCfgFile.Name(), "peer1,peer2"},
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
			[]interface{}{testCfgFile.Name(), "/gossamer/test/0"},
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
			[]interface{}{testCfgFile.Name(), "true"},
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
			[]interface{}{testCfgFile.Name(), "true"},
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
			[]interface{}{testCfgFile.Name(), "10.0.5.2"},
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
			[]interface{}{testCfgFile.Name(), "alice"},
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
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, cfg.Network)
		})
	}
}

// TestRPCConfigFromFlags tests createDotRPCConfig using relevant rpc flags
func TestRPCConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
			[]interface{}{testCfgFile.Name(), "true"},
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
			[]interface{}{testCfgFile.Name(), "false"},
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
			[]interface{}{testCfgFile.Name(), "true"},
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
			[]interface{}{testCfgFile.Name(), "false"},
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
			[]interface{}{testCfgFile.Name(), "testhost"}, // rpc must be enabled
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
			[]interface{}{testCfgFile.Name(), "5678"}, // rpc must be enabled
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
			[]interface{}{testCfgFile.Name(), "mod1,mod2"}, // rpc must be enabled
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
			[]interface{}{testCfgFile.Name(), "7070"},
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
			[]interface{}{testCfgFile.Name(), true},
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
			[]interface{}{testCfgFile.Name(), false},
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
			[]interface{}{testCfgFile.Name(), true},
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
			[]interface{}{testCfgFile.Name(), false},
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
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, cfg.RPC)
		})
	}
}

// TestUpdateConfigFromGenesisJSON tests updateDotConfigFromGenesisJSON
func TestUpdateConfigFromGenesisJSON(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	defer utils.RemoveTestDir(t)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile.Name(), genFile.Name(), testCfg.Global.Name},
	)
	require.Nil(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsPort:    testCfg.Global.MetricsPort,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init: dot.InitConfig{
			Genesis: genFile.Name(),
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: testCfg.Network,
		RPC:     testCfg.RPC,
		System:  testCfg.System,
		Pprof:   testCfg.Pprof,
	}

	cfg, err := createDotConfig(ctx)
	require.Nil(t, err)

	cfg.Init.Genesis = genFile.Name()
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

// TestUpdateConfigFromGenesisJSON_Default tests updateDotConfigFromGenesisJSON
// using the default genesis path if no genesis path is provided (ie, an empty
// genesis value provided in the toml configuration file or with --genesis "")
func TestUpdateConfigFromGenesisJSON_Default(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)

	defer utils.RemoveTestDir(t)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile.Name(), "", testCfg.Global.Name},
	)
	require.Nil(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsPort:    testCfg.Global.MetricsPort,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
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
	require.Nil(t, err)
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

func TestUpdateConfigFromGenesisData(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	defer utils.RemoveTestDir(t)

	ctx, err := newTestContext(
		t.Name(),
		[]string{"config", "genesis", "name"},
		[]interface{}{testCfgFile.Name(), genFile.Name(), testCfg.Global.Name},
	)
	require.Nil(t, err)

	expected := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           testCfg.Global.Name,
			ID:             testCfg.Global.ID,
			BasePath:       testCfg.Global.BasePath,
			LogLvl:         testCfg.Global.LogLvl,
			PublishMetrics: testCfg.Global.PublishMetrics,
			MetricsPort:    testCfg.Global.MetricsPort,
			TelemetryURLs:  testCfg.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init: dot.InitConfig{
			Genesis: genFile.Name(),
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
	require.Nil(t, err)

	cfg.Init.Genesis = genFile.Name()

	db, err := utils.SetupDatabase(cfg.Global.BasePath, false)
	require.Nil(t, err)

	gen, err := genesis.NewGenesisFromJSONRaw(genFile.Name())
	require.Nil(t, err)

	err = state.NewBaseState(db).StoreGenesisData(gen.GenesisData())
	require.Nil(t, err)

	err = db.Close()
	require.Nil(t, err)

	err = updateDotConfigFromGenesisData(ctx, cfg) // name should not be updated if provided as flag value
	require.Nil(t, err)
	cfg.System = types.SystemInfo{}
	require.Equal(t, expected, cfg)
}

func TestGlobalNodeName_WhenNodeAlreadyHasStoredName(t *testing.T) {
	// Initialise a node with a random name
	globalName := dot.RandomNodeName()

	cfg := dot.NewTestConfig(t)
	cfg.Global.Name = globalName
	require.NotNil(t, cfg)

	genPath := dot.NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genPath

	err := dot.InitNode(cfg)
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
			require.Nil(t, err)
			createdCfg, err := createDotConfig(ctx)
			require.Nil(t, err)
			require.Equal(t, c.expected, createdCfg.Global.Name)
		})
	}
}

func TestGlobalNodeNamePriorityOrder(t *testing.T) {
	cfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, cfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

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
		[]interface{}{cfg.Global.BasePath, "mydefinedname", testCfgFile.Name()},
		"mydefinedname",
	}

	c := whenNameFlagIsDefined
	t.Run(c.description, func(t *testing.T) {
		ctx, err := newTestContext(c.description, c.flags, c.values)
		require.Nil(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.Nil(t, err)
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
		[]interface{}{cfg.Global.BasePath, testCfgFile.Name()},
		cfg.Global.Name,
	}

	c = whenNameIsDefinedOnTomlConfig
	t.Run(c.description, func(t *testing.T) {
		ctx, err := newTestContext(c.description, c.flags, c.values)
		require.Nil(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.Nil(t, err)
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
		require.Nil(t, err)
		createdCfg, err := createDotConfig(ctx)
		require.Nil(t, err)
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
		"no value with default": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			defaultLevel: log.Error,
			level:        log.Error,
		},
		"flag integer value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "1"}),
			flagName:     "x",
			level:        log.Error,
		},
		"flag string value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "eror"}),
			flagName:     "x",
			level:        log.Error,
		},
		"flag bad string value": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "garbage"}),
			flagName:     "x",
			err:          errors.New("cannot parse log level string: level is not recognised: garbage"),
		},
		"toml integer value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "1",
			level:        log.Error,
		},
		"toml string value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "eror",
			level:        log.Error,
		},
		"toml bad string value": {
			flagsKVStore: newMockGetStringer(map[string]string{}),
			tomlValue:    "garbage",
			err:          errors.New("cannot parse log level string: level is not recognised: garbage"),
		},
		"flag takes precedence": {
			flagsKVStore: newMockGetStringer(map[string]string{"x": "eror"}),
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
		"empty string": {
			err: errors.New("cannot parse log level string: level is not recognised: "),
		},
		"valid integer": {
			logLevelString: "1",
			logLevel:       log.Error,
		},
		"minus one": {
			logLevelString: "-1",
			err:            errors.New("log level integer can only be between 0 and 5 included: log level given: -1"),
		},
		"over 5": {
			logLevelString: "6",
			err:            errors.New("log level integer can only be between 0 and 5 included: log level given: 6"),
		},
		"valid string": {
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
		"no value": {
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
				SyncLvl:           log.Info,
				NetworkLvl:        log.Info,
				RPCLvl:            log.Info,
				StateLvl:          log.Info,
				RuntimeLvl:        log.Info,
				BlockProducerLvl:  log.Info,
				FinalityGadgetLvl: log.Info,
			},
		},
		"some values": {
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
