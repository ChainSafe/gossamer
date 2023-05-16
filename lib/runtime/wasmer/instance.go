// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/offchain"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/wasmergo"
	"github.com/klauspost/compress/zstd"
)

// Name represents the name of the interpreter
const Name = "wasmer"

var (
	ErrCodeEmpty              = errors.New("code is empty")
	ErrWASMDecompress         = errors.New("wasm decompression failed")
	ErrInstanceIsStopped      = errors.New("instance is stopped")
	ErrExportFunctionNotFound = errors.New("export function not found")

	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "go-wasmer"),
	)
)

// Instance represents a runtime go-wasmer instance
type Instance struct {
	vm       *wasmergo.Instance
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
	fileBytes, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	return NewInstance(fileBytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg Config) (*Instance, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl), log.SetCallerFunc(true))
	if len(code) == 0 {
		return nil, ErrCodeEmpty
	}

	code, err := decompressWasm(code)
	if err != nil {
		// Note the sentinel error is wrapped here since the ztsd Go library
		// does not return any exported sentinel errors.
		return nil, fmt.Errorf("%w: %s", ErrWASMDecompress, err)
	}

	// Create engine and store with default values
	engine := wasmergo.NewEngine()
	store := wasmergo.NewStore(engine)

	// Compile the module
	module, err := wasmergo.NewModule(store, code)
	if err != nil {
		return nil, err
	}

	// Get memory descriptor from module, if it imports memory
	moduleImports := module.Imports()
	var memImport *wasmergo.ImportType
	for _, im := range moduleImports {
		if im.Name() == "memory" {
			memImport = im
			break
		}
	}

	var memoryType *wasmergo.MemoryType
	if memImport != nil {
		memoryType = memImport.Type().IntoMemoryType()
	}

	// Check if module exports memory
	hasExportedMemory := false
	moduleExports := module.Exports()
	for _, export := range moduleExports {
		if export.Name() == "memory" {
			hasExportedMemory = true
			break
		}
	}

	var memory *wasmergo.Memory
	// create memory to import, if it's expecting imported memory
	if !hasExportedMemory {
		if memoryType == nil {
			// values from newer kusama/polkadot runtimes
			lim, err := wasmergo.NewLimits(23, 4294967295)
			if err != nil {
				return nil, err
			}
			memoryType = wasmergo.NewMemoryType(lim)
		}

		memory = wasmergo.NewMemory(store, memoryType)
	}

	runtimeCtx := &runtime.Context{
		Storage:         cfg.Storage,
		Keystore:        cfg.Keystore,
		Validator:       cfg.Role == common.AuthorityRole,
		NodeStorage:     cfg.NodeStorage,
		Network:         cfg.Network,
		Transaction:     cfg.Transaction,
		SigVerifier:     crypto.NewSignatureVerifier(logger),
		OffchainHTTPSet: offchain.NewHTTPSet(),
	}

	imports := importsNodeRuntime(store, memory, runtimeCtx)
	if err != nil {
		return nil, fmt.Errorf("creating node runtime imports: %w", err)
	}
	wasmInstance, err := wasmergo.NewInstance(module, imports)
	if err != nil {
		return nil, err
	}

	if hasExportedMemory {
		memory, err = wasmInstance.Exports.GetMemory("memory")
		if err != nil {
			return nil, err
		}
	}

	runtimeCtx.Memory = &Memory{memory}

	// set heap base for allocator, start allocating at heap base
	heapBase, err := wasmInstance.Exports.Get("__heap_base")
	if err != nil {
		return nil, err
	}

	hb, err := heapBase.IntoGlobal().Get()
	if err != nil {
		return nil, err
	}

	runtimeCtx.Allocator = runtime.NewAllocator(runtimeCtx.Memory, uint32(hb.(int32)))
	instance := &Instance{
		vm:       wasmInstance,
		ctx:      runtimeCtx,
		codeHash: cfg.CodeHash,
	}

	if cfg.testVersion != nil {
		instance.ctx.Version = cfg.testVersion
	}

	return instance, nil
}

// decompressWasm decompresses a Wasm blob that may or may not be compressed with zstd
// ref: https://github.com/paritytech/substrate/blob/master/primitives/maybe-compressed-blob/src/lib.rs
func decompressWasm(code []byte) ([]byte, error) {
	compressionFlag := []byte{82, 188, 83, 118, 70, 219, 142, 5}
	if !bytes.HasPrefix(code, compressionFlag) {
		return code, nil
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("creating zstd reader: %s", err)
	}

	return decoder.DecodeAll(code[len(compressionFlag):], nil)
}

