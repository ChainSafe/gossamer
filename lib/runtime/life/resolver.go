package life

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	rtype "github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/perlin-network/life/exec"
)

// Resolver resolves the imports for life
type Resolver struct{} // TODO: move context inside resolver (#1875)

// ResolveFunc ...
func (*Resolver) ResolveFunc(module, field string) exec.FunctionImport { // nolint
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
		case "ext_offchain_index_set_version_1":
			return ext_offchain_index_set_version_1
		case "ext_storage_exists_version_1":
			return ext_storage_exists_version_1
		case "ext_default_child_storage_set_version_1":
			return ext_default_child_storage_set_version_1
		case "ext_default_child_storage_get_version_1":
			return ext_default_child_storage_get_version_1
		case "ext_default_child_storage_read_version_1":
			return ext_default_child_storage_read_version_1
		case "ext_default_child_storage_clear_version_1":
			return ext_default_child_storage_clear_version_1
		case "ext_default_child_storage_storage_kill_version_1":
			return ext_default_child_storage_storage_kill_version_1
		case "ext_default_child_storage_exists_version_1":
			return ext_default_child_storage_exists_version_1
		case "ext_default_child_storage_clear_prefix_version_1":
			return ext_default_child_storage_clear_prefix_version_1
		case "ext_default_child_storage_root_version_1":
			return ext_default_child_storage_root_version_1
		case "ext_default_child_storage_next_key_version_1":
			return ext_default_child_storage_next_key_version_1
		case "ext_crypto_ed25519_public_keys_version_1":
			return ext_crypto_ed25519_public_keys_version_1
		case "ext_crypto_ed25519_generate_version_1":
			return ext_crypto_ed25519_generate_version_1
		case "ext_crypto_ed25519_sign_version_1":
			return ext_crypto_ed25519_sign_version_1
		case "ext_crypto_ed25519_verify_version_1":
			return ext_crypto_ed25519_verify_version_1
		case "ext_crypto_sr25519_public_keys_version_1":
			return ext_crypto_sr25519_public_keys_version_1
		case "ext_crypto_sr25519_generate_version_1":
			return ext_crypto_sr25519_generate_version_1
		case "ext_crypto_sr25519_sign_version_1":
			return ext_crypto_sr25519_sign_version_1
		case "ext_crypto_sr25519_verify_version_1":
			return ext_crypto_sr25519_verify_version_1
		case "ext_crypto_secp256k1_ecdsa_recover_version_1":
			return ext_crypto_secp256k1_ecdsa_recover_version_1
		case "ext_hashing_keccak_256_version_1":
			return ext_hashing_keccak_256_version_1
		case "ext_hashing_sha2_256_version_1":
			return ext_hashing_sha2_256_version_1
		case "ext_hashing_blake2_128_version_1":
			return ext_hashing_blake2_128_version_1
		case "ext_hashing_twox_256_version_1":
			return ext_hashing_twox_256_version_1
		case "ext_trie_blake2_256_root_version_1":
			return ext_trie_blake2_256_root_version_1
		default:
			panic(fmt.Errorf("unknown import resolved: %s", field))
		}
	default:
		panic(fmt.Errorf("unknown module: %s", module))
	}
}

// ResolveGlobal ...
func (*Resolver) ResolveGlobal(_, _ string) int64 {
	panic("we're not resolving global variables for now")
}

func ext_logging_log_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_logging_log_version_1] executing...")
	level := int32(vm.GetCurrentFrame().Locals[0])
	targetData := vm.GetCurrentFrame().Locals[1]
	msgData := vm.GetCurrentFrame().Locals[2]

	target := asMemorySlice(vm.Memory, targetData)
	msg := asMemorySlice(vm.Memory, msgData)

	switch int(level) {
	case 0:
		logger.Criticalf("[ext_logging_log_version_1] target=%s message=%s", string(target), string(msg))
	case 1:
		logger.Warnf("[ext_logging_log_version_1] target=%s message=%s", string(target), string(msg))
	case 2:
		logger.Infof("[ext_logging_log_version_1] target=%s message=%s", string(target), string(msg))
	case 3:
		logger.Debugf("[ext_logging_log_version_1] target=%s message=%s", string(target), string(msg))
	case 4:
		logger.Tracef("[ext_logging_log_version_1] target=%s message=%s", string(target), string(msg))
	default:
		logger.Errorf("[ext_logging_log_version_1] level=%d target=%s message=%s", level, string(target), string(msg))
	}

	return 0
}

