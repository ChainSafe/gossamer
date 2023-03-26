package commands

import (
	"fmt"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/spf13/cobra"
)

func init() {
	importStateCmd.Flags().String("base-path", "", "base path")
	importStateCmd.Flags().String("chain", "", "chain id")
	importStateCmd.Flags().String("state-file", "", "path to state file")
	importStateCmd.Flags().String("header-file", "", "path to header file")
	importStateCmd.Flags().Uint64("first-slot", 0, "first slot to import")
}

var importStateCmd = &cobra.Command{
	Use:   "import-state",
	Short: "import-state",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := execImportState(cmd); err != nil {
			return err
		}
		return nil
	},
}

func execImportState(cmd *cobra.Command) error {
	chainID, err := cmd.Flags().GetString("chain")
	if err != nil {
		return fmt.Errorf("failed to get chain: %s", err)
	}

	basePath, err := cmd.Flags().GetString("base-path")
	if err != nil {
		return fmt.Errorf("failed to get base-path: %s", err)
	}

	if chainID == "" && basePath == "" {
		return fmt.Errorf("one of chain or base-path must be specified")
	}

	if basePath == "" {
		switch chainID {
		case "polkadot":
			basePath = polkadot.DefaultBasePath
		case "kusama":
			basePath = kusama.DefaultBasePath
		case "westend":
			basePath = westend.DefaultBasePath
		case "westend-dev":
			basePath = "~/.gossamer/westend-dev"
		default:
			return fmt.Errorf("chain %s not supported", chainID)
		}
	}

	firstSlot, err := cmd.Flags().GetUint64("first-slot")
	if err != nil {
		return fmt.Errorf("failed to get first-slot: %s", err)
	}

	stateFile, err := cmd.Flags().GetString("state-file")
	if err != nil {
		return fmt.Errorf("failed to get state-file: %s", err)
	}
	if stateFile == "" {
		return fmt.Errorf("state-file must be specified")
	}

	headerFile, err := cmd.Flags().GetString("header-file")
	if err != nil {
		return fmt.Errorf("failed to get header-file: %s", err)
	}
	if headerFile == "" {
		return fmt.Errorf("header-file must be specified")
	}

	basePath = utils.ExpandDir(basePath)

	return dot.ImportState(basePath, stateFile, headerFile, firstSlot)
}
