package runtime

// #include <stdlib.h>
//
// extern int memory;
// extern void ext_misc_print_utf8_version_1(void *context, int64_t a);
// extern void ext_misc_print_hex_version_1(void *context, int64_t a);
// extern int64_t ext_storage_read_version_1(void *context, int64_t a, int64_t b, int32_t c);
// extern int32_t ext_allocator_malloc_version_1(void *context, int32_t size);
// extern void ext_allocator_free_version_1(void *context, int32_t addr);
// extern void ext_logging_log_version_1(void *context, int32_t a, int64_t b, int64_t c);
// extern int32_t ext_trie_blake2_256_ordered_root_version_1(void *context, int64_t a);
// extern int64_t ext_offchain_submit_transaction_version_1(void *context, int64_t a);
// extern int32_t ext_crypto_ed25519_generate_version_1(void *context, int32_t a, int64_t b);
// extern int64_t ext_crypto_ed25519_public_keys_version_1(void *context, int32_t a);
// extern int64_t ext_crypto_ed25519_sign_version_1(void *context, int32_t a, int32_t b, int64_t c);
// extern int32_t ext_crypto_ed25519_verify_version_1(void *context, int32_t a, int64_t b, int32_t c);
// extern int32_t ext_crypto_sr25519_generate_version_1(void *context, int32_t a, int64_t b);
// extern int64_t ext_crypto_sr25519_public_keys_version_1(void *context, int32_t a);
// extern int64_t ext_crypto_sr25519_sign_version_1(void *context, int32_t a, int32_t b, int64_t c);
// extern int32_t ext_crypto_sr25519_verify_version_2(void *context, int32_t a, int64_t b, int32_t c);
// extern int32_t ext_hashing_twox_128_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_blake2_256_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_blake2_128_version_1(void *context, int64_t a);
// extern void ext_storage_set_version_1(void *context, int64_t a, int64_t b);
// extern void ext_storage_clear_version_1(void *context, int64_t a);
// extern int64_t ext_storage_get_version_1(void *context, int64_t a);
// extern int64_t ext_storage_changes_root_version_1(void *context, int64_t a);
// extern void ext_storage_child_set_version_1(void *context, int64_t a, int64_t b, int32_t c, int64_t d, int64_t e);
// extern int64_t ext_storage_child_read_version_1(void *context, int64_t a, int64_t b, int32_t c, int64_t d, int64_t e, int32_t f);
// extern int64_t ext_storage_root_version_1(void *context);
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
	//"github.com/ChainSafe/gossamer/lib/scale"
	// "github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
	// "github.com/OneOfOne/xxhash"
	// "github.com/ethereum/go-ethereum/crypto/secp256k1"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export memory
var memory, memErr = wasm.NewMemory(17, 0)

//export ext_misc_print_utf8_version_1
func ext_misc_print_utf8_version_1(context unsafe.Pointer, a C.int64_t) {
	log.Trace("[ext_misc_print_utf8_version_1] executing...")
}

//export ext_misc_print_hex_version_1
func ext_misc_print_hex_version_1(context unsafe.Pointer, a C.int64_t) {
	log.Trace("[ext_misc_print_hex_version_1] executing...")
}

//export ext_storage_read_version_1
func ext_storage_read_version_1(context unsafe.Pointer, a, b C.int64_t, c C.int32_t) C.int64_t {
	log.Trace("[ext_storage_read_version_1] executing...")
	return 0
}

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
		//panic(err)
	}
}

