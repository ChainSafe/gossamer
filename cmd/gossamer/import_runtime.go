// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/lib/genesis"
)

func createGenesisWithRuntime(fp string, genesisSpecFilePath string) (string, error) {
	runtime, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return "", err
	}

	genesis, err := genesis.NewGenesisSpecFromJSON(genesisSpecFilePath)
	if err != nil {
		return "", err
	}

	genesis.Genesis.Runtime["system"]["code"] = fmt.Sprintf("0x%x", runtime)
	bz, err := json.MarshalIndent(genesis, "", "\t")
	if err != nil {
		return "", err
	}

	return string(bz), nil
}