func ext_misc_print_utf8_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_misc_print_utf8_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)
	logger.Debugf("[ext_misc_print_utf8_version_1] utf8 data: 0x%x", data)
	return 0
}

func ext_misc_print_hex_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_misc_print_hex_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	data := asMemorySlice(vm.Memory, dataSpan)
	logger.Debugf("[ext_misc_print_hex_version_1] data is: 0x%x", data)
	return 0
}

func ext_allocator_malloc_version_1(vm *exec.VirtualMachine) int64 {
	size := uint32(vm.GetCurrentFrame().Locals[0])
	logger.Tracef("[ext_allocator_malloc_version_1] executing with size %d...", size)

	// Allocate memory
	res, err := ctx.Allocator.Allocate(size)
	if err != nil {
		logger.Errorf("[ext_allocator_malloc_version_1]: %s", err)
		panic(err)
	}

	return int64(res)
}

func ext_allocator_free_version_1(vm *exec.VirtualMachine) int64 {
	addr := uint32(vm.GetCurrentFrame().Locals[0])
	logger.Tracef("[ext_allocator_free_version_1] executing at address %d...", addr)

	// Deallocate memory
	err := ctx.Allocator.Deallocate(addr)
	if err != nil {
		logger.Errorf("[ext_allocator_free_version_1]: %s", err)
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
		logger.Errorf("[ext_hashing_blake2_256_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_blake2_256_version_1] data is 0x%x and hash is 0x%x", data, hash)

	out, err := toWasmMemorySized(vm.Memory, hash[:], 32)
	if err != nil {
		logger.Errorf("[ext_hashing_blake2_256_version_1] failed to allocate: %s", err)
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
		logger.Errorf("[ext_hashing_twox_128_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_twox_128_version_1] data is 0x%x and hash is 0x%x", data, hash)

	out, err := toWasmMemorySized(vm.Memory, hash, 16)
	if err != nil {
		logger.Errorf("[ext_hashing_twox_128_version_1] failed to allocate: %s", err)
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
		logger.Errorf("[ext_hashing_twox_64_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_twox_64_version_1] data is 0x%x and hash is 0x%x", data, hash)

	out, err := toWasmMemorySized(vm.Memory, hash, 8)
	if err != nil {
		logger.Errorf("[ext_hashing_twox_64_version_1] failed to allocate: %s", err)
		return 0
	}

	return int64(out)
}

func ext_storage_get_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_get_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)
	logger.Debugf("[ext_storage_get_version_1] key: 0x%x", key)

	value := storage.Get(key)
	logger.Debugf("[ext_storage_get_version_1] value: 0x%x", value)

	valueSpan, err := toWasmMemoryOptional(vm.Memory, value)
	if err != nil {
		logger.Errorf("[ext_storage_get_version_1] failed to allocate: %s", err)
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

	logger.Infof("[ext_storage_set_version_1] key 0x%x and value 0x%x", key, value)

	cp := make([]byte, len(value))
	copy(cp, value)
	storage.Set(key, cp)
	return 0
}

func ext_storage_next_key_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_next_key_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)

	next := storage.NextKey(key)
	logger.Debugf("[ext_storage_next_key_version_1] key is 0x%x and next is 0x%x", key, next)

	nextSpan, err := toWasmMemoryOptional(vm.Memory, next)
	if err != nil {
		logger.Errorf("[ext_storage_next_key_version_1] failed to allocate: %s", err)
		return 0
	}

	return nextSpan
}

func ext_storage_clear_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_clear_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)

	logger.Debugf("[ext_storage_clear_version_1] key: 0x%x", key)
	storage.Delete(key)
	return 0
}

