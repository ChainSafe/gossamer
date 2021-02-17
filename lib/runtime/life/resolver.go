package life

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/perlin-network/life/exec"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// Resolver resolves the imports for life
type Resolver struct{} // TODO: move context inside resolver

// ResolveFunc ...
func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "env":
		switch field {
		case "ext_logging_log_version_1":
			return ext_logging_log_version_1
		case "ext_misc_print_utf8_version_1":
			return ext_misc_print_utf8_version_1
		case "ext_misc_print_hex_version_1":
			return ext_misc_print_hex_version_1
		case "ext_allocator_malloc_version_1":
			return ext_allocator_malloc_version_1
		case "ext_allocator_free_version_1":
			return ext_allocator_free_version_1
		case "ext_hashing_blake2_256_version_1":
			return ext_hashing_blake2_256_version_1
		case "ext_hashing_twox_128_version_1":
			return ext_hashing_twox_128_version_1
		case "ext_storage_get_version_1":
			return ext_storage_get_version_1
		case "ext_storage_set_version_1":
			return ext_storage_set_version_1
		case "ext_storage_next_key_version_1":
			return ext_storage_next_key_version_1
		case "ext_hashing_twox_64_version_1":
			return ext_hashing_twox_64_version_1
		case "ext_storage_clear_version_1":
			return ext_storage_clear_version_1
		case "ext_storage_clear_prefix_version_1":
			return ext_storage_clear_prefix_version_1
		case "ext_storage_read_version_1":
			return ext_storage_read_version_1
		case "ext_storage_append_version_1":
			return ext_storage_append_version_1
		case "ext_trie_blake2_256_ordered_root_version_1":
			return ext_trie_blake2_256_ordered_root_version_1
		case "ext_storage_root_version_1":
			return ext_storage_root_version_1
		case "ext_storage_changes_root_version_1":
			return ext_storage_changes_root_version_1
		case "ext_crypto_start_batch_verify_version_1":
			return ext_crypto_start_batch_verify_version_1
		case "ext_crypto_finish_batch_verify_version_1":
			return ext_crypto_finish_batch_verify_version_1
		default:
			panic(fmt.Errorf("unknown import resolved: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

// ResolveGlobal ...
func (r *Resolver) ResolveGlobal(module, field string) int64 {
	panic("we're not resolving global variables for now")
}

func ext_logging_log_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_logging_log_version_1] executing...")
	level := int32(vm.GetCurrentFrame().Locals[0])
	targetData := vm.GetCurrentFrame().Locals[1]
	msgData := vm.GetCurrentFrame().Locals[2]

	target := fmt.Sprintf("%s", asMemorySlice(vm.Memory, targetData))
	msg := fmt.Sprintf("%s", asMemorySlice(vm.Memory, msgData))

	switch int(level) {
	case 0:
		logger.Crit("[ext_logging_log_version_1]", "target", target, "message", msg)
	case 1:
		logger.Warn("[ext_logging_log_version_1]", "target", target, "message", msg)
	case 2:
		logger.Info("[ext_logging_log_version_1]", "target", target, "message", msg)
	case 3:
		logger.Debug("[ext_logging_log_version_1]", "target", target, "message", msg)
	case 4:
		logger.Trace("[ext_logging_log_version_1]", "target", target, "message", msg)
	default:
		logger.Error("[ext_logging_log_version_1]", "level", int(level), "target", target, "message", msg)
	}

	return 0
}

func ext_misc_print_utf8_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_misc_print_utf8_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)
	logger.Debug("[ext_misc_print_utf8_version_1]", "utf8", fmt.Sprintf("%s", data))
	return 0
}

func ext_misc_print_hex_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_misc_print_hex_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)
	logger.Debug("[ext_misc_print_hex_version_1]", "hex", fmt.Sprintf("0x%x", data))
	return 0
}

