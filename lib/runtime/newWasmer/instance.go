// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package newWasmer

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/wasmerio/wasmer-go/wasmer"
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

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg *Config) (*Instance, error) {
	return newInstance(code, cfg)
}

func newInstance(code []byte, cfg *Config) (instance *Instance, err error) {
	logger.Patch(log.SetLevel(cfg.LogLvl), log.SetCallerFunc(true))

	// TODO fix return
	return nil, nil
}