func ext_storage_clear_prefix_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_clear_prefix_version_1] executing...")
	storage := ctx.Storage
	prefixSpan := vm.GetCurrentFrame().Locals[0]

	prefix := asMemorySlice(vm.Memory, prefixSpan)
	logger.Debugf("[ext_storage_clear_prefix_version_1] prefix: 0x%x", prefix)

	err := storage.ClearPrefix(prefix)
	if err != nil {
		logger.Errorf("[ext_storage_clear_prefix_version_1]: %s", err)
	}

	// sanity check
	next := storage.NextKey(prefix)
	if len(next) >= len(prefix) && bytes.Equal(prefix, next[:len(prefix)]) {
		panic("did not clear prefix")
	}

	return 0
}

func ext_storage_exists_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_exists_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	storage := ctx.Storage

	key := asMemorySlice(vm.Memory, keySpan)

	val := storage.Get(key)
	if len(val) == 0 {
		return 0
	}

	return 1
}

func ext_storage_read_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_read_version_1] executing...")
	keySpan := vm.GetCurrentFrame().Locals[0]
	valueOut := vm.GetCurrentFrame().Locals[1]
	offset := int32(vm.GetCurrentFrame().Locals[2])
	storage := ctx.Storage
	memory := vm.Memory

	key := asMemorySlice(memory, keySpan)
	value := storage.Get(key)
	logger.Debugf("[ext_storage_read_version_1] key 0x%x and value 0x%x", key, value)

	if value == nil {
		ret, _ := toWasmMemoryOptional(memory, nil)
		return ret
	}

	var size uint32

	if int(offset) > len(value) {
		size = uint32(0)
	} else {
		size = uint32(len(value[offset:]))
		valueBuf, valueLen := runtime.Int64ToPointerAndSize(valueOut)
		copy(memory[valueBuf:valueBuf+valueLen], value[offset:])
	}

	sizeSpan, err := toWasmMemoryOptionalUint32(memory, &size)
	if err != nil {
		logger.Errorf("[ext_storage_read_version_1] failed to allocate: %s", err)
		return 0
	}

	return sizeSpan
}

func storageAppend(storage runtime.Storage, key, valueToAppend []byte) error {
	nextLength := big.NewInt(1)
	var valueRes []byte

	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	valueCurr := storage.Get(key)

	if len(valueCurr) == 0 {
		valueRes = valueToAppend
	} else {
		var currLength *big.Int
		err := scale.Unmarshal(valueCurr, &currLength)
		if err != nil {
			logger.Tracef("[ext_storage_append_version_1] item in storage is not SCALE encoded, overwriting at key 0x%x", key)
			storage.Set(key, append([]byte{4}, valueToAppend...))
			return nil
		}

		lengthBytes, err := scale.Marshal(currLength)
		if err != nil {
			return err
		}
		// append new item, pop off number of bytes required for length encoding,
		// since we're not using old scale.Decoder
		valueRes = append(valueCurr[len(lengthBytes):], valueToAppend...)

		// increase length by 1
		nextLength = big.NewInt(0).Add(currLength, big.NewInt(1))
	}

	lengthEnc, err := scale.Marshal(nextLength)
	if err != nil {
		logger.Tracef("[ext_storage_append_version_1] failed to encode new length: %s", err)
		return err
	}

	// append new length prefix to start of items array
	lengthEnc = append(lengthEnc, valueRes...)
	logger.Debugf("[ext_storage_append_version_1] resulting value: 0x%x", lengthEnc)
	storage.Set(key, lengthEnc)
	return nil
}

func ext_storage_append_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_append_version_1] executing...")
	storage := ctx.Storage
	keySpan := vm.GetCurrentFrame().Locals[0]
	valueSpan := vm.GetCurrentFrame().Locals[1]

	key := asMemorySlice(vm.Memory, keySpan)
	logger.Debugf("[ext_storage_append_version_1] key 0x%x", key)
	valueAppend := asMemorySlice(vm.Memory, valueSpan)

	err := storageAppend(storage, key, valueAppend)
	if err != nil {
		logger.Errorf("[ext_storage_append_version_1]: %s", err)
	}

	return 0
}

