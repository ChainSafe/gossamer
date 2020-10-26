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

// #include <stdlib.h>
//
// extern int32_t ext_malloc(void *context, int32_t size);
// extern void ext_free(void *context, int32_t addr);
// extern void ext_print_utf8(void *context, int32_t utf8_data, int32_t utf8_len);
// extern void ext_print_hex(void *context, int32_t data, int32_t len);
// extern int32_t ext_get_storage_into(void *context, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen, int32_t valueOffset);
// extern void ext_set_storage(void *context, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen);
// extern void ext_blake2_256(void *context, int32_t data, int32_t len, int32_t out);
// extern void ext_clear_storage(void *context, int32_t keyData, int32_t keyLen);
// extern void ext_twox_64(void *context, int32_t data, int32_t len, int32_t out);
// extern void ext_twox_128(void *context, int32_t data, int32_t len, int32_t out);
// extern int32_t ext_get_allocated_storage(void *context, int32_t keyData, int32_t keyLen, int32_t writtenOut);
// extern void ext_storage_root(void *context, int32_t resultPtr);
// extern int32_t ext_storage_changes_root(void *context, int32_t a, int32_t b, int32_t c);
// extern void ext_clear_prefix(void *context, int32_t prefixData, int32_t prefixLen);
// extern int32_t ext_sr25519_verify(void *context, int32_t msgData, int32_t msgLen, int32_t sigData, int32_t pubkeyData);
// extern int32_t ext_ed25519_verify(void *context, int32_t msgData, int32_t msgLen, int32_t sigData, int32_t pubkeyData);
// extern void ext_blake2_256_enumerated_trie_root(void *context, int32_t valuesData, int32_t lensData, int32_t lensLen, int32_t result);
// extern void ext_print_num(void *context, int64_t data);
// extern void ext_keccak_256(void *context, int32_t data, int32_t len, int32_t out);
// extern int32_t ext_secp256k1_ecdsa_recover(void *context, int32_t msgData, int32_t sigData, int32_t pubkeyData);
// extern void ext_blake2_128(void *context, int32_t data, int32_t len, int32_t out);
// extern int32_t ext_is_validator(void *context);
// extern int32_t ext_local_storage_get(void *context, int32_t kind, int32_t key, int32_t keyLen, int32_t valueLen);
// extern int32_t ext_local_storage_compare_and_set(void *context, int32_t kind, int32_t key, int32_t keyLen, int32_t oldValue, int32_t oldValueLen, int32_t newValue, int32_t newValueLen);
// extern int32_t ext_sr25519_public_keys(void *context, int32_t idData, int32_t resultLen);
// extern int32_t ext_ed25519_public_keys(void *context, int32_t idData, int32_t resultLen);
// extern int32_t ext_network_state(void *context, int32_t writtenOut);
// extern int32_t ext_sr25519_sign(void *context, int32_t idData, int32_t pubkeyData, int32_t msgData, int32_t msgLen, int32_t out);
// extern int32_t ext_ed25519_sign(void *context, int32_t idData, int32_t pubkeyData, int32_t msgData, int32_t msgLen, int32_t out);
// extern int32_t ext_submit_transaction(void *context, int32_t data, int32_t len);
// extern void ext_local_storage_set(void *context, int32_t kind, int32_t key, int32_t keyLen, int32_t value, int32_t valueLen);
// extern void ext_ed25519_generate(void *context, int32_t idData, int32_t seed, int32_t seedLen, int32_t out);
// extern void ext_sr25519_generate(void *context, int32_t idData, int32_t seed, int32_t seedLen, int32_t out);
// extern void ext_set_child_storage(void *context, int32_t storageKeyData, int32_t storageKeyLen, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen);
// extern int32_t ext_get_child_storage_into(void *context, int32_t storageKeyData, int32_t storageKeyLen, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen, int32_t valueOffset);
// extern void ext_kill_child_storage(void *context, int32_t a, int32_t b);
// extern int32_t ext_sandbox_memory_new(void *context, int32_t a, int32_t b);
// extern void ext_sandbox_memory_teardown(void *context, int32_t a);
// extern int32_t ext_sandbox_instantiate(void *context, int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f);
// extern int32_t ext_sandbox_invoke(void *context, int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f, int32_t g, int32_t h);
// extern void ext_sandbox_instance_teardown(void *context, int32_t a);
// extern int32_t ext_get_allocated_child_storage(void *context, int32_t a, int32_t b, int32_t c, int32_t d, int32_t e);
// extern int32_t ext_child_storage_root(void *context, int32_t a, int32_t b, int32_t c);
// extern void ext_clear_child_storage(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern int32_t ext_secp256k1_ecdsa_recover_compressed(void *context, int32_t a, int32_t b, int32_t c);
// extern int32_t ext_sandbox_memory_get(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern int32_t ext_sandbox_memory_set(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern void ext_log(void *context, int32_t a, int32_t b, int32_t c, int32_t d, int32_t e);
// extern void ext_twox_256(void *context, int32_t a, int32_t b, int32_t c);
// extern int32_t ext_exists_storage(void *context, int32_t a, int32_t b);
// extern int32_t ext_exists_child_storage(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern void ext_clear_child_prefix(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"unsafe"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_print_num
func ext_print_num(context unsafe.Pointer, data C.int64_t) {
	logger.Trace("[ext_print_num] executing...")
	logger.Debug("[ext_print_num]", "message", fmt.Sprintf("%d", data))
}

