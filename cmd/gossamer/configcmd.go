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
	"github.com/ChainSafe/gossamer/cmd/utils"
	"github.com/ChainSafe/gossamer/common"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot"
	api "github.com/ChainSafe/gossamer/internal"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
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
func makeNode(ctx *cli.Context) (*dot.Dot, error) {
	fig, err := getConfig(ctx)
	if err != nil {
		log.Crit("unable to extract required config", "err", err)
		return nil, err
	}

	var services []common.Service

	// P2P
	p2pSrvc := createP2PService(ctx, fig.P2PConfig)
	services = append(services, p2pSrvc)

	// DB
	dataDir := getDatabaseDir(ctx, fig)
	dbSrvc, err := polkadb.NewBadgerService(dataDir)
	if err != nil {
		return nil, err
	}
	services = append(services, dbSrvc)

	// API
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	services = append(services, apiSrvc)

	// RPC
	rpcSrvr := rpc.NewHttpServer(apiSrvc.Api, &json2.Codec{}, fig.RPCConfig)

	return dot.NewDot(services, rpcSrvr), nil
}

// setConfig checks for config.toml if --config flag is specified
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

// getDatabaseDir initializes directory for BadgerService logs
func getDatabaseDir(ctx *cli.Context, fig *cfg.Config) string {
	if fig.DbConfig.DataDir != "" {
		return fig.DbConfig.DataDir
	} else if file := ctx.GlobalString(utils.DataDirFlag.Name); file != "" {
		return file
	} else {
		return cfg.DefaultDataDir()
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
		log.Crit("config file err ","err", err)
		os.Exit(1)
	}
	if info.IsDir() {
		log.Crit("cannot pass in a directory, expecting file ")
		os.Exit(1)
	}
	/* #nosec */
	f, err := os.Open(filep)
	if err != nil {
		log.Crit("opening file err ", "err",err)
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

// createP2PService starts a p2p network layer from provided config
func createP2PService(ctx *cli.Context,cfg *p2p.Config) *p2p.Service {
	setBootstrapNodes(ctx, cfg)
	srvc, err := p2p.NewService(cfg)
	if err != nil {
		log.Error("error starting p2p", "err", err.Error())
	}
	return srvc
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.Config) {
	var urls []string
	switch {
	case ctx.GlobalIsSet(utils.BootnodesFlag.Name):
		urls = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
	case cfg.BootstrapNodes != nil:
		return // already set, don't apply defaults.
	}
	cfg.BootstrapNodes = append(cfg.BootstrapNodes, urls...)
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