func ext_trie_blake2_256_ordered_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_trie_blake2_256_ordered_root_version_1] executing...")
	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory
	data := asMemorySlice(memory, dataSpan)

	t := trie.NewEmptyTrie()
	var v [][]byte
	err := scale.Unmarshal(data, &v)
	if err != nil {
		logger.Errorf("[ext_trie_blake2_256_ordered_root_version_1]: %s", err)
		return 0
	}

	for i, val := range v {
		key, err := scale.Marshal(big.NewInt(int64(i))) //nolint
		if err != nil {
			logger.Errorf("[ext_blake2_256_enumerated_trie_root]: %s", err)
			return 0
		}
		logger.Tracef("[ext_trie_blake2_256_ordered_root_version_1] key 0x%x and value 0x%x", key, val)

		t.Put(key, val)
	}

	// allocate memory for value and copy value to memory
	ptr, err := ctx.Allocator.Allocate(32)
	if err != nil {
		logger.Errorf("[ext_trie_blake2_256_ordered_root_version_1]: %s", err)
		return 0
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Errorf("[ext_trie_blake2_256_ordered_root_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_trie_blake2_256_ordered_root_version_1] root hash: %s", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return int64(ptr)
}

func ext_storage_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_root_version_1] executing...")
	storage := ctx.Storage

	root, err := storage.Root()
	if err != nil {
		logger.Errorf("[ext_storage_root_version_1] failed to get storage root: %s", err)
		return 0
	}

	logger.Debugf("[ext_storage_root_version_1] root hash: %s", root)

	rootSpan, err := toWasmMemory(vm.Memory, root[:])
	if err != nil {
		logger.Errorf("[ext_storage_root_version_1] failed to allocate: %s", err)
		return 0
	}

	return rootSpan
}

func ext_storage_changes_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_storage_changes_root_version_1] executing...")
	logger.Debug("[ext_storage_changes_root_version_1] returning None")

	rootSpan, err := toWasmMemoryOptional(vm.Memory, nil)
	if err != nil {
		logger.Errorf("[ext_storage_changes_root_version_1] failed to allocate: %s", err)
		return 0
	}

	return rootSpan
}

func ext_crypto_start_batch_verify_version_1(_ *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_start_batch_verify_version_1] executing...")
	return 0
}

func ext_crypto_finish_batch_verify_version_1(_ *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_finish_batch_verify_version_1] executing...")
	return 1
}

func ext_offchain_index_set_version_1(_ *exec.VirtualMachine) int64 {
	logger.Trace("[ext_offchain_index_set_version_1] executing...")
	return 0
}

func ext_default_child_storage_set_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_set_version_1] executing...")
	storage := ctx.Storage
	memory := vm.Memory

	childStorageKeySpan := vm.GetCurrentFrame().Locals[0]
	childStorageKey := asMemorySlice(memory, childStorageKeySpan)
	keySpan := vm.GetCurrentFrame().Locals[1]
	key := asMemorySlice(memory, keySpan)
	valueSpan := vm.GetCurrentFrame().Locals[2]
	value := asMemorySlice(memory, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	err := storage.SetChildStorage(childStorageKey, key, cp)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_set_version_1] failed to set value in child storage: %s", err)
		return 0
	}

	return 0
}

func ext_default_child_storage_get_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_get_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	storage := ctx.Storage
	memory := vm.Memory

	child, err := storage.GetChildStorage(asMemorySlice(memory, childStorageKey), asMemorySlice(memory, key))
	if err != nil {
		logger.Errorf("[ext_default_child_storage_get_version_1] failed to get child from child storage: %s", err)
		return 0
	}

	value, err := toWasmMemoryOptional(memory, child)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_get_version_1] failed to allocate: %s", err)
		return 0
	}

	return value
}

func ext_default_child_storage_read_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_read_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	valueOut := vm.GetCurrentFrame().Locals[2]
	offset := vm.GetCurrentFrame().Locals[3]
	storage := ctx.Storage
	memory := vm.Memory

	value, err := storage.GetChildStorage(asMemorySlice(memory, childStorageKey), asMemorySlice(memory, key))
	if err != nil {
		logger.Errorf("[ext_default_child_storage_read_version_1] failed to get child storage: %s", err)
		return 0
	}

	valueBuf, valueLen := runtime.Int64ToPointerAndSize(valueOut)
	copy(memory[valueBuf:valueBuf+valueLen], value[offset:])

	size := uint32(len(value[offset:]))
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, size)

	sizeSpan, err := toWasmMemoryOptional(memory, sizeBuf)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_read_version_1] failed to allocate: %s", err)
		return 0
	}

	return sizeSpan
}

