// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package newWasmer

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/wasmerio/wasmer-go/wasmer"
	"os"
	"path/filepath"
	"sync"
)

// Name represents the name of the interpreter
const Name = "wasmer"

var (
	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "go-wasmer"),
	)
)

// Instance represents a runtime go-wasmer instance
type Instance struct {
	vm       wasmer.Instance
	ctx      *runtime.Context
	isClosed bool
	codeHash common.Hash
	mutex    sync.Mutex
}

// NewRuntimeFromGenesis creates a runtime instance from the genesis data
func NewRuntimeFromGenesis(cfg Config) (instance *Instance, err error) {
	if cfg.Storage == nil {
		return nil, errors.New("storage is nil")
	}

	code := cfg.Storage.LoadCode()
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in state")
	}

	return NewInstance(code, cfg)
}

// NewInstanceFromTrie returns a new runtime instance with the code provided in the given trie
func NewInstanceFromTrie(t *trie.Trie, cfg Config) (*Instance, error) {
	code := t.Get(common.CodeKey)
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in trie")
	}

	return NewInstance(code, cfg)
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
// TODO should cfg be a pointer?
func NewInstance(code []byte, cfg Config) (*Instance, error) {
	return newInstance(code, cfg)
}

func newInstance(code []byte, cfg Config) (instance *Instance, err error) {
	logger.Patch(log.SetLevel(cfg.LogLvl), log.SetCallerFunc(true))

	// TODO fix return
	return nil, nil
}
