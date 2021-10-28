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
// extern void ext_logging_log_version_1(void *context, int32_t level, int64_t target, int64_t msg);
// extern int32_t ext_logging_max_level_version_1(void *context);
//
// extern void ext_sandbox_instance_teardown_version_1(void *context, int32_t a);
// extern int32_t ext_sandbox_instantiate_version_1(void *context, int32_t a, int64_t b, int64_t c, int32_t d);
// extern int32_t ext_sandbox_invoke_version_1(void *context, int32_t a, int64_t b, int64_t c, int32_t d, int32_t e, int32_t f);
// extern int32_t ext_sandbox_memory_get_version_1(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern int32_t ext_sandbox_memory_new_version_1(void *context, int32_t a, int32_t b);
// extern int32_t ext_sandbox_memory_set_version_1(void *context, int32_t a, int32_t b, int32_t c, int32_t d);
// extern void ext_sandbox_memory_teardown_version_1(void *context, int32_t a);
//
// extern int32_t ext_crypto_ed25519_generate_version_1(void *context, int32_t a, int64_t b);
// extern int64_t ext_crypto_ed25519_public_keys_version_1(void *context, int32_t a);
// extern int64_t ext_crypto_ed25519_sign_version_1(void *context, int32_t a, int32_t b, int64_t c);
// extern int32_t ext_crypto_ed25519_verify_version_1(void *context, int32_t a, int64_t b, int32_t c);
// extern int32_t ext_crypto_finish_batch_verify_version_1(void *context);
// extern int64_t ext_crypto_secp256k1_ecdsa_recover_version_1(void *context, int32_t a, int32_t b);
// extern int64_t ext_crypto_secp256k1_ecdsa_recover_version_2(void *context, int32_t a, int32_t b);
// extern int64_t ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(void *context, int32_t a, int32_t b);
// extern int64_t ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(void *context, int32_t a, int32_t b);
// extern int32_t ext_crypto_ecdsa_verify_version_2(void *context, int32_t a, int64_t b, int32_t c);
// extern int32_t ext_crypto_sr25519_generate_version_1(void *context, int32_t a, int64_t b);
// extern int64_t ext_crypto_sr25519_public_keys_version_1(void *context, int32_t a);
// extern int64_t ext_crypto_sr25519_sign_version_1(void *context, int32_t a, int32_t b, int64_t c);
// extern int32_t ext_crypto_sr25519_verify_version_1(void *context, int32_t a, int64_t b, int32_t c);
// extern int32_t ext_crypto_sr25519_verify_version_2(void *context, int32_t a, int64_t b, int32_t c);
// extern void ext_crypto_start_batch_verify_version_1(void *context);
//
// extern int32_t ext_trie_blake2_256_root_version_1(void *context, int64_t a);
// extern int32_t ext_trie_blake2_256_ordered_root_version_1(void *context, int64_t a);
// extern int32_t ext_trie_blake2_256_verify_proof_version_1(void *context, int32_t a, int64_t b, int64_t c, int64_t d);
//
// extern int64_t ext_misc_runtime_version_version_1(void *context, int64_t a);
// extern void ext_misc_print_hex_version_1(void *context, int64_t a);
// extern void ext_misc_print_num_version_1(void *context, int64_t a);
// extern void ext_misc_print_utf8_version_1(void *context, int64_t a);
//
// extern void ext_default_child_storage_clear_version_1(void *context, int64_t a, int64_t b);
// extern int64_t ext_default_child_storage_get_version_1(void *context, int64_t a, int64_t b);
// extern int64_t ext_default_child_storage_next_key_version_1(void *context, int64_t a, int64_t b);
// extern int64_t ext_default_child_storage_read_version_1(void *context, int64_t a, int64_t b, int64_t c, int32_t d);
// extern int64_t ext_default_child_storage_root_version_1(void *context, int64_t a);
// extern void ext_default_child_storage_set_version_1(void *context, int64_t a, int64_t b, int64_t c);
// extern void ext_default_child_storage_storage_kill_version_1(void *context, int64_t a);
// extern int32_t ext_default_child_storage_storage_kill_version_2(void *context, int64_t a, int64_t b);
// extern int64_t ext_default_child_storage_storage_kill_version_3(void *context, int64_t a, int64_t b);
// extern void ext_default_child_storage_clear_prefix_version_1(void *context, int64_t a, int64_t b);
// extern int32_t ext_default_child_storage_exists_version_1(void *context, int64_t a, int64_t b);
//
// extern void ext_allocator_free_version_1(void *context, int32_t a);
// extern int32_t ext_allocator_malloc_version_1(void *context, int32_t a);
//
// extern int32_t ext_hashing_blake2_128_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_blake2_256_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_keccak_256_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_sha2_256_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_twox_256_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_twox_128_version_1(void *context, int64_t a);
// extern int32_t ext_hashing_twox_64_version_1(void *context, int64_t a);
//
// extern void ext_offchain_index_set_version_1(void *context, int64_t a, int64_t b);
// extern int32_t ext_offchain_is_validator_version_1(void *context);
// extern void ext_offchain_local_storage_clear_version_1(void *context, int32_t a, int64_t b);
// extern int32_t ext_offchain_local_storage_compare_and_set_version_1(void *context, int32_t a, int64_t b, int64_t c, int64_t d);
// extern int64_t ext_offchain_local_storage_get_version_1(void *context, int32_t a, int64_t b);
// extern void ext_offchain_local_storage_set_version_1(void *context, int32_t a, int64_t b, int64_t c);
// extern int64_t ext_offchain_network_state_version_1(void *context);
// extern int32_t ext_offchain_random_seed_version_1(void *context);
// extern int64_t ext_offchain_submit_transaction_version_1(void *context, int64_t a);
// extern int64_t ext_offchain_timestamp_version_1(void *context);
// extern void ext_offchain_sleep_until_version_1(void *context, int64_t a);
//
// extern void ext_storage_append_version_1(void *context, int64_t a, int64_t b);
// extern int64_t ext_storage_changes_root_version_1(void *context, int64_t a);
// extern void ext_storage_clear_version_1(void *context, int64_t a);
// extern void ext_storage_clear_prefix_version_1(void *context, int64_t a);
// extern int64_t ext_storage_clear_prefix_version_2(void *context, int64_t a, int64_t b);
// extern void ext_storage_commit_transaction_version_1(void *context);
// extern int32_t ext_storage_exists_version_1(void *context, int64_t a);
// extern int64_t ext_storage_get_version_1(void *context, int64_t a);
// extern int64_t ext_storage_next_key_version_1(void *context, int64_t a);
// extern int64_t ext_storage_read_version_1(void *context, int64_t a, int64_t b, int32_t c);
// extern void ext_storage_rollback_transaction_version_1(void *context);
// extern int64_t ext_storage_root_version_1(void *context);
// extern void ext_storage_set_version_1(void *context, int64_t a, int64_t b);
// extern void ext_storage_start_transaction_version_1(void *context);
//
// extern void ext_transaction_index_index_version_1(void *context, int32_t a, int32_t b, int32_t c);
// extern void ext_transaction_index_renew_version_1(void *context, int32_t a, int32_t b);
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"unsafe"

	"github.com/ChainSafe/gossamer/lib/common"
	rtype "github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_logging_log_version_1
