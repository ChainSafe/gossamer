package main

import (
	log "github.com/ChainSafe/log15"
)

func handleAccounts(ctx *cli.Context) error {
	err := startLogger(ctx)
	if err != nil {
		log.Error("account", "error", err)
		return err
	}

	log.Info("account")
	return nil
}
