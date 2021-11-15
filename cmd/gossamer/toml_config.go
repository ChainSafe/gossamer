// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"unicode"

	"github.com/naoina/toml"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
)

// loadConfig loads the values from the toml configuration file into the provided configuration
func loadConfig(cfg *ctoml.Config, fp string) error {
	fp, err := filepath.Abs(fp)
	if err != nil {
		logger.Errorf("failed to create absolute path for toml configuration file: %s", err)
		return err
	}

	file, err := os.Open(filepath.Clean(fp))
	if err != nil {
		logger.Errorf("failed to open toml configuration file: %s", err)
		return err
	}

	var tomlSettings = toml.Config{
		NormFieldName: func(rt reflect.Type, key string) string {
			return key
		},
		FieldToKey: func(rt reflect.Type, field string) string {
			return field
		},
		MissingField: func(rt reflect.Type, field string) error {
			link := ""
			if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
				link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
			}
			return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
		},
	}

	if err = tomlSettings.NewDecoder(file).Decode(&cfg); err != nil {
		logger.Errorf("failed to decode configuration: %s", err)
		return err
	}

	return nil
}

// exportConfig exports a dot configuration to a toml configuration file
func exportConfig(cfg *ctoml.Config, fp string) *os.File {
	var (
		newFile *os.File
		err     error
		raw     []byte
	)

	if raw, err = toml.Marshal(*cfg); err != nil {
		logger.Errorf("failed to marshal configuration: %s", err)
		os.Exit(1)
	}

	newFile, err = os.Create(filepath.Clean(fp))
	if err != nil {
		logger.Errorf("failed to create configuration file: %s", err)
		os.Exit(1)
	}

	_, err = newFile.Write(raw)
	if err != nil {
		logger.Errorf("failed to write to configuration file: %s", err)
		os.Exit(1)
	}

	if err := newFile.Close(); err != nil {
		logger.Errorf("failed to close configuration file: %s", err)
		os.Exit(1)
	}

	return newFile
}
