package runtime

// #include <stdlib.h>
//
// extern int32_t ext_malloc(void *context, int32_t x);
// extern void ext_free(void *context, int32_t addr);
// extern void ext_print_utf8(void *context, int32_t offset, int32_t size);
// extern void ext_print_hex(void *context, int32_t data, int32_t len);
// extern int32_t ext_get_storage_into(void *context, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen, int32_t valueOffset);
// extern void ext_set_storage(void *context, int32_t keyData, int32_t keyLen, int32_t valueData, int32_t valueLen);
// extern void ext_blake2_256(void *context, int32_t data, int32_t len, int32_t out);
// extern void ext_clear_storage(void *context, int32_t keyData, int32_t keyLen);
// extern void ext_twox_128(void *context, int32_t data, int32_t len, int32_t out);
// extern int32_t ext_get_allocated_storage(void *context, int32_t keyData, int32_t keyLen, int32_t writtenOut);
// extern void ext_storage_root(void *context, int32_t resultPtr);
// extern int32_t ext_storage_changes_root(void *context, int32_t a, int32_t b, int32_t c);
// extern void ext_clear_prefix(void *context, int32_t prefixData, int32_t prefixLen);
// extern int32_t ext_sr25519_verify(void *context, int32_t msgData, int32_t msgLen, int32_t sigData, int32_t pubkeyData);
// extern int32_t ext_ed25519_verify(void *context, int32_t msgData, int32_t msgLen, int32_t sigData, int32_t pubkeyData);
// extern void ext_blake2_256_enumerated_trie_root(void *context, int32_t valuesData, int32_t lensData, int32_t lensLen, int32_t result);
// // extern void ext_print_num(void *context, int64_t data);
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
	trie "github.com/ChainSafe/gossamer/trie"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_sr25519_verify
func ext_sr25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData int32) int32{
	return 0
}

//export ext_ed25519_verify
func ext_ed25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData int32) int32{
	return 0
}

//export ext_blake2_256_enumerated_trie_root
func ext_blake2_256_enumerated_trie_root(context unsafe.Pointer, valuesData, lensData, lensLen, result int32) {
	return
}

//export ext_malloc
func ext_malloc(context unsafe.Pointer, x int32) int32 {
	return 100
}

//export ext_free
func ext_free(context unsafe.Pointer, addr int32) {
	return
}

//export ext_print_utf8
func ext_print_utf8(context unsafe.Pointer, offset, size int32) {
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	fmt.Println(memory[offset:offset+size])
	return
}

//export ext_print_hex
func ext_print_hex(context unsafe.Pointer, data, len int32) {
	return
}

//export ext_get_storage_into
func ext_get_storage_into(context unsafe.Pointer, keyData, keyLen, valueData, valueLen, valueOffset int32) int32 {
	return 0
}

//export ext_set_storage
func ext_set_storage(context unsafe.Pointer, keyData, keyLen, valueData, valueLen int32) {
	return
}

//export ext_storage_root
func ext_storage_root(context unsafe.Pointer, resultPtr int32) {
	return
}

//export ext_storage_changes_root
func ext_storage_changes_root(context unsafe.Pointer, a, b, c int32) int32 {
	return 0
}

//export ext_get_allocated_storage
func ext_get_allocated_storage(context unsafe.Pointer, keyData, keyLen, writtenOut int32) int32 {
	return 0
}

//export ext_blake2_256
func ext_blake2_256(context unsafe.Pointer, data, len, out int32) {
	return
}

//export ext_clear_storage
func ext_clear_storage(context unsafe.Pointer, keyData, keyLen int32) {
	return
}

//export ext_clear_prefix
func ext_clear_prefix(context unsafe.Pointer, prefixData, prefixLen int32) {
	return
}

//export ext_twox_128
func ext_twox_128(context unsafe.Pointer, data, len, out int32) {
	return
}

//export ext_print_num
func ext_print_num(context unsafe.Pointer, data int64) {
	return
}

func Exec(t *trie.Trie) ([]byte, error) {
	// Reads the WebAssembly module as bytes.
	bytes, err := wasm.ReadBytes("polkadot_runtime.compact.wasm")
	if err != nil {
		return nil, err
	}
	
	imports, err := wasm.NewImports().Append("ext_malloc", ext_malloc, C.ext_malloc)
	if err != nil {
		return nil, err
	}

	imports.Append("ext_free", ext_free, C.ext_free)
	imports.Append("ext_print_utf8", ext_print_utf8, C.ext_print_utf8)
	imports.Append("ext_print_hex", ext_print_hex, C.ext_print_hex)
	//imports.Append("ext_print_num", ext_print_num, C.ext_print_num)
	imports.Append("ext_get_storage_into", ext_get_storage_into, C.ext_get_storage_into)
	imports.Append("ext_get_allocated_storage", ext_get_allocated_storage, C.ext_get_allocated_storage)
	imports.Append("ext_set_storage", ext_set_storage, C.ext_set_storage)
	imports.Append("ext_blake2_256", ext_blake2_256, C.ext_blake2_256)
	imports.Append("ext_blake2_256_enumerated_trie_root", ext_blake2_256_enumerated_trie_root, C.ext_blake2_256_enumerated_trie_root)
	imports.Append("ext_clear_storage", ext_clear_storage, C.ext_clear_storage)
	imports.Append("ext_clear_prefix", ext_clear_prefix, C.ext_clear_prefix)
	imports.Append("ext_twox_128", ext_twox_128, C.ext_twox_128)
	imports.Append("ext_storage_root", ext_storage_root, C.ext_storage_root)
	imports.Append("ext_storage_changes_root", ext_storage_changes_root, C.ext_storage_changes_root)
	imports.Append("ext_sr25519_verify", ext_sr25519_verify, C.ext_sr25519_verify)
	imports.Append("ext_ed25519_verify", ext_ed25519_verify, C.ext_ed25519_verify)

	// Instantiates the WebAssembly module.
	instance, err := wasm.NewInstanceWithImports(bytes, imports)
	if err != nil {
		return nil, err
	}
	defer instance.Close()

	data := unsafe.Pointer(t)
	instance.SetContextData(data)

	version, ok := instance.Exports["Core_version"]
	if !ok {
		return nil, errors.New("could not find exported function")
	}

	fmt.Printf("%T", version)
	res, err := version()
	if err != nil {
		return nil, err
	}
	resi := res.ToI64()

	offset := int32(resi >> 32)
	length :=  int32(resi)
	fmt.Printf("offset %d length %d", offset, length)
	return instance.Memory.Data()[offset:offset+length], err
}