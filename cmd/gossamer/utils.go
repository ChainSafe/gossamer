// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	terminal "golang.org/x/term"
)

const confirmCharacter = "Y"

// setupLogger sets up the global Gossamer logger.
func setupLogger(ctx *cli.Context) (level log.Level, err error) {
	level, err = getLogLevel(ctx, LogFlag.Name, "", log.Info)
	if err != nil {
		return level, err
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
func newTestConfig(t *testing.T, chainConfig *dot.Config) *dot.Config {
	dir := t.TempDir()

	//westendDevConfig := dot.WestendDevConfig()
	cfg := &dot.Config{
		Global: dot.GlobalConfig{
			Name:           chainConfig.Global.Name,
			ID:             chainConfig.Global.ID,
			BasePath:       dir,
			LogLvl:         log.Info,
			PublishMetrics: chainConfig.Global.PublishMetrics,
			MetricsAddress: chainConfig.Global.MetricsAddress,
			RetainBlocks:   chainConfig.Global.RetainBlocks,
			Pruning:        chainConfig.Global.Pruning,
			TelemetryURLs:  chainConfig.Global.TelemetryURLs,
		},
		Log: dot.LogConfig{
			CoreLvl:           log.Info,
			DigestLvl:         log.Info,
			SyncLvl:           log.Info,
			NetworkLvl:        log.Info,
			RPCLvl:            log.Info,
			StateLvl:          log.Info,
			RuntimeLvl:        log.Info,
			BlockProducerLvl:  log.Info,
			FinalityGadgetLvl: log.Info,
		},
		Init:    chainConfig.Init,
		Account: chainConfig.Account,
		Core:    chainConfig.Core,
		Network: chainConfig.Network,
		RPC:     chainConfig.RPC,
		System:  chainConfig.System,
		Pprof:   chainConfig.Pprof,
	}

	return cfg
}

// newTestConfigWithFile returns a new test configuration and a temporary configuration file
func newTestConfigWithFile(t *testing.T, chainConfig *dot.Config) (cfg *dot.Config, configPath string) {
	t.Helper()

	cfg = newTestConfig(t, chainConfig)

	tomlCfg := dotConfigToToml(cfg)

	configPath = filepath.Join(cfg.Global.BasePath, "config.toml")
	err := exportConfig(tomlCfg, configPath)
	require.NoError(t, err)

	return cfg, configPath
}
