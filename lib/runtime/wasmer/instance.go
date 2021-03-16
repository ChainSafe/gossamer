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

	cfg.Imports = ImportsNodeRuntime
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

	// modImports := module.Imports()
	// var memImport *wasm.ImportType
	// for _, im := range modImports {
	// 	fmt.Println(im.Name())
	// 	if im.Name() == "memory" {
	// 		memImport = im
	// 		break
	// 	}
	// }

	// if memImport == nil {
	// 	panic("memImport is nil")
	// }
	// memType := memImport.Type().IntoMemoryType()

	// TODO: determine memory descriptor size that the runtime wants from the wasm.
	// should be doable w/ wasmer 1.0.0.
	lim, err := wasm.NewLimits(23, 256) // TODO: determine maximum
	if err != nil {
		return nil, err
	}
	memType := wasm.NewMemoryType(lim)
	memory := wasm.NewMemory(store, memType)
	mem := &Memory{memory}

	ctx := &runtime.Context{
		Storage:     cfg.Storage,
		Keystore:    cfg.Keystore,
		Validator:   cfg.Role == byte(4),
		NodeStorage: cfg.NodeStorage,
		Network:     cfg.Network,
		Transaction: cfg.Transaction,
		SigVerifier: runtime.NewSignatureVerifier(),
		Memory:      mem,
	}

	imports := cfg.Imports(store, memory, ctx)

	instance, err := wasm.NewInstance(module, imports)
	if err != nil {
		return nil, err
	}

	heapBase, err := instance.Exports.Get("__heap_base")
	if err != nil {
		return nil, err
	}

	hb, err := heapBase.IntoGlobal().Get()
	if err != nil {
		return nil, err
	}

	ctx.Allocator = runtime.NewAllocator(*mem, uint32(hb.(int32)))

	logger.Debug("NewInstance", "ctx", ctx)
	//instance.SetContextData(ctx)

	inst := &Instance{
		vm:      instance,
		ctx:     ctx,
		imports: cfg.Imports,
	}

	inst.version, _ = inst.Version()
	return inst, nil
}

// UpdateRuntimeCode updates the runtime instance to run the given code
func (in *Instance) UpdateRuntimeCode(code []byte) error {
	in.Stop()

	// TODO: can we re-use engine and store?
	engine := wasm.NewEngine()
	store := wasm.NewStore(engine)

	lim, err := wasm.NewLimits(23, 256)
	if err != nil {
		return err
	}
	memType := wasm.NewMemoryType(lim)
	memory := wasm.NewMemory(store, memType)

	imports := in.imports(store, memory, in.ctx)

	module, err := wasm.NewModule(store, code)
	if err != nil {
		return err
	}

	instance, err := wasm.NewInstance(module, imports)
	if err != nil {
		return err
	}

	// TODO: get __heap_base exported value from runtime.
	// wasmer 0.3.x does not support this, but wasmer 1.0.0 does
	heapBase := runtime.DefaultHeapBase

	in.ctx.Allocator = runtime.NewAllocator(Memory{memory}, heapBase)
	//instance.SetContextData(in.ctx)

	inst := &Instance{
		vm:      instance,
		ctx:     in.ctx,
		imports: in.imports,
	}

	inst.version, _ = inst.Version()
	return nil
}

// SetContextStorage sets the runtime's storage. It should be set before calls to the below functions.
func (in *Instance) SetContextStorage(s runtime.Storage) {
	in.ctx.Storage = s
	//in.vm.SetContextData(in.ctx)
}

// Stop func
func (in *Instance) Stop() {}

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

	logger.Info("instance.exec", "ptr", ptr, "datalen", datalen)

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
func int64ToPointerAndSize(in int64) (ptr int32, length int32) {
	return int32(in), int32(in >> 32)
}

// pointerAndSizeToInt64 converts int32 pointer and size to a int64
func pointerAndSizeToInt64(ptr, size int32) int64 {
	return int64(ptr) | (int64(size) << 32)
}