func ext_logging_log_version_1(context unsafe.Pointer, level C.int32_t, targetData, msgData C.int64_t) {
	logger.Trace("[ext_logging_log_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	target := string(asMemorySlice(instanceContext, targetData))
	msg := string(asMemorySlice(instanceContext, msgData))

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
}

//export ext_logging_max_level_version_1
func ext_logging_max_level_version_1(context unsafe.Pointer) C.int32_t {
	logger.Trace("[ext_logging_max_level_version_1] executing...")
	return 4
}

//export ext_transaction_index_index_version_1
func ext_transaction_index_index_version_1(context unsafe.Pointer, a, b, c C.int32_t) {
	logger.Trace("[ext_transaction_index_index_version_1] executing...")
	logger.Warn("[ext_transaction_index_index_version_1] unimplemented")
}

//export ext_transaction_index_renew_version_1
func ext_transaction_index_renew_version_1(context unsafe.Pointer, a, b C.int32_t) {
	logger.Trace("[ext_transaction_index_renew_version_1] executing...")
	logger.Warn("[ext_transaction_index_renew_version_1] unimplemented")
}

//export ext_sandbox_instance_teardown_version_1
func ext_sandbox_instance_teardown_version_1(context unsafe.Pointer, a C.int32_t) {
	logger.Trace("[ext_sandbox_instance_teardown_version_1] executing...")
	logger.Warn("[ext_sandbox_instance_teardown_version_1] unimplemented")
}

//export ext_sandbox_instantiate_version_1
func ext_sandbox_instantiate_version_1(context unsafe.Pointer, a C.int32_t, x, y C.int64_t, z C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_instantiate_version_1] executing...")
	logger.Warn("[ext_sandbox_instantiate_version_1] unimplemented")
	return 0
}

//export ext_sandbox_invoke_version_1
func ext_sandbox_invoke_version_1(context unsafe.Pointer, a C.int32_t, x, y C.int64_t, z, d, e C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_invoke_version_1] executing...")
	logger.Warn("[ext_sandbox_invoke_version_1] unimplemented")
	return 0
}

//export ext_sandbox_memory_get_version_1
func ext_sandbox_memory_get_version_1(context unsafe.Pointer, a, z, d, e C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_get_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_get_version_1] unimplemented")
	return 0
}

//export ext_sandbox_memory_new_version_1
func ext_sandbox_memory_new_version_1(context unsafe.Pointer, a, z C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_new_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_new_version_1] unimplemented")
	return 0
}

//export ext_sandbox_memory_set_version_1
func ext_sandbox_memory_set_version_1(context unsafe.Pointer, a, z, d, e C.int32_t) C.int32_t {
	logger.Trace("[ext_sandbox_memory_set_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_set_version_1] unimplemented")
	return 0
}

//export ext_sandbox_memory_teardown_version_1
func ext_sandbox_memory_teardown_version_1(context unsafe.Pointer, a C.int32_t) {
	logger.Trace("[ext_sandbox_memory_teardown_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_teardown_version_1] unimplemented")
}

//export ext_crypto_ed25519_generate_version_1
func ext_crypto_ed25519_generate_version_1(context unsafe.Pointer, keyTypeID C.int32_t, seedSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_crypto_ed25519_generate_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(instanceContext, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] cannot generate key", "error", err)
		return 0
	}

	var kp crypto.Keypair

	if seed != nil {
		kp, err = ed25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = ed25519.GenerateKeypair()
	}

	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] cannot generate key", "error", err)
		return 0
	}

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1]", "name", id, "error", err)
		return 0
	}

	ks.Insert(kp)

	ret, err := toWasmMemorySized(instanceContext, kp.Public().Encode(), 32)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] failed to allocate memory", "error", err)
		return 0
	}

	logger.Debug("[ext_crypto_ed25519_generate_version_1] generated ed25519 keypair", "public", kp.Public().Hex())
	return C.int32_t(ret)
}

//export ext_crypto_ed25519_public_keys_version_1
func ext_crypto_ed25519_public_keys_version_1(context unsafe.Pointer, keyTypeID C.int32_t) C.int64_t {
	logger.Debug("[ext_crypto_ed25519_public_keys_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_public_keys_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warn("[ext_crypto_ed25519_public_keys_version_1]", "name", id, "error", "keystore type is not ed25519", "type", ks.Type())
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	ret, err := toWasmMemory(instanceContext, append(prefix, encodedKeys...))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory", err)
		ret, _ = toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	return C.int64_t(ret)
}

//export ext_crypto_ed25519_sign_version_1
func ext_crypto_ed25519_sign_version_1(context unsafe.Pointer, keyTypeID, key C.int32_t, msg C.int64_t) C.int64_t {
	logger.Debug("[ext_crypto_ed25519_sign_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	id := memory[keyTypeID : keyTypeID+4]

	pubKeyData := memory[key : key+32]
	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] failed to get public keys", "error", err)
		return 0
	}

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_sign_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemoryOptional(instanceContext, nil)
		return C.int64_t(ret)
	}

	var ret int64
	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not find public key in keystore", "error", pubKey)
		ret, err = toWasmMemoryOptional(instanceContext, nil)
		if err != nil {
			logger.Error("[ext_crypto_ed25519_sign_version_1] failed to allocate memory", err)
			return 0
		}
		return C.int64_t(ret)
	}

	sig, err := signingKey.Sign(asMemorySlice(instanceContext, msg))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not sign message")
	}

	ret, err = toWasmMemoryFixedSizeOptional(instanceContext, sig)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] failed to allocate memory", err)
		return 0
	}

	return C.int64_t(ret)
}

//export ext_crypto_ed25519_verify_version_1
func ext_crypto_ed25519_verify_version_1(context unsafe.Pointer, sig C.int32_t, msg C.int64_t, key C.int32_t) C.int32_t {
	logger.Debug("[ext_crypto_ed25519_verify_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	signature := memory[sig : sig+64]
	message := asMemorySlice(instanceContext, msg)
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

//export ext_crypto_secp256k1_ecdsa_recover_version_1
func ext_crypto_secp256k1_ecdsa_recover_version_1(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	pub, err := secp256k1.RecoverPublicKey(message, signature)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to recover public key", "error", err)
		var ret int64
		ret, err = toWasmMemoryResult(instanceContext, nil)
		if err != nil {
			logger.Error("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to allocate memory", "error", err)
			return 0
		}
		return C.int64_t(ret)
	}

	logger.Debug("[ext_crypto_secp256k1_ecdsa_recover_version_1]", "len", len(pub), "recovered public key", fmt.Sprintf("0x%x", pub))

	ret, err := toWasmMemoryResult(instanceContext, pub[1:])
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to allocate memory", "error", err)
		return 0
	}

	return C.int64_t(ret)
}

//export ext_crypto_secp256k1_ecdsa_recover_version_2
func ext_crypto_secp256k1_ecdsa_recover_version_2(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_version_2] executing...")
	return ext_crypto_secp256k1_ecdsa_recover_version_1(context, sig, msg)
}

//export ext_crypto_ecdsa_verify_version_2
func ext_crypto_ecdsa_verify_version_2(context unsafe.Pointer, sig C.int32_t, msg C.int64_t, key C.int32_t) C.int32_t {
	logger.Trace("executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]
	pubKey := memory[key : key+33]

	pub := new(secp256k1.PublicKey)
	err := pub.Decode(pubKey)
	if err != nil {
		logger.Error("failed to decode public key", "error", err)
		return C.int32_t(0)
	}

	logger.Debug("", "pub", pub.Hex(),
		"message", fmt.Sprintf("0x%x", message),
		"signature", fmt.Sprintf("0x%x", signature),
	)

	hash, err := common.Blake2bHash(message)
	if err != nil {
		logger.Error("failed to hash message", "error", err)
		return C.int32_t(0)
	}

	if sigVerifier.IsStarted() {
		signature := runtime.Signature{
			PubKey:    pub.Encode(),
			Sign:      signature,
			Msg:       hash[:],
			KeyTypeID: crypto.Secp256k1Type,
		}
		sigVerifier.Add(&signature)
		return C.int32_t(1)
	}

	if ok, err := pub.Verify(hash[:], signature); err != nil || !ok {
		logger.Error("failed to validate signature", "error", err)
		return C.int32_t(0)
	}

	logger.Debug("validated signature")
	return C.int32_t(1)
}

//export ext_crypto_secp256k1_ecdsa_recover_compressed_version_1
func ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	cpub, err := secp256k1.RecoverPublicKeyCompressed(message, signature)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] failed to recover public key", "error", err)
		ret, _ := toWasmMemoryResult(instanceContext, nil)
		return C.int64_t(ret)
	}

	logger.Debug("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1]", "len", len(cpub), "recovered public key", fmt.Sprintf("0x%x", cpub))

	ret, err := toWasmMemoryResult(instanceContext, cpub)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] failed to allocate memory", "error", err)
		return 0
	}

	return C.int64_t(ret)
}

