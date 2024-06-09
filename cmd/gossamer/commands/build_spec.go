// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/spf13/cobra"
)

func init() {
	BuildSpecCmd.Flags().Bool("raw", false, "print raw genesis json")
	BuildSpecCmd.Flags().
		String("output-path", "", "path to output the recently created chain-spec JSON file")
}

// BuildSpecCmd is the command to generate genesis JSON
var BuildSpecCmd = &cobra.Command{
	Use:   "build-spec",
	Short: "Generates chain-spec JSON data, and can convert to raw chain-spec data",
	Long: `The build-spec command outputs current chain-spec JSON data.
Usage: gossamer build-spec
To generate raw chain-spec file from default:
	gossamer build-spec --raw --output chain-spec.json
To generate raw chain-spec file from specific chain-spec file:
	gossamer build-spec --raw --chain chain-spec.json --output-path chain-spec-raw.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execBuildSpec(cmd)
	},
}

// execBuildSpec executes the build-spec command
func execBuildSpec(cmd *cobra.Command) error {
	var err error
	raw, err := cmd.Flags().GetBool("raw")
	if err != nil {
		return fmt.Errorf("failed to get raw value: %s", err)
	}

	chainSpec, err := cmd.Flags().GetString("chain")
	if err != nil {
		return fmt.Errorf("failed to get genesis-spec value: %s", err)
	}

	basePath, err = cmd.Flags().GetString("base-path")
	if err != nil {
		return fmt.Errorf("failed to get base-path value: %s", err)
	}

	if chainSpec == "" && basePath == "" {
		return fmt.Errorf("one of chain or base-path must be specified")
	}

	outputPath, err := cmd.Flags().GetString("output-path")
	if err != nil {
		return fmt.Errorf("failed to get output-path value: %s", err)
	}

	var bs *dot.BuildSpec

	if chainSpec != "" {
		bs, err = dot.BuildFromGenesis(chainSpec, 0)
		if err != nil {
			return err
		}
	} else {
		bs, err = dot.BuildFromDB(basePath)
		if err != nil {
			return fmt.Errorf("error building spec from database, "+
				"init must be run before build-spec or run build-spec "+
				"with --genesis flag Error %s", err)
		}
	}

	if bs == nil {
		return fmt.Errorf("error building genesis")
	}

	var res []byte
	if raw {
		res, err = bs.ToJSONRaw()
	} else {
		res, err = bs.ToJSON()
	}

	if err != nil {
		return err
	}

	if outputPath != "" {
		if err = dot.WriteGenesisSpecFile(res, outputPath); err != nil {
			return fmt.Errorf("cannot write genesis spec file: %w", err)
		}
		return nil
	}

	fmt.Printf("%s\n", res)
	return nil
}
