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
	"strconv"

	"github.com/ChainSafe/gossamer/cmd/utils"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

var (
	app       = cli.NewApp()
	nodeFlags = []cli.Flag{
		utils.DataDirFlag,
		configFileFlag,
	}
	rpcFlags = []cli.Flag{
		utils.RpcEnabledFlag,
		utils.RpcListenAddrFlag,
		utils.RpcPortFlag,
		utils.RpcHostFlag,
		utils.RpcModuleFlag,
	}
	cliFlags = []cli.Flag{
		utils.VerbosityFlag,
	}
)

// init initializes CLI
func init() {
	app.Action = gossamer
	app.Copyright = "Copyright 2019 ChainSafe Systems Authors"
	app.Name = "gossamer"
	app.Usage = "Official gossamer command-line interface"
	app.Author = "ChainSafe Systems 2019"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		dumpConfigCommand,
	}
	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, cliFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Error("error starting app", "output", os.Stderr, "err", err)
		os.Exit(1)
	}
}

func LvlFromInt(lvlUint int) (log.Lvl, error) {
	switch lvlUint {
	case 5:
		return log.LvlTrace, nil
	case 4:
		return log.LvlDebug, nil
	case 3:
		return log.LvlInfo, nil
	case 2:
		return log.LvlWarn, nil
	case 1:
		return log.LvlError, nil
	case 0:
		return log.LvlCrit, nil
	default:
		return log.LvlDebug, fmt.Errorf("Unknown level: %v", lvlUint)
	}
}

func StartLogger(ctx *cli.Context) error {
	logger := log.Root()

	level, err := strconv.Atoi(ctx.String("verbosity"))
	if err != nil {

		lvl, err := log.LvlFromString(ctx.String("verbosity"))
		if err != nil {
			return err
		}

		handler := logger.GetHandler()
		log.Root().SetHandler(log.LvlFilterHandler(lvl, handler))

	} else {

		lvl, err := LvlFromInt(level)
		if err != nil {
			return err
		}

		handler := logger.GetHandler()
		log.Root().SetHandler(log.LvlFilterHandler(lvl, handler))
	}

	return nil
}

// gossamer is the main entrypoint into the gossamer system
func gossamer(ctx *cli.Context) error {

	err := StartLogger(ctx)
	if err != nil {
		log.Error("verbosity level error", "err", err)
	}

	node, _, err := makeNode(ctx)
	if err != nil {
		// TODO: Need to manage error propagation and exit smoothly
		log.Error("error making node", "err", err)
	}
	// srvlog.Info("🕸️Starting node...")
	log.Info("🕸️Starting node...")
	node.Start()

	return nil
}