//export ext_crypto_secp256k1_ecdsa_recover_compressed_version_2
func ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_2] executing...")
	return ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(context, sig, msg)
}

//export ext_crypto_sr25519_generate_version_1
func ext_crypto_sr25519_generate_version_1(context unsafe.Pointer, keyTypeID C.int32_t, seedSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_crypto_sr25519_generate_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(instanceContext, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_generate_version_1] cannot generate key", "error", err)
		return 0
	}

	var kp crypto.Keypair
	if seed != nil {
		kp, err = sr25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = sr25519.GenerateKeypair()
	}

	if err != nil {
		logger.Trace("[ext_crypto_sr25519_generate_version_1] cannot generate key", "error", err)
		panic(err)
	}

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_generate_version_1]", "name", id, "error", err)
		return 0
	}

	ks.Insert(kp)
	ret, err := toWasmMemorySized(instanceContext, kp.Public().Encode(), 32)
	if err != nil {
		logger.Error("[ext_crypto_sr25519_generate_version_1] failed to allocate memory", "error", err)
		return 0
	}

	logger.Debug("[ext_crypto_sr25519_generate_version_1] generated sr25519 keypair", "public", kp.Public().Hex())
	return C.int32_t(ret)
}

//export ext_crypto_sr25519_public_keys_version_1
func ext_crypto_sr25519_public_keys_version_1(context unsafe.Pointer, keyTypeID C.int32_t) C.int64_t {
	logger.Debug("[ext_crypto_sr25519_public_keys_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_public_keys_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	if ks.Type() != crypto.Sr25519Type && ks.Type() != crypto.UnknownType {
		logger.Warn("[ext_crypto_sr25519_public_keys_version_1]", "name", id, "error", "keystore type is not sr25519")
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Error("[ext_crypto_sr25519_public_keys_version_1] failed to allocate memory", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	ret, err := toWasmMemory(instanceContext, append(prefix, encodedKeys...))
	if err != nil {
		logger.Error("[ext_crypto_sr25519_public_keys_version_1] failed to allocate memory", err)
		ret, _ = toWasmMemory(instanceContext, []byte{0})
		return C.int64_t(ret)
	}

	return C.int64_t(ret)
}

//export ext_crypto_sr25519_sign_version_1
func ext_crypto_sr25519_sign_version_1(context unsafe.Pointer, keyTypeID, key C.int32_t, msg C.int64_t) C.int64_t {
	logger.Debug("[ext_crypto_sr25519_sign_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	memory := instanceContext.Memory().Data()

	emptyRet, _ := toWasmMemoryOptional(instanceContext, nil)

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_sign_version_1]", "name", id, "error", err)
		return C.int64_t(emptyRet)
	}

	var ret int64
	pubKey, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] failed to get public key", "error", err)
		return C.int64_t(emptyRet)
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] could not find public key in keystore", "error", pubKey)
		return C.int64_t(emptyRet)
	}

	msgData := asMemorySlice(instanceContext, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] could not sign message", "error", err)
		return C.int64_t(emptyRet)
	}

	ret, err = toWasmMemoryFixedSizeOptional(instanceContext, sig)
	if err != nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] failed to allocate memory", "error", err)
		return C.int64_t(emptyRet)
	}

	return C.int64_t(ret)
}

//export ext_crypto_sr25519_verify_version_1
func ext_crypto_sr25519_verify_version_1(context unsafe.Pointer, sig C.int32_t, msg C.int64_t, key C.int32_t) C.int32_t {
	logger.Debug("[ext_crypto_sr25519_verify_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_verify_version_1] invalid sr25519 public key")
		return 0
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_1]", "pub", pub.Hex(),
		"message", fmt.Sprintf("0x%x", message),
		"signature", fmt.Sprintf("0x%x", signature),
	)

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
		logger.Debug("[ext_crypto_sr25519_verify_version_1] failed to validate signature", "error", err)
		// this fails at block 3876, which seems to be expected, based on discussions
		return 1
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_1] verified sr25519 signature")
	return 1
}

//export ext_crypto_sr25519_verify_version_2
func ext_crypto_sr25519_verify_version_2(context unsafe.Pointer, sig C.int32_t, msg C.int64_t, key C.int32_t) C.int32_t {
	logger.Trace("[ext_crypto_sr25519_verify_version_2] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_verify_version_2] invalid sr25519 public key")
		return 0
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_2]", "pub", pub.Hex(),
		"message", fmt.Sprintf("0x%x", message),
		"signature", fmt.Sprintf("0x%x", signature),
	)

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

	if ok, err := pub.Verify(message, signature); err != nil || !ok {
		logger.Error("[ext_crypto_sr25519_verify_version_2] failed to validate signature", "error", err)
		return 0
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_2] validated signature")
	return C.int32_t(1)
}

//export ext_crypto_start_batch_verify_version_1
func ext_crypto_start_batch_verify_version_1(context unsafe.Pointer) {
	logger.Debug("[ext_crypto_start_batch_verify_version_1] executing...")

	// TODO: fix and re-enable signature verification (#1405)
	// beginBatchVerify(context)
}

func beginBatchVerify(context unsafe.Pointer) { //nolint
	instanceContext := wasm.IntoInstanceContext(context)
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	if sigVerifier.IsStarted() {
		logger.Error("[ext_crypto_start_batch_verify_version_1] previous batch verification is not finished")
		return
	}

	sigVerifier.Start()
}

//export ext_crypto_finish_batch_verify_version_1
func ext_crypto_finish_batch_verify_version_1(context unsafe.Pointer) C.int32_t {
	logger.Debug("[ext_crypto_finish_batch_verify_version_1] executing...")

	// TODO: fix and re-enable signature verification (#1405)
	// return finishBatchVerify(context)
	return 1
}

func finishBatchVerify(context unsafe.Pointer) C.int32_t { //nolint
	instanceContext := wasm.IntoInstanceContext(context)
	sigVerifier := instanceContext.Data().(*runtime.Context).SigVerifier

	if !sigVerifier.IsStarted() {
		logger.Error("[ext_crypto_finish_batch_verify_version_1] batch verification is not started", "error")
		panic("batch verification is not started")
	}

	if sigVerifier.Finish() {
		return 1
	}
	logger.Error("[ext_crypto_finish_batch_verify_version_1] failed to batch verify; invalid signature")
	return 0
}

//export ext_trie_blake2_256_root_version_1
func ext_trie_blake2_256_root_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Debug("[ext_trie_blake2_256_root_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	data := asMemorySlice(instanceContext, dataSpan)

	t := trie.NewEmptyTrie()

	type kv struct {
		Key, Value []byte
	}

	// this function is expecting an array of (key, value) tuples
	var kvs []kv
	if err := scale.Unmarshal(data, &kvs); err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return 0
	}

	for _, kv := range kvs {
		t.Put(kv.Key, kv.Value)
	}

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(32)
	if err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return 0
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_trie_blake2_256_root_version_1]", "root", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return C.int32_t(ptr)
}

