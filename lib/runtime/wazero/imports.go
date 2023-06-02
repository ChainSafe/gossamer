package wazero_runtime

import (
	"context"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tetratelabs/wazero/api"
)

var (
	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "wazero"),
	)
)

// splitPointerSize converts an int64 pointer size to an
// uint32 pointer and an uint32 size.
func splitPointerSize(pointerSize int64) (ptr, size uint32) {
	return uint32(pointerSize), uint32(pointerSize >> 32)
}

// asMemorySlice converts a 64 bit pointer size to a Go byte slice.
func asMemorySlice(m api.Module, pointerSize int64) (data []byte) {
	// memory := context.Memory().Data()
	ptr, size := splitPointerSize(pointerSize)
	data, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("wtf?")
	}
	return data
}

// toWasmMemorySized copies a Go byte slice to wasm memory and returns the corresponding
// 32 bit pointer. Note the data must have a well known fixed length in the runtime.
func toWasmMemorySized(m api.Module, allocator *runtime.FreeingBumpHeapAllocator, data []byte) (pointer uint32, err error) {
	size := uint32(len(data))
	pointer, err = allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	ok := m.Memory().Write(pointer, data)
	if !ok {
		return 0, fmt.Errorf("out of range")
	}
	return pointer, nil
}

//export ext_logging_log_version_1
func ext_logging_log_version_1(ctx context.Context, m api.Module, level int32, targetData, msgData int64) {
	logger.Trace("executing...")
	// instanceContext := wasm.IntoInstanceContext(context)

	// target := string(asMemorySlice(instanceContext, targetData))
	// msg := string(asMemorySlice(instanceContext, msgData))
	target := string(asMemorySlice(m, targetData))
	msg := string(asMemorySlice(m, msgData))

	switch int(level) {
	case 0:
		logger.Critical("target=" + target + " message=" + msg)
	case 1:
		logger.Warn("target=" + target + " message=" + msg)
	case 2:
		logger.Info("target=" + target + " message=" + msg)
	case 3:
		logger.Debug("target=" + target + " message=" + msg)
	case 4:
		logger.Trace("target=" + target + " message=" + msg)
	default:
		logger.Errorf("level=%d target=%s message=%s", int(level), target, msg)
	}
}

//export ext_crypto_ed25519_generate_version_1
func ext_crypto_ed25519_generate_version_1(ctx context.Context, m api.Module, keyTypeID int32, seedSpan int64) uint32 {
	logger.Trace("executing...")

	// instanceContext := wasm.IntoInstanceContext(context)
	// runtimeCtx := instanceContext.Data().(*runtime.Context)
	// memory := instanceContext.Memory().Data()

	// id := memory[keyTypeID : keyTypeID+4]
	id, ok := m.Memory().Read(uint32(keyTypeID), 4)
	if !ok {
		panic("wtf?")
	}
	seedBytes := asMemorySlice(m, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return 0
	}

	var kp *ed25519.Keypair

	if seed != nil {
		kp, err = ed25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = ed25519.GenerateKeypair()
	}

	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return 0
	}

	rtCtx := ctx.Value("context").(*runtime.Context)
	if rtCtx == nil {
		panic("wtf?")
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return 0
	}

	err = ks.Insert(kp)
	if err != nil {
		logger.Warnf("failed to insert key: %s", err)
		return 0
	}

	ret, err := toWasmMemorySized(m, rtCtx.Allocator, kp.Public().Encode())
	if err != nil {
		logger.Warnf("failed to allocate memory: %s", err)
		return 0
	}

	logger.Debug("generated ed25519 keypair with public key: " + kp.Public().Hex())
	return ret
}

func ext_allocator_free_version_1(ctx context.Context, m api.Module, addr int32) {
	// fmt.Println("in here!! ext_allocator_free_version_1")
	allocator := ctx.Value("allocator").(*runtime.FreeingBumpHeapAllocator)

	// Deallocate memory
	err := allocator.Deallocate(uint32(addr))
	if err != nil {
		panic(err)
	}
}

func ext_allocator_malloc_version_1(ctx context.Context, m api.Module, a int32) int32 {
	// fmt.Println("in here!! ext_allocator_malloc_version_1")
	size := a

	allocator := ctx.Value("allocator").(*runtime.FreeingBumpHeapAllocator)

	// Allocate memory
	res, err := allocator.Allocate(uint32(size))
	if err != nil {
		panic(err)
	}

	return int32(res)
}
