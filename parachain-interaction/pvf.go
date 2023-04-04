package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/klauspost/compress/zstd"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

var (
	ErrCodeEmpty              = errors.New("code is empty")
	ErrWASMDecompress         = errors.New("wasm decompression failed")
	ErrExportFunctionNotFound = errors.New("export function not found")
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

	// Provide importable memory for newer runtimes
	// TODO: determine memory descriptor size that the runtime wants from the wasm.
	// should be doable w/ wasmer 1.0.0. (#1268)
	memory, err := wasm.NewMemory(23, 0)
	if err != nil {
		return instance, nil, fmt.Errorf("creating web assembly memory: %w", err)
	}

	// TODO: Do I need any imports here? Not sure!

	// Instantiates the WebAssembly module.
	instance, err = wasm.NewInstance(code)
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

	allocator = runtime.NewAllocator(instance.Memory, heapBase)

	return instance, allocator, nil
}

type Instance struct {
	vm wasm.Instance
	// ctx      *runtime.Context
	Allocator *runtime.FreeingBumpHeapAllocator
	// codeHash common.Hash
	mutex sync.Mutex
}

// Exec calls the given function with the given data
func (in *Instance) Exec(function string, data []byte) (result []byte, err error) {
	in.mutex.Lock()
	defer in.mutex.Unlock()

	dataLength := uint32(len(data))
	inputPtr, err := in.Allocator.Allocate(dataLength)
	if err != nil {
		return nil, fmt.Errorf("allocating input memory: %w", err)
	}

	defer in.Allocator.Clear()

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
