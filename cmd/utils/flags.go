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

package utils

import (
	"fmt"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/urfave/cli"
	"strings"
)

var (
	// BadgerDB directory
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the database",
		Value: cfg.DefaultDataDir(),
	}
	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpscaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: cfg.DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: cfg.DefaultHTTPPort,
	}
	// P2P service settings
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value: "",
	}
)

// SetP2PConfig sets up the configurations required for P2P service
func SetP2PConfig(ctx *cli.Context, cfg *p2p.ServiceConfig) *p2p.Service {
	setBootstrapNodes(ctx, cfg)
	srv := startP2PService(cfg)
	return srv
}

// startP2PService starts a p2p network layer from provided config
func startP2PService(cfg *p2p.ServiceConfig) *p2p.Service {
	srv, err := p2p.NewService(cfg)
	if err != nil {
		fmt.Printf("error starting p2p %s", err.Error())
	}
	return srv
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.ServiceConfig) {
	var urls []string
	switch {
	case ctx.GlobalIsSet(BootnodesFlag.Name):
		urls = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
	case cfg.BootstrapNodes != nil:
		return // already set, don't apply defaults.
	}

	for _, url := range urls {
		cfg.BootstrapNodes = append(cfg.BootstrapNodes, url)
	}
}