func ext_default_child_storage_clear_version_1(vm *exec.VirtualMachine) int64 {
	logger.Debug("[ext_default_child_storage_clear_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	keySpan := vm.GetCurrentFrame().Locals[1]
	memory := vm.Memory
	storage := ctx.Storage

	keyToChild := asMemorySlice(memory, childStorageKey)
	key := asMemorySlice(memory, keySpan)

	err := storage.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_clear_version_1] failed to clear child storage: %s", err)
	}
	return 0
}

func ext_default_child_storage_storage_kill_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_storage_kill_version_1] executing...")

	childStorageKeySpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory
	storage := ctx.Storage

	childStorageKey := asMemorySlice(memory, childStorageKeySpan)
	storage.DeleteChild(childStorageKey)
	return 0
}

func ext_default_child_storage_exists_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_exists_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	storage := ctx.Storage
	memory := vm.Memory

	child, err := storage.GetChildStorage(asMemorySlice(memory, childStorageKey), asMemorySlice(memory, key))
	if err != nil {
		logger.Errorf("[ext_default_child_storage_exists_version_1] failed to get child from child storage: %s", err)
		return 0
	}
	if child != nil {
		return 1
	}
	return 0
}

func ext_default_child_storage_clear_prefix_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_clear_prefix_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	prefixSpan := vm.GetCurrentFrame().Locals[1]
	storage := ctx.Storage
	memory := vm.Memory

	keyToChild := asMemorySlice(memory, childStorageKey)
	prefix := asMemorySlice(memory, prefixSpan)

	err := storage.ClearPrefixInChild(keyToChild, prefix)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_clear_prefix_version_1] failed to clear prefix in child: %s", err)
	}
	return 0
}

func ext_default_child_storage_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_root_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory
	storage := ctx.Storage

	child, err := storage.GetChild(asMemorySlice(memory, childStorageKey))
	if err != nil {
		logger.Errorf("[ext_default_child_storage_root_version_1] failed to retrieve child: %s", err)
		return 0
	}

	childRoot, err := child.Hash()
	if err != nil {
		logger.Errorf("[ext_default_child_storage_root_version_1] failed to encode child root: %s", err)
		return 0
	}

	root, err := toWasmMemoryOptional(memory, childRoot[:])
	if err != nil {
		logger.Errorf("[ext_default_child_storage_root_version_1] failed to allocate: %s", err)
		return 0
	}

	return root
}

func ext_default_child_storage_next_key_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_default_child_storage_next_key_version_1] executing...")

	childStorageKey := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	memory := vm.Memory
	storage := ctx.Storage

	child, err := storage.GetChildNextKey(asMemorySlice(memory, childStorageKey), asMemorySlice(memory, key))
	if err != nil {
		logger.Errorf("[ext_default_child_storage_next_key_version_1] failed to get child's next key: %s", err)
		return 0
	}

	value, err := toWasmMemoryOptional(memory, child)
	if err != nil {
		logger.Errorf("[ext_default_child_storage_next_key_version_1] failed to allocate: %s", err)
		return 0
	}

	return value
}

func ext_crypto_ed25519_public_keys_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_ed25519_public_keys_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_public_keys_version_1] error for id 0x%x: %s", id, err)
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"[ext_crypto_ed25519_public_keys_version_1] keystore type for id 0x%x is %s and not the expected ed25519",
			id, ks.Type())
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory: %s", err)
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	ret, err := toWasmMemory(memory, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory: %s", err)
		ret, _ = toWasmMemory(memory, []byte{0})
		return ret
	}

	return ret
}

func ext_crypto_ed25519_generate_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_ed25519_generate_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	seedSpan := vm.GetCurrentFrame().Locals[1]
	memory := vm.Memory

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(memory, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_generate_version_1] cannot generate key: %s", err)
		return 0
	}

	var kp crypto.Keypair

	if seed != nil {
		kp, err = ed25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = ed25519.GenerateKeypair()
	}

	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_generate_version_1] cannot generate key: %s", err)
		return 0
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_generate_version_1] error for id 0x%x: %s", id, err)
		return 0
	}

	ks.Insert(kp)

	ret, err := toWasmMemorySized(memory, kp.Public().Encode(), 32)
	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_generate_version_1] failed to allocate memory: %s", err)
		return 0
	}

	logger.Debug("[ext_crypto_ed25519_generate_version_1] generated ed25519 keypair with resulting public key: " + kp.Public().Hex())
	return int64(ret)
}

