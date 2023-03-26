package commands

import (
	"bufio"
	"fmt"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	westend_dev "github.com/ChainSafe/gossamer/chain/westend-dev"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const confirmCharacter = "Y"

func init() {
	initCmd.Flags().String("chain", "", "chain id")
	initCmd.Flags().Bool("force", false, "force node initialization")
	initCmd.Flags().String("base-path", "", "base path")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init",
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
	if basePath != "" {
		config.BaseConfig.BasePath = basePath
	}

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

	if err := dot.InitNode(config); err != nil {
		return err
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
