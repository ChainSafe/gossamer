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

package wasmer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
	wasm "github.com/wasmerio/wasmer-go/wasmer"
)

// Name represents the name of the interpreter
const Name = "wasmer"

// Check that runtime interfaces are satisfied
var (
	_ runtime.Instance = (*Instance)(nil)
	_ runtime.Memory   = (*Memory)(nil)

	logger = log.New("pkg", "runtime", "module", "go-wasmer")
)

// ImportsFunc is type that defines a function that should return the runtime imports
type ImportsFunc func(*wasm.Store, *wasm.Memory, *runtime.Context) *wasm.ImportObject

// Config represents a wasmer configuration
type Config struct {
	runtime.InstanceConfig
	Imports ImportsFunc
}

// Instance represents a v0.8 runtime go-wasmer instance
type Instance struct {
	vm      *wasm.Instance
	ctx     *runtime.Context
	mutex   sync.Mutex
	version runtime.Version
	imports ImportsFunc
}

// NewRuntimeFromGenesis creates a runtime instance from the genesis data
func NewRuntimeFromGenesis(g *genesis.Genesis, cfg *Config) (runtime.Instance, error) { // TODO: simplify, get :code from storage
	codeStr := g.GenesisFields().Raw["top"][common.BytesToHex(common.CodeKey)]
	if codeStr == "" {
		return nil, fmt.Errorf("cannot find :code in genesis")
	}

	code := common.MustHexToBytes(codeStr)

	cfg.Imports = ImportsNodeRuntime
	return NewInstance(code, cfg)
}

// NewInstanceFromTrie returns a new runtime instance with the code provided in the given trie
func NewInstanceFromTrie(t *trie.Trie, cfg *Config) (*Instance, error) {
	code := t.Get(common.CodeKey)
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in trie")
	}

	//cfg.Imports = ImportsNodeRuntime
	return NewInstance(code, cfg)
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg *Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg *Config) (*Instance, error) {
	// TODO: verify that v0.8 specific funcs are available
	return newInstance(code, cfg)
}

func newInstance(code []byte, cfg *Config) (*Instance, error) {
	if len(code) == 0 {
		return nil, errors.New("code is empty")
	}

	// if cfg.LogLvl set to < 0, then don't change package log level
	if cfg.LogLvl >= 0 {
		h := log.StreamHandler(os.Stdout, log.TerminalFormat())
		h = log.CallerFileHandler(h)
		logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	}

	engine := wasm.NewEngine()
	store := wasm.NewStore(engine)

	module, err := wasm.NewModule(store, code)
	if err != nil {
		return nil, err
	}

	// get memory descriptor from module, if it imports memory
	modImports := module.Imports()
	var memImport *wasm.ImportType
	for _, im := range modImports {
		if im.Name() == "memory" {
			memImport = im
			break
		}
	}

	var memType *wasm.MemoryType
	if memImport != nil {
		memType = memImport.Type().IntoMemoryType()
	}

	// check if module exports memory
	hasExportedMemory := false
	for _, export := range module.Exports() {
		if export.Name() == "memory" {
			hasExportedMemory = true
			break
		}
	}

	var memory *wasm.Memory
	// create memory to import, if it's expecting imported memory
	if !hasExportedMemory {
		if memType == nil {
			// values from newer kusama/polkadot runtimes
			lim, err := wasm.NewLimits(23, 4294967295) //nolint
			if err != nil {
				return nil, err
			}
			memType = wasm.NewMemoryType(lim)
		}

		memory = wasm.NewMemory(store, memType)
	}

	ctx := &runtime.Context{
		Storage:     cfg.Storage,
		Keystore:    cfg.Keystore,
		Validator:   cfg.Role == byte(4),
		NodeStorage: cfg.NodeStorage,
		Network:     cfg.Network,
		Transaction: cfg.Transaction,
		SigVerifier: runtime.NewSignatureVerifier(),
	}

	imports := cfg.Imports(store, memory, ctx)
	instance, err := wasm.NewInstance(module, imports)
	if err != nil {
		return nil, err
	}

	logger.Info("instantiated runtime!!!")

	if hasExportedMemory {
		memory, err = instance.Exports.GetMemory("memory")
		if err != nil {
			return nil, err
		}
	}

	ctx.Memory = &Memory{memory}

	// set heap base for allocator, start allocating at heap base
	heapBase, err := instance.Exports.Get("__heap_base")
	if err != nil {
		return nil, err
	}

	hb, err := heapBase.IntoGlobal().Get()
	if err != nil {
		return nil, err
	}

	ctx.Allocator = runtime.NewAllocator(ctx.Memory, uint32(hb.(int32)))
	inst := &Instance{
		vm:      instance,
		ctx:     ctx,
		imports: cfg.Imports,
	}

	inst.version, err = inst.Version()
	if err != nil {
		return nil, err
	}

	logger.Info("instantiated runtime", "name", inst.version.SpecName(), "specification version", inst.version.SpecVersion())
	return inst, nil
}