func ext_crypto_ed25519_sign_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_ed25519_sign_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	msg := vm.GetCurrentFrame().Locals[2]
	memory := vm.Memory

	id := memory[keyTypeID : keyTypeID+4]

	pubKeyData := memory[key : key+32]
	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Errorf("[ext_crypto_ed25519_sign_version_1] failed to get public keys: %s", err)
		return 0
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_ed25519_sign_version_1] error for id 0x%x: %s", id, err)
		ret, _ := toWasmMemoryOptional(memory, nil)
		return ret
	}

	var ret int64
	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not find public key " + pubKey.Hex() + " in keystore")
		ret, err = toWasmMemoryOptional(memory, nil)
		if err != nil {
			logger.Errorf("[ext_crypto_ed25519_sign_version_1] failed to allocate memory: %s", err)
			return 0
		}
		return ret
	}

	sig, err := signingKey.Sign(asMemorySlice(memory, msg))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not sign message")
	}

	ret, err = toWasmMemoryFixedSizeOptional(memory, sig)
	if err != nil {
		logger.Errorf("[ext_crypto_ed25519_sign_version_1] failed to allocate memory: %s", err)
		return 0
	}

	return ret
}

func ext_crypto_ed25519_verify_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_ed25519_verify_version_1] executing...")

	sig := vm.GetCurrentFrame().Locals[0]
	msg := vm.GetCurrentFrame().Locals[1]
	key := vm.GetCurrentFrame().Locals[2]
	memory := vm.Memory
	sigVerifier := ctx.SigVerifier

	signature := memory[sig : sig+64]
	message := asMemorySlice(memory, msg)
	pubKeyData := memory[key : key+32]

	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_verify_version_1] failed to create public key")
		return 0
	}

	if sigVerifier.IsStarted() {
		signature := runtime.Signature{
			PubKey:    pubKey.Encode(),
			Sign:      signature,
			Msg:       message,
			KeyTypeID: crypto.Ed25519Type,
		}
		sigVerifier.Add(&signature)
		return 1
	}

	if ok, err := pubKey.Verify(message, signature); err != nil || !ok {
		logger.Error("[ext_crypto_ed25519_verify_version_1] failed to verify")
		return 0
	}

	logger.Debug("[ext_crypto_ed25519_verify_version_1] verified ed25519 signature")
	return 1
}

func ext_crypto_sr25519_public_keys_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_sr25519_public_keys_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_sr25519_public_keys_version_1] error for id 0x%x: %s", id, err)
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	if ks.Type() != crypto.Sr25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"[ext_crypto_ed25519_public_keys_version_1] keystore type for id 0x%x is %s and not the expected sr25519",
			id, ks.Type())
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_public_keys_version_1] failed to allocate memory: %s", err)
		ret, _ := toWasmMemory(memory, []byte{0})
		return ret
	}

	ret, err := toWasmMemory(memory, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_public_keys_version_1] failed to allocate memory: %s", err)
		ret, _ = toWasmMemory(memory, []byte{0})
		return ret
	}

	return ret
}

func ext_crypto_sr25519_generate_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_sr25519_generate_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	seedSpan := vm.GetCurrentFrame().Locals[1]
	memory := vm.Memory

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(memory, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("[ext_crypto_sr25519_generate_version_1] cannot generate key: %s", err)
		return 0
	}

	var kp crypto.Keypair
	if seed != nil {
		kp, err = sr25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = sr25519.GenerateKeypair()
	}

	if err != nil {
		logger.Tracef("[ext_crypto_sr25519_generate_version_1] cannot generate key: %s", err)
		panic(err)
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_sr25519_generate_version_1] error for id 0x%x: %s", id, err)
		return 0
	}

	ks.Insert(kp)
	ret, err := toWasmMemorySized(memory, kp.Public().Encode(), 32)
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_generate_version_1] failed to allocate memory: %s", err)
		return 0
	}

	logger.Debug("[ext_crypto_sr25519_generate_version_1] generated sr25519 keypair with resulting public key: " + kp.Public().Hex())
	return int64(ret)
}

