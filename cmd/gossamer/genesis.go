package main

import (
	"fmt"

	"github.com/ChainSafe/gossamer/cmd/utils"
	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/trie"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

func loadGenesis(ctx *cli.Context) error {
	fig, err := getConfig(ctx)
	if err != nil {
		return err
	}

	// read genesis file
	fp := getGenesisPath(ctx)
	gen, err := genesis.LoadGenesisJsonFile(fp)
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "genesisfile", fp)

	// DB: Create database dir and initialize stateDB and blockDB
	dataDir := getDataDir(ctx, fig)
	dbSrv, err := polkadb.NewDbService(dataDir)
	if err != nil {
		return err
	}

	err = dbSrv.Start()
	if err != nil {
		return err
	}

	defer func() {
		err := dbSrv.Stop()
		if err != nil {
			log.Error("error stopping database service")
		}
	}()

	tdb := &trie.Database{
		Db: dbSrv.StateDB.Db,
	}

	// create and load storage trie with initial genesis state
	t := trie.NewEmptyTrie(tdb)

	err = t.Load(gen.Genesis.Raw)
	if err != nil {
		return fmt.Errorf("cannot load trie with initial state: %s", err)
	}

	// write initial genesis data to DB
	err = t.StoreInDB()
	if err != nil {
		return err
	}

	err = t.StoreHash()
	if err != nil {
		return err
	}

	// store node name, ID, p2p protocol, bootnodes in DB
	return storeGenesisInfo(tdb, gen)
}

// getGenesisPath gets the path to the genesis file
func getGenesisPath(ctx *cli.Context) string {
	if file := ctx.GlobalString(utils.GenesisFlag.Name); file != "" {
		return file
	} else {
		return cfg.DefaultGenesisPath
	}
}

func storeGenesisInfo(db *trie.Database, gen *genesis.Genesis) error {
	err := db.Store(common.NodeName, []byte(gen.Name))
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "name", gen.Name)

	err = db.Store(common.NodeId, []byte(gen.Id))
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "id", gen.Id)

	err = db.Store(common.NodeProtocolId, []byte(gen.ProtocolId))
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "protocolID", gen.ProtocolId)

	encBootnodes, err := scale.Encode(gen.Bootnodes)
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "bootnodes", gen.Bootnodes)

	return db.Store(common.NodeBootnodes, encBootnodes)
}