//export ext_malloc
func ext_malloc(context unsafe.Pointer, size C.int32_t) C.int32_t {
	logger.Trace("[ext_malloc] executing...", "size", size)
	instanceContext := wasm.IntoInstanceContext(context)
	data := instanceContext.Data()
	runtimeCtx, ok := data.(*runtime.Context)
	if !ok {
		panic(fmt.Sprintf("%#v", data))
	}

	// Allocate memory
	res, err := runtimeCtx.Allocator.Allocate(uint32(size))
	if err != nil {
		logger.Error("[ext_malloc]", "Error:", err)
		panic(err)
	}

	return C.int32_t(res)
}

//export ext_free
func ext_free(context unsafe.Pointer, addr C.int32_t) {
	logger.Trace("[ext_free] executing...", "addr", addr)
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	// Deallocate memory
	err := runtimeCtx.Allocator.Deallocate(uint32(addr))
	if err != nil {
		logger.Error("[ext_free] Error:", "Error", err)
		//panic(err)
	}
}

// prints string located in memory at location `offset` with length `size`
//export ext_print_utf8
func ext_print_utf8(context unsafe.Pointer, utf8_data, utf8_len C.int32_t) {
	logger.Trace("[ext_print_utf8] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	logger.Debug("[ext_print_utf8]", "message", fmt.Sprintf("%s", memory[utf8_data:utf8_data+utf8_len]))
}

// prints hex formatted bytes located in memory at location `offset` with length `size`
//export ext_print_hex
func ext_print_hex(context unsafe.Pointer, offset, size C.int32_t) {
	logger.Trace("[ext_print_hex] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	logger.Debug("[ext_print_hex]", "message", fmt.Sprintf("%x", memory[offset:offset+size]))
}

// gets the key stored at memory location `keyData` with length `keyLen` and stores the value in memory at
// location `valueData`. the value can have up to value `valueLen` and the returned value starts at value[valueOffset:]
//export ext_get_storage_into
func ext_get_storage_into(context unsafe.Pointer, keyData, keyLen, valueData, valueLen, valueOffset C.int32_t) C.int32_t {
	logger.Trace("[ext_get_storage_into] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	key := memory[keyData : keyData+keyLen]
	val, err := s.Get(key)
	if err != nil {
		logger.Warn("[ext_get_storage_into]", "err", err)
		ret := 1<<32 - 1
		return C.int32_t(ret)
	} else if val == nil {
		logger.Debug("[ext_get_storage_into]", "err", "value is nil")
		ret := 1<<32 - 1
		return C.int32_t(ret)
	}

	if len(val) > int(valueLen) {
		logger.Debug("[ext_get_storage_into]", "error", "value exceeds allocated buffer length")
		return 0
	}

	copy(memory[valueData:valueData+valueLen], val[valueOffset:])
	return C.int32_t(len(val[valueOffset:]))
}

// puts the key at memory location `keyData` with length `keyLen` and value at memory location `valueData`
// with length `valueLen` into the storage trie
//export ext_set_storage
func ext_set_storage(context unsafe.Pointer, keyData, keyLen, valueData, valueLen C.int32_t) {
	logger.Trace("[ext_set_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	key := memory[keyData : keyData+keyLen]
	val := memory[valueData : valueData+valueLen]
	logger.Trace("[ext_set_storage]", "key", fmt.Sprintf("0x%x", key), "val", val)
	err := s.Set(key, val)
	if err != nil {
		logger.Error("[ext_set_storage]", "error", err)
		return
	}
}

//export ext_set_child_storage
func ext_set_child_storage(context unsafe.Pointer, storageKeyData, storageKeyLen, keyData, keyLen, valueData, valueLen C.int32_t) {
	logger.Trace("[ext_set_child_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	keyToChild := memory[storageKeyData : storageKeyData+storageKeyLen]
	key := memory[keyData : keyData+keyLen]
	value := memory[valueData : valueData+valueLen]

	err := s.SetChildStorage(keyToChild, key, value)
	if err != nil {
		logger.Error("[ext_set_child_storage]", "error", err)
	}
}

//export ext_get_child_storage_into
func ext_get_child_storage_into(context unsafe.Pointer, storageKeyData, storageKeyLen, keyData, keyLen, valueData, valueLen, valueOffset C.int32_t) C.int32_t {
	logger.Trace("[ext_get_child_storage_into] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	keyToChild := memory[storageKeyData : storageKeyData+storageKeyLen]
	key := memory[keyData : keyData+keyLen]

	value, err := s.GetChildStorage(keyToChild, key)
	if err != nil {
		logger.Error("[ext_get_child_storage_into]", "error", err)
		return -(1 << 31)
	}

	copy(memory[valueData:valueData+valueLen], value[valueOffset:])
	return C.int32_t(len(value[valueOffset:]))
}

// returns the trie root in the memory location `resultPtr`
//export ext_storage_root
func ext_storage_root(context unsafe.Pointer, resultPtr C.int32_t) {
	logger.Trace("[ext_storage_root] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	root, err := s.Root()
	if err != nil {
		logger.Error("[ext_storage_root]", "error", err)
		return
	}

	copy(memory[resultPtr:resultPtr+32], root[:])
}

//export ext_storage_changes_root
func ext_storage_changes_root(context unsafe.Pointer, a, b, c C.int32_t) C.int32_t {
	logger.Trace("[ext_storage_changes_root] executing...")
	logger.Debug("[ext_storage_changes_root] Not yet implemented.")
	return 0
}

// gets value stored at key at memory location `keyData` with length `keyLen` and returns the location
// in memory where it's stored and stores its length in `writtenOut`
//export ext_get_allocated_storage
func ext_get_allocated_storage(context unsafe.Pointer, keyData, keyLen, writtenOut C.int32_t) C.int32_t {
	logger.Trace("[ext_get_allocated_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	key := memory[keyData : keyData+keyLen]
	logger.Trace("[ext_get_allocated_storage]", "key", fmt.Sprintf("0x%x", key))

	val, err := s.Get(key)
	if err != nil {
		logger.Error("[ext_get_allocated_storage]", "error", err)
		copy(memory[writtenOut:writtenOut+4], []byte{0xff, 0xff, 0xff, 0xff})
		return 0
	}

	if len(val) >= (1 << 32) {
		logger.Error("[ext_get_allocated_storage]", "error", "retrieved value length exceeds 2^32")
		copy(memory[writtenOut:writtenOut+4], []byte{0xff, 0xff, 0xff, 0xff})
		return 0
	}

	if val == nil {
		logger.Trace("[ext_get_allocated_storage]", "value", "nil")
		copy(memory[writtenOut:writtenOut+4], []byte{0xff, 0xff, 0xff, 0xff})
		return 0
	}

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(uint32(len(val)))
	if err != nil {
		logger.Error("[ext_get_allocated_storage]", "error", err)
		copy(memory[writtenOut:writtenOut+4], []byte{0xff, 0xff, 0xff, 0xff})
		return 0
	}

	logger.Trace("[ext_get_allocated_storage]", "value", val)
	copy(memory[ptr:ptr+uint32(len(val))], val)

	// copy length to memory
	byteLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(byteLen, uint32(len(val)))
	// writtenOut stores the location of the memory that was allocated
	copy(memory[writtenOut:writtenOut+4], byteLen)

	// return ptr to value
	return C.int32_t(ptr)
}

// deletes the trie entry with key at memory location `keyData` with length `keyLen`
//export ext_clear_storage
func ext_clear_storage(context unsafe.Pointer, keyData, keyLen C.int32_t) {
	logger.Trace("[ext_clear_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	key := memory[keyData : keyData+keyLen]
	err := s.Delete(key)
	if err != nil {
		logger.Error("[ext_clear_storage]", "error", err)
	}
}

// deletes all entries in the trie that have a key beginning with the prefix stored at `prefixData`
//export ext_clear_prefix
func ext_clear_prefix(context unsafe.Pointer, prefixData, prefixLen C.int32_t) {
	logger.Trace("[ext_clear_prefix] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	prefix := memory[prefixData : prefixData+prefixLen]
	entries := s.Entries()
	for k := range entries {
		if bytes.Equal([]byte(k)[:prefixLen], prefix) {
			err := s.Delete([]byte(k))
			if err != nil {
				logger.Error("[ext_clear_prefix]", "err", err)
			}
		}
	}
}

// accepts an array of values, puts them into a trie, and returns the root
// the keys to the values are their position in the array
//export ext_blake2_256_enumerated_trie_root
func ext_blake2_256_enumerated_trie_root(context unsafe.Pointer, valuesData, lensData, lensLen, result C.int32_t) {
	logger.Trace("[ext_blake2_256_enumerated_trie_root] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	t := trie.NewEmptyTrie()
	var i C.int32_t
	var pos C.int32_t = 0
	logger.Trace("[ext_blake2_256_enumerated_trie_root]", "valuesData", valuesData, "lensData", lensData, "lensLen", lensLen)
	for i = 0; i < lensLen; i++ {
		valueLenBytes := memory[lensData+i*4 : lensData+(i+1)*4]
		valueLen := C.int32_t(binary.LittleEndian.Uint32(valueLenBytes))
		value := memory[valuesData+pos : valuesData+pos+valueLen]
		logger.Trace("[ext_blake2_256_enumerated_trie_root]", "key", i, "value", fmt.Sprintf("%d", value), "valueLen", valueLen)
		pos += valueLen

		// encode the key
		encodedOutput, err := scale.Encode(big.NewInt(int64(i)))
		if err != nil {
			logger.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
			return
		}
		logger.Trace("[ext_blake2_256_enumerated_trie_root]", "key", i, "key value", encodedOutput)
		err = t.Put(encodedOutput, value)
		if err != nil {
			logger.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
			return
		}
	}
	root, err := t.Hash()
	if err != nil {
		logger.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
		return
	}
	logger.Trace("[ext_blake2_256_enumerated_trie_root]", "root", root)
	copy(memory[result:result+32], root[:])
}

// performs blake2b 256-bit hash of the byte array at memory location `data` with length `length` and saves the
// hash at memory location `out`
//export ext_blake2_256
func ext_blake2_256(context unsafe.Pointer, data, length, out C.int32_t) {
	logger.Trace("[ext_blake2_256] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	hash, err := common.Blake2bHash(memory[data : data+length])
	if err != nil {
		logger.Error("[ext_blake2_256]", "error", err)
		return
	}

	copy(memory[out:out+32], hash[:])
}

//export ext_blake2_128
func ext_blake2_128(context unsafe.Pointer, data, length, out C.int32_t) {
	logger.Trace("[ext_blake2_128] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	hash, err := common.Blake2b128(memory[data : data+length])
	if err != nil {
		logger.Error("[ext_blake2_128]", "error", err)
		return
	}

	logger.Trace("[ext_blake2_128]", "hash", fmt.Sprintf("0x%x", hash))
	copy(memory[out:out+16], hash[:])
}

//export ext_keccak_256
func ext_keccak_256(context unsafe.Pointer, data, length, out C.int32_t) {
	logger.Trace("[ext_keccak_256] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	hash, err := common.Keccak256(memory[data : data+length])
	if err != nil {
		logger.Error("[ext_keccak_256]", "error", err)
		return
	}

	logger.Trace("[ext_keccak_256]", "hash", hash)
	copy(memory[out:out+32], hash[:])
}

//export ext_twox_64
func ext_twox_64(context unsafe.Pointer, data, len, out C.int32_t) {
	logger.Trace("[ext_twox_64] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	logger.Trace("[ext_twox_64] hashing...", "value", memory[data:data+len])

	hash, err := common.Twox64(memory[data : data+len])
	if err != nil {
		logger.Error("[ext_twox_64]", "error", err)
		return
	}
	copy(memory[out:out+8], hash)
}

//export ext_twox_128
func ext_twox_128(context unsafe.Pointer, data, len, out C.int32_t) {
	logger.Trace("[ext_twox_128] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	logger.Trace("[ext_twox_128] hashing...", "value", fmt.Sprintf("%s", memory[data:data+len]))

	res, err := common.Twox128Hash(memory[data : data+len])
	if err != nil {
		logger.Trace("error hashing in ext_twox_128", "error", err)
	}

	copy(memory[out:out+16], res)
}

//export ext_sr25519_generate
func ext_sr25519_generate(context unsafe.Pointer, idData, seed, seedLen, out C.int32_t) {
	logger.Trace("[ext_sr25519_generate] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	// TODO: key types not yet implemented
	// id := memory[idData:idData+4]

	seedBytes := memory[seed : seed+seedLen]

	kp, err := sr25519.NewKeypairFromSeed(seedBytes)
	if err != nil {
		logger.Trace("ext_sr25519_generate cannot generate key", "error", err)
	}

	logger.Trace("ext_sr25519_generate", "address", kp.Public().Address())

	runtimeCtx.Keystore.Insert(kp)

	copy(memory[out:out+32], kp.Public().Encode())
}

//export ext_ed25519_public_keys
func ext_ed25519_public_keys(context unsafe.Pointer, idData, resultLen C.int32_t) C.int32_t {
	logger.Trace("[ext_ed25519_public_keys] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	keys := runtimeCtx.Keystore.Ed25519PublicKeys()
	// TODO: when do deallocate?
	offset, err := runtimeCtx.Allocator.Allocate(uint32(len(keys) * 32))
	if err != nil {
		logger.Error("[ext_ed25519_public_keys]", "error", err)
		return -1
	}

	for i, key := range keys {
		copy(memory[offset+uint32(i*32):offset+uint32((i+1)*32)], key.Encode())
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(keys)))
	copy(memory[resultLen:resultLen+4], buf)
	return C.int32_t(offset)
}

//export ext_sr25519_public_keys
func ext_sr25519_public_keys(context unsafe.Pointer, idData, resultLen C.int32_t) C.int32_t {
	logger.Trace("[ext_sr25519_public_keys] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	keys := runtimeCtx.Keystore.Sr25519PublicKeys()

	offset, err := runtimeCtx.Allocator.Allocate(uint32(len(keys) * 32))
	if err != nil {
		logger.Error("[ext_sr25519_public_keys]", "error", err)
		return -1
	}

	for i, key := range keys {
		copy(memory[offset+uint32(i*32):offset+uint32((i+1)*32)], key.Encode())
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(keys)))
	copy(memory[resultLen:resultLen+4], buf)
	return C.int32_t(offset)
}

//export ext_ed25519_sign
func ext_ed25519_sign(context unsafe.Pointer, idData, pubkeyData, msgData, msgLen, out C.int32_t) C.int32_t {
	logger.Trace("[ext_ed25519_sign] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	pubkeyBytes := memory[pubkeyData : pubkeyData+32]
	pubkey, err := ed25519.NewPublicKey(pubkeyBytes)
	if err != nil {
		logger.Error("[ext_ed25519_sign]", "error", err)
		return 1
	}

	signingKey := runtimeCtx.Keystore.GetKeypair(pubkey)
	if signingKey == nil {
		logger.Error("[ext_ed25519_sign] could not find key in keystore", "public key", pubkey)
		return 1
	}

	msgLenBytes := memory[msgLen : msgLen+4]
	msgLength := binary.LittleEndian.Uint32(msgLenBytes)
	msg := memory[msgData : msgData+C.int32_t(msgLength)]
	sig, err := signingKey.Sign(msg)
	if err != nil {
		logger.Error("[ext_ed25519_sign] could not sign message")
		return 1
	}

	copy(memory[out:out+64], sig)
	return 0
}

//export ext_sr25519_sign
func ext_sr25519_sign(context unsafe.Pointer, idData, pubkeyData, msgData, msgLen, out C.int32_t) C.int32_t {
	logger.Trace("[ext_sr25519_sign] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	pubkeyBytes := memory[pubkeyData : pubkeyData+32]
	pubkey, err := sr25519.NewPublicKey(pubkeyBytes)
	if err != nil {
		logger.Error("[ext_sr25519_sign]", "error", err)
		return 1
	}

	signingKey := runtimeCtx.Keystore.GetKeypair(pubkey)

	if signingKey == nil {
		logger.Error("[ext_sr25519_sign] could not find key in keystore", "public key", pubkey)
		return 1
	}

	msgLenBytes := memory[msgLen : msgLen+4]
	msgLength := binary.LittleEndian.Uint32(msgLenBytes)
	msg := memory[msgData : msgData+C.int32_t(msgLength)]
	sig, err := signingKey.Sign(msg)
	if err != nil {
		logger.Error("[ext_sr25519_sign] could not sign message")
		return 1
	}

	copy(memory[out:out+64], sig)
	return 0
}

//export ext_sr25519_verify
func ext_sr25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData C.int32_t) C.int32_t {
	logger.Trace("[ext_sr25519_verify] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	msg := memory[msgData : msgData+msgLen]
	sig := memory[sigData : sigData+64]
	logger.Trace("[ext_sr25519_verify]", "msg", msg)
	pub, err := sr25519.NewPublicKey(memory[pubkeyData : pubkeyData+32])
	if err != nil {
		return 1
	}

	if ok, err := pub.Verify(msg, sig); err != nil || !ok {
		return 1
	}

	return 0
}

//export ext_ed25519_generate
func ext_ed25519_generate(context unsafe.Pointer, idData, seed, seedLen, out C.int32_t) {
	logger.Trace("[ext_ed25519_generate] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	// TODO: key types not yet implemented
	// id := memory[idData:idData+4]

	seedBytes := memory[seed : seed+seedLen]

	kp, err := ed25519.NewKeypairFromSeed(seedBytes)
	if err != nil {
		logger.Trace("ext_ed25519_generate cannot generate key", "error", err)
	}

	logger.Trace("ext_ed25519_generate", "address", kp.Public().Address())

	runtimeCtx.Keystore.Insert(kp)

	copy(memory[out:out+32], kp.Public().Encode())
}

//export ext_ed25519_verify
func ext_ed25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData C.int32_t) C.int32_t {
	logger.Trace("[ext_ed25519_verify] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	msg := memory[msgData : msgData+msgLen]
	sig := memory[sigData : sigData+64]
	pubkey, err := ed25519.NewPublicKey(memory[pubkeyData : pubkeyData+32])
	if err != nil {
		return 1
	}

	if ok, err := pubkey.Verify(msg, sig); err != nil || !ok {
		return 1
	}

	return 0
}

//export ext_secp256k1_ecdsa_recover
func ext_secp256k1_ecdsa_recover(context unsafe.Pointer, msgData, sigData, pubkeyData C.int32_t) C.int32_t {
	logger.Trace("[ext_secp256k1_ecdsa_recover] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element.
	msg := memory[msgData : msgData+32]
	sig := memory[sigData : sigData+65]

	pub, err := secp256k1.RecoverPubkey(msg, sig)
	if err != nil {
		return 1
	}

	copy(memory[pubkeyData:pubkeyData+65], pub)
	return 0
}

//export ext_is_validator
func ext_is_validator(context unsafe.Pointer) C.int32_t {
	logger.Trace("[ext_is_validator] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Validator {
		return 1
	}
	return 0
}

//export ext_local_storage_get
func ext_local_storage_get(context unsafe.Pointer, kind, key, keyLen, valueLen C.int32_t) C.int32_t {
	logger.Trace("[ext_local_storage_get] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	keyM := memory[key : key+keyLen]
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	var res []byte
	var err error
	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		res, err = runtimeCtx.NodeStorage.PersistentStorage.Get(keyM)
	case runtime.NodeStorageTypeLocal:
		res, err = runtimeCtx.NodeStorage.LocalStorage.Get(keyM)
	}

	if err != nil {
		logger.Error("[ext_local_storage_get]", "error", err)
		return 0
	}
	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(uint32(valueLen))
	if err != nil {
		logger.Error("[ext_local_storage_get]", "error", err)
		return 0
	}
	copy(memory[ptr:ptr+uint32(valueLen)], res[:])
	return C.int32_t(ptr)
}

//export ext_local_storage_compare_and_set
func ext_local_storage_compare_and_set(context unsafe.Pointer, kind, keyPtr, keyLen, oldValuePtr, oldValueLen, newValuePtr, newValueLen C.int32_t) C.int32_t {
	logger.Trace("[ext_local_storage_compare_and_set] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	key := memory[keyPtr : keyPtr+keyLen]
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	var storedValue []byte
	var err error
	var nodeStorage runtime.BasicStorage

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		nodeStorage = runtimeCtx.NodeStorage.PersistentStorage
		storedValue, err = nodeStorage.Get(key)
	case runtime.NodeStorageTypeLocal:
		nodeStorage = runtimeCtx.NodeStorage.LocalStorage
		storedValue, err = nodeStorage.Get(key)
	}

	if err != nil {
		logger.Error("[ext_local_storage_compare_and_set]", "error", err)
		return 1
	}

	oldValue := memory[oldValuePtr : oldValuePtr+oldValueLen]

	if reflect.DeepEqual(storedValue, oldValue) {
		newValue := memory[newValuePtr : newValuePtr+newValueLen]
		err := nodeStorage.Put(key, newValue)
		if err != nil {
			logger.Error("[ext_local_storage_compare_and_set]", "error", err)
			return 1
		}
		return 0
	}
	return 1
}

//export ext_network_state
func ext_network_state(context unsafe.Pointer, writtenOut C.int32_t) C.int32_t {
	logger.Trace("[ext_network_state] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Network == nil {
		return 0
	}

	nsEnc, err := scale.Encode(runtimeCtx.Network.NetworkState())
	if err != nil {
		logger.Error("[ext_network_state]", "error", err)
		return 0
	}

	// copy network state length to memory writtenOut location
	nsEncLen := uint32(len(nsEnc))
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, nsEncLen)
	copy(memory[writtenOut:writtenOut+4], buf)

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(nsEncLen)
	if err != nil {
		logger.Error("[ext_network_state]", "error", err)
		return 0
	}
	copy(memory[ptr:ptr+nsEncLen], nsEnc)
	return C.int32_t(ptr)
}

//export ext_submit_transaction
func ext_submit_transaction(context unsafe.Pointer, data, len C.int32_t) C.int32_t {
	logger.Trace("[ext_submit_transaction] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	extBytes := memory[data : data+len]

	ext := types.Extrinsic(extBytes)

	// validate the transaction
	txv := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	vtx := transaction.NewValidTransaction(ext, txv)

	runtimeCtx.Transaction.AddToPool(vtx)
	return 0
}

//export ext_local_storage_set
func ext_local_storage_set(context unsafe.Pointer, kind, key, keyLen, value, valueLen C.int32_t) {
	logger.Trace("[ext_local_storage_set] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	keyM := memory[key : key+keyLen]
	valueM := memory[value : value+valueLen]

	runtimeCtx := instanceContext.Data().(*runtime.Context)

	var err error
	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		err = runtimeCtx.NodeStorage.PersistentStorage.Put(keyM, valueM)
	case runtime.NodeStorageTypeLocal:
		err = runtimeCtx.NodeStorage.LocalStorage.Put(keyM, valueM)
	}
	if err != nil {
		logger.Error("[ext_local_storage_set]", "error", err)
	}
}

//export ext_kill_child_storage
func ext_kill_child_storage(context unsafe.Pointer, storageKeyData, storageKeyLen C.int32_t) {
	logger.Trace("[ext_kill_child_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	keyToChild := memory[storageKeyData : storageKeyData+storageKeyLen]

	err := s.DeleteChildStorage(keyToChild)
	if err != nil {
		logger.Error("[ext_kill_child_storage]", "error", err)
	}
}

//export ext_sandbox_memory_new
func ext_sandbox_memory_new(context unsafe.Pointer, a, b C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_new] executing...")
	logger.Warn("[ext_sandbox_memory_new] not yet implemented")
	return 0
}

//export ext_sandbox_memory_teardown
func ext_sandbox_memory_teardown(context unsafe.Pointer, a C.int32_t) {
	logger.Trace("[ext_sandbox_memory_teardown] executing...")
	logger.Warn("[ext_sandbox_memory_teardown] not yet implemented")
}

//export ext_sandbox_instantiate
func ext_sandbox_instantiate(context unsafe.Pointer, a, b, c, d, e, f C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_instantiate] executing...")
	logger.Warn("[ext_sandbox_instantiate] not yet implemented")
	return 0
}

//export ext_sandbox_invoke
func ext_sandbox_invoke(context unsafe.Pointer, a, b, c, d, e, f, g, h C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_invoke] executing...")
	logger.Warn("[ext_sandbox_invoke] not yet implemented")
	return 0
}

//export ext_sandbox_instance_teardown
func ext_sandbox_instance_teardown(context unsafe.Pointer, a C.int32_t) {
	logger.Trace("[ext_sandbox_instance_teardown] executing...")
	logger.Warn("[ext_sandbox_instance_teardown] not yet implemented")
}

//export ext_get_allocated_child_storage
func ext_get_allocated_child_storage(context unsafe.Pointer, storageKeyData, storageKeyLen, keyData, keyLen, writtenOut C.int32_t) C.int32_t {
	logger.Trace("[ext_get_allocated_child_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	keyToChild := memory[storageKeyData : storageKeyData+storageKeyLen]
	key := memory[keyData : keyData+keyLen]

	value, err := s.GetChildStorage(keyToChild, key)
	if err != nil {
		logger.Error("[ext_get_allocated_child_storage]", "error", err)
		return 0
	}
	valueLen := uint32(len(value))
	if valueLen == 0 {
		copy(memory[writtenOut:writtenOut+4], []byte{0xff, 0xff, 0xff, 0xff})
		return 0
	}

	// copy length to memory
	byteLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(byteLen, valueLen)
	copy(memory[writtenOut:writtenOut+4], byteLen)

	resPtr, err := runtimeCtx.Allocator.Allocate(valueLen)
	if err != nil {
		logger.Error("[ext_get_allocated_child_storage]", "error", err)
		return 0
	}
	copy(memory[resPtr:resPtr+valueLen], value)
	return C.int32_t(resPtr)
}

//export ext_child_storage_root
func ext_child_storage_root(context unsafe.Pointer, a, b, c C.int32_t) C.int32_t {
	logger.Trace("[ext_child_storage_root] executing...")
	logger.Warn("[ext_child_storage_root] not yet implemented")
	return 0
}

//export ext_clear_child_storage
func ext_clear_child_storage(context unsafe.Pointer, storageKeyData, storageKeyLen, keyData, keyLen C.int32_t) {
	logger.Trace("[ext_clear_child_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	s := runtimeCtx.Storage

	keyToChild := memory[storageKeyData : storageKeyData+storageKeyLen]
	key := memory[keyData : keyData+keyLen]
	err := s.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Error("[ext_clear_child_storage]", "error", err)
	}
}

//export ext_secp256k1_ecdsa_recover_compressed
func ext_secp256k1_ecdsa_recover_compressed(context unsafe.Pointer, a, b, c C.int32_t) C.int32_t {
	logger.Trace("[ext_secp256k1_ecdsa_recover_compressed] executing...")
	logger.Warn("[ext_secp256k1_ecdsa_recover_compressed] not yet implemented")
	return 0
}

//export ext_sandbox_memory_get
func ext_sandbox_memory_get(context unsafe.Pointer, a, b, c, d C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_get] executing...")
	logger.Warn("[ext_sandbox_memory_get] not yet implemented")
	return 0
}

//export ext_sandbox_memory_set
func ext_sandbox_memory_set(context unsafe.Pointer, a, b, c, d C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_set] executing...")
	logger.Warn("[ext_sandbox_memory_set] not yet implemented")
	return 0
}

//export ext_log
func ext_log(context unsafe.Pointer, a, b, c, d, e C.int32_t) {
	logger.Trace("[ext_log] executing...")
	logger.Warn("[ext_log] not yet implemented")
}

//export ext_twox_256
func ext_twox_256(context unsafe.Pointer, data, len, out C.int32_t) {
	logger.Trace("[ext_twox_256] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	logger.Trace("[ext_twox_256] hashing...", "value", fmt.Sprintf("%s", memory[data:data+len]))

	hash, err := common.Twox256(memory[data : data+len])
	if err != nil {
		logger.Error("[ext_twox_256]", "error", err)
		return
	}

	copy(memory[out:out+32], hash[:])
}

//export ext_exists_storage
func ext_exists_storage(context unsafe.Pointer, a, b C.int32_t) C.int32_t {
	logger.Trace("[ext_exists_storage] executing...")
	logger.Warn("[ext_exists_storage] not yet implemented")
	return 0
}

//export ext_exists_child_storage
func ext_exists_child_storage(context unsafe.Pointer, a, b, c, d C.int32_t) C.int32_t {
	logger.Trace("[ext_exists_child_storage] executing...")
	logger.Warn("[ext_exists_child_storage] not yet implemented")
	return 0
}

//export ext_clear_child_prefix
func ext_clear_child_prefix(context unsafe.Pointer, a, b, c, d C.int32_t) {
	logger.Trace("[ext_clear_child_prefix] executing...")
	logger.Warn("[ext_clear_child_prefix] not yet implemented")
}

// ImportsLegacyNodeRuntime returns the wasm imports for the substrate v0.6 node runtime
func ImportsLegacyNodeRuntime() (*wasm.Imports, error) { //nolint
	imports, err := wasm.NewImports().Append("ext_malloc", ext_malloc, C.ext_malloc)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_free", ext_free, C.ext_free)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_utf8", ext_print_utf8, C.ext_print_utf8)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_hex", ext_print_hex, C.ext_print_hex)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_num", ext_print_num, C.ext_print_num)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_storage_into", ext_get_storage_into, C.ext_get_storage_into)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_allocated_storage", ext_get_allocated_storage, C.ext_get_allocated_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_set_storage", ext_set_storage, C.ext_set_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_256", ext_blake2_256, C.ext_blake2_256)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_256_enumerated_trie_root", ext_blake2_256_enumerated_trie_root, C.ext_blake2_256_enumerated_trie_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_storage", ext_clear_storage, C.ext_clear_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_prefix", ext_clear_prefix, C.ext_clear_prefix)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_twox_128", ext_twox_128, C.ext_twox_128)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_root", ext_storage_root, C.ext_storage_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_changes_root", ext_storage_changes_root, C.ext_storage_changes_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_verify", ext_sr25519_verify, C.ext_sr25519_verify)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_verify", ext_ed25519_verify, C.ext_ed25519_verify)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_keccak_256", ext_keccak_256, C.ext_keccak_256)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_secp256k1_ecdsa_recover", ext_secp256k1_ecdsa_recover, C.ext_secp256k1_ecdsa_recover)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_128", ext_blake2_128, C.ext_blake2_128)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_is_validator", ext_is_validator, C.ext_is_validator)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_get", ext_local_storage_get, C.ext_local_storage_get)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_compare_and_set", ext_local_storage_compare_and_set, C.ext_local_storage_compare_and_set)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_public_keys", ext_ed25519_public_keys, C.ext_ed25519_public_keys)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_public_keys", ext_sr25519_public_keys, C.ext_sr25519_public_keys)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_network_state", ext_network_state, C.ext_network_state)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_sign", ext_sr25519_sign, C.ext_sr25519_sign)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_sign", ext_ed25519_sign, C.ext_ed25519_sign)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_submit_transaction", ext_submit_transaction, C.ext_submit_transaction)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_set", ext_local_storage_set, C.ext_local_storage_set)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_generate", ext_ed25519_generate, C.ext_ed25519_generate)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_generate", ext_sr25519_generate, C.ext_sr25519_generate)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_twox_64", ext_twox_64, C.ext_twox_64)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_set_child_storage", ext_set_child_storage, C.ext_set_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_child_storage_into", ext_get_child_storage_into, C.ext_get_child_storage_into)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_kill_child_storage", ext_kill_child_storage, C.ext_kill_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_new", ext_sandbox_memory_new, C.ext_sandbox_memory_new)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_teardown", ext_sandbox_memory_teardown, C.ext_sandbox_memory_teardown)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_instantiate", ext_sandbox_instantiate, C.ext_sandbox_instantiate)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_invoke", ext_sandbox_invoke, C.ext_sandbox_invoke)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_instance_teardown", ext_sandbox_instance_teardown, C.ext_sandbox_instance_teardown)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_allocated_child_storage", ext_get_allocated_child_storage, C.ext_get_allocated_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_child_storage_root", ext_child_storage_root, C.ext_child_storage_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_child_storage", ext_clear_child_storage, C.ext_clear_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_secp256k1_ecdsa_recover_compressed", ext_secp256k1_ecdsa_recover_compressed, C.ext_secp256k1_ecdsa_recover_compressed)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_get", ext_sandbox_memory_get, C.ext_sandbox_memory_get)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_set", ext_sandbox_memory_set, C.ext_sandbox_memory_set)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_log", ext_log, C.ext_log)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_twox_256", ext_twox_256, C.ext_twox_256)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_exists_storage", ext_exists_storage, C.ext_exists_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_exists_child_storage", ext_exists_child_storage, C.ext_exists_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_child_prefix", ext_clear_child_prefix, C.ext_clear_child_prefix)
	if err != nil {
		return nil, err
	}

	return imports, nil
}