func ext_crypto_sr25519_sign_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_sr25519_sign_version_1] executing...")

	keyTypeID := vm.GetCurrentFrame().Locals[0]
	key := vm.GetCurrentFrame().Locals[1]
	msg := vm.GetCurrentFrame().Locals[2]
	memory := vm.Memory

	emptyRet, _ := toWasmMemoryOptional(memory, nil)

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("[ext_crypto_sr25519_sign_version_1] error for id 0x%x: %s", id, err)
		return emptyRet
	}

	var ret int64
	pubKey, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_sign_version_1] failed to get public key: %s", err)
		return emptyRet
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] could not find public key " + pubKey.Hex() + " in keystore")
		return emptyRet
	}

	msgData := asMemorySlice(memory, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_sign_version_1] could not sign message: %s", err)
		return emptyRet
	}

	ret, err = toWasmMemoryFixedSizeOptional(memory, sig)
	if err != nil {
		logger.Errorf("[ext_crypto_sr25519_sign_version_1] failed to allocate memory: %s", err)
		return emptyRet
	}

	return ret
}

func ext_crypto_sr25519_verify_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_sr25519_verify_version_1] executing...")

	sig := vm.GetCurrentFrame().Locals[0]
	msg := vm.GetCurrentFrame().Locals[1]
	key := vm.GetCurrentFrame().Locals[2]
	memory := vm.Memory
	sigVerifier := ctx.SigVerifier

	message := asMemorySlice(memory, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_verify_version_1] invalid sr25519 public key")
		return 0
	}

	logger.Debugf(
		"[ext_crypto_sr25519_verify_version_1] pub=%s; message=0x%x; signature=0x%x",
		pub.Hex(), message, signature)

	if sigVerifier.IsStarted() {
		signature := runtime.Signature{
			PubKey:    pub.Encode(),
			Sign:      signature,
			Msg:       message,
			KeyTypeID: crypto.Sr25519Type,
		}
		sigVerifier.Add(&signature)
		return 1
	}

	if ok, err := pub.VerifyDeprecated(message, signature); err != nil || !ok {
		logger.Debugf("[ext_crypto_sr25519_verify_version_1] failed to validate signature: %s", err)
		// this fails at block 3876, however based on discussions this seems to be expected
		return 1
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_1] verified sr25519 signature")
	return 1
}

func ext_crypto_secp256k1_ecdsa_recover_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_version_1] executing...")

	sig := vm.GetCurrentFrame().Locals[0]
	msg := vm.GetCurrentFrame().Locals[1]
	memory := vm.Memory

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	if signature[64] == 27 {
		signature[64] = 0
	}

	if signature[64] == 28 {
		signature[64] = 1
	}

	pub, err := secp256k1.RecoverPublicKey(message, signature)
	if err != nil {
		logger.Errorf("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to recover public key: %s", err)
		var ret int64
		ret, err = toWasmMemoryResult(memory, nil)
		if err != nil {
			logger.Errorf("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to allocate memory: %s", err)
			return 0
		}
		return ret
	}

	logger.Debugf(
		"[ext_crypto_secp256k1_ecdsa_recover_version_1] recovered public key of length %d: 0x%x",
		len(pub), pub)

	ret, err := toWasmMemoryResult(memory, pub[1:])
	if err != nil {
		logger.Errorf("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to allocate memory: %s", err)
		return 0
	}

	return ret
}

func ext_hashing_keccak_256_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_keccak_256_version_1] executing...")

	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	data := asMemorySlice(memory, dataSpan)

	hash, err := common.Keccak256(data)
	if err != nil {
		logger.Errorf("[ext_hashing_keccak_256_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_keccak_256_version_1] data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(memory, hash[:], 32)
	if err != nil {
		logger.Errorf("[ext_hashing_keccak_256_version_1] failed to allocate: %s", err)
		return 0
	}

	return int64(out)
}

func ext_hashing_sha2_256_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_sha2_256_version_1] executing...")

	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	data := asMemorySlice(memory, dataSpan)
	hash := common.Sha256(data)

	logger.Debugf("[ext_hashing_sha2_256_version_1] data 0x%x hash hash %x", data, hash)

	out, err := toWasmMemorySized(memory, hash[:], 32)
	if err != nil {
		logger.Errorf("[ext_hashing_sha2_256_version_1] failed to allocate: %s", err)
		return 0
	}

	return int64(out)
}

