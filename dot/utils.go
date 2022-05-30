// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/cosmos/go-bip39"
	"github.com/naoina/toml"
)

// exportConfig exports a dot configuration to a toml configuration file
func exportConfig(cfg *Config, fp string) {
	raw, err := toml.Marshal(*cfg)
	if err != nil {
		logger.Errorf("failed to marshal configuration: %s", err)
		os.Exit(1)
	}
	if err := os.WriteFile(fp, raw, 0600); err != nil {
		logger.Errorf("failed to write file: %s", err)
		os.Exit(1)
	}
}

// ExportTomlConfig exports a dot configuration to a toml configuration file
func ExportTomlConfig(cfg *ctoml.Config, fp string) {
	raw, err := toml.Marshal(*cfg)
	if err != nil {
		logger.Errorf("failed to marshal configuration: %s", err)
		os.Exit(1)
	}
	if err := os.WriteFile(fp, raw, 0600); err != nil {
		logger.Errorf("failed to write file: %s", err)
		os.Exit(1)
	}
}

// CreateJSONRawFile will generate a JSON genesis file with raw storage
func CreateJSONRawFile(bs *BuildSpec, fp string) {
	data, err := bs.ToJSONRaw()
	if err != nil {
		logger.Errorf("failed to convert into raw json: %s", err)
		os.Exit(1)
	}

	if err := os.WriteFile(fp, data, 0600); err != nil {
		logger.Errorf("failed to write file: %s", err)
		os.Exit(1)
	}
}

// RandomNodeName generates a new random name if there is no name configured for the node
func RandomNodeName() string {
	entropy, _ := bip39.NewEntropy(128)
	randomNamesString, _ := bip39.NewMnemonic(entropy)
	randomNames := strings.Split(randomNamesString, " ")
	number := binary.BigEndian.Uint16(entropy)
	return randomNames[0] + "-" + randomNames[1] + "-" + fmt.Sprint(number)
}
