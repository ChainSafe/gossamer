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
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

// makeNode sets up node; opening badgerDB instance and returning the Dot container
func makeNode(ctx *cli.Context) (*dot.Dot, *cfg.Config, error) {
	fig, err := getConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	var srvcs []services.Service

	// DB: Create database dir and initialize stateDB and blockDB
	dataDir := getDataDir(ctx, fig)
	dbSrv, err := polkadb.NewDbService(dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create db service: %s", err)
	}

	err = dbSrv.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot start db service: %s", err)
	}

	// Trie: load most recent state from DB
	state := trie.NewEmptyTrie(&trie.Database{
		Db: dbSrv.StateDB.Db,
	})

	latestState, err := state.LoadHash()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load latest state root hash: %s", err)
	}

	err = state.LoadFromDB(latestState)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load latest state: %s", err)
	}

	// Runtime: load runtime code from trie and create runtime executor
	code, err := state.Get([]byte(":code"))
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving :code from trie: %s", err)
	}
	r, err := runtime.NewRuntime(code, state)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating runtime executor: %s", err)
	}
	srvcs = append(srvcs, dbSrv)

	// TODO: BABE

	// P2P
	fig.P2pCfg = setP2pConfig(ctx, fig.P2pCfg)
	p2pSrvc, msgChan := createP2PService(fig.P2pCfg)
	srvcs = append(srvcs, p2pSrvc)

	// core.Service
	coreSrvc := core.NewService(r, nil, msgChan)
	srvcs = append(srvcs, coreSrvc)

	// API
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	srvcs = append(srvcs, apiSrvc)

	// RPC
	setRpcConfig(ctx, fig.RpcCfg)
	rpcSrvr := startRpc(ctx, fig.RpcCfg, apiSrvc)

	return dot.NewDot(srvcs, rpcSrvr), fig, nil
}

// getConfig checks for config.toml if --config flag is specified
func getConfig(ctx *cli.Context) (*cfg.Config, error) {
	var fig *cfg.Config
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		config, err := loadConfig(file)
		if err != nil {
			log.Warn("err loading toml file", "err", err.Error())
			return fig, err
		}
		return config, nil
	} else {
		return cfg.DefaultConfig(), nil
	}
}

// loadConfig loads the contents from config.toml and inits Config object
func loadConfig(file string) (*cfg.Config, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		log.Warn("error finding working directory", "err", err)
	}
	filep := filepath.Join(filepath.Clean(fp))
	info, err := os.Lstat(filep)
	if err != nil {
		log.Crit("config file err ", "err", err)
		os.Exit(1)
	}
	if info.IsDir() {
		log.Crit("cannot pass in a directory, expecting file ")
		os.Exit(1)
	}
	/* #nosec */
	f, err := os.Open(filep)
	if err != nil {
		log.Crit("opening file err ", "err", err)
		os.Exit(1)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Warn("err closing conn", "err", err.Error())
		}
	}()
	var config *cfg.Config
	if err = tomlSettings.NewDecoder(f).Decode(&config); err != nil {
		log.Error("decoding toml error", "err", err.Error())
	}
	return config, err
}

// getDataDir initializes directory for gossamer data
func getDataDir(ctx *cli.Context, fig *cfg.Config) string {
	if file := ctx.GlobalString(utils.DataDirFlag.Name); file != "" {
		fig.DbCfg.DataDir = file
		return file
	} else if fig.DbCfg.DataDir != "" {
		return fig.DbCfg.DataDir
	} else {
		return cfg.DefaultDataDir()
	}
}

func setP2pConfig(ctx *cli.Context, fig p2p.Config) p2p.Config {
	// Bootnodes
	if bnodes := ctx.GlobalString(utils.BootnodesFlag.Name); bnodes != "" {
		fig.BootstrapNodes = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
	}

	if port := ctx.GlobalUint(utils.P2pPortFlag.Name); port != 0 {
		fig.Port = uint32(port)
	}

	// NoBootstrap
	if off := ctx.GlobalBool(utils.NoBootstrapFlag.Name); off {
		fig.NoBootstrap = true
	}

	// NoMdns
	if off := ctx.GlobalBool(utils.NoMdnsFlag.Name); off {
		fig.NoMdns = true
	}
	return fig
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

func setRpcConfig(ctx *cli.Context, fig rpc.Config) rpc.Config {
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
	return fig
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
	fig, err := getConfig(ctx)
	if err != nil {
		return err
	}

	comment := ""

	out, err := tomlSettings.Marshal(&fig)
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