//export ext_trie_blake2_256_ordered_root_version_1
func ext_trie_blake2_256_ordered_root_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Debug("[ext_trie_blake2_256_ordered_root_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	data := asMemorySlice(instanceContext, dataSpan)

	t := trie.NewEmptyTrie()
	var values [][]byte
	err := scale.Unmarshal(data, &values)
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return 0
	}

	for i, val := range values {
		key, err := scale.Marshal(big.NewInt(int64(i))) //nolint
		if err != nil {
			logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
			return 0
		}
		logger.Trace("[ext_trie_blake2_256_ordered_root_version_1]", "key", key, "value", val)

		t.Put(key, val)
	}

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(32)
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
	return C.int32_t(ptr)
}

//export ext_trie_blake2_256_verify_proof_version_1
func ext_trie_blake2_256_verify_proof_version_1(context unsafe.Pointer, a C.int32_t, b, c, d C.int64_t) C.int32_t {
	logger.Debug("[ext_trie_blake2_256_verify_proof_version_1] executing...")
	logger.Warn("[ext_trie_blake2_256_verify_proof_version_1] unimplemented")
	return 0
}

//export ext_misc_print_hex_version_1
func ext_misc_print_hex_version_1(context unsafe.Pointer, dataSpan C.int64_t) {
	logger.Trace("[ext_misc_print_hex_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	data := asMemorySlice(instanceContext, dataSpan)
	logger.Debug("[ext_misc_print_hex_version_1]", "hex", fmt.Sprintf("0x%x", data))
}

//export ext_misc_print_num_version_1
func ext_misc_print_num_version_1(_ unsafe.Pointer, data C.int64_t) {
	logger.Trace("[ext_misc_print_num_version_1] executing...")

	logger.Debug("[ext_misc_print_num_version_1]", "num", fmt.Sprintf("%d", int64(data)))
}

//export ext_misc_print_utf8_version_1
func ext_misc_print_utf8_version_1(context unsafe.Pointer, dataSpan C.int64_t) {
	logger.Trace("[ext_misc_print_utf8_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	data := asMemorySlice(instanceContext, dataSpan)
	logger.Debug("[ext_misc_print_utf8_version_1]", "utf8", string(data))
}

//export ext_misc_runtime_version_version_1
func ext_misc_runtime_version_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int64_t {
	logger.Trace("[ext_misc_runtime_version_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	data := asMemorySlice(instanceContext, dataSpan)

	cfg := &Config{
		Imports: ImportsNodeRuntime,
	}
	cfg.LogLvl = -1 // don't change log level
	cfg.Storage, _ = rtstorage.NewTrieState(nil)

	instance, err := NewInstance(data, cfg)
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to create instance", "error", err)
		return 0
	}

	// instance version is set and cached in NewInstance
	version := instance.version
	logger.Debug("[ext_misc_runtime_version_version_1]", "version", version)

	if version == nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to get runtime version")
		out, _ := toWasmMemoryOptional(instanceContext, nil)
		return C.int64_t(out)
	}

	encodedData, err := version.Encode()
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to encode result", "error", err)
		return 0
	}

	out, err := toWasmMemoryOptional(instanceContext, encodedData)
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(out)
}

