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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...)),
		Category:    "CONFIGURATION DEBUGGING",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// makeNode sets up node; opening badgerDB instance and returning the Dot container
func makeNode(ctx *cli.Context, gen *genesis.GenesisState) (*dot.Dot, *cfg.Config, error) {
	fig, err := getConfig(ctx)
	if err != nil {
		log.Crit("unable to extract required config", "err", err)
		return nil, nil, err
	}

	var srvcs []services.Service

	setGlobalConfig(ctx, &fig.GlobalCfg)

	// DB
	dbSrv, err := polkadb.NewDbService(fig.GlobalCfg.DataDir)
	if err != nil {
		return nil, nil, err
	}
	srvcs = append(srvcs, dbSrv)

	// TODO: trie and runtime

	// TODO: BABE

	// P2P
	setP2pConfig(ctx, fig)
	p2pSrvc, msgChan := createP2PService(fig.P2pCfg)
	srvcs = append(srvcs, p2pSrvc)

	// core.Service
	coreSrvc := core.NewService(nil, nil, msgChan)
	srvcs = append(srvcs, coreSrvc)

	// API
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	srvcs = append(srvcs, apiSrvc)

	// RPC
	setRpcConfig(ctx, &fig.RpcCfg)
	rpcSrvr := startRpc(ctx, fig.RpcCfg, apiSrvc)

	return dot.NewDot(srvcs, rpcSrvr), fig, nil
}

// getConfig checks for config.toml if --config flag is specified
func getConfig(ctx *cli.Context) (*cfg.Config, error) {
	fig := cfg.DefaultConfig()
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		err := loadConfig(file, fig)
		if err != nil {
			log.Warn("err loading toml file", "err", err.Error())
			return fig, err
		}
		return fig, nil
	} else {
		return cfg.DefaultConfig(), nil
	}
}

// loadConfig loads the contents from config toml and inits Config object
func loadConfig(file string, config *cfg.Config) error {
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	log.Debug("Loading configuration", "path", filepath.Clean(fp))
	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}
	if err = tomlSettings.NewDecoder(f).Decode(&config); err != nil {
		return err
	}
	return nil
}

func setGlobalConfig(ctx *cli.Context, fig *cfg.GlobalConfig) {
	if dir := ctx.GlobalString(utils.DataDirFlag.Name); dir != "" {
		fig.DataDir = dir
	}
	fig.DataDir, _ = filepath.Abs(fig.DataDir)
}

func setP2pConfig(ctx *cli.Context, fig *cfg.Config) {
	// Bootnodes
	if bnodes := ctx.GlobalString(utils.BootnodesFlag.Name); bnodes != "" {
		fig.P2pCfg.BootstrapNodes = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
	}

	if port := ctx.GlobalUint(utils.P2pPortFlag.Name); port != 0 {
		fig.P2pCfg.Port = uint32(port)
	}

	// NoBootstrap
	if off := ctx.GlobalBool(utils.NoBootstrapFlag.Name); off {
		fig.P2pCfg.NoBootstrap = true
	}

	// NoMdns
	if off := ctx.GlobalBool(utils.NoMdnsFlag.Name); off {
		fig.P2pCfg.NoMdns = true
	}

	fig.P2pCfg.DataDir = fig.GlobalCfg.DataDir
}

// createP2PService starts a p2p network layer from provided config
func createP2PService(fig p2p.Config) (*p2p.Service, chan []byte) {
	msgChan := make(chan []byte)

	srvc, err := p2p.NewService(&fig, msgChan)
	if err != nil {
		log.Error("error starting p2p", "err", err.Error())
	}
	return srvc, msgChan
}

func setRpcConfig(ctx *cli.Context, fig *rpc.Config) {
	// Modules
	if mods := ctx.GlobalString(utils.RpcModuleFlag.Name); mods != "" {
		fig.Modules = strToMods(strings.Split(ctx.GlobalString(utils.RpcModuleFlag.Name), ","))
	}

	// Host
	if host := ctx.GlobalString(utils.RpcHostFlag.Name); host != "" {
		fig.Host = host
	}

	// Port
	if port := ctx.GlobalUint(utils.RpcPortFlag.Name); port != 0 {
		fig.Port = uint32(port)
	}

}

func startRpc(ctx *cli.Context, fig rpc.Config, apiSrvc *api.Service) *rpc.HttpServer {
	if ctx.GlobalBool(utils.RpcEnabledFlag.Name) {
		return rpc.NewHttpServer(apiSrvc.Api, &json2.Codec{}, fig)
	}
	return nil
}

// strToMods casts a []strings to []api.Module
func strToMods(strs []string) []api.Module {
	var res []api.Module
	for _, str := range strs {
		res = append(res, api.Module(str))
	}
	return res
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, fig, err := makeNode(ctx, nil)
	if err != nil {
		return err
	}
	comment := ""

	out, err := toml.Marshal(fig)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		/* #nosec */
		dump, err = os.OpenFile(filepath.Clean(ctx.Args().Get(0)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		defer func() {
			err = dump.Close()
			if err != nil {
				log.Warn("err closing conn", "err", err.Error())
			}
		}()
	}
	_, err = dump.WriteString(comment)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	_, err = dump.Write(out)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	return nil
}

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}
