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

// Check that runtime interfaces are satisfied
var (
	_ runtime.Instance = (*Instance)(nil)
	_ runtime.Memory   = (*wasm.Memory)(nil)

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
	sync.Mutex
}

// NewRuntimeFromGenesis creates a runtime instance from the genesis data
func NewRuntimeFromGenesis(cfg runtime.InstanceConfig) (instance runtime.Instance, err error) {
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
func NewInstanceFromTrie(t *trie.Trie, cfg runtime.InstanceConfig) (*Instance, error) {
	code := t.Get(common.CodeKey)
	if len(code) == 0 {
		return nil, fmt.Errorf("cannot find :code in trie")
	}

	return NewInstance(code, cfg)
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg runtime.InstanceConfig) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := wasm.ReadBytes(fp)
	if err != nil {
		return nil, err
	}

	return NewInstance(bytes, cfg)
}

// NewInstance instantiates a runtime from raw wasm bytecode
func NewInstance(code []byte, cfg runtime.InstanceConfig) (*Instance, error) {
	if len(code) == 0 {
		return nil, errors.New("code is empty")
	}

	var err error
	code, err = decompressWasm(code)
	if err != nil {
		return nil, fmt.Errorf("cannot decompress WASM code: %w", err)
	}

	logger.Patch(log.SetLevel(cfg.LogLvl), log.SetCallerFunc(true))

	imports, err := importsNodeRuntime()
	if err != nil {
		return nil, fmt.Errorf("creating node runtime imports: %w", err)
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
		Storage:         cfg.Storage,
		Allocator:       allocator,
		Keystore:        cfg.Keystore,
		Validator:       cfg.Role == byte(4),
		NodeStorage:     cfg.NodeStorage,
		Network:         cfg.Network,
		Transaction:     cfg.Transaction,
		SigVerifier:     crypto.NewSignatureVerifier(logger),
		OffchainHTTPSet: offchain.NewHTTPSet(),
	}

	logger.Debugf("NewInstance called with runtimeCtx: %v", runtimeCtx)
	instance.SetContextData(runtimeCtx)

	inst := &Instance{
		vm:       instance,
		ctx:      runtimeCtx,
		codeHash: cfg.CodeHash,
	}

	return inst, nil
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
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
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

// UpdateRuntimeCode updates the runtime instance to run the given code
func (in *Instance) UpdateRuntimeCode(code []byte) error {
	in.Stop()

	err := in.setupInstanceVM(code)
	if err != nil {
		return err
	}

	return nil
}

// CheckRuntimeVersion calculates runtime Version for runtime blob passed in
func (in *Instance) CheckRuntimeVersion(code []byte) (runtime.Version, error) {
	tmp := &Instance{
		ctx: in.ctx,
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
	imports, err := importsNodeRuntime()
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

// Load load
func (in *Instance) load(location, length int32) []byte {
	mem := in.vm.Memory.Data()
	return mem[location : location+length]
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) ([]byte, error) {
	in.Lock()
	defer in.Unlock()

	if in.isClosed {
		return nil, errors.New("instance is stopped")
	}

	dataLength := uint32(len(data))
	inputPtr, err := in.ctx.Allocator.Allocate(dataLength)
	if err != nil {
		return nil, err
	}

	defer in.ctx.Allocator.Clear()

	// Store the data into memory
	memory := in.vm.Memory.Data()
	copy(memory[inputPtr:inputPtr+dataLength], data)

	runtimeFunc, ok := in.vm.Exports[function]
	if !ok {
		return nil, fmt.Errorf("could not find exported function %s", function)
	}

	wasmValue, err := runtimeFunc(int32(inputPtr), int32(dataLength))
	if err != nil {
		return nil, err
	}

	outputPtr, outputLength := runtime.Int64ToPointerAndSize(wasmValue.ToI64())
	return in.load(outputPtr, outputLength), nil
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