// ImportsTestRuntime registers the wasm imports for the v0.6 substrate test runtime
func ImportsTestRuntime() (*wasm.Imports, error) {
	imports, err := wasm.NewImports().Append("ext_malloc", ext_malloc, C.ext_malloc)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_free", ext_free, C.ext_free)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_utf8", ext_print_utf8, C.ext_print_utf8)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_hex", ext_print_hex, C.ext_print_hex)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_print_num", ext_print_num, C.ext_print_num)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_storage_into", ext_get_storage_into, C.ext_get_storage_into)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_allocated_storage", ext_get_allocated_storage, C.ext_get_allocated_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_set_storage", ext_set_storage, C.ext_set_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_256", ext_blake2_256, C.ext_blake2_256)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_256_enumerated_trie_root", ext_blake2_256_enumerated_trie_root, C.ext_blake2_256_enumerated_trie_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_storage", ext_clear_storage, C.ext_clear_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_clear_prefix", ext_clear_prefix, C.ext_clear_prefix)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_twox_128", ext_twox_128, C.ext_twox_128)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_root", ext_storage_root, C.ext_storage_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_changes_root", ext_storage_changes_root, C.ext_storage_changes_root)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_verify", ext_sr25519_verify, C.ext_sr25519_verify)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_verify", ext_ed25519_verify, C.ext_ed25519_verify)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_keccak_256", ext_keccak_256, C.ext_keccak_256)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_secp256k1_ecdsa_recover", ext_secp256k1_ecdsa_recover, C.ext_secp256k1_ecdsa_recover)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_blake2_128", ext_blake2_128, C.ext_blake2_128)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_is_validator", ext_is_validator, C.ext_is_validator)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_get", ext_local_storage_get, C.ext_local_storage_get)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_compare_and_set", ext_local_storage_compare_and_set, C.ext_local_storage_compare_and_set)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_public_keys", ext_ed25519_public_keys, C.ext_ed25519_public_keys)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_public_keys", ext_sr25519_public_keys, C.ext_sr25519_public_keys)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_network_state", ext_network_state, C.ext_network_state)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_sign", ext_sr25519_sign, C.ext_sr25519_sign)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_sign", ext_ed25519_sign, C.ext_ed25519_sign)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_submit_transaction", ext_submit_transaction, C.ext_submit_transaction)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_local_storage_set", ext_local_storage_set, C.ext_local_storage_set)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_ed25519_generate", ext_ed25519_generate, C.ext_ed25519_generate)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sr25519_generate", ext_sr25519_generate, C.ext_sr25519_generate)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_twox_64", ext_twox_64, C.ext_twox_64)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_set_child_storage", ext_set_child_storage, C.ext_set_child_storage)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_get_child_storage_into", ext_get_child_storage_into, C.ext_get_child_storage_into)
	if err != nil {
		return nil, err
	}

	return imports, nil
}