func ext_allocator_malloc_version_1(vm *exec.VirtualMachine) int64 {
	size := uint32(vm.GetCurrentFrame().Locals[0])
	logger.Trace("[ext_allocator_malloc_version_1] executing...", "size", size)

	// Allocate memory
	res, err := ctx.Allocator.Allocate(size)
	if err != nil {
		logger.Error("[ext_allocator_malloc_version_1]", "error", err)
		panic(err)
	}

	return int64(res)
}

func ext_allocator_free_version_1(vm *exec.VirtualMachine) int64 {
	addr := uint32(vm.GetCurrentFrame().Locals[0])
	logger.Trace("[ext_allocator_free_version_1] executing...", "addr", addr)

	// Deallocate memory
	err := ctx.Allocator.Deallocate(addr)
	if err != nil {
		logger.Error("[ext_allocator_free_version_1]", "error", err)
		panic(err)
	}

	return 0
}

func ext_hashing_blake2_256_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_blake2_256_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]

	data := asMemorySlice(vm.Memory, dataSpan)

	hash, err := common.Blake2bHash(data)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_blake2_256_version_1]", "data", data, "hash", hash)

	out, err := toWasmMemorySized(vm.Memory, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1] failed to allocate", "error", err)
		return 0
	}

	return int64(out)
}

func ext_hashing_twox_128_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_twox_128_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)

	hash, err := common.Twox128Hash(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_twox_128_version_1]", "data", fmt.Sprintf("%s", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(vm.Memory, hash, 16)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1] failed to allocate", "error", err)
		return 0
	}

	return int64(out)
}

func ext_hashing_twox_64_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_twox_64_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)

	hash, err := common.Twox64(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_twox_64_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(vm.Memory, hash, 8)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1] failed to allocate", "error", err)
		return 0
	}

	return int64(out)
}

func ext_storage_get_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_get_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)
	logger.Debug("[ext_storage_get_version_1]", "key", fmt.Sprintf("0x%x", key))

	value, err := storage.Get(key)
	if err != nil {
		logger.Error("[ext_storage_get_version_1]", "error", err)
		ptr, _ := toWasmMemoryOptional(vm.Memory, nil)
		return ptr
	}

	logger.Debug("[ext_storage_get_version_1]", "value", fmt.Sprintf("0x%x", value))

	valueSpan, err := toWasmMemoryOptional(vm.Memory, value)
	if err != nil {
		logger.Error("[ext_storage_get_version_1] failed to allocate", "error", err)
		ptr, _ := toWasmMemoryOptional(vm.Memory, nil)
		return ptr
	}

	return valueSpan
}

func ext_storage_set_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_set_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	valueSpan := vm.GetCurrentFrame().Locals[1]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)
	value := asMemorySlice(vm.Memory, valueSpan)

	if ctx.TransactionStorageChanges != nil {
		ctx.TransactionStorageChanges = append(ctx.TransactionStorageChanges, &runtime.TransactionStorageChange{
			Operation: runtime.SetOp,
			Key:       key,
			Value:     value,
		})
		return 0
	}

	logger.Debug("[ext_storage_set_version_1]", "key", fmt.Sprintf("0x%x", key), "val", fmt.Sprintf("0x%x", value))

	cp := make([]byte, len(value))
	copy(cp, value)
	err := storage.Set(key, cp)
	if err != nil {
		logger.Error("[ext_storage_set_version_1]", "error", err)
		return 0
	}

	return 0
}

func ext_storage_next_key_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_next_key_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)

	next := storage.NextKey(key)
	logger.Debug("[ext_storage_next_key_version_1]", "key", fmt.Sprintf("0x%x", key), "next", fmt.Sprintf("0x%x", next))

	nextSpan, err := toWasmMemoryOptional(vm.Memory, next)
	if err != nil {
		logger.Error("[ext_storage_next_key_version_1] failed to allocate", "error", err)
		return 0
	}

	return nextSpan
}

func ext_storage_clear_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_clear_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)

	logger.Debug("[ext_storage_clear_version_1]", "key", fmt.Sprintf("0x%x", key))

	if ctx.TransactionStorageChanges != nil {
		ctx.TransactionStorageChanges = append(ctx.TransactionStorageChanges, &runtime.TransactionStorageChange{
			Operation: runtime.ClearOp,
			Key:       key,
		})
		return 0
	}

	_ = storage.Delete(key)
	return 0
}

