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

	"github.com/ChainSafe/chaindb"
	"github.com/dgraph-io/badger/v2"
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

// GetGssmrGenesisRawPath gets the gssmr raw genesis path
func GetGssmrGenesisRawPath() string {
	chainDir, err := locateChainDirectory()
	if err != nil {
		panic(err)
	}

	return filepath.Join(chainDir, "gssmr/genesis.json")
}

// GetGssmrGenesisPath gets the gssmr human-readable genesis path
func GetGssmrGenesisPath() string {
	chainDir, err := locateChainDirectory()
	if err != nil {
		panic(err)
	}

	return filepath.Join(chainDir, "gssmr/genesis-spec.json")
}

// GetKusamaGenesisPath gets the kusama genesis path
func GetKusamaGenesisPath() string {
	chainDir, err := locateChainDirectory()
	if err != nil {
		panic(err)
	}

	return filepath.Join(chainDir, "kusama/genesis.json")
}

// GetDevGenesisPath gets the dev genesis path
func GetDevGenesisPath() string {
	chainDir, err := locateChainDirectory()
	if err != nil {
		panic(err)
	}

	return filepath.Join(chainDir, "dev/genesis.json")
}

var (
	ErrChainDirectoryPathNotValid = errors.New("chain directory path is not valid")
	ErrChainDirectoryPathNotFound = errors.New("chain directory path not found")
)

func locateChainDirectory() (chainDir string, err error) {
	pathsToTry := []string{
		"../../internal/chain/", // from cmd/gossamer
		"../chain/",             // from internal/package
		"../../chain/",          // from internal/package/subpackage
		"../../../chain/",       // from internal/package/subpackage/subpackage
	}

	for _, pathToTry := range pathsToTry {
		if PathExists(pathToTry) {
			path, err := filepath.Abs(pathToTry)
			if err != nil {
				return "", fmt.Errorf("%w: %s", ErrChainDirectoryPathNotValid, pathToTry)
			}
			return path, nil
		}
	}

	return "", ErrChainDirectoryPathNotFound
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
