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
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"
	log "github.com/inconshreveable/log15"
	common "github.com/ChainSafe/gossamer/common"
	trie "github.com/ChainSafe/gossamer/trie"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_print_num
// func ext_print_num(context unsafe.Pointer, data int64) {
// 	log.Debug("[ext_print_num] executing...")
// 	log.Debug("[ext_print_num]", "message", fmt.Sprintf("%d", data))
// 	return
// }

//export ext_malloc
func ext_malloc(context unsafe.Pointer, x int32) int32 {
	log.Debug("[ext_malloc] executing...")
	return 1
}

//export ext_free
func ext_free(context unsafe.Pointer, addr int32) {
	log.Debug("[ext_free] executing...")
	return
}

// prints string located in memory at location `offset` with length `size`
//export ext_print_utf8
func ext_print_utf8(context unsafe.Pointer, offset, size int32) {
	log.Debug("[ext_print_utf8] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	log.Debug("[ext_print_utf8]", "message", fmt.Sprintf("%s", memory[offset:offset+size]))
	return
}

// prints hex formatted bytes located in memory at location `offset` with length `size`
//export ext_print_hex
func ext_print_hex(context unsafe.Pointer, offset, size int32) {
	log.Debug("[ext_print_hex] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	log.Debug("[ext_print_hex]", "message", fmt.Sprintf("%x", memory[offset:offset+size]))
	return
}

// gets the key stored at memory location `keyData` with length `keyLen` and stores the value in memory at
// location `valueData`. the value can have up to value `valueLen` and the returned value starts at value[valueOffset:]
//export ext_get_storage_into
func ext_get_storage_into(context unsafe.Pointer, keyData, keyLen, valueData, valueLen, valueOffset int32) int32 {
	log.Debug("[ext_get_storage_into] executing...")

	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := (*trie.Trie)(instanceContext.Data()) 

	key := memory[keyData:keyData+keyLen]
	val, err := t.Get(key)
	if err != nil || val == nil {
		return 2^32 -1
	}

	if len(val) > int(valueLen) {
		log.Error("[ext_get_storage_into]", "error", "value exceeds allocated buffer length")
	}

	copy(memory[valueData:valueData+valueLen], val[valueOffset:])
	return int32(len(val[valueOffset:]))
}

// puts the key at memory location `keyData` with length `keyLen` and value at memory location `valueData`
// with length `valueLen` into the storage trie
//export ext_set_storage
func ext_set_storage(context unsafe.Pointer, keyData, keyLen, valueData, valueLen int32) {
	log.Debug("[ext_set_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := (*trie.Trie)(instanceContext.Data()) 

	key := memory[keyData:keyData+keyLen]
	val := memory[valueData:valueData+valueLen]
	err := t.Put(key, val)
	if err != nil {
		log.Error("[ext_set_storage]", "error", err)
	}

	return
}

// returns the trie root in the memory location `resultPtr`
//export ext_storage_root
func ext_storage_root(context unsafe.Pointer, resultPtr int32) {
	log.Debug("[ext_storage_root] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := (*trie.Trie)(instanceContext.Data()) 

	root, err := t.Hash()
	if err != nil {
		log.Error("[ext_storage_root]", "error", err)
	}

	copy(memory[resultPtr:resultPtr+32], root[:])
	return
}

//export ext_storage_changes_root
func ext_storage_changes_root(context unsafe.Pointer, a, b, c int32) int32 {
	log.Debug("[ext_storage_changes_root] executing...")
	return 0
}

// gets value stored at key at memory location `keyData` with length `keyLen` and returns the location
// in memory where it's stored and stores its length in `writtenOut` 
//export ext_get_allocated_storage
func ext_get_allocated_storage(context unsafe.Pointer, keyData, keyLen, writtenOut int32) int32 {
	log.Debug("[ext_get_allocated_storage] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := (*trie.Trie)(instanceContext.Data()) 

	key := memory[keyData:keyData+keyLen]
	val, err := t.Get(key)
	if err == nil && len(val) >= (1 << 32) {
		err = errors.New("retrieved value length exceeds 2^32")
	}
	if err != nil {
		log.Error("[ext_get_allocated_storage]", "error", err)
		return 0
	}		

	ptr := 1
	copy(memory[ptr:ptr+len(val)], val)
	return int32(len(val))
}

// deletes the trie entry with key at memory location `keyData` with length `keyLen`
//export ext_clear_storage
func ext_clear_storage(context unsafe.Pointer, keyData, keyLen int32) {
	log.Debug("[ext_sr25519_verify] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := (*trie.Trie)(instanceContext.Data()) 

	key := memory[keyData:keyData+keyLen]
	err := t.Delete(key)
	if err != nil {
		log.Error("[ext_storage_root]", "error", err)
	}

	return
}

//export ext_clear_prefix
func ext_clear_prefix(context unsafe.Pointer, prefixData, prefixLen int32) {
	log.Debug("[ext_clear_prefix] executing...")
	return
}

// accepts an array of keys and values, puts them into a trie, and returns the root
//export ext_blake2_256_enumerated_trie_root
func ext_blake2_256_enumerated_trie_root(context unsafe.Pointer, valuesData, lensData, lensLen, result int32) {
	log.Debug("[ext_blake2_256_enumerated_trie_root] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	t := &trie.Trie{}

	var i int32
	var pos int32 = 0
	for i = 0; i < lensLen; i++ {
		valueLenBytes := memory[lensData + i*32 : lensData + (i+1)*32]
		valueLen := int32(binary.LittleEndian.Uint32(valueLenBytes))
		value := memory[valuesData + pos : valuesData + pos + valueLen]
		pos += valueLen

		// todo: do we get key/value pairs or just values??
		err := t.Put(value, nil)
		if err != nil {
			log.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
		}
	}

	root, err := t.Hash()
	if err != nil {
		log.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
	}

	copy(memory[result:result+32], root[:])
	return
}

// performs blake2b 256-bit hash of the byte array at memory location `data` with length `length` and saves the
// hash at memory location `out`
//export ext_blake2_256
func ext_blake2_256(context unsafe.Pointer, data, length, out int32) {
	log.Debug("[ext_blake2_256] executing...")
	instanceContext := wasm.IntoInstanceContext(context) 
	memory := instanceContext.Memory().Data() 
	hash, err := common.Blake2bHash(memory[data:data+length])
	if err != nil {
		log.Error("[ext_blake2_256]", "error", err)
	}

	copy(memory[out:out+32], hash[:])
	return
}

//export ext_twox_128
func ext_twox_128(context unsafe.Pointer, data, len, out int32) {
	log.Debug("[ext_twox_128] executing...")
	return
}

//export ext_sr25519_verify
func ext_sr25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData int32) int32 {
	log.Debug("[ext_sr25519_verify] executing...")
	return 0
}

//export ext_ed25519_verify
func ext_ed25519_verify(context unsafe.Pointer, msgData, msgLen, sigData, pubkeyData int32) int32{
	log.Debug("[ext_ed25519_verify] executing...")
	return 0
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