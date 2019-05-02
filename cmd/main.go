package main

import (
	"os"
	"time"

	api "github.com/ChainSafe/gossamer/internal"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	log "github.com/inconshreveable/log15"
	"github.com/naoina/toml"
)

func main() {
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

	srvlog := log.New(log.Ctx{"blockchain": "gossamer"})
	var config cfg.Config
	if err = toml.NewDecoder(f).Decode(&config); err != nil {
		srvlog.Warn("toml error::: %s", err.Error())
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
	time.Sleep(1 * time.Minute)
	count := srv.PeerCount()
	log.Info("Peercount....", "count", count)
	time.Sleep(30 * time.Second)
	log.Warn("BYE👋👋👋")
}
