// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"bufio"
	"fmt"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westend_dev "github.com/ChainSafe/gossamer/chain/westend-dev"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const confirmCharacter = "Y"

func init() {
	initCmd.Flags().String("chain", "", "the default chain configuration to load. Example: --chain kusama")
	initCmd.Flags().Bool("force", false, "force reinitialization of node")
	initCmd.Flags().String("base-path", "", "the path to the node's base directory. Example: --base-path /my/custom/path")
	initCmd.Flags().String("genesis", "", "the path to the genesis configuration to load. Example: --genesis genesis.json")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise node databases and load genesis data to state",
	Long: `The init command initialises the node databases and loads the genesis data from the genesis file to state.
Example: 
	gossamer init --genesis genesis.json
	gossamer init --chain westend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := execInit(cmd); err != nil {
			return err
		}
		return nil
	},
}

func execInit(cmd *cobra.Command) error {
	chain, err := cmd.Flags().GetString("chain")
	if err != nil {
		return fmt.Errorf("failed to get --chain: %s", err)
	}

	switch chain {
	case "polkadot":
		config = polkadot.DefaultConfig()
	case "kusama":
		config = kusama.DefaultConfig()
	case "westend":
		config = westend.DefaultConfig()
	case "westend-dev":
		config = westend_dev.DefaultConfig()
	default:
		return fmt.Errorf("chain %s not supported", chain)
	}

	basePath, err := cmd.Flags().GetString("base-path")
	if err != nil {
		return fmt.Errorf("failed to get --base-path: %s", err)
	}
	if config.BasePath == "" && basePath == "" {
		return fmt.Errorf("base-path not set")
	}
	if basePath != "" {
		config.BasePath = basePath
	}
	config.BasePath = utils.ExpandDir(config.BasePath)

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("failed to get --force: %s", err)
	}

	if dot.IsNodeInitialised(config.BasePath) {
		// prompt user to confirm reinitialization
		if force || confirmMessage("Are you sure you want to reinitialise the node? [Y/n]") {
			logger.Info("reinitialising node at base path " + config.BasePath + "...")
		} else {
			logger.Warn("exiting without reinitialising the node at base path " + config.BasePath + "...")
			return nil // exit if reinitialization is not confirmed
		}
	}

	if err := config.ValidateBasic(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
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
