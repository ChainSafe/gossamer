// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.
package life

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"

	log "github.com/ChainSafe/log15"
	"github.com/perlin-network/life/exec"
	wasm_validation "github.com/perlin-network/life/wasm-validation"
)

// Name represents the name of the interpreter
const Name = "life"

// Check that runtime interfaces are satisfied
var (
	_      runtime.Instance = (*Instance)(nil)
	logger                  = log.New("pkg", "runtime", "module", "perlin/life")
	ctx    *runtime.Context
)

// Config represents a life configuration
type Config struct {
	runtime.InstanceConfig
	Resolver exec.ImportResolver
}

// Instance represents a v0.8 runtime life instance
type Instance struct {
	vm      *exec.VirtualMachine
	mu      sync.Mutex
	version runtime.Version
}

// GetCodeHash returns code hash of the runtime
func (*Instance) GetCodeHash() common.Hash {
	return common.Hash{}
}

// NewRuntimeFromGenesis creates a runtime instance from the genesis data
func NewRuntimeFromGenesis(cfg *Config) (runtime.Instance, error) {
	if cfg.Storage == nil {
		return nil, errors.New("storage is nil")
	}

	code := cfg.Storage.LoadCode()
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in state")
	}

	cfg.Resolver = new(Resolver)
	return NewInstance(code, cfg)
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg *Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	if err = wasm_validation.ValidateWasm(bytes); err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance ...
func NewInstance(code []byte, cfg *Config) (*Instance, error) {
	if len(code) == 0 {
		return nil, errors.New("code is empty")
	}

	// if cfg.LogLvl set to < 0, then don't change package log level
	if cfg.LogLvl >= 0 {
		h := log.StreamHandler(os.Stdout, log.TerminalFormat())
		h = log.CallerFileHandler(h)
		logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	}

	vmCfg := exec.VMConfig{
		DefaultMemoryPages: 23,
	}

	instance, err := exec.NewVirtualMachine(code, vmCfg, cfg.Resolver, nil)
	if err != nil {
		return nil, err
	}

	memory := &Memory{
		memory: instance.Memory,
	}

	// TODO: use __heap_base (#1874)
	allocator := runtime.NewAllocator(memory, 0)

	runtimeCtx := &runtime.Context{
		Storage:     cfg.Storage,
		Allocator:   allocator,
		Keystore:    cfg.Keystore,
		Validator:   cfg.Role == byte(4),
		NodeStorage: cfg.NodeStorage,
		Network:     cfg.Network,
		Transaction: cfg.Transaction,
		SigVerifier: runtime.NewSignatureVerifier(),
	}

	logger.Debug("creating new runtime instance", "context", runtimeCtx)

	inst := &Instance{
		vm: instance,
	}

	ctx = runtimeCtx
	inst.version, err = inst.Version()
	if err != nil {
		logger.Error("error checking instance version", "error", err)
	}
	return inst, nil
}

// Memory is a thin wrapper around life's memory to support
// Gossamer runtime.Memory interface
type Memory struct {
	memory []byte
}

// Data returns the memory's data
func (m *Memory) Data() []byte {
	return m.memory
}

// Length returns the memory's length
func (m *Memory) Length() uint32 {
	return uint32(len(m.memory))
}

// Grow ...
func (m *Memory) Grow(numPages uint32) error {
	m.memory = append(m.memory, make([]byte, runtime.PageSize*numPages)...)
	return nil
}

// UpdateRuntimeCode ...
func (*Instance) UpdateRuntimeCode(_ []byte) error {
	return errors.New("unimplemented")
}

// CheckRuntimeVersion ...
func (*Instance) CheckRuntimeVersion(_ []byte) (runtime.Version, error) {
	return nil, errors.New("unimplemented")
}

// SetContextStorage sets the runtime's storage. It should be set before calls to the below functions.
func (*Instance) SetContextStorage(s runtime.Storage) {
	ctx.Storage = s
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) ([]byte, error) {
	in.mu.Lock()
	defer in.mu.Unlock()

	ptr, err := ctx.Allocator.Allocate(uint32(len(data)))
	if err != nil {
		return nil, err
	}
	defer ctx.Allocator.Clear()

	copy(in.vm.Memory[ptr:ptr+uint32(len(data))], data)

	fnc, ok := in.vm.GetFunctionExport(function)
	if !ok {
		return nil, fmt.Errorf("could not find exported function %s", function)
	}

	ret, err := in.vm.Run(fnc, int64(ptr), int64(len(data)))
	if err != nil {
		fmt.Println(in.vm.StackTrace)
		return nil, err
	}

	offset, length := runtime.Int64ToPointerAndSize(ret)
	return in.vm.Memory[offset : offset+length], nil
}

// Stop ...
func (*Instance) Stop() {}

// NodeStorage to get reference to runtime node service
func (*Instance) NodeStorage() runtime.NodeStorage {
	return ctx.NodeStorage
}

// NetworkService to get referernce to runtime network service
func (*Instance) NetworkService() runtime.BasicNetwork {
	return ctx.Network
}

// Validator returns the context's Validator
func (*Instance) Validator() bool {
	return ctx.Validator
}

// Keystore to get reference to runtime keystore
func (*Instance) Keystore() *keystore.GlobalKeystore {
	return ctx.Keystore
}
