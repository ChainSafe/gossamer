package main

import (
	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/trie"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

func loadGenesis(ctx *cli.Context) (*genesis.GenesisState, error) {
	fig, err := getConfig(ctx)
	if err != nil {
		log.Crit("unable to extract required config", "err", err)
		return nil, err
	}

	// read genesis file
	fp := getGenesisPath(ctx)
	gen, err := genesis.LoadGenesisJsonFile(fp)
	if err != nil {
		log.Crit("cannot read genesis file", "err", err)
		return nil, err
	}

	// DB: Create database dir and initialize stateDB and blockDB
	dataDir := getDatabaseDir(ctx, fig)
	dbSrv, err := polkadb.NewDatabaseService(dataDir)
	if err != nil {
		log.Crit("error creating DB service", "error", err)
	}

	trieStateDB, err := trie.NewStateDB(dbSrv.StateDB)
	if err != nil {
		log.Crit("error creating trie state DB", "error", err)
	}
	t := trie.NewEmptyTrie(trieStateDB)
	err = loadTrie(t, gen.Genesis.Raw)
	if err != nil {
		log.Crit("error loading genesis state", "error", err)
	}

	// write state to DB
	err = commitToDb(t)
	if err != nil {
		log.Crit("error writing genesis state to DB", "error", err)
	}

	// TODO: load genesis trie and create initial p2p config
	return &genesis.GenesisState{
		Name:        gen.Name,
		Id:          gen.Id,
		GenesisTrie: t,
		Db:          dbSrv,
	}, nil
}

// getGenesisPath gets the path to the genesis file
func getGenesisPath(ctx *cli.Context) string {
	if file := ctx.GlobalString(utils.GenesisFlag.Name); file != "" {
		return file
	} else {
		return cfg.DefaultGenesisPath
	}
}
