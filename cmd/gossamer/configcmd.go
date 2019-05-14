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

	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/goss"
	"github.com/ChainSafe/gossamer/polkadb"
	log "github.com/inconshreveable/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

var (
	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// makeNode sets up node; opening badgerDB instance and returning the Goss container
func makeNode(ctx *cli.Context) (*goss.Goss, error) {
	fig, err := setConfig(ctx)
	if err != nil {
		log.Error("unable to extract required config", "err", err)
	}
	srv := utils.SetP2PConfig(ctx, fig.ServiceConfig)
	datadir := setDatabaseDir(ctx, fig)
	db, err := polkadb.NewBadgerDB(datadir)
	if err != nil {
		fmt.Println(err)
	}
	return &goss.Goss{
		ServerConfig: fig.ServiceConfig,
		Server:       srv,
		Polkadb:      db,
	}, nil
}

// setConfig checks for config.toml if --config flag is specified
func setConfig(ctx *cli.Context) (*cfg.Config, error) {
	var fig *cfg.Config
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		config, err := loadConfig(file)
		if err != nil {
			fmt.Println("err", err.Error())
			return fig, err
		}
		return config, nil
	}
	return fig, nil
}

// setDatabaseDir initializes directory for BadgerDB logs
func setDatabaseDir(ctx *cli.Context, cfg *cfg.Config) string {
	if cfg.BadgerDB.Datadir != "" {
		return cfg.BadgerDB.Datadir
	} else if file := ctx.GlobalString(utils.DataDirFlag.Name); file != "" {
		return file
	} else {
		log.Error("must specify data directory")
		return ""
	}
}

// loadConfig loads the contents from config.toml and inits Config object
func loadConfig(file string) (*cfg.Config, error) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			fmt.Println("err closing conn", "err", err.Error())
		}
	}()

	var config *cfg.Config
	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		log.Error("decoding toml error", "err", err.Error())
	}
	return config, err
}
