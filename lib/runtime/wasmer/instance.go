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
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

// Name represents the name of the interpreter
const Name = "wasmer"

// Check that runtime interfaces are satisfied
var (
	_ runtime.Instance = (*Instance)(nil)
	_ runtime.Memory   = (*wasm.Memory)(nil)

	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "go-wasmer"),
	)
)

// Config represents a wasmer configuration
type Config struct {
	runtime.InstanceConfig
	Imports func() (*wasm.Imports, error)
}

// Instance represents a v0.8 runtime go-wasmer instance
type Instance struct {
	vm       wasm.Instance
	ctx      *runtime.Context
	version  runtime.Version
	imports  func() (*wasm.Imports, error)
	isClosed bool
	codeHash common.Hash
	sync.Mutex
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

	cfg.Imports = ImportsNodeRuntime
	return NewInstance(code, cfg)
}

// NewInstanceFromTrie returns a new runtime instance with the code provided in the given trie
func NewInstanceFromTrie(t *trie.Trie, cfg *Config) (*Instance, error) {
	code := t.Get(common.CodeKey)
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in trie")
	}

	cfg.Imports = ImportsNodeRuntime
	return NewInstance(code, cfg)
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg *Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := wasm.ReadBytes(fp)
	if err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg *Config) (*Instance, error) {
	if len(code) == 0 {
		return nil, errors.New("code is empty")
	}

	logger.PatchLevel(cfg.LogLvl)
	logger.PatchCallerFunc(true)

	imports, err := cfg.Imports()
	if err != nil {
		return nil, err
	}

	// Provide importable memory for newer runtimes
	// TODO: determine memory descriptor size that the runtime wants from the wasm. (#1268)
	// should be doable w/ wasmer 1.0.0.
	memory, err := wasm.NewMemory(23, 0)
	if err != nil {
		return nil, err
	}

	_, err = imports.AppendMemory("memory", memory)
	if err != nil {
		return nil, err
	}

	// Instantiates the WebAssembly module.
	instance, err := wasm.NewInstanceWithImports(code, imports)
	if err != nil {
		return nil, err
	}

	// TODO: get __heap_base exported value from runtime.
	// wasmer 0.3.x does not support this, but wasmer 1.0.0 does (#1268)
	heapBase := runtime.DefaultHeapBase

	// Assume imported memory is used if runtime does not export any
	if !instance.HasMemory() {
		instance.Memory = memory
	}

	allocator := runtime.NewAllocator(instance.Memory, heapBase)

	runtimeCtx := &runtime.Context{
		Storage:     cfg.Storage,
		Allocator:   allocator,
		Keystore:    cfg.Keystore,
		Validator:   cfg.Role == byte(4),
		NodeStorage: cfg.NodeStorage,
		Network:     cfg.Network,
		Transaction: cfg.Transaction,
		SigVerifier: runtime.NewSignatureVerifier(logger),
	}

	logger.Debugf("NewInstance called with runtimeCtx: %v", runtimeCtx)
	instance.SetContextData(runtimeCtx)

	inst := &Instance{
		vm:       instance,
		ctx:      runtimeCtx,
		imports:  cfg.Imports,
		codeHash: cfg.CodeHash,
	}

	inst.version, _ = inst.Version()
	return inst, nil
}

// GetCodeHash returns the code of the instance
func (in *Instance) GetCodeHash() common.Hash {
	return in.codeHash
}

// UpdateRuntimeCode updates the runtime instance to run the given code
func (in *Instance) UpdateRuntimeCode(code []byte) error {
	in.Stop()

	err := in.setupInstanceVM(code)
	if err != nil {
		return err
	}

	in.version = nil
	in.version, err = in.Version()
	if err != nil {
		return err
	}

	return nil
}

// CheckRuntimeVersion calculates runtime Version for runtime blob passed in
func (in *Instance) CheckRuntimeVersion(code []byte) (runtime.Version, error) {
	tmp := &Instance{
		imports: in.imports,
		ctx:     in.ctx,
	}

	in.Lock()
	defer in.Unlock()

	err := tmp.setupInstanceVM(code)
	if err != nil {
		return nil, err
	}

	return tmp.Version()
}

func (in *Instance) setupInstanceVM(code []byte) error {
	imports, err := in.imports()
	if err != nil {
		return err
	}

	// TODO: determine memory descriptor size that the runtime wants from the wasm.
	// should be doable w/ wasmer 1.0.0. (#1268)
	memory, err := wasm.NewMemory(23, 0)
	if err != nil {
		return err
	}

	_, err = imports.AppendMemory("memory", memory)
	if err != nil {
		return err
	}

	// Instantiates the WebAssembly module.
	in.vm, err = wasm.NewInstanceWithImports(code, imports)
	if err != nil {
		return err
	}

	// Assume imported memory is used if runtime does not export any
	if !in.vm.HasMemory() {
		in.vm.Memory = memory
	}

	// TODO: get __heap_base exported value from runtime.
	// wasmer 0.3.x does not support this, but wasmer 1.0.0 does (#1268)
	heapBase := runtime.DefaultHeapBase

	in.ctx.Allocator = runtime.NewAllocator(in.vm.Memory, heapBase)
	in.vm.SetContextData(in.ctx)
	return nil
}

// SetContextStorage sets the runtime's storage. It should be set before calls to the below functions.
func (in *Instance) SetContextStorage(s runtime.Storage) {
	in.Lock()
	defer in.Unlock()
	in.ctx.Storage = s
}

// Stop func
func (in *Instance) Stop() {
	in.Lock()
	defer in.Unlock()
	if !in.isClosed {
		in.vm.Close()
		in.isClosed = true
	}
}

// Store func
func (in *Instance) store(data []byte, location int32) {
	mem := in.vm.Memory.Data()
	copy(mem[location:location+int32(len(data))], data)
}

// Load load
func (in *Instance) load(location, length int32) []byte {
	mem := in.vm.Memory.Data()
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

	in.Lock()
	defer in.Unlock()

	if in.isClosed {
		return nil, errors.New("instance is stopped")
	}

	ptr, err := in.malloc(uint32(len(data)))
	if err != nil {
		return nil, err
	}

	defer in.clear()

	// Store the data into memory
	in.store(data, int32(ptr))
	datalen := int32(len(data))

	runtimeFunc, ok := in.vm.Exports[function]
	if !ok {
		return nil, fmt.Errorf("could not find exported function %s", function)
	}

	res, err := runtimeFunc(int32(ptr), datalen)
	if err != nil {
		return nil, err
	}

	offset, length := runtime.Int64ToPointerAndSize(res.ToI64())
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

// Keystore to get reference to runtime keystore
func (in *Instance) Keystore() *keystore.GlobalKeystore {
	return in.ctx.Keystore
}

// Validator returns the context's Validator
func (in *Instance) Validator() bool {
	return in.ctx.Validator
}
