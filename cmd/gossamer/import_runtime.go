// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/internal/lib/genesis"
)

var defaultGenesisSpecPath = "./internal/chain/gssmr/genesis-spec.json"

func createGenesisWithRuntime(fp string) (string, error) {
	runtime, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return "", err
	}

	genesis, err := genesis.NewGenesisSpecFromJSON(defaultGenesisSpecPath)
	if err != nil {
		return "", err
	}

	genesis.Genesis.Runtime["System"]["code"] = fmt.Sprintf("0x%x", runtime)
	bz, err := json.MarshalIndent(genesis, "", "\t")
	if err != nil {
		return "", err
	}

	return string(bz), nil
}
