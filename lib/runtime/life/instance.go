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
	"os"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	log "github.com/ChainSafe/log15"
	"github.com/perlin-network/life/exec"
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

// NewRuntimeFromGenesis creates a runtime instance from the genesis data
func NewRuntimeFromGenesis(g *genesis.Genesis, cfg *Config) (runtime.Instance, error) { // TODO: simplify, get :code from storage
	codeStr := g.GenesisFields().Raw["top"][common.BytesToHex(common.CodeKey)]
	if codeStr == "" {
		return nil, fmt.Errorf("cannot find :code in genesis")
	}

	code := common.MustHexToBytes(codeStr)
	cfg.Resolver = new(Resolver)
	return NewInstance(code, cfg)
}

// NewInstance ...
func NewInstance(code []byte, cfg *Config) (runtime.Instance, error) {
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
		DefaultMemoryPages: 20,
	}

	instance, err := exec.NewVirtualMachine(code, vmCfg, cfg.Resolver, nil)
	if err != nil {
		return nil, err
	}

	memory := &Memory{
		memory: instance.Memory,
	}

	// TODO: use __heap_base
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
	inst.version, _ = inst.Version()
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
func (in *Instance) UpdateRuntimeCode(_ []byte) error {
	return errors.New("unimplemented")
}

// SetContextStorage sets the runtime's storage. It should be set before calls to the below functions.
func (in *Instance) SetContextStorage(s runtime.Storage) {
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
		panic("entry function not found")
	}

	ret, err := in.vm.Run(fnc, int64(ptr), int64(len(data)))
	if err != nil {
		fmt.Println(in.vm.StackTrace)
		return nil, err
	}

	offset, length := int64ToPointerAndSize(ret)
	return in.vm.Memory[offset : offset+length], nil
}

// Stop ...
func (in *Instance) Stop() {}

// NodeStorage to get reference to runtime node service
func (in *Instance) NodeStorage() runtime.NodeStorage {
	return ctx.NodeStorage
}

// NetworkService to get referernce to runtime network service
func (in *Instance) NetworkService() runtime.BasicNetwork {
	return ctx.Network
}

// TODO: move below to lib/runtime

// int64ToPointerAndSize converts an int64 into a int32 pointer and a int32 length
func int64ToPointerAndSize(in int64) (ptr int32, length int32) {
	return int32(in), int32(in >> 32)
}

// pointerAndSizeToInt64 converts int32 pointer and size to a int64
func pointerAndSizeToInt64(ptr, size int32) int64 {
	return int64(ptr) | (int64(size) << 32)
}
