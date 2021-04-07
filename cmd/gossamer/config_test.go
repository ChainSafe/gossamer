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

import (
	"io/ioutil"
	"testing"

	"github.com/ChainSafe/gossamer/chain/gssmr"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestConfigFromChainFlag tests createDotConfig using the --chain flag
func TestConfigFromChainFlag(t *testing.T) {
	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    *dot.Config
	}{
		{
			"Test gossamer --chain gssmr",
			[]string{"chain", "name"},
			[]interface{}{"gssmr", dot.GssmrConfig().Global.Name},
			dot.GssmrConfig(),
		},
		{
			"Test gossamer --chain kusama",
			[]string{"chain", "name"},
			[]interface{}{"kusama", dot.KusamaConfig().Global.Name},
			dot.KusamaConfig(),
		},
		{
			"Test gossamer --chain polkadot",
			[]string{"chain", "name"},
			[]interface{}{"polkadot", dot.PolkadotConfig().Global.Name},
			dot.PolkadotConfig(),
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)
			cfg, err := createDotConfig(ctx)
			require.Nil(t, err)
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
	testApp.Writer = ioutil.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    dot.InitConfig
	}{
		{
			"Test gossamer --genesis",
			[]string{"config", "genesis"},
			[]interface{}{testCfgFile.Name(), "test_genesis"},
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
	testApp.Writer = ioutil.Discard

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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
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
				LogLvl:         log.LvlInfo,
				PublishMetrics: testCfg.Global.PublishMetrics,
				MetricsPort:    testCfg.Global.MetricsPort,
				NoTelemetry:    true,
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

// TestAccountConfigFromFlags tests createDotAccountConfig using relevant account flags
func TestAccountConfigFromFlags(t *testing.T) {
	testCfg, testCfgFile := newTestConfigWithFile(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)

	defer utils.RemoveTestDir(t)

	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard

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
	testApp.Writer = ioutil.Discard

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
	testApp.Writer = ioutil.Discard

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
				Port:        1234,
				Bootnodes:   testCfg.Network.Bootnodes,
				ProtocolID:  testCfg.Network.ProtocolID,
				NoBootstrap: testCfg.Network.NoBootstrap,
				NoMDNS:      testCfg.Network.NoMDNS,
			},
		},
		{
			"Test gossamer --bootnodes",
			[]string{"config", "bootnodes"},
			[]interface{}{testCfgFile.Name(), "peer1,peer2"},
			dot.NetworkConfig{
				Port:        testCfg.Network.Port,
				Bootnodes:   []string{"peer1", "peer2"},
				ProtocolID:  testCfg.Network.ProtocolID,
				NoBootstrap: testCfg.Network.NoBootstrap,
				NoMDNS:      testCfg.Network.NoMDNS,
			},
		},
		{
			"Test gossamer --protocol",
			[]string{"config", "protocol"},
			[]interface{}{testCfgFile.Name(), "/gossamer/test/0"},
			dot.NetworkConfig{
				Port:        testCfg.Network.Port,
				Bootnodes:   testCfg.Network.Bootnodes,
				ProtocolID:  "/gossamer/test/0",
				NoBootstrap: testCfg.Network.NoBootstrap,
				NoMDNS:      testCfg.Network.NoMDNS,
			},
		},
		{
			"Test gossamer --nobootstrap",
			[]string{"config", "nobootstrap"},
			[]interface{}{testCfgFile.Name(), "true"},
			dot.NetworkConfig{
				Port:        testCfg.Network.Port,
				Bootnodes:   testCfg.Network.Bootnodes,
				ProtocolID:  testCfg.Network.ProtocolID,
				NoBootstrap: true,
				NoMDNS:      testCfg.Network.NoMDNS,
			},
		},
		{
			"Test gossamer --nomdns",
			[]string{"config", "nomdns"},
			[]interface{}{testCfgFile.Name(), "true"},
			dot.NetworkConfig{
				Port:        testCfg.Network.Port,
				Bootnodes:   testCfg.Network.Bootnodes,
				ProtocolID:  testCfg.Network.ProtocolID,
				NoBootstrap: testCfg.Network.NoBootstrap,
				NoMDNS:      true,
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
	testApp.Writer = ioutil.Discard

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
		},
		Log: dot.LogConfig{
			CoreLvl:           log.LvlInfo,
			SyncLvl:           log.LvlInfo,
			NetworkLvl:        log.LvlInfo,
			RPCLvl:            log.LvlInfo,
			StateLvl:          log.LvlInfo,
			RuntimeLvl:        log.LvlInfo,
			BlockProducerLvl:  log.LvlInfo,
			FinalityGadgetLvl: log.LvlInfo,
		},
		Init: dot.InitConfig{
			Genesis: genFile.Name(),
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: testCfg.Network,
		RPC:     testCfg.RPC,
		System:  testCfg.System,
	}

	cfg, err := createDotConfig(ctx)
	require.Nil(t, err)

	cfg.Init.Genesis = genFile.Name()
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
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
		},
		Log: dot.LogConfig{
			CoreLvl:           log.LvlInfo,
			SyncLvl:           log.LvlInfo,
			NetworkLvl:        log.LvlInfo,
			RPCLvl:            log.LvlInfo,
			StateLvl:          log.LvlInfo,
			RuntimeLvl:        log.LvlInfo,
			BlockProducerLvl:  log.LvlInfo,
			FinalityGadgetLvl: log.LvlInfo,
		},
		Init: dot.InitConfig{
			Genesis: DefaultCfg().Init.Genesis,
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: testCfg.Network,
		RPC:     testCfg.RPC,
		System:  testCfg.System,
	}

	expected.Core.BabeThresholdNumerator = 0
	expected.Core.BabeThresholdDenominator = 0

	cfg, err := createDotConfig(ctx)
	require.Nil(t, err)
	updateDotConfigFromGenesisJSONRaw(*dotConfigToToml(testCfg), cfg)
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
		},
		Log: dot.LogConfig{
			CoreLvl:           log.LvlInfo,
			SyncLvl:           log.LvlInfo,
			NetworkLvl:        log.LvlInfo,
			RPCLvl:            log.LvlInfo,
			StateLvl:          log.LvlInfo,
			RuntimeLvl:        log.LvlInfo,
			BlockProducerLvl:  log.LvlInfo,
			FinalityGadgetLvl: log.LvlInfo,
		},
		Init: dot.InitConfig{
			Genesis: genFile.Name(),
		},
		Account: testCfg.Account,
		Core:    testCfg.Core,
		Network: dot.NetworkConfig{
			Port:        testCfg.Network.Port,
			Bootnodes:   []string{}, // TODO: improve cmd tests #687
			ProtocolID:  testCfg.Network.ProtocolID,
			NoBootstrap: testCfg.Network.NoBootstrap,
			NoMDNS:      testCfg.Network.NoMDNS,
		},
		RPC:    testCfg.RPC,
		System: testCfg.System,
	}

	cfg, err := createDotConfig(ctx)
	require.Nil(t, err)

	cfg.Init.Genesis = genFile.Name()
	expected.Core.BabeThresholdNumerator = 0
	expected.Core.BabeThresholdDenominator = 0

	db, err := chaindb.NewBadgerDB(&chaindb.Config{
		DataDir: cfg.Global.BasePath,
	})
	require.Nil(t, err)

	gen, err := genesis.NewGenesisFromJSONRaw(genFile.Name())
	require.Nil(t, err)

	err = state.StoreGenesisData(db, gen.GenesisData())
	require.Nil(t, err)

	err = db.Close()
	require.Nil(t, err)

	err = updateDotConfigFromGenesisData(ctx, cfg) // name should not be updated if provided as flag value
	require.Nil(t, err)

	require.Equal(t, expected, cfg)
}
