// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/require"
)

// DefaultDatabaseDir directory inside basepath where database contents are stored
const DefaultDatabaseDir = "db"

// SetupDatabase will return an instance of database based on basepath
func SetupDatabase(basepath string, inMemory bool) (chaindb.Database, error) {
	return chaindb.NewBadgerDB(&chaindb.Config{
		DataDir:  filepath.Join(basepath, DefaultDatabaseDir),
		InMemory: inMemory,
	})
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

	keys := []string{}

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

// GetGssmrGenesisRawPathTest gets the gssmr raw genesis path
// and fails the test if it cannot find it.
func GetGssmrGenesisRawPathTest(t *testing.T) string {
	t.Helper()
	path, err := GetGssmrGenesisRawPath()
	require.NoError(t, err)
	return path
}

// GetGssmrGenesisRawPath gets the gssmr raw genesis path
// and returns an error if it cannot find it.
func GetGssmrGenesisRawPath() (path string, err error) {
	rootPath, err := GetProjectRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, "./chain/gssmr/genesis.json"), nil
}

// GetGssmrGenesisPathTest gets the gssmr genesis path
// and fails the test if it cannot find it.
func GetGssmrGenesisPathTest(t *testing.T) string {
	t.Helper()
	path, err := GetGssmrGenesisPath()
	require.NoError(t, err)
	return path
}

// GetGssmrV3SubstrateGenesisPathTest gets the v3 substrate gssmr genesis path
// and fails the test if it cannot find it.
func GetGssmrV3SubstrateGenesisPathTest(t *testing.T) string {
	t.Helper()
	path, err := GetGssmrV3SubstrateGenesisPath()
	require.NoError(t, err)
	return path
}

// GetGssmrV3SubstrateGenesisRawPathTest gets the v3 substrate gssmr raw genesis path
// and fails the test if it cannot find it.
func GetGssmrV3SubstrateGenesisRawPathTest(t *testing.T) string {
	t.Helper()
	path, err := GetGssmrV3SubstrateGenesisRawPath()
	require.NoError(t, err)
	return path
}

// GetGssmrV3SubstrateGenesisRawPath gets the v3 substrate gssmr raw genesis path
// and returns an error if it cannot find it.
func GetGssmrV3SubstrateGenesisRawPath() (path string, err error) {
	rootPath, err := GetProjectRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, "./chain/gssmr-v3substrate/genesis.json"), nil
}

// GetDevGenesisPath gets the dev genesis path
func GetDevGenesisPath(t *testing.T) string {
	return filepath.Join(GetProjectRootPathTest(t), "./chain/dev/genesis.json")
}

// GetDevV3SubstrateGenesisPath gets the v3 substrate dev genesis path
func GetDevV3SubstrateGenesisPath(t *testing.T) string {
	return filepath.Join(GetProjectRootPathTest(t), "./chain/dev-v3substrate/genesis.json")
}

// GetDevGenesisSpecPathTest gets the dev genesis spec path
func GetDevGenesisSpecPathTest(t *testing.T) string {
	return filepath.Join(GetProjectRootPathTest(t), "./chain/dev/genesis-spec.json")
}

// GetGssmrGenesisPath gets the gssmr genesis path
// and returns an error if it cannot find it.
func GetGssmrGenesisPath() (path string, err error) {
	rootPath, err := GetProjectRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, "./chain/gssmr/genesis-spec.json"), nil
}

// GetGssmrV3SubstrateGenesisPath gets the v3 substrate gssmr genesis path
// and returns an error if it cannot find it.
func GetGssmrV3SubstrateGenesisPath() (path string, err error) {
	rootPath, err := GetProjectRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, "./chain/gssmr-v3substrate/genesis-spec.json"), nil
}

// GetKusamaGenesisPath gets the Kusama genesis path
func GetKusamaGenesisPath(t *testing.T) string {
	return filepath.Join(GetProjectRootPathTest(t), "./chain/kusama/genesis.json")
}

// GetPolkadotGenesisPath gets the Polkadot genesis path
func GetPolkadotGenesisPath(t *testing.T) string {
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
	ErrFindProjectRoot = fmt.Errorf("cannot find project root")
)

// GetProjectRootPath finds the root of the project where `go.mod` is
// and returns it as an absolute path.
func GetProjectRootPath() (rootPath string, err error) {
	_, fullpath, _, _ := runtime.Caller(0)
	rootPath = path.Dir(fullpath)

	const directoryToFind = "chain"
	const subPathsToFind = "dev,gssmr,kusama,polkadot"

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
func LoadChainDB(basePath string) (*chaindb.BadgerDB, error) {
	cfg := &chaindb.Config{
		DataDir: basePath,
	}

	// Open already existing DB
	db, err := chaindb.NewBadgerDB(cfg)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// LoadBadgerDB load the db at the given path.
func LoadBadgerDB(basePath string) (*badger.DB, error) {
	opts := badger.DefaultOptions(basePath)
	// Open already existing DB
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return db, nil
}
