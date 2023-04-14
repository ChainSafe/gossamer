package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/klauspost/compress/zstd"
	"github.com/wasmerio/wasmer-go/wasmer"
)

var (
	ErrCodeEmpty         = errors.New("code is empty")
	ErrWASMDecompress    = errors.New("wasm decompression failed")
	ErrInstanceIsStopped = errors.New("instance is stopped")

	ErrExportFunctionNotFound = errors.New("export function not found")
	errMemoryValueOutOfBounds = errors.New("memory value is out of bounds")
)

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

// Memory is a thin wrapper around Wasmer memory to support
// Gossamer runtime.Memory interface
type Memory struct {
	memory *wasmer.Memory
}

func setupVM(code []byte) (*wasmer.Instance, error) {
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
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compile the module
	module, err := wasmer.NewModule(store, code)
	if err != nil {
		return nil, err
	}

	// Get memory descriptor from module, if it imports memory
	// moduleImports := module.Imports()
	// var memImport *wasmer.ImportType
	// for _, im := range moduleImports {
	// 	if im.Name() == "memory" {
	// 		memImport = im
	// 		break
	// 	}
	// }

	// var memoryType *wasmer.MemoryType
	// if memImport != nil {
	// 	memoryType = memImport.Type().IntoMemoryType()
	// }

	// // Check if module exports memory
	// hasExportedMemory := false
	// moduleExports := module.Exports()
	// for _, export := range moduleExports {
	// 	if export.Name() == "memory" {
	// 		hasExportedMemory = true
	// 		break
	// 	}
	// }

	// var memory *wasmer.Memory
	// // create memory to import, if it's expecting imported memory
	// if !hasExportedMemory {
	// 	if memoryType == nil {
	// 		// values from newer kusama/polkadot runtimes
	// 		lim, err := wasmer.NewLimits(23, 4294967295)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		memoryType = wasmer.NewMemoryType(lim)
	// 	}

	// 	memory = wasmer.NewMemory(store, memoryType)
	// }

	// runtimeCtx := &runtime.Context{
	// 	Storage:         cfg.Storage,
	// 	Keystore:        cfg.Keystore,
	// 	Validator:       cfg.Role == common.AuthorityRole,
	// 	NodeStorage:     cfg.NodeStorage,
	// 	Network:         cfg.Network,
	// 	Transaction:     cfg.Transaction,
	// 	OffchainHTTPSet: offchain.NewHTTPSet(),
	// }

	// imports := runtimewasmer.ImportsNodeRuntime(store, memory, runtimeCtx)
	// if err != nil {
	// 	return nil, fmt.Errorf("creating node runtime imports: %w", err)
	// }

	// wasmInstance, err := wasmer.NewInstance(module, imports)
	// if err != nil {
	// 	return nil, err
	// }

	wasmInstance, err := wasmer.NewInstance(module, wasmer.NewImportObject())
	if err != nil {
		return nil, err
	}
	// if hasExportedMemory {
	// 	memory, err = wasmInstance.Exports.GetMemory("memory")
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// runtimeCtx.Memory = Memory{memory}

	// set heap base for allocator, start allocating at heap base
	// heapBase, err := wasmInstance.Exports.Get("__heap_base")
	// if err != nil {
	// 	return nil, err
	// }

	// hb, err := heapBase.IntoGlobal().Get()
	// if err != nil {
	// 	return nil, err
	// }

	// runtimeCtx.Allocator = runtime.NewAllocator(runtimeCtx.Memory, uint32(hb.(int32)))
	// instance := &Instance{
	// 	vm: wasmInstance,
	// 	// ctx:      runtimeCtx,
	// 	// codeHash: cfg.CodeHash,
	// }

	return wasmInstance, nil
}

type Instance struct {
	vm       *wasmer.Instance
	ctx      *runtime.Context
	isClosed bool
	codeHash common.Hash
	mutex    sync.Mutex
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
		return nil, fmt.Errorf("running runtime function: %w", err)
	}

	wasmValueAsI64 := wasmer.NewI64(wasmValue)
	outputPtr, outputLength := splitPointerSize(wasmValueAsI64.I64())
	memory = in.ctx.Memory.Data() // call Data() again to get larger slice
	return memory[outputPtr : outputPtr+outputLength], nil
}

func safeCastInt32(value uint32) (int32, error) {
	if value > math.MaxInt32 {
		return 0, fmt.Errorf("%w", errMemoryValueOutOfBounds)
	}
	return int32(value), nil
}

// splitPointerSize converts an int64 pointer size to an
// uint32 pointer and an uint32 size.
func splitPointerSize(pointerSize int64) (ptr, size uint32) {
	return uint32(pointerSize), uint32(pointerSize >> 32)
}

func (in *Instance) ValidateBlock(params ValidationParameters) (
	*ValidationResult, error) {

	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(params)
	if err != nil {
		return nil, fmt.Errorf("encoding validation parameters: %w", err)
	}

	encodedValidationResult, err := in.Exec("validate_block", buffer.Bytes())
	if err != nil {
		return nil, err
	}

	validationResult := ValidationResult{}
	err = scale.Unmarshal(encodedValidationResult, &validationResult)
	if err != nil {
		return nil, fmt.Errorf("scale decoding: %w", err)
	}
	return &validationResult, nil
}
