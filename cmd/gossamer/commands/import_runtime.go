// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	substrategenesis "github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/os"
	"github.com/spf13/cobra"
)

func init() {
	ImportRuntimeCmd.Flags().String("wasm-file", "", "path to wasm runtime binary file")
	ImportRuntimeCmd.Flags().String("genesis-file", "", "path to genesis file")
}

// ImportRuntimeCmd is the command to import a runtime binary into a genesis file
var ImportRuntimeCmd = &cobra.Command{
	Use:   "import-runtime",
	Short: "Appends the given .wasm runtime binary to a genesis file",
	Long: `The import-runtime command appends the given .wasm runtime binary to a genesis file.
Example: 
	gossamer import-runtime --wasm-file runtime.wasm --genesis-file genesis.json > updated_genesis.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := execImportRuntime(cmd); err != nil {
			return err
		}
		return nil
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

	genesisFile, err := cmd.Flags().GetString("genesis-file")
	if err != nil {
		return fmt.Errorf("failed to get genesis-file: %s", err)
	}
	if genesisFile == "" {
		return fmt.Errorf("genesis-file must be specified")
	}

	out, err := createGenesisWithRuntime(wasmFile, genesisFile)
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

	genesisSpec, err := substrategenesis.NewGenesisSpecFromJSON(genesisSpecFilePath)
	if err != nil {
		return "", err
	}

	genesisSpec.Genesis.Runtime["system"]["code"] = fmt.Sprintf("0x%x", runtime)
	bz, err := json.MarshalIndent(genesisSpec, "", "\t")
	if err != nil {
		return "", err
	}

	return string(bz), nil
}
