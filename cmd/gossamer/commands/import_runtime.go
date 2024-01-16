// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/gossamer/lib/genesis"

	"github.com/ChainSafe/gossamer/lib/os"
	"github.com/spf13/cobra"
)

func init() {
	ImportRuntimeCmd.Flags().String("wasm-file", "", "path to wasm runtime binary file")
}

// ImportRuntimeCmd is the command to import a runtime binary into a genesis file
var ImportRuntimeCmd = &cobra.Command{
	Use:   "import-runtime",
	Short: "Appends the given .wasm runtime binary to a chain-spec",
	Long: `The import-runtime command appends the given .wasm runtime binary to a chain-spec.
Example: 
	gossamer import-runtime --wasm-file runtime.wasm --chain chain-spec.json > chain-spec-new.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execImportRuntime(cmd)
	},
}

// execImportRuntime executes the import-runtime command
func execImportRuntime(cmd *cobra.Command) error {
	wasmFile, err := cmd.Flags().GetString("wasm-file")
	if err != nil {
		return fmt.Errorf("failed to get wasm-file: %s", err)
	}
	if wasmFile == "" {
		return fmt.Errorf("wasm-file must be specified")
	}

	chainSpec, err := cmd.Flags().GetString("chain")
	if err != nil {
		return fmt.Errorf("failed to get chain-spec: %s", err)
	}
	if chainSpec == "" {
		return fmt.Errorf("chain must be specified")
	}

	out, err := createGenesisWithRuntime(wasmFile, chainSpec)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

// createGenesisWithRuntime creates a genesis file with the given runtime
func createGenesisWithRuntime(fp string, genesisSpecFilePath string) (string, error) {
	runtime, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return "", err
	}

	chainSpec, err := genesis.NewGenesisSpecFromJSON(genesisSpecFilePath)
	if err != nil {
		return "", err
	}

	chainSpec.Genesis.Runtime.System.Code = fmt.Sprintf("0x%x", runtime)
	jsonSpec, err := json.MarshalIndent(chainSpec, "", "\t")
	if err != nil {
		return "", err
	}

	return string(jsonSpec), nil
}
