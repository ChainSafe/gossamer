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
	"errors"
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
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	"github.com/ChainSafe/gossamer/runtime"
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
		return nil, nil, err
	}

	var srvcs []services.Service

	// set up message channel for p2p -> core.Service
	msgChan := make(chan []byte)

	if gen == nil {
		return nil, nil, errors.New("genesis is nil")
	}

	if gen.GenesisTrie == nil {
		return nil, nil, errors.New("no genesis trie exists")
	}

	// load runtime code from trie and create runtime executor
	code, err := gen.GenesisTrie.Get([]byte(":code"))
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("error retrieving :code from trie: %s", err))
	}
	r, err := runtime.NewRuntime(code, gen.GenesisTrie)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("error creating runtime executor: %s", err))
	}

	// TODO: BABE

	// core.Service
	coreSrvc := core.NewService(r, nil, msgChan)
	srvcs = append(srvcs, coreSrvc)

	// P2P
	setBootstrapNodes(ctx, fig.P2pCfg)
	setNoBootstrap(ctx, fig.P2pCfg)
	p2pSrvc := createP2PService(fig.P2pCfg, msgChan)
	srvcs = append(srvcs, p2pSrvc)

	// API
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	srvcs = append(srvcs, apiSrvc)

	// RPC
	setRpcModules(ctx, fig.RpcCfg)
	setRpcHost(ctx, fig.RpcCfg)
	setRpcPort(ctx, fig.RpcCfg)
	rpcSrvr := rpc.NewHttpServer(apiSrvc.Api, &json2.Codec{}, fig.RpcCfg)

	return dot.NewDot(srvcs, rpcSrvr, coreSrvc), fig, nil
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
		return cfg.DefaultConfig, nil
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

// getDatabaseDir initializes directory for BadgerService logs
func getDatabaseDir(ctx *cli.Context, fig *cfg.Config) string {
	if file := ctx.GlobalString(utils.DataDirFlag.Name); file != "" {
		fig.DbCfg.DataDir = file
		return file
	} else if fig.DbCfg.DataDir != "" {
		return fig.DbCfg.DataDir
	} else {
		return cfg.DefaultDataDir()
	}
}

// createP2PService starts a p2p network layer from provided config
func createP2PService(fig *p2p.Config, msgChan chan<- []byte) *p2p.Service {
	srvc, err := p2p.NewService(fig, msgChan)
	if err != nil {
		log.Error("error starting p2p", "err", err.Error())
	}
	return srvc
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, fig *p2p.Config) {
	var urls []string

	if bnodes := ctx.GlobalString(utils.BootnodesFlag.Name); bnodes != "" {
		urls = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
		fig.BootstrapNodes = append(fig.BootstrapNodes, urls...)
		return
	} else if fig.BootstrapNodes != nil {
		return // set in config, dont use defaults
	} else {
		fig.BootstrapNodes = cfg.DefaultP2PBootstrap
	}
}

// setNoBootsrap sets config to flag value if true, or default value if not set in config
func setNoBootstrap(ctx *cli.Context, fig *p2p.Config) {
	if off := ctx.GlobalBool(utils.NoBootstrapFlag.Name); off {
		fig.NoBootstrap = true
		return
	} else if fig.NoBootstrap {
		return // set in config, dont use defaults
	} else {
		fig.NoBootstrap = cfg.DefaultNoBootstrap
	}
}

// setRpcModules checks the context for rpc modes and applies them to `cfg`, unless some are already set
func setRpcModules(ctx *cli.Context, fig *rpc.Config) {
	var strs []string

	if mods := ctx.GlobalString(utils.RpcModuleFlag.Name); mods != "" {
		strs = strings.Split(ctx.GlobalString(utils.RpcModuleFlag.Name), ",")
		fig.Modules = append(fig.Modules, strToMods(strs)...)
		return
	} else if fig.Modules != nil {
		return // set in config, dont use defaults
	} else {
		fig.Modules = cfg.DefaultRpcModules
	}
}

// setRpcHost checks the context for a hostname and applies it to `cfg`, unless one is already set
func setRpcHost(ctx *cli.Context, fig *rpc.Config) {
	if host := ctx.GlobalString(utils.RpcHostFlag.Name); host != "" {
		fig.Host = host
		return
	} else if fig.Host != "" {
		return
	} else {
		fig.Host = cfg.DefaultRpcHttpHost
	}
}

// setRpcPort checks the context for a port and applies it to `cfg`, unless one is already set
func setRpcPort(ctx *cli.Context, fig *rpc.Config) {
	if port := ctx.GlobalUint(utils.RpcPortFlag.Name); port != 0 {
		fig.Port = uint32(port)
		return
	} else if fig.Port != 0 {
		return
	} else {
		fig.Port = cfg.DefaultRpcHttpPort
	}
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
	gen, err := loadGenesis(ctx)
	if err != nil {
		return err
	}

	_, fig, err := makeNode(ctx, gen)
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
