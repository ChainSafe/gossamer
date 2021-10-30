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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
	terminal "golang.org/x/term"
)

const confirmCharacter = "Y"

// setupLogger sets up the global Gossamer logger.
func setupLogger(ctx *cli.Context) (level log.Level, err error) {
	if lvlToInt, err := strconv.Atoi(ctx.String(LogFlag.Name)); err == nil {
		level = log.Level(lvlToInt)
	} else if level, err = log.ParseLevel(ctx.String(LogFlag.Name)); err != nil {
		return 0, err
	}

	log.Patch(
		log.SetWriter(os.Stdout),
		log.SetFormat(log.FormatConsole),
		log.SetCallerFile(true),
		log.SetCallerLine(true),
		log.SetLevel(level),
	)

	return level, nil
}

// getPassword prompts user to enter password
func getPassword(msg string) []byte {
	for {
		fmt.Println(msg)
		fmt.Print("> ")
		password, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("invalid input: %s\n", err)
		} else {
			fmt.Printf("\n")
			return password
		}
	}
}

// confirmMessage prompts user to confirm message and returns true if "Y"
func confirmMessage(msg string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(msg)
	fmt.Print("> ")
	for {
		text, _ := reader.ReadString('\n')
		text = strings.ReplaceAll(text, "\n", "")
		return strings.Compare(confirmCharacter, strings.ToUpper(text)) == 0
	}
}

// newTestConfig returns a new test configuration using the provided basepath
func newTestConfig(t *testing.T) *dot.Config {
	dir := utils.NewTestDir(t)

	cfg := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           dot.GssmrConfig().Global.Name,
			ID:             dot.GssmrConfig().Global.ID,
			BasePath:       dir,
			LogLvl:         log.Info,
			PublishMetrics: dot.GssmrConfig().Global.PublishMetrics,
			MetricsPort:    dot.GssmrConfig().Global.MetricsPort,
			RetainBlocks:   dot.GssmrConfig().Global.RetainBlocks,
			Pruning:        dot.GssmrConfig().Global.Pruning,
			TelemetryURLs:  dot.GssmrConfig().Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init:    dot.GssmrConfig().Init,
		Account: dot.GssmrConfig().Account,
		Core:    dot.GssmrConfig().Core,
		Network: dot.GssmrConfig().Network,
		RPC:     dot.GssmrConfig().RPC,
		System:  dot.GssmrConfig().System,
	}

	return cfg
}

// newTestConfigWithFile returns a new test configuration and a temporary configuration file
func newTestConfigWithFile(t *testing.T) (*dot.Config, *os.File) {
	cfg := newTestConfig(t)

	file, err := ioutil.TempFile(cfg.Global.BasePath, "config-")
	require.NoError(t, err)

	tomlCfg := dotConfigToToml(cfg)
	cfgFile := exportConfig(tomlCfg, file.Name())
	return cfg, cfgFile
}
