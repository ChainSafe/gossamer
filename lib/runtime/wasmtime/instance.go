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

package wasmtime

import (
	"errors"
	"os"
	"runtime"
	"sync"

	gssmrruntime "github.com/ChainSafe/gossamer/lib/runtime"

	log "github.com/ChainSafe/log15"
	"github.com/bytecodealliance/wasmtime-go"
)

// Name represents the name of the interpreter
const Name = "wasmtime"

var (
	_ gssmrruntime.Instance = (*Instance)(nil)

	ctx    gssmrruntime.Context
	logger = log.New("pkg", "runtime", "module", "go-wasmtime")
)

// ImportsFunc returns a linker with the module imports
type ImportsFunc func(*wasmtime.Store, *wasmtime.Memory) (*wasmtime.Linker, error)

// Config represents a wasmer configuration
type Config struct {
	gssmrruntime.InstanceConfig
	Imports ImportsFunc
}

// Instance represents a v0.8 runtime go-wasmtime instance
type Instance struct {
	vm  *wasmtime.Instance
	mu  sync.Mutex
	mem *wasmtime.Memory
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg *Config) (*Instance, error) {
	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModuleFromFile(engine, fp)
	if err != nil {
		return nil, err
	}

	return newInstanceFromModule(module, engine, cfg)
}

// NewInstance instantiates a runtime from the given wasm bytecode
func NewInstance(code []byte, cfg *Config) (*Instance, error) {
	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModule(engine, code)
	if err != nil {
		return nil, err
	}

	return newInstanceFromModule(module, engine, cfg)
}

func newInstanceFromModule(module *wasmtime.Module, engine *wasmtime.Engine, cfg *Config) (*Instance, error) {
	// if cfg.LogLvl set to < 0, then don't change package log level
	if cfg.LogLvl >= 0 {
		h := log.StreamHandler(os.Stdout, log.TerminalFormat())
		h = log.CallerFileHandler(h)
		logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	}
	store := wasmtime.NewStore(engine)

	lim := wasmtime.Limits{
		Min: 20,
		Max: wasmtime.LimitsMaxNone,
	}
	mem := wasmtime.NewMemory(store, wasmtime.NewMemoryType(lim))

	linker, err := cfg.Imports(store, mem)
	if err != nil {
		return nil, err
	}

	instance, err := linker.Instantiate(module)
	if err != nil {
		return nil, err
	}

	// TODO: use __heap_base
	allocator := gssmrruntime.NewAllocator(Memory{mem}, 0)

	ctx = gssmrruntime.Context{
		Storage:     cfg.Storage,
		Allocator:   allocator,
		Keystore:    cfg.Keystore,
		Validator:   cfg.Role == byte(4),
		NodeStorage: cfg.NodeStorage,
		Network:     cfg.Network,
	}

	return &Instance{
		vm:  instance,
		mem: mem,
	}, nil
}

// UpdateRuntimeCode ...
func (in *Instance) UpdateRuntimeCode(_ []byte) error {
	return errors.New("unimplemented")
}

// SetContextStorage sets the runtime context's Storage
func (in *Instance) SetContextStorage(s gssmrruntime.Storage) {
	ctx.Storage = s
}

// Stop ...
func (in *Instance) Stop() {}

// NodeStorage returns the context's NodeStorage
func (in *Instance) NodeStorage() gssmrruntime.NodeStorage {
	return ctx.NodeStorage
}

// NetworkService returns the context's NetworkService
func (in *Instance) NetworkService() gssmrruntime.BasicNetwork {
	return ctx.Network
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) ([]byte, error) {
	return in.exec(function, data)
}

func (in *Instance) exec(function string, data []byte) ([]byte, error) {
	in.mu.Lock()
	defer in.mu.Unlock()

	ptr, err := ctx.Allocator.Allocate(uint32(len(data)))
	if err != nil {
		return nil, err
	}
	defer ctx.Allocator.Clear()

	memdata := in.mem.UnsafeData()
	copy(memdata[ptr:ptr+uint32(len(data))], data)

	run := in.vm.GetExport(function).Func()
	resi, err := run.Call(int32(ptr), int32(len(data)))
	if err != nil {
		return nil, err
	}

	if resi == nil {
		return []byte{}, err
	}

	ret := resi.(int64)
	length := int32(ret >> 32)
	offset := int32(ret)

	runtime.KeepAlive(in.mem)
	return memdata[offset : offset+length], nil
}
