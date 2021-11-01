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
	"github.com/ChainSafe/gossamer/lib/utils"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestExportCommand test "gossamer export --config"
func TestExportCommand(t *testing.T) {
	testDir := utils.NewTestDir(t)
	testCfg, testConfigFile := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	defer utils.RemoveTestDir(t)

	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard

	testName := "testnode"
	testBootnode := "bootnode"
	testProtocol := "/protocol/test/0"
	testConfig := testConfigFile.Name()

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    *dot.Config
	}{
		{
			"Test gossamer export --config --genesis --basepath --name --log --force",
			[]string{"config", "genesis", "basepath", "name", "log", "force"},
			[]interface{}{testConfig, genFile.Name(), testDir, testName, log.Info.String(), "true"},
			&dot.Config{
				Global: dot.GlobalConfig{
					Name:           testName,
					ID:             testCfg.Global.ID,
					BasePath:       testCfg.Global.BasePath,
					LogLvl:         log.Info,
					PublishMetrics: testCfg.Global.PublishMetrics,
					MetricsPort:    testCfg.Global.MetricsPort,
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
					Bootnodes:         testCfg.Network.Bootnodes,
					ProtocolID:        testCfg.Network.ProtocolID,
					NoBootstrap:       testCfg.Network.NoBootstrap,
					NoMDNS:            testCfg.Network.NoMDNS,
					DiscoveryInterval: testCfg.Network.DiscoveryInterval,
					MinPeers:          testCfg.Network.MinPeers,
				},
				RPC: testCfg.RPC,
			},
		},
		{
			"Test gossamer export --config --genesis --bootnodes --log --force",
			[]string{"config", "genesis", "bootnodes", "name", "force", "pruning", "retain-blocks"},
			[]interface{}{testConfig, genFile.Name(), testBootnode, "Gossamer", "true", gssmr.DefaultPruningMode, gssmr.DefaultRetainBlocks},
			&dot.Config{
				Global: testCfg.Global,
				Init: dot.InitConfig{
					Genesis: genFile.Name(),
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
				Account: testCfg.Account,
				Core:    testCfg.Core,
				Network: dot.NetworkConfig{
					Port:              testCfg.Network.Port,
					Bootnodes:         []string{testBootnode},
					ProtocolID:        testCfg.Network.ProtocolID,
					NoBootstrap:       testCfg.Network.NoBootstrap,
					NoMDNS:            testCfg.Network.NoMDNS,
					DiscoveryInterval: testCfg.Network.DiscoveryInterval,
					MinPeers:          testCfg.Network.MinPeers,
				},
				RPC: testCfg.RPC,
			},
		},
		{
			"Test gossamer export --config --genesis --protocol --log --force",
			[]string{"config", "genesis", "protocol", "force", "name", "pruning", "retain-blocks"},
			[]interface{}{testConfig, genFile.Name(), testProtocol, "true", "Gossamer", gssmr.DefaultPruningMode, gssmr.DefaultRetainBlocks},
			&dot.Config{
				Global: testCfg.Global,
				Init: dot.InitConfig{
					Genesis: genFile.Name(),
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
				Account: testCfg.Account,
				Core:    testCfg.Core,
				Network: dot.NetworkConfig{
					Port:              testCfg.Network.Port,
					Bootnodes:         []string{testBootnode},
					ProtocolID:        testProtocol,
					NoBootstrap:       testCfg.Network.NoBootstrap,
					NoMDNS:            testCfg.Network.NoMDNS,
					DiscoveryInterval: testCfg.Network.DiscoveryInterval,
					MinPeers:          testCfg.Network.MinPeers,
				},
				RPC: testCfg.RPC,
			},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)

			err = exportAction(ctx)
			require.Nil(t, err)

			config := ctx.GlobalString(ConfigFlag.Name)

			cfg := new(ctoml.Config)
			err = loadConfig(cfg, config)
			require.Nil(t, err)

			require.Equal(t, dotConfigToToml(c.expected), cfg)
		})
	}
}
