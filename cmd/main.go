package main

import (
	"fmt"
	api "github.com/ChainSafe/gossamer/internal"
	"os"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	log "github.com/inconshreveable/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

var app = cli.NewApp()

func init() {
	app.Action = gossamer
	app.Copyright = "Copyright 2019 ChainSafe Systems Authors"
	app.Name = "gossamer"
	app.Usage = "Official gossamer command-line interface"
	app.Author = "Chainsafe Systems 2019"
	app.Version = "0.0.1"
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func gossamer(ctx *cli.Context) error {
	srvlog := log.New(log.Ctx{"blockchain": "gossamer"})
	f, err := os.Open("../config.toml")
	if err != nil {
		panic(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Error("err closing conn", "err", err.Error())
		}
	}()


	var config cfg.Config
	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		srvlog.Warn("toml error::: %s", err.Error())
	}

	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	srv, err := p2p.NewService(config.ServiceConfig)
	if err != nil {
		srvlog.Warn("error starting p2p %s", err.Error())
	}
	srvlog.Info("🕸️ starting gossamer blockchain...", log.Ctx{"datadir": config.BadgerDB.Datadir})
	srv.Start()

	_, err = polkadb.NewBadgerDB(config.BadgerDB.Datadir)
	rpcSvr := rpc.NewServer()
	rpcSvr.RegisterCodec(json2.NewCodec())
	err = rpcSvr.RegisterService(new(api.PublicRPC), "PublicRPC")
	if err != nil {
		srvlog.Warn("could not register service: %s", err)
	}
	srvlog.Info("gossamer blockchain started...")

	return nil
}
