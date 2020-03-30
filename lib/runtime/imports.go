package runtime

// #include <stdlib.h>
//
// extern int32_t ext_allocator_malloc_version_1(void *context, int32_t size);
// extern void ext_allocator_free_version_1(void *context, int32_t addr);
/*
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
*/
import "C"

import (
	// "bytes"
	// "encoding/binary"
	"fmt"
	// "math/big"
	"unsafe"

	// "github.com/ChainSafe/gossamer/lib/common"
	// "github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	// "github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	// "github.com/ChainSafe/gossamer/lib/scale"
	// "github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
	// "github.com/OneOfOne/xxhash"
	// "github.com/ethereum/go-ethereum/crypto/secp256k1"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_allocator_malloc_version_1
func ext_allocator_malloc_version_1(context unsafe.Pointer, size int32) int32 {
	log.Trace("[ext_allocator_malloc_version_1] executing...", "size", size)
	instanceContext := wasm.IntoInstanceContext(context)
	data := instanceContext.Data()
	runtimeCtx, ok := data.(*Ctx)
	if !ok {
		panic(fmt.Sprintf("%#v", data))
	}

	// Allocate memory
	res, err := runtimeCtx.allocator.Allocate(uint32(size))
	if err != nil {
		log.Error("[ext_allocator_malloc_version_1]", "Error:", err)
		panic(err)
	}

	return int32(res)
}

//export ext_allocator_free_version_1
func ext_allocator_free_version_1(context unsafe.Pointer, addr int32) {
	log.Trace("[ext_allocator_free_version_1] executing...", "addr", addr)
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*Ctx)

	// Deallocate memory
	err := runtimeCtx.allocator.Deallocate(uint32(addr))
	if err != nil {
		log.Error("[ext_allocator_free_version_1] Error:", "Error", err)
		panic(err)
	}
}

func RegisterImports() (*wasm.Imports, error) {
	return registerImports()
}

func registerImports() (*wasm.Imports, error) {
	imports, err := wasm.NewImports().Append("ext_allocator_malloc_version_1", ext_allocator_malloc_version_1, C.ext_allocator_malloc_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_allocator_free_version_1", ext_allocator_free_version_1, C.ext_allocator_free_version_1)
	if err != nil {
		return nil, err
	}
	return imports, nil
}