func ext_storage_clear_prefix_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_clear_prefix_version_1] executing...")
	storage := ctx.Storage
	prefixSpan := vm.GetCurrentFrame().Locals[0]

	prefix := asMemorySlice(vm.Memory, prefixSpan)
	logger.Debug("[ext_storage_clear_prefix_version_1]", "prefix", fmt.Sprintf("0x%x", prefix))

	if ctx.TransactionStorageChanges != nil {
		ctx.TransactionStorageChanges = append(ctx.TransactionStorageChanges, &runtime.TransactionStorageChange{
			Operation: runtime.ClearPrefixOp,
			Prefix:    prefix,
		})
		return 0
	}

	err := storage.ClearPrefix(prefix)
	if err != nil {
		logger.Error("[ext_storage_clear_prefix_version_1]", "error", err)
	}

	// sanity check
	next := storage.NextKey(prefix)
	if len(next) >= len(prefix) && bytes.Equal(prefix, next[:len(prefix)]) {
		panic("did not clear prefix")
	}

	return 0
}

func ext_storage_read_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_read_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	valueOut := vm.GetCurrentFrame().Locals[1]
	offset := int32(vm.GetCurrentFrame().Locals[2])
	storage := ctx.Storage
	memory := vm.Memory

	key := asMemorySlice(memory, keySpan)
	value, err := storage.Get(key)
	if err != nil {
		logger.Error("[ext_storage_read_version_1]", "error", err)
		ret, _ := toWasmMemoryOptional(memory, nil)
		return ret
	}

	logger.Debug("[ext_storage_read_version_1]", "key", fmt.Sprintf("0x%x", key), "value", fmt.Sprintf("0x%x", value))

	if value == nil {
		ret, _ := toWasmMemoryOptional(memory, nil)
		return ret
	}

	var size uint32

	if int(offset) > len(value) {
		size = uint32(0)
	} else {
		size = uint32(len(value[offset:]))
		valueBuf, valueLen := int64ToPointerAndSize(valueOut)
		copy(memory[valueBuf:valueBuf+valueLen], value[offset:])
	}

	sizeSpan, err := toWasmMemoryOptionalUint32(memory, &size)
	if err != nil {
		logger.Error("[ext_storage_read_version_1] failed to allocate", "error", err)
		return 0
	}

	return sizeSpan
}

func storageAppend(storage runtime.Storage, key, valueToAppend []byte) error {
	nextLength := big.NewInt(1)
	var valueRes []byte

	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	valueCurr, err := storage.Get(key)
	if err != nil {
		return err
	}

	if len(valueCurr) == 0 {
		valueRes = valueToAppend
	} else {
		// remove length prefix from existing value
		r := &bytes.Buffer{}
		_, _ = r.Write(valueCurr)
		dec := &scale.Decoder{Reader: r}
		currLength, err := dec.DecodeBigInt() //nolint
		if err != nil {
			logger.Trace("[ext_storage_append_version_1] item in storage is not SCALE encoded, overwriting", "key", key)
			return storage.Set(key, append([]byte{4}, valueToAppend...))
		}

		// append new item
		valueRes = append(r.Bytes(), valueToAppend...)

		// increase length by 1
		nextLength = big.NewInt(0).Add(currLength, big.NewInt(1))
	}

	lengthEnc, err := scale.Encode(nextLength)
	if err != nil {
		logger.Trace("[ext_storage_append_version_1] failed to encode new length", "error", err)
	}

	// append new length prefix to start of items array
	finalVal := append(lengthEnc, valueRes...)
	logger.Debug("[ext_storage_append_version_1]", "resulting value", fmt.Sprintf("0x%x", finalVal))
	return storage.Set(key, finalVal)
}

