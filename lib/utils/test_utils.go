// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"fmt"
	"os"
	"path"
	"testing"
)

// TestDir test data directory
const TestDir = "./test_data"

// NewTestDir create new test data directory
func NewTestDir(t *testing.T) string {
	testDir := path.Join(TestDir, t.Name())

	err := os.Mkdir(TestDir, os.ModePerm)
	if err != nil && !PathExists(TestDir) {
		fmt.Println(fmt.Errorf("failed to create test directory: %s", err))
	}

	err = os.Mkdir(testDir, os.ModePerm)
	if err != nil && !PathExists(testDir) {
		fmt.Println(fmt.Errorf("failed to create test directory: %s", err))
	}

	return testDir
}

// NewTestBasePath create new test data directory
func NewTestBasePath(t *testing.T, name string) string {
	testDir := path.Join(TestDir, t.Name())
	basePath := path.Join(testDir, name)

	err := os.Mkdir(TestDir, os.ModePerm)
	if err != nil && !PathExists(TestDir) {
		fmt.Println(fmt.Errorf("failed to create test directory: %s", err))
	}

	err = os.Mkdir(testDir, os.ModePerm)
	if err != nil && !PathExists(testDir) {
		fmt.Println(fmt.Errorf("failed to create test directory: %s", err))
	}

	err = os.Mkdir(basePath, os.ModePerm)
	if err != nil && !PathExists(basePath) {
		fmt.Println(fmt.Errorf("failed to create test data directory: %s", err))
	}

	return basePath
}

// RemoveTestDir removes the test data directory
func RemoveTestDir(t *testing.T) {
	testDir := path.Join(TestDir, t.Name())
	err := os.RemoveAll(testDir)
	if err != nil || PathExists(testDir) {
		fmt.Println(fmt.Errorf("failed to remove test directory: %s", err))
	}
}
