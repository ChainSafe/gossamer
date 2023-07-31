// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/offchain"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ChainSafe/gossamer/lib/crypto"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"

	"github.com/klauspost/compress/zstd"
)

// Name represents the name of the interpreter
const Name = "wasmer"

var (
	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "go-wasmer"),
	)
)

// Instance represents a v0.8 runtime go-wasmer instance
type Instance struct {
	vm       wasm.Instance
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
	bytes, err := wasm.ReadBytes(fp)
	if err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg Config) (instance *Instance, err error) {
	logger.Patch(log.SetLevel(cfg.LogLvl), log.SetCallerFunc(true))

	wasmInstance, allocator, err := setupVM(code)
	if err != nil {
		return nil, fmt.Errorf("setting up VM: %w", err)
	}

	runtimeCtx := &runtime.Context{
		Storage:         cfg.Storage,
		Allocator:       allocator,
		Keystore:        cfg.Keystore,
		Validator:       cfg.Role == common.AuthorityRole,
		NodeStorage:     cfg.NodeStorage,
		Network:         cfg.Network,
		Transaction:     cfg.Transaction,
		SigVerifier:     crypto.NewSignatureVerifier(logger),
		OffchainHTTPSet: offchain.NewHTTPSet(),
	}
	wasmInstance.SetContextData(runtimeCtx)

	instance = &Instance{
		vm:       wasmInstance,
		ctx:      runtimeCtx,
		codeHash: cfg.CodeHash,
	}

	if cfg.testVersion != nil {
		instance.ctx.Version = cfg.testVersion
	}

	wasmInstance.SetContextData(instance.ctx)

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

	version, err = instance.Version()
	if err != nil {
		return version, fmt.Errorf("running runtime: %w", err)
	}

	return version, nil
}

var (
	ErrCodeEmpty      = errors.New("code is empty")
	ErrWASMDecompress = errors.New("wasm decompression failed")
)

func setupVM(code []byte) (instance wasm.Instance,
	allocator *runtime.FreeingBumpHeapAllocator, err error) {
	if len(code) == 0 {
		return instance, nil, ErrCodeEmpty
	}

	code, err = decompressWasm(code)
	if err != nil {
		// Note the sentinel error is wrapped here since the ztsd Go library
		// does not return any exported sentinel errors.
		return instance, nil, fmt.Errorf("%w: %s", ErrWASMDecompress, err)
	}

	imports, err := importsNodeRuntime()
	if err != nil {
		return instance, nil, fmt.Errorf("creating node runtime imports: %w", err)
	}

	// Provide importable memory for newer runtimes
	// TODO: determine memory descriptor size that the runtime wants from the wasm.
	// should be doable w/ wasmer 1.0.0. (#1268)
	memory, err := wasm.NewMemory(23, 0)
	if err != nil {
		return instance, nil, fmt.Errorf("creating web assembly memory: %w", err)
	}

	_, err = imports.AppendMemory("memory", memory)
	if err != nil {
		return instance, nil, fmt.Errorf("appending memory to imports: %w", err)
	}

	// Instantiates the WebAssembly module.
	instance, err = wasm.NewInstanceWithImports(code, imports)
	if err != nil {
		return instance, nil, fmt.Errorf("creating web assembly instance: %w", err)
	}

	// Assume imported memory is used if runtime does not export any
	if !instance.HasMemory() {
		instance.Memory = memory
	}

	// TODO: get __heap_base exported value from runtime.
	// wasmer 0.3.x does not support this, but wasmer 1.0.0 does (#1268)
	heapBase := runtime.DefaultHeapBase

	allocator = runtime.NewAllocator(&memoryShim{instance.Memory}, heapBase)

	return instance, allocator, nil
}

type memoryShim struct {
	*wasm.Memory
}

func (ms *memoryShim) Size() uint32 { return ms.Memory.Length() }
func (ms *memoryShim) Grow(deltaPages uint32) (previousPages uint32, ok bool) {
	err := ms.Memory.Grow(deltaPages)
	if err != nil {
		return 0, false
	}
	return 0, true
}
func (ms *memoryShim) ReadByte(offset uint32) (byte, bool) { //nolint:govet
	if offset >= ms.Memory.Length() {
		return 0, false
	}
	return ms.Memory.Data()[offset], true
}
func (ms *memoryShim) Read(offset, byteCount uint32) ([]byte, bool) {
	if offset+byteCount >= ms.Memory.Length() {
		return nil, false
	}
	return ms.Memory.Data()[offset : offset+byteCount], true
}
func (ms *memoryShim) WriteByte(offset uint32, v byte) bool { //nolint:govet
	if offset >= ms.Memory.Length() {
		return false
	}
	ms.Memory.Data()[offset] = v
	return true
}
func (ms *memoryShim) Write(offset uint32, v []byte) bool {
	if offset+uint32(len(v)) >= ms.Memory.Length() {
		return false
	}
	copy(ms.Data()[offset:offset+uint32(len(v))], v)
	return true
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

var (
	ErrInstanceIsStopped      = errors.New("instance is stopped")
	ErrExportFunctionNotFound = errors.New("export function not found")
)

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
	memory := in.vm.Memory.Data()
	copy(memory[inputPtr:inputPtr+dataLength], data)

	runtimeFunc, ok := in.vm.Exports[function]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrExportFunctionNotFound, function)
	}

	wasmValue, err := runtimeFunc(int32(inputPtr), int32(dataLength))
	if err != nil {
		return nil, fmt.Errorf("running runtime function: %w", err)
	}

	outputPtr, outputLength := splitPointerSize(wasmValue.ToI64())
	memory = in.vm.Memory.Data() // call Data() again to get larger slice
	return memory[outputPtr : outputPtr+outputLength], nil
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
