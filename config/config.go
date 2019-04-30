package cfg

import (
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
)

type Config struct {
	ServiceConfig *p2p.ServiceConfig
	BadgerDB polkadb.BadgerDB
}