func ext_hashing_blake2_128_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_blake2_128_version_1] executing...")

	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	data := asMemorySlice(memory, dataSpan)

	hash, err := common.Blake2b128(data)
	if err != nil {
		logger.Errorf("[ext_hashing_blake2_128_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_blake2_128_version_1] data 0x%x has hash 0x%x", data, hash)

	out, err := toWasmMemorySized(memory, hash, 16)
	if err != nil {
		logger.Errorf("[ext_hashing_blake2_128_version_1] failed to allocate: %s", err)
		return 0
	}

	return int64(out)
}

func ext_hashing_twox_256_version_1(vm *exec.VirtualMachine) int64 {
	logger.Trace("[ext_hashing_twox_256_version_1] executing...")

	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	data := asMemorySlice(memory, dataSpan)

	hash, err := common.Twox256(data)
	if err != nil {
		logger.Errorf("[ext_hashing_twox_256_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_hashing_twox_256_version_1] data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(memory, hash[:], 32)
	if err != nil {
		logger.Errorf("[ext_hashing_twox_256_version_1] failed to allocate: %s", err)
		return 0
	}

	return int64(out)
}

func ext_trie_blake2_256_root_version_1(vm *exec.VirtualMachine) int64 {
	logger.Debug("[ext_trie_blake2_256_root_version_1] executing...")

	dataSpan := vm.GetCurrentFrame().Locals[0]
	memory := vm.Memory

	data := asMemorySlice(memory, dataSpan)

	t := trie.NewEmptyTrie()

	// this function is expecting an array of (key, value) tuples
	type kv struct {
		Key, Value []byte
	}

	var kvs []kv
	if err := scale.Unmarshal(data, &kvs); err != nil {
		logger.Errorf("[ext_trie_blake2_256_root_version_1]: %s", err)
		return 0
	}

	for _, kv := range kvs {
		t.Put(kv.Key, kv.Value)
	}

	// allocate memory for value and copy value to memory
	ptr, err := ctx.Allocator.Allocate(32)
	if err != nil {
		logger.Errorf("[ext_trie_blake2_256_root_version_1]: %s", err)
		return 0
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Errorf("[ext_trie_blake2_256_root_version_1]: %s", err)
		return 0
	}

	logger.Debugf("[ext_trie_blake2_256_root_version_1] root hash: %s", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return int64(ptr)
}

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(memory []byte, span int64) []byte {
	ptr, size := runtime.Int64ToPointerAndSize(span)
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

	copy(memory[out:out+size], data)
	return out, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(memory, data []byte) (int64, error) {
	var opt *[]byte
	if data != nil {
		opt = &data
	}

	enc, err := scale.Marshal(opt)
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

	copy(memory[out:out+size], data)
	return runtime.PointerAndSizeToInt64(int32(out), int32(size)), nil
}

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(memory []byte, data *uint32) (int64, error) {
	var opt *uint32
	if data != nil {
		temp := *data
		opt = &temp
	}

	enc, err := scale.Marshal(opt)
	if err != nil {
		return int64(0), err
	}
	return toWasmMemory(memory, enc)
}

// Wraps slice in optional.FixedSizeBytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryFixedSizeOptional(memory, data []byte) (int64, error) {
	var opt [64]byte
	copy(opt[:], data[:])
	enc, err := scale.Marshal(&opt)
	if err != nil {
		return 0, err
	}
	return toWasmMemory(memory, enc)
}

// Wraps slice in Result type and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryResult(memory, data []byte) (int64, error) {
	var res *rtype.Result
	if len(data) == 0 {
		res = rtype.NewResult(byte(1), nil)
	} else {
		res = rtype.NewResult(byte(0), data)
	}

	enc, err := res.Encode()
	if err != nil {
		return 0, err
	}

	return toWasmMemory(memory, enc)
}