//export ext_logging_log_version_1
func ext_logging_log_version_1(context unsafe.Pointer, level C.int32_t, target, message C.int64_t) {
	log.Trace("[ext_logging_log_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	targetData := memory[uint32(target) : uint32(target)+uint32(target>>32)]
	messageData := memory[uint32(message) : uint32(message)+uint32(message>>32)]

	log.Info("[ext_logging_log_version_1]", "target", string(targetData), "message", string(messageData))
}

//export ext_trie_blake2_256_ordered_root_version_1
func ext_trie_blake2_256_ordered_root_version_1(context unsafe.Pointer, a C.int64_t) C.int32_t {
	log.Trace("[ext_trie_blake2_256_ordered_root_version_1] executing...")
	return 0
}

//export ext_offchain_submit_transaction_version_1
func ext_offchain_submit_transaction_version_1(context unsafe.Pointer, a C.int64_t) C.int64_t {
	log.Trace("[ext_offchain_submit_transaction_version_1] executing...")
	return 0
}

//export ext_crypto_ed25519_generate_version_1
func ext_crypto_ed25519_generate_version_1(context unsafe.Pointer, a C.int32_t, b C.int64_t) C.int32_t {
	log.Trace("[ext_crypto_ed25519_generate_version_1] executing...")
	return 0
}

//export ext_crypto_ed25519_public_keys_version_1
func ext_crypto_ed25519_public_keys_version_1(context unsafe.Pointer, a C.int32_t) C.int64_t {
	log.Trace("[ext_crypto_ed25519_public_keys_version_1] executing...")
	return 0
}

//export ext_crypto_ed25519_sign_version_1
func ext_crypto_ed25519_sign_version_1(context unsafe.Pointer, a, b C.int32_t, c C.int64_t) C.int64_t {
	log.Trace("[ext_crypto_ed25519_sign_version_1] executing...")
	return 0
}

//export ext_crypto_ed25519_verify_version_1
func ext_crypto_ed25519_verify_version_1(context unsafe.Pointer, a C.int32_t, b C.int64_t, c C.int32_t) C.int32_t {
	log.Trace("[ext_crypto_ed25519_verify_version_1] executing...")
	return 0
}

//export ext_crypto_sr25519_generate_version_1
func ext_crypto_sr25519_generate_version_1(context unsafe.Pointer, a C.int32_t, b C.int64_t) C.int32_t {
	log.Trace("[ext_crypto_sr25519_generate_version_1] executing...")
	return 0
}

//export ext_crypto_sr25519_public_keys_version_1
func ext_crypto_sr25519_public_keys_version_1(context unsafe.Pointer, a C.int32_t) C.int64_t {
	log.Trace("[ext_crypto_sr25519_public_keys_version_1] executing...")
	return 0
}

//export ext_crypto_sr25519_sign_version_1
func ext_crypto_sr25519_sign_version_1(context unsafe.Pointer, a, b C.int32_t, c C.int64_t) C.int64_t {
	log.Trace("[ext_crypto_sr25519_sign_version_1] executing...")
	return 0
}

//export ext_crypto_sr25519_verify_version_2
func ext_crypto_sr25519_verify_version_2(context unsafe.Pointer, a C.int32_t, b C.int64_t, c C.int32_t) C.int32_t {
	log.Trace("[ext_crypto_sr25519_verify_version_2] executing...")
	return 0
}

//export ext_hashing_twox_128_version_1
func ext_hashing_twox_128_version_1(context unsafe.Pointer, a C.int64_t) C.int32_t {
	log.Trace("[ext_hashing_twox_128_version_1] executing...")
	return 0
}

//export ext_hashing_blake2_256_version_1
func ext_hashing_blake2_256_version_1(context unsafe.Pointer, a C.int64_t) C.int32_t {
	log.Trace("[ext_hashing_blake2_256_version_1] executing...")
	return 0
}

//export ext_hashing_blake2_128_version_1
func ext_hashing_blake2_128_version_1(context unsafe.Pointer, a C.int64_t) C.int32_t {
	log.Trace("[ext_hashing_blake2_128_version_1] executing...")
	return 0
}

//export ext_storage_set_version_1
func ext_storage_set_version_1(context unsafe.Pointer, a, b C.int64_t) {
	log.Trace("[ext_storage_set_version_1] executing...")
}

//export ext_storage_clear_version_1
func ext_storage_clear_version_1(context unsafe.Pointer, a C.int64_t) {
	log.Trace("[ext_storage_clear_version_1] executing...")
}

//export ext_storage_get_version_1
func ext_storage_get_version_1(context unsafe.Pointer, a C.int64_t) C.int64_t {
	log.Trace("[ext_storage_get_version_1] executing...")
	return 0
}

//export ext_storage_changes_root_version_1
func ext_storage_changes_root_version_1(context unsafe.Pointer, a C.int64_t) C.int64_t {
	log.Trace("[ext_storage_changes_root_version_1] executing...")
	return 0
}

//export ext_storage_child_set_version_1
func ext_storage_child_set_version_1(context unsafe.Pointer, a, b C.int64_t, c C.int32_t, d, e C.int64_t) {
	log.Trace("[ext_storage_child_set_version_1] executing...")
}

//export ext_storage_child_read_version_1
func ext_storage_child_read_version_1(context unsafe.Pointer, a, b C.int64_t, c C.int32_t, d, e C.int64_t, f C.int32_t) C.int64_t {
	log.Trace("[ext_storage_child_read_version_1] executing...")
	return 0
}

//export ext_storage_root_version_1
func ext_storage_root_version_1(context unsafe.Pointer) C.int64_t {
	log.Trace("[ext_storage_root_version_1] executing...")
	return 0
}

// RegisterImports registers the wasm imports for the most recent substrate test runtime.
func RegisterImports() (*wasm.Imports, error) {
	return registerImports()
}

func registerImports() (*wasm.Imports, error) {
	// check for memory error
	if memErr != nil {
		return nil, memErr
	}

	imports, err := wasm.NewImports().Append("ext_allocator_malloc_version_1", ext_allocator_malloc_version_1, C.ext_allocator_malloc_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_allocator_free_version_1", ext_allocator_free_version_1, C.ext_allocator_free_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.AppendMemory("memory", memory)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_misc_print_hex_version_1", ext_misc_print_hex_version_1, C.ext_misc_print_hex_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_misc_print_utf8_version_1", ext_misc_print_utf8_version_1, C.ext_misc_print_utf8_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_read_version_1", ext_storage_read_version_1, C.ext_storage_read_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_logging_log_version_1", ext_logging_log_version_1, C.ext_logging_log_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_trie_blake2_256_ordered_root_version_1", ext_trie_blake2_256_ordered_root_version_1, C.ext_trie_blake2_256_ordered_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_submit_transaction_version_1", ext_offchain_submit_transaction_version_1, C.ext_offchain_submit_transaction_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_ed25519_generate_version_1", ext_crypto_ed25519_generate_version_1, C.ext_crypto_ed25519_generate_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_ed25519_public_keys_version_1", ext_crypto_ed25519_public_keys_version_1, C.ext_crypto_ed25519_public_keys_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_ed25519_sign_version_1", ext_crypto_ed25519_sign_version_1, C.ext_crypto_ed25519_sign_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_ed25519_verify_version_1", ext_crypto_ed25519_verify_version_1, C.ext_crypto_ed25519_verify_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_sr25519_generate_version_1", ext_crypto_sr25519_generate_version_1, C.ext_crypto_sr25519_generate_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_sr25519_public_keys_version_1", ext_crypto_sr25519_public_keys_version_1, C.ext_crypto_sr25519_public_keys_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_sr25519_sign_version_1", ext_crypto_sr25519_sign_version_1, C.ext_crypto_sr25519_sign_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_sr25519_verify_version_2", ext_crypto_sr25519_verify_version_2, C.ext_crypto_sr25519_verify_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_twox_128_version_1", ext_hashing_twox_128_version_1, C.ext_hashing_twox_128_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_blake2_256_version_1", ext_hashing_blake2_256_version_1, C.ext_hashing_blake2_256_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_blake2_128_version_1", ext_hashing_blake2_128_version_1, C.ext_hashing_blake2_128_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_set_version_1", ext_storage_set_version_1, C.ext_storage_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_clear_version_1", ext_storage_clear_version_1, C.ext_storage_clear_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_get_version_1", ext_storage_get_version_1, C.ext_storage_get_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_changes_root_version_1", ext_storage_changes_root_version_1, C.ext_storage_changes_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_child_set_version_1", ext_storage_child_set_version_1, C.ext_storage_child_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_child_read_version_1", ext_storage_child_read_version_1, C.ext_storage_child_read_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_root_version_1", ext_storage_root_version_1, C.ext_storage_root_version_1)
	if err != nil {
		return nil, err
	}
	return imports, nil
}
