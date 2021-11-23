// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/urfave/cli"
)

// exportAction is the action for the "export" subcommand
func exportAction(ctx *cli.Context) error {
	// use --config value as export destination
	config := ctx.GlobalString(ConfigFlag.Name)

	// check if --config value is set
	if config == "" {
		return fmt.Errorf("export destination undefined: --config value required")
	}

	// check if configuration file already exists at export destination
	if utils.PathExists(config) {
		logger.Warn("toml configuration file " + config + "already exists")

		// use --force value to force overwrite the toml configuration file
		force := ctx.Bool(ForceFlag.Name)

		// prompt user to confirm overwriting existing toml configuration file
		if force || confirmMessage("Are you sure you want to overwrite the file? [Y/n]") {
			logger.Warn("overwriting toml configuration file " + config)
		} else {
			logger.Warn(
				"exiting without exporting toml configuration file " + config,
			)
			return nil // exit if reinitialization is not confirmed
		}
	}

	cfg, err := createExportConfig(ctx)
	if err != nil {
		return err
	}

	tomlCfg := dotConfigToToml(cfg)
	file := exportConfig(tomlCfg, config)
	// export config will exit and log error on error

	logger.Info("exported toml configuration to " + file.Name())

	return nil
}

func dotConfigToToml(dcfg *dot.Config) *ctoml.Config {
	cfg := &ctoml.Config{
		Pprof: ctoml.PprofConfig{
			Enabled:          dcfg.Pprof.Enabled,
			ListeningAddress: dcfg.Pprof.Settings.ListeningAddress,
			BlockRate:        dcfg.Pprof.Settings.BlockProfileRate,
			MutexRate:        dcfg.Pprof.Settings.MutexProfileRate,
		},
	}

	cfg.Global = ctoml.GlobalConfig{
		Name:         dcfg.Global.Name,
		ID:           dcfg.Global.ID,
		BasePath:     dcfg.Global.BasePath,
		LogLvl:       dcfg.Global.LogLvl.String(),
		MetricsPort:  dcfg.Global.MetricsPort,
		RetainBlocks: dcfg.Global.RetainBlocks,
		Pruning:      string(dcfg.Global.Pruning),
	}

	cfg.Log = ctoml.LogConfig{
		CoreLvl:           dcfg.Log.CoreLvl.String(),
		SyncLvl:           dcfg.Log.SyncLvl.String(),
		NetworkLvl:        dcfg.Log.NetworkLvl.String(),
		RPCLvl:            dcfg.Log.RPCLvl.String(),
		StateLvl:          dcfg.Log.StateLvl.String(),
		RuntimeLvl:        dcfg.Log.RuntimeLvl.String(),
		BlockProducerLvl:  dcfg.Log.BlockProducerLvl.String(),
		FinalityGadgetLvl: dcfg.Log.FinalityGadgetLvl.String(),
	}

	cfg.Init = ctoml.InitConfig{
		Genesis: dcfg.Init.Genesis,
	}

	cfg.Account = ctoml.AccountConfig{
		Key:    dcfg.Account.Key,
		Unlock: dcfg.Account.Unlock,
	}

	cfg.Core = ctoml.CoreConfig{
		Roles:            dcfg.Core.Roles,
		BabeAuthority:    dcfg.Core.BabeAuthority,
		GrandpaAuthority: dcfg.Core.GrandpaAuthority,
		GrandpaInterval:  uint32(dcfg.Core.GrandpaInterval / time.Second),
	}

	cfg.Network = ctoml.NetworkConfig{
		Port:              dcfg.Network.Port,
		Bootnodes:         dcfg.Network.Bootnodes,
		ProtocolID:        dcfg.Network.ProtocolID,
		NoBootstrap:       dcfg.Network.NoBootstrap,
		NoMDNS:            dcfg.Network.NoMDNS,
		DiscoveryInterval: int(dcfg.Network.DiscoveryInterval / time.Second),
		MinPeers:          dcfg.Network.MinPeers,
		MaxPeers:          dcfg.Network.MaxPeers,
	}

	cfg.RPC = ctoml.RPCConfig{
		Enabled:    dcfg.RPC.Enabled,
		External:   dcfg.RPC.External,
		Port:       dcfg.RPC.Port,
		Host:       dcfg.RPC.Host,
		Modules:    dcfg.RPC.Modules,
		WSPort:     dcfg.RPC.WSPort,
		WS:         dcfg.RPC.WS,
		WSExternal: dcfg.RPC.WSExternal,
	}

	return cfg
}