// CheckRuntimeVersion calculates runtime Version for runtime blob passed in
func (in *Instance) CheckRuntimeVersion(code []byte) (runtime.Version, error) {
	// TODO: make sure this works
	cfg := &Config{
		Imports: in.imports,
	}
	cfg.LogLvl = -1
	cfg.Storage = in.ctx.Storage

	tmp, err := newInstance(code, cfg)
	if err != nil {
		return nil, err
	}

	return tmp.Version()
}

// UpdateRuntimeCode updates the runtime instance to run the given code
func (in *Instance) UpdateRuntimeCode(code []byte) error {
	cfg := &Config{
		Imports: in.imports,
	}
	cfg.LogLvl = -1
	cfg.Storage = in.ctx.Storage
	cfg.Keystore = in.ctx.Keystore
	cfg.Role = 1 // TODO: set properly
	cfg.NodeStorage = in.ctx.NodeStorage
	cfg.Network = in.ctx.Network
	cfg.Transaction = in.ctx.Transaction

	next, err := newInstance(code, cfg)
	if err != nil {
		return err
	}

	// in.ctx.Allocator = next.ctx.Allocator
	// in.ctx.Memory = next.ctx.Memory
	in.ctx = next.ctx
	in.vm = next.vm
	in.version = next.version
	logger.Info("updated runtime", "specification version", in.version.SpecVersion())
	return nil
}

// SetContextStorage sets the runtime's storage. It should be set before calls to the below functions.
func (in *Instance) SetContextStorage(s runtime.Storage) {
	in.ctx.Storage = s
}

// Store func
func (in *Instance) store(data []byte, location int32) {
	memory := in.ctx.Memory
	mem := memory.Data()
	copy(mem[location:location+int32(len(data))], data)
}

// Load load
func (in *Instance) load(location, length int32) []byte {
	memory := in.ctx.Memory
	mem := memory.Data()
	return mem[location : location+length]
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) ([]byte, error) {
	return in.exec(function, data)
}

// Exec func
func (in *Instance) exec(function string, data []byte) ([]byte, error) {
	if in.ctx.Storage == nil {
		return nil, runtime.ErrNilStorage
	}

	in.mutex.Lock()
	defer in.mutex.Unlock()

	ptr, err := in.malloc(uint32(len(data)))
	if err != nil {
		return nil, err
	}

	defer in.clear()

	// Store the data into memory
	in.store(data, int32(ptr))
	datalen := int32(len(data))

	runtimeFunc, err := in.vm.Exports.GetFunction(function)
	if err != nil {
		return nil, fmt.Errorf("could not find exported function %s: %w", function, err)
	}

	res, err := runtimeFunc(int32(ptr), datalen)
	if err != nil {
		return nil, err
	}

	offset, length := int64ToPointerAndSize(res.(int64)) // TODO: are all returns int64?
	return in.load(offset, length), nil
}

func (in *Instance) malloc(size uint32) (uint32, error) {
	return in.ctx.Allocator.Allocate(size)
}

func (in *Instance) clear() {
	in.ctx.Allocator.Clear()
}

// NodeStorage to get reference to runtime node service
func (in *Instance) NodeStorage() runtime.NodeStorage {
	return in.ctx.NodeStorage
}

// NetworkService to get referernce to runtime network service
func (in *Instance) NetworkService() runtime.BasicNetwork {
	return in.ctx.Network
}

// int64ToPointerAndSize converts an int64 into a int32 pointer and a int32 length
func int64ToPointerAndSize(in int64) (ptr, length int32) {
	return int32(in), int32(in >> 32)
}

// pointerAndSizeToInt64 converts int32 pointer and size to a int64
func pointerAndSizeToInt64(ptr, size int32) int64 {
	return int64(ptr) | (int64(size) << 32)
}