// GetCodeHash returns the code of the instance
func (in *Instance) GetCodeHash() common.Hash {
	return in.codeHash
}

// GetContext returns the context of the instance
func (in *Instance) GetContext() *runtime.Context {
	return in.ctx
}

// GetRuntimeVersion finds the runtime version by initiating a temporary
// runtime instance using the WASM code provided, and querying it.
func GetRuntimeVersion(code []byte) (version runtime.Version, err error) {
	config := Config{
		LogLvl: log.DoNotChange,
	}
	instance, err := NewInstance(code, config)
	if err != nil {
		return version, fmt.Errorf("creating runtime instance: %w", err)
	}
	defer instance.Stop()

	logger.Info("instantiated runtime!!!")

	version, err = instance.Version()
	if err != nil {
		return version, fmt.Errorf("running runtime: %w", err)
	}

	return version, nil
}

// Exec calls the given function with the given data
func (in *Instance) ExecWithoutReturning(function string, data []byte) (err error) {
	in.mutex.Lock()
	defer in.mutex.Unlock()

	if in.isClosed {
		return ErrInstanceIsStopped
	}

	dataLength := uint32(len(data))
	inputPtr, err := in.ctx.Allocator.Allocate(dataLength)
	if err != nil {
		return fmt.Errorf("allocating input memory: %w", err)
	}

	defer in.ctx.Allocator.Clear()

	// Store the data into memory
	memory := in.ctx.Memory.Data()
	copy(memory[inputPtr:inputPtr+dataLength], data)

	runtimeFunc, err := in.vm.Exports.GetFunction(function)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrExportFunctionNotFound, function)
	}

	castedInputPointer, err := safeCastInt32(inputPtr)
	if err != nil {
		panic(err)
	}

	castedDataLength, err := safeCastInt32(dataLength)
	if err != nil {
		panic(err)
	}

	_, err = runtimeFunc(castedInputPointer, castedDataLength)
	if err != nil {
		if errors.Is(err, errMemoryValueOutOfBounds) {
			panic(fmt.Errorf("executing runtime function: %v", err))
		}
		return fmt.Errorf("running runtime function: %w", err)
	}

	return nil
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) (result []byte, err error) {
	in.mutex.Lock()
	defer in.mutex.Unlock()

	if in.isClosed {
		return nil, ErrInstanceIsStopped
	}

	dataLength := uint32(len(data))
	inputPtr, err := in.ctx.Allocator.Allocate(dataLength)
	if err != nil {
		return nil, fmt.Errorf("allocating input memory: %w", err)
	}

	defer in.ctx.Allocator.Clear()

	// Store the data into memory
	memory := in.ctx.Memory.Data()
	copy(memory[inputPtr:inputPtr+dataLength], data)

	runtimeFunc, err := in.vm.Exports.GetFunction(function)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExportFunctionNotFound, function)
	}

	castedInputPointer, err := safeCastInt32(inputPtr)
	if err != nil {
		panic(err)
	}

	castedDataLength, err := safeCastInt32(dataLength)
	if err != nil {
		panic(err)
	}

	wasmValue, err := runtimeFunc(castedInputPointer, castedDataLength)
	if err != nil {
		if errors.Is(err, errMemoryValueOutOfBounds) {
			panic(fmt.Errorf("executing runtime function: %v", err))
		}
		return nil, fmt.Errorf("running runtime function: %w", err)
	}

	wasmValueAsI64 := wasmergo.NewI64(wasmValue)
	outputPtr, outputLength := splitPointerSize(wasmValueAsI64.I64())
	memory = in.ctx.Memory.Data() // call Data() again to get larger slice

	allocatedData := make([]byte, outputLength)
	copy(allocatedData[:], memory[outputPtr:outputPtr+outputLength])
	return allocatedData, nil
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

// SetContextStorage sets the runtime's storage.
func (in *Instance) SetContextStorage(s runtime.Storage) {
	in.mutex.Lock()
	defer in.mutex.Unlock()
	in.ctx.Storage = s
}

// Stop closes the WASM instance, its imports and clears
// the context allocator in a thread-safe way.
func (in *Instance) Stop() {
	in.mutex.Lock()
	defer in.mutex.Unlock()
	in.close()
}

// close closes the wasm instance (and its imports)
// and clears the context allocator. If the instance
// has previously been closed, it simply returns.
// It is NOT THREAD SAFE to use.
func (in *Instance) close() {
	if in.isClosed {
		return
	}

	in.vm.Close()
	in.ctx.Allocator.Clear()
	in.isClosed = true
}