func ext_storage_append_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_append_version_1] executing...")
	storage := ctx.Storage
	keySpan := vm.GetCurrentFrame().Locals[0]
	valueSpan := vm.GetCurrentFrame().Locals[0]

	key := asMemorySlice(vm.Memory, keySpan)
	logger.Debug("[ext_storage_append_version_1]", "key", fmt.Sprintf("0x%x", key))
	valueAppend := asMemorySlice(vm.Memory, valueSpan)

	if ctx.TransactionStorageChanges != nil {
		ctx.TransactionStorageChanges = append(ctx.TransactionStorageChanges, &runtime.TransactionStorageChange{
			Operation: runtime.AppendOp,
			Key:       key,
			Value:     valueAppend,
		})
		return 0
	}

	err := storageAppend(storage, key, valueAppend)
	if err != nil {
		logger.Error("[ext_storage_append_version_1]", "error", err)
	}

	return 0
}

func ext_trie_blake2_256_ordered_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_trie_blake2_256_ordered_root_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory
	data := asMemorySlice(memory, dataSpan)

	t := trie.NewEmptyTrie()
	v, err := scale.Decode(data, [][]byte{})
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return 0
	}

	values := v.([][]byte)

	for i, val := range values {
		key, err := scale.Encode(big.NewInt(int64(i))) //nolint
		if err != nil {
			logger.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
			return 0
		}
		logger.Trace("[ext_trie_blake2_256_ordered_root_version_1]", "key", key, "value", val)

		t.Put(key, val)
	}

	// allocate memory for value and copy value to memory
	ptr, err := ctx.Allocator.Allocate(32)
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return 0
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_trie_blake2_256_ordered_root_version_1]", "root", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return int64(ptr)
}

func ext_storage_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_root_version_1] executing...")
	storage := ctx.Storage

	root, err := storage.Root()
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to get storage root", "error", err)
		return 0
	}

	logger.Debug("[ext_storage_root_version_1]", "root", root)

	rootSpan, err := toWasmMemory(vm.Memory, root[:])
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to allocate", "error", err)
		return 0
	}

	return rootSpan
}

func ext_storage_changes_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_changes_root_version_1] executing...")
	logger.Debug("[ext_storage_changes_root_version_1] returning None")

	rootSpan, err := toWasmMemoryOptional(vm.Memory, nil)
	if err != nil {
		logger.Error("[ext_storage_changes_root_version_1] failed to allocate", "error", err)
		return 0
	}

	return rootSpan
}

func ext_crypto_start_batch_verify_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_start_batch_verify_version_1] executing...")
	return 0
}

func ext_crypto_finish_batch_verify_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_finish_batch_verify_version_1] executing...")
	return 1
}

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(memory []byte, span int64) []byte {
	ptr, size := int64ToPointerAndSize(span)
	return memory[ptr : ptr+size]
}

// Copy a byte slice of a fixed size to wasm memory and return resulting pointer
func toWasmMemorySized(memory, data []byte, size uint32) (uint32, error) {
	if int(size) != len(data) {
		return 0, errors.New("internal byte array size missmatch")
	}

	allocator := ctx.Allocator
	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	copy(memory[out:out+size], data[:])
	return out, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(memory, data []byte) (int64, error) {
	var opt *optional.Bytes
	if data == nil {
		opt = optional.NewBytes(false, nil)
	} else {
		opt = optional.NewBytes(true, data)
	}

	enc, err := opt.Encode()
	if err != nil {
		return 0, err
	}

	return toWasmMemory(memory, enc)
}

// Copy a byte slice to wasm memory and return the resulting 64bit span descriptor
func toWasmMemory(memory, data []byte) (int64, error) {
	allocator := ctx.Allocator
	size := uint32(len(data))

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	copy(memory[out:out+size], data[:])
	return pointerAndSizeToInt64(int32(out), int32(size)), nil
}

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(memory []byte, data *uint32) (int64, error) {
	var opt *optional.Uint32
	if data == nil {
		opt = optional.NewUint32(false, 0)
	} else {
		opt = optional.NewUint32(true, *data)
	}

	enc := opt.Encode()
	return toWasmMemory(memory, enc)
}
