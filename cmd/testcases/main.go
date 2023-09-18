// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Replace all spaces in strings in test cases with underscores.
// See https://github.com/ChainSafe/gossamer/issues/3039.
func main() {
	err := filepath.WalkDir(".", walk)
	if err != nil {
		fmt.Printf("error: %s", err)
		os.Exit(1)
	}
}

var (
	regexSubtestStringWithSpaces = regexp.MustCompile(`\tt\.Run\(".+ .+"?\)`)
	regexMapStringKeyWithSpaces  = regexp.MustCompile(`\t".+ .+"?: \{`)
	regexSliceStringWithSpaces   = regexp.MustCompile(`(name|test)( |\t)*: ".+ .+",`)
	regexStringWithSpaces        = regexp.MustCompile(`".+( .+)+"`)
)

func walk(path string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if entry.IsDir() {
		return nil
	}

	fileName := entry.Name()
	if !strings.HasSuffix(fileName, "_test.go") {
		return nil
	}

	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}
	existingPerms := info.Mode().Perm()

	existing, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	lines := strings.Split(string(existing), "\n")

	foundAtLeastOne := false
	for i, line := range lines {
		var toReplace string

		switch {
		case regexSubtestStringWithSpaces.MatchString(line):
			toReplace = regexSubtestStringWithSpaces.FindString(line)
		case regexMapStringKeyWithSpaces.MatchString(line):
			toReplace = regexMapStringKeyWithSpaces.FindString(line)
		case regexSliceStringWithSpaces.MatchString(line):
			toReplace = regexSliceStringWithSpaces.FindString(line)
		}

		if toReplace == "" {
			continue
		}

		foundAtLeastOne = true
		toReplace = regexStringWithSpaces.FindString(toReplace)
		replaced := strings.ReplaceAll(toReplace, " ", "_")
		lines[i] = strings.Replace(line, toReplace, replaced, 1)
	}

	if !foundAtLeastOne {
		return nil
	}

	err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), existingPerms)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
