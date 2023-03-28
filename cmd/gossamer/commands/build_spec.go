// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/spf13/cobra"
)

func init() {
	buildSpecCmd.Flags().Bool("raw", false, "print raw genesis json")
	buildSpecCmd.Flags().String("genesis", "", "path to human-readable genesis JSON file")
	buildSpecCmd.Flags().String("base-path", "", "path to node's base directory")
	buildSpecCmd.Flags().String("output-path", "", "path to output the recently created genesis JSON file")
}

var buildSpecCmd = &cobra.Command{
	Use:   "build-spec",
	Short: "Generates genesis JSON data, and can convert to raw genesis data",
	Long: `The build-spec command outputs current genesis JSON data.
Usage: gossamer build-spec
To generate raw genesis file from default:
	gossamer build-spec --raw --output genesis.json
To generate raw genesis file from specific genesis file:
	gossamer build-spec --raw --genesis genesis-spec.json --output-path genesis.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := execBuildSpec(cmd); err != nil {
			return err
		}
		return nil
	},
}

func execBuildSpec(cmd *cobra.Command) error {
	raw, err := cmd.Flags().GetBool("raw")
	if err != nil {
		return fmt.Errorf("failed to get raw value: %s", err)
	}

	genesisSpec, err := cmd.Flags().GetString("genesis-spec")
	if err != nil {
		return fmt.Errorf("failed to get genesis-spec value: %s", err)
	}

	basePath, err := cmd.Flags().GetString("base-path")
	if err != nil {
		return fmt.Errorf("failed to get base-path value: %s", err)
	}

	if genesisSpec == "" && basePath == "" {
		return fmt.Errorf("one of genesis-spec or base-path must be specified")
	}

	outputPath, err := cmd.Flags().GetString("output-path")
	if err != nil {
		return fmt.Errorf("failed to get output-path value: %s", err)
	}

	var bs *dot.BuildSpec

	if genesisSpec != "" {
		bs, err = dot.BuildFromGenesis(genesisSpec, 0)
		if err != nil {
			return err
		}
	} else {
		bs, err = dot.BuildFromDB(basePath)
		if err != nil {
			return fmt.Errorf("error building spec from database, init must be run before build-spec or run build-spec with --genesis flag Error %s", err)
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
		err = dot.WriteGenesisSpecFile(res, outputPath)
		if err != nil {
			return fmt.Errorf("cannot write genesis spec file: %w", err)
		}
	} else {
		fmt.Printf("%s\n", res)
	}

	return nil
}
