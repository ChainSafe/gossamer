// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/spf13/cobra"
)

const confirmCharacter = "Y"

func init() {
	InitCmd.Flags().Bool("force",
		false,
		"force reinitialization of node")
}

// InitCmd is the command to initialise the node
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise node databases and load genesis data to state",
	Long: `The init command initialises the node databases and loads the genesis data from the genesis file to state.
Examples: 
	gossamer init --genesis genesis.json
	gossamer init --chain westend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInit(cmd)
	},
}

// execInit executes the init command
func execInit(cmd *cobra.Command) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("failed to get --force: %s", err)
	}

	isInitialised, err := dot.IsNodeInitialised(config.BasePath)
	if err != nil {
		return fmt.Errorf("checking if node is initialised: %w", err)
	}

	if isInitialised {
		// prompt user to confirm reinitialization
		if force || confirmMessage("Are you sure you want to reinitialise the node? [Y/n]") {
			logger.Info("reinitialising node at base path " + config.BasePath + "...")
		} else {
			logger.Warn("exiting without reinitialising the node at base path " + config.BasePath + "...")
			return nil // exit if reinitialization is not confirmed
		}
	}

	// Write the config to the base path
	if err := cfg.WriteConfigFile(config.BasePath, config); err != nil {
		return fmt.Errorf("failed to ensure root: %s", err)
	}

	if err := dot.InitNode(config); err != nil {
		return fmt.Errorf("failed to initialise node: %s", err)
	}

	logger.Info("node initialised at: " + config.BasePath)
	return nil
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
