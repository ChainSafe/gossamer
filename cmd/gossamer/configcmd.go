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
	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot"
	api "github.com/ChainSafe/gossamer/internal"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	log "github.com/inconshreveable/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strings"
)

var (
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

	// P2P
	p2pSrvc := setupP2PService(ctx, fig.P2PConfig)

	// DB
	datadir := getDatabaseDir(ctx, fig)
	dbSrvc, err := polkadb.NewBadgerDB(datadir)
	if err != nil {

		return nil, err
	}

	// API
	apiSrvc := api.NewApiService(p2pSrvc)

	// RPC
	rpc := rpc.NewHttpServer(apiSrvc, module, cfg)
	return dot.NewDot(p2pSrvc, dbSrvc, apiSrvc, rpc), nil
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

// setDatabaseDir initializes directory for BadgerDB logs
func getDatabaseDir(ctx *cli.Context, fig *cfg.Config) string {
	if fig.DbConfig.Datadir != "" {
		return fig.DbConfig.Datadir
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
	/* #nosec */
	f, err := os.Open(filep)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Warn("err closing conn", "err", err.Error())
		}
	}()
	var config *cfg.Config
	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		log.Error("decoding toml error", "err", err.Error())
	}
	return config, err
}

// setupP2PService starts a p2p network layer from provided config
func setupP2PService(ctx *cli.Context,cfg *p2p.Config) *p2p.Service {
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

func setupRPC(cfg)
