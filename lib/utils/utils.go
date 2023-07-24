// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/stretchr/testify/require"
)

// DefaultDatabaseDir directory inside basepath where database contents are stored
const DefaultDatabaseDir = "db"

// SetupDatabase will return an instance of database based on basepath
func SetupDatabase(basepath string, inMemory bool) (database.Database, error) {
	basepath = filepath.Join(basepath, DefaultDatabaseDir)
	return database.NewPebble(basepath, inMemory)
}

// PathExists returns true if the named file or directory exists, otherwise false
func PathExists(p string) bool {
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// HomeDir returns the user's current HOME directory
func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

// ExpandDir expands a tilde prefix path to a full home path
func ExpandDir(targetPath string) string {
	if strings.HasPrefix(targetPath, "~\\") || strings.HasPrefix(targetPath, "~/") {
		if homeDir := HomeDir(); homeDir != "" {
			targetPath = homeDir + targetPath[1:]
		}
	} else if strings.HasPrefix(targetPath, ".\\") || strings.HasPrefix(targetPath, "./") {
		targetPath, _ = filepath.Abs(targetPath)
	}
	return path.Clean(os.ExpandEnv(targetPath))
}

// BasePath attempts to create a data directory using the given name within the
// gossamer directory within the user's HOME directory, returns absolute path
// or, if unable to locate HOME directory, returns within current directory
func BasePath(name string) string {
	home := HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Gossamer", name)
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Gossamer", name)
		} else {
			return filepath.Join(home, ".gossamer", name)
		}
	}
	return name
}

// KeystoreDir returns the absolute filepath of the keystore directory
func KeystoreDir(basepath string) (keystorepath string, err error) {
	// basepath specified, set keystore filepath to absolute path of [basepath]/keystore
	if basepath != "" {
		basepath = ExpandDir(basepath)
		keystorepath, err = filepath.Abs(basepath + "/keystore")
		if err != nil {
			return "", fmt.Errorf("failed to create absolute filepath: %s", err)
		}
	}

	// if basepath does not exist, create it
	if _, err = os.Stat(keystorepath); os.IsNotExist(err) {
		err = os.Mkdir(keystorepath, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create data directory: %s", err)
		}
	}

	// if basepath/keystore does not exist, create it
	if _, err = os.Stat(keystorepath); os.IsNotExist(err) {
		err = os.Mkdir(keystorepath, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create keystore directory: %s", err)
		}
	}

	return keystorepath, nil
}

// KeystoreFiles returns the filenames of all the keys in the basepath's keystore
func KeystoreFiles(basepath string) ([]string, error) {
	keystorepath, err := KeystoreDir(basepath)
	if err != nil {
		return nil, fmt.Errorf("failed to get keystore directory: %s", err)
	}

	files, err := os.ReadDir(keystorepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore directory: %s", err)
	}

	var keys []string

	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext == ".key" {
			keys = append(keys, f.Name())
		}
	}

	return keys, nil
}

// KeystoreFilepaths lists all the keys in the basepath/keystore/ directory and returns them as a list of filepaths
func KeystoreFilepaths(basepath string) ([]string, error) {
	keys, err := KeystoreFiles(basepath)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		fmt.Printf("[%d] %s\n", i, key)
	}

	return keys, nil
}

// GetWestendDevHumanReadableGenesisPath gets the westend-dev human readable spec filepath
func GetWestendDevHumanReadableGenesisPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(GetProjectRootPathTest(t), "./chain/westend-dev/westend-dev-spec.json")
}

// GetWestendDevRawGenesisPath gets the westend-dev genesis raw path
func GetWestendDevRawGenesisPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(GetProjectRootPathTest(t), "./chain/westend-dev/westend-dev-spec-raw.json")
}

// GetWestendLocalRawGenesisPath gets the westend-local genesis raw path
func GetWestendLocalRawGenesisPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(GetProjectRootPathTest(t), "./chain/westend-local/westend-local-spec-raw.json")
}

// GetKusamaGenesisPath gets the Kusama genesis path
func GetKusamaGenesisPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(GetProjectRootPathTest(t), "./chain/kusama/genesis.json")
}

// GetPolkadotGenesisPath gets the Polkadot genesis path
func GetPolkadotGenesisPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(GetProjectRootPathTest(t), "./chain/polkadot/genesis.json")
}

// GetProjectRootPathTest finds the root of the project where `go.mod` is
// and returns it as an absolute path. It fails the test if it's not found.
func GetProjectRootPathTest(t *testing.T) (rootPath string) {
	t.Helper()
	rootPath, err := GetProjectRootPath()
	require.NoError(t, err)
	return rootPath
}

var (
	ErrFindProjectRoot = errors.New("cannot find project root")
)

// GetProjectRootPath finds the root of the project where directory `cmd`
// and subdirectory `gossamer` is and returns it as an absolute path.
func GetProjectRootPath() (rootPath string, err error) {
	_, fullpath, _, _ := runtime.Caller(0)
	rootPath = path.Dir(fullpath)

	const directoryToFind = "cmd"
	const subPathsToFind = "gossamer"

	subPaths := strings.Split(subPathsToFind, ",")

	for {
		pathToCheck := path.Join(rootPath, directoryToFind)
		stats, err := os.Stat(pathToCheck)

		if os.IsNotExist(err) {
			previousRootPath := rootPath
			rootPath = path.Dir(rootPath)

			if rootPath == previousRootPath {
				return "", ErrFindProjectRoot
			}

			continue
		}

		if err != nil {
			return "", err
		}

		if !stats.IsDir() {
			continue
		}

		dirEntries, err := os.ReadDir(pathToCheck)
		if err != nil {
			return "", err
		}

		subPathsSet := make(map[string]struct{}, len(subPaths))
		for _, subPath := range subPaths {
			subPathsSet[subPath] = struct{}{}
		}

		for _, dirEntry := range dirEntries {
			delete(subPathsSet, dirEntry.Name())
		}

		if len(subPathsSet) > 0 {
			continue
		}

		break
	}

	rootPath, err = filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}

	return rootPath, nil
}

// LoadChainDB load the db at the given path.
func LoadChainDB(basePath string) (database.Database, error) {
	// Open already existing DB
	db, err := database.NewPebble(basePath, false)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return db, nil
}