//export ext_default_child_storage_read_version_1
func ext_default_child_storage_read_version_1(context unsafe.Pointer, childStorageKey, key, valueOut C.int64_t, offset C.int32_t) C.int64_t {
	logger.Debug("[ext_default_child_storage_read_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage
	memory := instanceContext.Memory().Data()

	value, err := storage.GetChildStorage(asMemorySlice(instanceContext, childStorageKey), asMemorySlice(instanceContext, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_read_version_1] failed to get child storage", "error", err)
		return 0
	}

	valueBuf, valueLen := runtime.Int64ToPointerAndSize(int64(valueOut))
	copy(memory[valueBuf:valueBuf+valueLen], value[offset:])

	size := uint32(len(value[offset:]))
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, size)

	sizeSpan, err := toWasmMemoryOptional(instanceContext, sizeBuf)
	if err != nil {
		logger.Error("[ext_default_child_storage_read_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(sizeSpan)
}

//export ext_default_child_storage_clear_version_1
func ext_default_child_storage_clear_version_1(context unsafe.Pointer, childStorageKey, keySpan C.int64_t) {
	logger.Debug("[ext_default_child_storage_clear_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	key := asMemorySlice(instanceContext, keySpan)

	err := storage.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Error("[ext_default_child_storage_clear_version_1] failed to clear child storage", "error", err)
	}
}

//export ext_default_child_storage_clear_prefix_version_1
func ext_default_child_storage_clear_prefix_version_1(context unsafe.Pointer, childStorageKey, prefixSpan C.int64_t) {
	logger.Debug("[ext_default_child_storage_clear_prefix_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	prefix := asMemorySlice(instanceContext, prefixSpan)

	err := storage.ClearPrefixInChild(keyToChild, prefix)
	if err != nil {
		logger.Error("[ext_default_child_storage_clear_prefix_version_1] failed to clear prefix in child", "error", err)
	}
}

//export ext_default_child_storage_exists_version_1
func ext_default_child_storage_exists_version_1(context unsafe.Pointer, childStorageKey, key C.int64_t) C.int32_t {
	logger.Debug("[ext_default_child_storage_exists_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	child, err := storage.GetChildStorage(asMemorySlice(instanceContext, childStorageKey), asMemorySlice(instanceContext, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_exists_version_1] failed to get child from child storage", "error", err)
		return 0
	}
	if child != nil {
		return 1
	}
	return 0
}

//export ext_default_child_storage_get_version_1
func ext_default_child_storage_get_version_1(context unsafe.Pointer, childStorageKey, key C.int64_t) C.int64_t {
	logger.Debug("[ext_default_child_storage_get_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	child, err := storage.GetChildStorage(asMemorySlice(instanceContext, childStorageKey), asMemorySlice(instanceContext, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_get_version_1] failed to get child from child storage", "error", err)
		return 0
	}

	value, err := toWasmMemoryOptional(instanceContext, child)
	if err != nil {
		logger.Error("[ext_default_child_storage_get_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(value)
}

//export ext_default_child_storage_next_key_version_1
func ext_default_child_storage_next_key_version_1(context unsafe.Pointer, childStorageKey, key C.int64_t) C.int64_t {
	logger.Debug("[ext_default_child_storage_next_key_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	child, err := storage.GetChildNextKey(asMemorySlice(instanceContext, childStorageKey), asMemorySlice(instanceContext, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_next_key_version_1] failed to get child's next key", "error", err)
		return 0
	}

	value, err := toWasmMemoryOptional(instanceContext, child)
	if err != nil {
		logger.Error("[ext_default_child_storage_next_key_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(value)
}

//export ext_default_child_storage_root_version_1
func ext_default_child_storage_root_version_1(context unsafe.Pointer, childStorageKey C.int64_t) C.int64_t {
	logger.Debug("[ext_default_child_storage_root_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	child, err := storage.GetChild(asMemorySlice(instanceContext, childStorageKey))
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to retrieve child", "error", err)
		return 0
	}

	childRoot, err := child.Hash()
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to encode child root", "error", err)
		return 0
	}

	root, err := toWasmMemoryOptional(instanceContext, childRoot[:])
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(root)
}

//export ext_default_child_storage_set_version_1
func ext_default_child_storage_set_version_1(context unsafe.Pointer, childStorageKeySpan, keySpan, valueSpan C.int64_t) {
	logger.Debug("[ext_default_child_storage_set_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)
	key := asMemorySlice(instanceContext, keySpan)
	value := asMemorySlice(instanceContext, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	err := storage.SetChildStorage(childStorageKey, key, cp)
	if err != nil {
		logger.Error("[ext_default_child_storage_set_version_1] failed to set value in child storage", "error", err)
		return
	}
}

//export ext_default_child_storage_storage_kill_version_1
func ext_default_child_storage_storage_kill_version_1(context unsafe.Pointer, childStorageKeySpan C.int64_t) {
	logger.Debug("[ext_default_child_storage_storage_kill_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)
	storage.DeleteChild(childStorageKey)
}

//export ext_default_child_storage_storage_kill_version_2
func ext_default_child_storage_storage_kill_version_2(context unsafe.Pointer, childStorageKeySpan, lim C.int64_t) C.int32_t {
	logger.Debug("[ext_default_child_storage_storage_kill_version_2] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage
	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit *[]byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warn("[ext_default_child_storage_storage_kill_version_2] cannot generate limit", "error", err)
		return 0
	}

	_, all, err := storage.DeleteChildLimit(childStorageKey, limit)
	if err != nil {
		logger.Warn("[ext_default_child_storage_storage_kill_version_2] cannot get child storage", "error", err)
	}

	if all {
		return 1
	}

	return 0
}

type noneRemain uint32
type someRemain uint32

func (noneRemain) Index() uint {
	return 0
}
func (someRemain) Index() uint {
	return 1
}

//export ext_default_child_storage_storage_kill_version_3
func ext_default_child_storage_storage_kill_version_3(context unsafe.Pointer, childStorageKeySpan, lim C.int64_t) C.int64_t {
	logger.Debug("executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage
	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit *[]byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warn("cannot generate limit", "error", err)
	}

	deleted, all, err := storage.DeleteChildLimit(childStorageKey, limit)
	if err != nil {
		logger.Warn("cannot get child storage", "error", err)
		return C.int64_t(0)
	}

	vdt, err := scale.NewVaryingDataType(noneRemain(0), someRemain(0))
	if err != nil {
		logger.Warn("cannot create new varying data type", "error", err)
	}

	if all {
		err = vdt.Set(noneRemain(deleted))
	} else {
		err = vdt.Set(someRemain(deleted))
	}
	if err != nil {
		logger.Warn("cannot set varying data type", "error", err)
		return C.int64_t(0)
	}

	encoded, err := scale.Marshal(vdt)
	if err != nil {
		logger.Warn("problem marshaling varying data type", "error", err)
		return C.int64_t(0)
	}

	out, err := toWasmMemoryOptional(instanceContext, encoded)
	if err != nil {
		logger.Warn("failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(out)
}

//export ext_allocator_free_version_1
func ext_allocator_free_version_1(context unsafe.Pointer, addr C.int32_t) {
	logger.Trace("[ext_allocator_free_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	// Deallocate memory
	err := runtimeCtx.Allocator.Deallocate(uint32(addr))
	if err != nil {
		logger.Error("[ext_allocator_free_version_1] failed to free memory", "error", err)
	}
}

//export ext_allocator_malloc_version_1
func ext_allocator_malloc_version_1(context unsafe.Pointer, size C.int32_t) C.int32_t {
	logger.Trace("[ext_allocator_malloc_version_1] executing...", "size", size)

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)

	// Allocate memory
	res, err := ctx.Allocator.Allocate(uint32(size))
	if err != nil {
		logger.Crit("[ext_allocator_malloc_version_1] failed to allocate memory", "error", err)
		panic(err)
	}

	return C.int32_t(res)
}

//export ext_hashing_blake2_128_version_1
func ext_hashing_blake2_128_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_blake2_128_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Blake2b128(data)
	if err != nil {
		logger.Error("[ext_hashing_blake2_128_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_blake2_128_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(instanceContext, hash, 16)
	if err != nil {
		logger.Error("[ext_hashing_blake2_128_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_blake2_256_version_1
func ext_hashing_blake2_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_blake2_256_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Blake2bHash(data)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_blake2_256_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", hash)

	out, err := toWasmMemorySized(instanceContext, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_keccak_256_version_1
func ext_hashing_keccak_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_keccak_256_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Keccak256(data)
	if err != nil {
		logger.Error("[ext_hashing_keccak_256_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_keccak_256_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", hash)

	out, err := toWasmMemorySized(instanceContext, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_keccak_256_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_sha2_256_version_1
func ext_hashing_sha2_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_sha2_256_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)
	hash := common.Sha256(data)

	logger.Debug("[ext_hashing_sha2_256_version_1]", "data", data, "hash", hash)

	out, err := toWasmMemorySized(instanceContext, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_sha2_256_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_256_version_1
func ext_hashing_twox_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_twox_256_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox256(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_256_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_twox_256_version_1]", "data", data, "hash", hash)

	out, err := toWasmMemorySized(instanceContext, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_twox_256_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_128_version_1
func ext_hashing_twox_128_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_twox_128_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox128Hash(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_twox_128_version_1]", "data", string(data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(instanceContext, hash, 16)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_64_version_1
func ext_hashing_twox_64_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("[ext_hashing_twox_64_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox64(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1]", "error", err)
		return 0
	}

	logger.Debug("[ext_hashing_twox_64_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(instanceContext, hash, 8)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_offchain_index_set_version_1
func ext_offchain_index_set_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("[ext_offchain_index_set_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	storageKey := asMemorySlice(instanceContext, keySpan)
	newValue := asMemorySlice(instanceContext, valueSpan)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	err := runtimeCtx.NodeStorage.BaseDB.Put(storageKey, cp)
	if err != nil {
		logger.Error("[ext_offchain_index_set_version_1] failed to set value in raw storage", "error", err)
	}
}

//export ext_offchain_local_storage_clear_version_1
func ext_offchain_local_storage_clear_version_1(context unsafe.Pointer, kind C.int32_t, key C.int64_t) {
	logger.Trace("[ext_offchain_local_storage_clear_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	storageKey := asMemorySlice(instanceContext, key)

	memory := instanceContext.Memory().Data()
	kindInt := binary.LittleEndian.Uint32(memory[kind : kind+4])

	var err error

	switch runtime.NodeStorageType(kindInt) {
	case runtime.NodeStorageTypePersistent:
		err = runtimeCtx.NodeStorage.PersistentStorage.Del(storageKey)
	case runtime.NodeStorageTypeLocal:
		err = runtimeCtx.NodeStorage.LocalStorage.Del(storageKey)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_clear_version_1] failed to clear value from storage", "error", err)
	}
}

//export ext_offchain_is_validator_version_1
func ext_offchain_is_validator_version_1(context unsafe.Pointer) C.int32_t {
	logger.Debug("[ext_offchain_is_validator_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Validator {
		return 1
	}
	return 0
}

//export ext_offchain_local_storage_compare_and_set_version_1
func ext_offchain_local_storage_compare_and_set_version_1(context unsafe.Pointer, kind C.int32_t, key, oldValue, newValue C.int64_t) C.int32_t {
	logger.Debug("[ext_offchain_local_storage_compare_and_set_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	storageKey := asMemorySlice(instanceContext, key)

	var storedValue []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		storedValue, err = runtimeCtx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		storedValue, err = runtimeCtx.NodeStorage.LocalStorage.Get(storageKey)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_compare_and_set_version_1] failed to get value from storage", "error", err)
		return 0
	}

	oldVal := asMemorySlice(instanceContext, oldValue)
	newVal := asMemorySlice(instanceContext, newValue)
	if reflect.DeepEqual(storedValue, oldVal) {
		cp := make([]byte, len(newVal))
		copy(cp, newVal)
		err = runtimeCtx.NodeStorage.LocalStorage.Put(storageKey, cp)
		if err != nil {
			logger.Error("[ext_offchain_local_storage_compare_and_set_version_1] failed to set value in storage", "error", err)
			return 0
		}
	}

	return 1
}

//export ext_offchain_local_storage_get_version_1
func ext_offchain_local_storage_get_version_1(context unsafe.Pointer, kind C.int32_t, key C.int64_t) C.int64_t {
	logger.Debug("[ext_offchain_local_storage_get_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	storageKey := asMemorySlice(instanceContext, key)

	var res []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		res, err = runtimeCtx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		res, err = runtimeCtx.NodeStorage.LocalStorage.Get(storageKey)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_get_version_1] failed to get value from storage", "error", err)
	}
	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemoryOptional(instanceContext, res)
	if err != nil {
		logger.Error("[ext_offchain_local_storage_get_version_1] failed to allocate memory", "error", err)
		return 0
	}
	return C.int64_t(ptr)
}

//export ext_offchain_local_storage_set_version_1
func ext_offchain_local_storage_set_version_1(context unsafe.Pointer, kind C.int32_t, key, value C.int64_t) {
	logger.Debug("[ext_offchain_local_storage_set_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	storageKey := asMemorySlice(instanceContext, key)
	newValue := asMemorySlice(instanceContext, value)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	var err error
	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		err = runtimeCtx.NodeStorage.PersistentStorage.Put(storageKey, cp)
	case runtime.NodeStorageTypeLocal:
		err = runtimeCtx.NodeStorage.LocalStorage.Put(storageKey, cp)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_set_version_1] failed to set value in storage", "error", err)
	}
}

//export ext_offchain_network_state_version_1
func ext_offchain_network_state_version_1(context unsafe.Pointer) C.int64_t {
	logger.Debug("[ext_offchain_network_state_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Network == nil {
		return 0
	}

	nsEnc, err := scale.Marshal(runtimeCtx.Network.NetworkState())
	if err != nil {
		logger.Error("[ext_offchain_network_state_version_1] failed at encoding network state", "error", err)
		return 0
	}

	// copy network state length to memory writtenOut location
	nsEncLen := uint32(len(nsEnc))
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, nsEncLen)

	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemorySized(instanceContext, nsEnc, nsEncLen)
	if err != nil {
		logger.Error("[ext_offchain_network_state_version_1] failed to allocate memory", "error", err)
		return 0
	}

	return C.int64_t(ptr)
}

//export ext_offchain_random_seed_version_1
func ext_offchain_random_seed_version_1(context unsafe.Pointer) C.int32_t {
	logger.Debug("[ext_offchain_random_seed_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)

	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		logger.Error("[ext_offchain_random_seed_version_1] failed to generate random seed", "error", err)
	}
	ptr, err := toWasmMemorySized(instanceContext, seed, 32)
	if err != nil {
		logger.Error("[ext_offchain_random_seed_version_1] failed to allocate memory", "error", err)
	}
	return C.int32_t(ptr)
}

//export ext_offchain_submit_transaction_version_1
func ext_offchain_submit_transaction_version_1(context unsafe.Pointer, data C.int64_t) C.int64_t {
	logger.Debug("[ext_offchain_submit_transaction_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	extBytes := asMemorySlice(instanceContext, data)

	var extrinsic []byte
	err := scale.Unmarshal(extBytes, &extrinsic)
	if err != nil {
		logger.Error("[ext_offchain_submit_transaction_version_1] failed to decode extrinsic data", "error", err)
	}

	// validate the transaction
	txv := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	vtx := transaction.NewValidTransaction(extrinsic, txv)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	runtimeCtx.Transaction.AddToPool(vtx)

	ptr, err := toWasmMemoryOptional(instanceContext, nil)
	if err != nil {
		logger.Error("[ext_offchain_submit_transaction_version_1] failed to allocate memory", "error", err)
	}
	return C.int64_t(ptr)
}

//export ext_offchain_timestamp_version_1
func ext_offchain_timestamp_version_1(context unsafe.Pointer) C.int64_t {
	logger.Trace("[ext_offchain_timestamp_version_1] executing...")
	logger.Warn("[ext_offchain_timestamp_version_1] unimplemented")
	return 0
}

//export ext_offchain_sleep_until_version_1
func ext_offchain_sleep_until_version_1(_ unsafe.Pointer, deadline C.int64_t) {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
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
			logger.Trace("[ext_storage_append_version_1] item in storage is not SCALE encoded, overwriting", "key", key)
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
		logger.Trace("[ext_storage_append_version_1] failed to encode new length", "error", err)
		return err
	}

	// append new length prefix to start of items array
	lengthEnc = append(lengthEnc, valueRes...)
	logger.Debug("[ext_storage_append_version_1]", "resulting value", fmt.Sprintf("0x%x", lengthEnc))
	storage.Set(key, lengthEnc)
	return nil
}

//export ext_storage_append_version_1
func ext_storage_append_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("[ext_storage_append_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)
	valueAppend := asMemorySlice(instanceContext, valueSpan)
	logger.Debug("[ext_storage_append_version_1]", "key", fmt.Sprintf("0x%x", key), "value to append", fmt.Sprintf("0x%x", valueAppend))

	cp := make([]byte, len(valueAppend))
	copy(cp, valueAppend)

	err := storageAppend(storage, key, cp)
	if err != nil {
		logger.Error("[ext_storage_append_version_1]", "error", err)
	}
}

//export ext_storage_changes_root_version_1
func ext_storage_changes_root_version_1(context unsafe.Pointer, parentHashSpan C.int64_t) C.int64_t {
	logger.Trace("[ext_storage_changes_root_version_1] executing...")
	logger.Debug("[ext_storage_changes_root_version_1] returning None")

	instanceContext := wasm.IntoInstanceContext(context)

	rootSpan, err := toWasmMemoryOptional(instanceContext, nil)
	if err != nil {
		logger.Error("[ext_storage_changes_root_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(rootSpan)
}

//export ext_storage_clear_version_1
func ext_storage_clear_version_1(context unsafe.Pointer, keySpan C.int64_t) {
	logger.Trace("[ext_storage_clear_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)

	logger.Debug("[ext_storage_clear_version_1]", "key", fmt.Sprintf("0x%x", key))
	storage.Delete(key)
}

//export ext_storage_clear_prefix_version_1
func ext_storage_clear_prefix_version_1(context unsafe.Pointer, prefixSpan C.int64_t) {
	logger.Trace("[ext_storage_clear_prefix_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	prefix := asMemorySlice(instanceContext, prefixSpan)
	logger.Debug("[ext_storage_clear_prefix_version_1]", "prefix", fmt.Sprintf("0x%x", prefix))

	err := storage.ClearPrefix(prefix)
	if err != nil {
		logger.Error("[ext_storage_clear_prefix_version_1]", "error", err)
	}
}

//export ext_storage_clear_prefix_version_2
func ext_storage_clear_prefix_version_2(context unsafe.Pointer, prefixSpan, lim C.int64_t) C.int64_t {
	logger.Trace("[ext_storage_clear_prefix_version_2] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	prefix := asMemorySlice(instanceContext, prefixSpan)
	logger.Debug("[ext_storage_clear_prefix_version_2]", "prefix", fmt.Sprintf("0x%x", prefix))

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit []byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warn("[ext_storage_clear_prefix_version_2] cannot generate limit", "error", err)
		ret, _ := toWasmMemory(instanceContext, nil)
		return C.int64_t(ret)
	}

	limitUint := binary.LittleEndian.Uint32(limit)
	numRemoved, all := storage.ClearPrefixLimit(prefix, limitUint)
	encBytes, err := toKillStorageResultEnum(all, numRemoved)

	if err != nil {
		logger.Error("[ext_storage_clear_prefix_version_2] failed to allocate memory", err)
		ret, _ := toWasmMemory(instanceContext, nil)
		return C.int64_t(ret)
	}

	valueSpan, err := toWasmMemory(instanceContext, encBytes)
	if err != nil {
		logger.Error("[ext_storage_clear_prefix_version_2] failed to allocate", "error", err)
		ptr, _ := toWasmMemory(instanceContext, nil)
		return C.int64_t(ptr)
	}

	return C.int64_t(valueSpan)
}

//export ext_storage_exists_version_1
func ext_storage_exists_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int32_t {
	logger.Trace("[ext_storage_exists_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)
	logger.Debug("[ext_storage_exists_version_1]", "key", fmt.Sprintf("0x%x", key))

	val := storage.Get(key)
	if len(val) > 0 {
		return 1
	}

	return 0
}

//export ext_storage_get_version_1
func ext_storage_get_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int64_t {
	logger.Trace("[ext_storage_get_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)
	logger.Debug("[ext_storage_get_version_1]", "key", fmt.Sprintf("0x%x", key))

	value := storage.Get(key)
	logger.Debug("[ext_storage_get_version_1]", "value", fmt.Sprintf("0x%x", value))

	valueSpan, err := toWasmMemoryOptional(instanceContext, value)
	if err != nil {
		logger.Error("[ext_storage_get_version_1] failed to allocate", "error", err)
		ptr, _ := toWasmMemoryOptional(instanceContext, nil)
		return C.int64_t(ptr)
	}

	return C.int64_t(valueSpan)
}

//export ext_storage_next_key_version_1
func ext_storage_next_key_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int64_t {
	logger.Trace("[ext_storage_next_key_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)

	next := storage.NextKey(key)
	logger.Debug("[ext_storage_next_key_version_1]", "key", fmt.Sprintf("0x%x", key), "next", fmt.Sprintf("0x%x", next))

	nextSpan, err := toWasmMemoryOptional(instanceContext, next)
	if err != nil {
		logger.Error("[ext_storage_next_key_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(nextSpan)
}

//export ext_storage_read_version_1
func ext_storage_read_version_1(context unsafe.Pointer, keySpan, valueOut C.int64_t, offset C.int32_t) C.int64_t {
	logger.Trace("[ext_storage_read_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage
	memory := instanceContext.Memory().Data()

	key := asMemorySlice(instanceContext, keySpan)
	value := storage.Get(key)
	logger.Debug("[ext_storage_read_version_1]", "key", fmt.Sprintf("0x%x", key), "value", fmt.Sprintf("0x%x", value))

	if value == nil {
		ret, _ := toWasmMemoryOptional(instanceContext, nil)
		return C.int64_t(ret)
	}

	var size uint32

	if int(offset) > len(value) {
		size = uint32(0)
	} else {
		size = uint32(len(value[offset:]))
		valueBuf, valueLen := runtime.Int64ToPointerAndSize(int64(valueOut))
		copy(memory[valueBuf:valueBuf+valueLen], value[offset:])
	}

	sizeSpan, err := toWasmMemoryOptionalUint32(instanceContext, &size)
	if err != nil {
		logger.Error("[ext_storage_read_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(sizeSpan)
}

//export ext_storage_root_version_1
func ext_storage_root_version_1(context unsafe.Pointer) C.int64_t {
	logger.Trace("[ext_storage_root_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	root, err := storage.Root()
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to get storage root", "error", err)
		return 0
	}

	logger.Debug("[ext_storage_root_version_1]", "root", root)

	rootSpan, err := toWasmMemory(instanceContext, root[:])
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to allocate", "error", err)
		return 0
	}

	return C.int64_t(rootSpan)
}

//export ext_storage_set_version_1
func ext_storage_set_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("[ext_storage_set_version_1] executing...")

	instanceContext := wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)
	value := asMemorySlice(instanceContext, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	logger.Debug("[ext_storage_set_version_1]", "key", fmt.Sprintf("0x%x", key), "val", fmt.Sprintf("0x%x", value))
	storage.Set(key, cp)
}

//export ext_storage_start_transaction_version_1
func ext_storage_start_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("[ext_storage_start_transaction_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.BeginStorageTransaction()
}

//export ext_storage_rollback_transaction_version_1
func ext_storage_rollback_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("[ext_storage_rollback_transaction_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.RollbackStorageTransaction()
}

//export ext_storage_commit_transaction_version_1
func ext_storage_commit_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("[ext_storage_commit_transaction_version_1] executing...")
	instanceContext := wasm.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.CommitStorageTransaction()
}

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(context wasm.InstanceContext, span C.int64_t) []byte {
	memory := context.Memory().Data()
	ptr, size := runtime.Int64ToPointerAndSize(int64(span))
	return memory[ptr : ptr+size]
}

// Copy a byte slice to wasm memory and return the resulting 64bit span descriptor
func toWasmMemory(context wasm.InstanceContext, data []byte) (int64, error) {
	allocator := context.Data().(*runtime.Context).Allocator
	size := uint32(len(data))

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := context.Memory().Data()

	if uint32(len(memory)) < out+size {
		panic(fmt.Sprintf("length of memory is less than expected, want %d have %d", out+size, len(memory)))
	}

	copy(memory[out:out+size], data)
	return runtime.PointerAndSizeToInt64(int32(out), int32(size)), nil
}

// Copy a byte slice of a fixed size to wasm memory and return resulting pointer
func toWasmMemorySized(context wasm.InstanceContext, data []byte, size uint32) (uint32, error) {
	if int(size) != len(data) {
		return 0, errors.New("internal byte array size missmatch")
	}

	allocator := context.Data().(*runtime.Context).Allocator

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := context.Memory().Data()
	copy(memory[out:out+size], data)

	return out, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(context wasm.InstanceContext, data []byte) (int64, error) {
	var opt *[]byte
	if data != nil {
		temp := data
		opt = &temp
	}

	enc, err := scale.Marshal(opt)
	if err != nil {
		return 0, err
	}

	return toWasmMemory(context, enc)
}

// Wraps slice in Result type and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryResult(context wasm.InstanceContext, data []byte) (int64, error) {
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

	return toWasmMemory(context, enc)
}

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(context wasm.InstanceContext, data *uint32) (int64, error) {
	var opt *uint32
	if data != nil {
		temp := *data
		opt = &temp
	}

	enc, err := scale.Marshal(opt)
	if err != nil {
		return int64(0), err
	}
	return toWasmMemory(context, enc)
}

// toKillStorageResult returns enum encoded value
func toKillStorageResultEnum(allRemoved bool, numRemoved uint32) ([]byte, error) {
	var b, sbytes []byte
	sbytes, err := scale.Marshal(numRemoved)
	if err != nil {
		return nil, err
	}

	if allRemoved {
		// No key remains in the child trie.
		b = append(b, byte(0))
	} else {
		// At least one key still resides in the child trie due to the supplied limit.
		b = append(b, byte(1))
	}

	b = append(b, sbytes...)

	return b, err
}

// Wraps slice in optional.FixedSizeBytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryFixedSizeOptional(context wasm.InstanceContext, data []byte) (int64, error) {
	var opt [64]byte
	copy(opt[:], data[:])
	enc, err := scale.Marshal(&opt)
	if err != nil {
		return 0, err
	}
	return toWasmMemory(context, enc)
}

// ImportsNodeRuntime returns the imports for the v0.8 runtime
func ImportsNodeRuntime() (*wasm.Imports, error) { //nolint
	var err error

	imports := wasm.NewImports()

	_, err = imports.Append("ext_allocator_free_version_1", ext_allocator_free_version_1, C.ext_allocator_free_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_allocator_malloc_version_1", ext_allocator_malloc_version_1, C.ext_allocator_malloc_version_1)
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
	_, err = imports.Append("ext_crypto_finish_batch_verify_version_1", ext_crypto_finish_batch_verify_version_1, C.ext_crypto_finish_batch_verify_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_secp256k1_ecdsa_recover_version_1", ext_crypto_secp256k1_ecdsa_recover_version_1, C.ext_crypto_secp256k1_ecdsa_recover_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_secp256k1_ecdsa_recover_version_2", ext_crypto_secp256k1_ecdsa_recover_version_2, C.ext_crypto_secp256k1_ecdsa_recover_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_secp256k1_ecdsa_recover_compressed_version_1", ext_crypto_secp256k1_ecdsa_recover_compressed_version_1, C.ext_crypto_secp256k1_ecdsa_recover_compressed_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_secp256k1_ecdsa_recover_compressed_version_2", ext_crypto_secp256k1_ecdsa_recover_compressed_version_2, C.ext_crypto_secp256k1_ecdsa_recover_compressed_version_2)
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
	_, err = imports.Append("ext_crypto_sr25519_verify_version_1", ext_crypto_sr25519_verify_version_1, C.ext_crypto_sr25519_verify_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_sr25519_verify_version_2", ext_crypto_sr25519_verify_version_2, C.ext_crypto_sr25519_verify_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_ecdsa_verify_version_2", ext_crypto_ecdsa_verify_version_2, C.ext_crypto_ecdsa_verify_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_crypto_start_batch_verify_version_1", ext_crypto_start_batch_verify_version_1, C.ext_crypto_start_batch_verify_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_clear_version_1", ext_default_child_storage_clear_version_1, C.ext_default_child_storage_clear_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_clear_prefix_version_1", ext_default_child_storage_clear_prefix_version_1, C.ext_default_child_storage_clear_prefix_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_exists_version_1", ext_default_child_storage_exists_version_1, C.ext_default_child_storage_exists_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_get_version_1", ext_default_child_storage_get_version_1, C.ext_default_child_storage_get_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_next_key_version_1", ext_default_child_storage_next_key_version_1, C.ext_default_child_storage_next_key_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_read_version_1", ext_default_child_storage_read_version_1, C.ext_default_child_storage_read_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_root_version_1", ext_default_child_storage_root_version_1, C.ext_default_child_storage_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_set_version_1", ext_default_child_storage_set_version_1, C.ext_default_child_storage_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_storage_kill_version_1", ext_default_child_storage_storage_kill_version_1, C.ext_default_child_storage_storage_kill_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_storage_kill_version_2", ext_default_child_storage_storage_kill_version_2, C.ext_default_child_storage_storage_kill_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_default_child_storage_storage_kill_version_3", ext_default_child_storage_storage_kill_version_3, C.ext_default_child_storage_storage_kill_version_3)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_hashing_blake2_128_version_1", ext_hashing_blake2_128_version_1, C.ext_hashing_blake2_128_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_blake2_256_version_1", ext_hashing_blake2_256_version_1, C.ext_hashing_blake2_256_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_keccak_256_version_1", ext_hashing_keccak_256_version_1, C.ext_hashing_keccak_256_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_sha2_256_version_1", ext_hashing_sha2_256_version_1, C.ext_hashing_sha2_256_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_twox_256_version_1", ext_hashing_twox_256_version_1, C.ext_hashing_twox_256_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_twox_128_version_1", ext_hashing_twox_128_version_1, C.ext_hashing_twox_128_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_hashing_twox_64_version_1", ext_hashing_twox_64_version_1, C.ext_hashing_twox_64_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_logging_log_version_1", ext_logging_log_version_1, C.ext_logging_log_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_logging_max_level_version_1", ext_logging_max_level_version_1, C.ext_logging_max_level_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_misc_print_hex_version_1", ext_misc_print_hex_version_1, C.ext_misc_print_hex_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_misc_print_num_version_1", ext_misc_print_num_version_1, C.ext_misc_print_num_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_misc_print_utf8_version_1", ext_misc_print_utf8_version_1, C.ext_misc_print_utf8_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_misc_runtime_version_version_1", ext_misc_runtime_version_version_1, C.ext_misc_runtime_version_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_offchain_index_set_version_1", ext_offchain_index_set_version_1, C.ext_offchain_index_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_is_validator_version_1", ext_offchain_is_validator_version_1, C.ext_offchain_is_validator_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_local_storage_clear_version_1", ext_offchain_local_storage_clear_version_1, C.ext_offchain_local_storage_clear_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_local_storage_compare_and_set_version_1", ext_offchain_local_storage_compare_and_set_version_1, C.ext_offchain_local_storage_compare_and_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_local_storage_get_version_1", ext_offchain_local_storage_get_version_1, C.ext_offchain_local_storage_get_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_local_storage_set_version_1", ext_offchain_local_storage_set_version_1, C.ext_offchain_local_storage_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_network_state_version_1", ext_offchain_network_state_version_1, C.ext_offchain_network_state_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_random_seed_version_1", ext_offchain_random_seed_version_1, C.ext_offchain_random_seed_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_submit_transaction_version_1", ext_offchain_submit_transaction_version_1, C.ext_offchain_submit_transaction_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_timestamp_version_1", ext_offchain_timestamp_version_1, C.ext_offchain_timestamp_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_offchain_sleep_until_version_1", ext_offchain_sleep_until_version_1, C.ext_offchain_sleep_until_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_instance_teardown_version_1", ext_sandbox_instance_teardown_version_1, C.ext_sandbox_instance_teardown_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_instantiate_version_1", ext_sandbox_instantiate_version_1, C.ext_sandbox_instantiate_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_invoke_version_1", ext_sandbox_invoke_version_1, C.ext_sandbox_invoke_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_get_version_1", ext_sandbox_memory_get_version_1, C.ext_sandbox_memory_get_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_new_version_1", ext_sandbox_memory_new_version_1, C.ext_sandbox_memory_new_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_set_version_1", ext_sandbox_memory_set_version_1, C.ext_sandbox_memory_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_sandbox_memory_teardown_version_1", ext_sandbox_memory_teardown_version_1, C.ext_sandbox_memory_teardown_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_storage_append_version_1", ext_storage_append_version_1, C.ext_storage_append_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_changes_root_version_1", ext_storage_changes_root_version_1, C.ext_storage_changes_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_clear_version_1", ext_storage_clear_version_1, C.ext_storage_clear_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_clear_prefix_version_1", ext_storage_clear_prefix_version_1, C.ext_storage_clear_prefix_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_clear_prefix_version_2", ext_storage_clear_prefix_version_2, C.ext_storage_clear_prefix_version_2)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_commit_transaction_version_1", ext_storage_commit_transaction_version_1, C.ext_storage_commit_transaction_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_exists_version_1", ext_storage_exists_version_1, C.ext_storage_exists_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_get_version_1", ext_storage_get_version_1, C.ext_storage_get_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_next_key_version_1", ext_storage_next_key_version_1, C.ext_storage_next_key_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_read_version_1", ext_storage_read_version_1, C.ext_storage_read_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_rollback_transaction_version_1", ext_storage_rollback_transaction_version_1, C.ext_storage_rollback_transaction_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_root_version_1", ext_storage_root_version_1, C.ext_storage_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_set_version_1", ext_storage_set_version_1, C.ext_storage_set_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_storage_start_transaction_version_1", ext_storage_start_transaction_version_1, C.ext_storage_start_transaction_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_trie_blake2_256_ordered_root_version_1", ext_trie_blake2_256_ordered_root_version_1, C.ext_trie_blake2_256_ordered_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_trie_blake2_256_root_version_1", ext_trie_blake2_256_root_version_1, C.ext_trie_blake2_256_root_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_trie_blake2_256_verify_proof_version_1", ext_trie_blake2_256_verify_proof_version_1, C.ext_trie_blake2_256_verify_proof_version_1)
	if err != nil {
		return nil, err
	}

	_, err = imports.Append("ext_transaction_index_index_version_1", ext_transaction_index_index_version_1, C.ext_transaction_index_index_version_1)
	if err != nil {
		return nil, err
	}
	_, err = imports.Append("ext_transaction_index_renew_version_1", ext_transaction_index_renew_version_1, C.ext_transaction_index_renew_version_1)
	if err != nil {
		return nil, err
	}

	return imports, nil
}
