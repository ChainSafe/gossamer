// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package newWasmer

import "C" //skipcq: SCC-compile

import (
	"encoding/binary"
	"fmt"
	"github.com/wasmerio/wasmer-go/wasmer"
	"math/big"
	"math/rand"
	"reflect"
	"time"
	"unsafe"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/proof"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const (
	validateSignatureFail = "failed to validate signature"
)

//export ext_logging_log_version_1
func ext_logging_log_version_1(env interface{}, args []wasmer.Value) {
	logger.Trace("executing...")
	ctx := env.(*Context)

	level, ok := args[0].Unwrap().(int32)
	if !ok {
		logger.Criticalf("[ext_logging_log_version_1]", "error", "addr cannot be converted to int32")
	}

	targetData := args[1].I64()
	msgData := args[2].I64()

	target := string(asMemorySlice(ctx, targetData))
	msg := string(asMemorySlice(ctx, msgData))

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

//export ext_logging_max_level_version_1
func ext_logging_max_level_version_1(_ interface{}) []wasmer.Value {
	logger.Trace("executing...")
	return []wasmer.Value{wasmer.NewI32(4)}
}

//export ext_transaction_index_index_version_1
func ext_transaction_index_index_version_1(_ interface{}, _ []wasmer.Value) {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
}

//export ext_transaction_index_renew_version_1
func ext_transaction_index_renew_version_1(_ interface{}, _ []wasmer.Value) {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
}

//export ext_sandbox_instance_teardown_version_1
func ext_sandbox_instance_teardown_version_1(_ interface{}, _ []wasmer.Value) {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
}

//export ext_sandbox_instantiate_version_1
func ext_sandbox_instantiate_version_1(_ interface{}, _ []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
	return []wasmer.Value{wasmer.NewI32(0)}
}

//export ext_sandbox_invoke_version_1
func ext_sandbox_invoke_version_1(_ interface{}, _ []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
	return []wasmer.Value{wasmer.NewI32(0)}
}

//export ext_sandbox_memory_get_version_1
func ext_sandbox_memory_get_version_1(_ interface{}, _ []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
	return []wasmer.Value{wasmer.NewI32(0)}
}

//export ext_sandbox_memory_new_version_1
func ext_sandbox_memory_new_version_1(_ interface{}, _ []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
	return []wasmer.Value{wasmer.NewI32(0)}
}

//export ext_sandbox_memory_set_version_1
func ext_sandbox_memory_set_version_1(_ interface{}, _ []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
	return []wasmer.Value{wasmer.NewI32(0)}
}

//export ext_sandbox_memory_teardown_version_1
func ext_sandbox_memory_teardown_version_1(_ interface{}, _ []wasmer.Value) {
	logger.Trace("executing...")
	logger.Warn("unimplemented")
}

//export ext_crypto_ed25519_generate_version_1
func ext_crypto_ed25519_generate_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")

	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()

	keyTypeID := args[0].I32()
	seedSpan := args[1].I64()

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(runtimeCtx, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	var kp KeyPair

	if seed != nil {
		kp, err = ed25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = ed25519.GenerateKeypair()
	}

	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	err = ks.Insert(kp)
	if err != nil {
		logger.Warnf("failed to insert key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	ret, err := toWasmMemorySized(runtimeCtx, kp.Public().Encode())
	if err != nil {
		logger.Warnf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debug("generated ed25519 keypair with public key: " + kp.Public().Hex())
	return []wasmer.Value{wasmer.NewI32(ret)}
}

//export ext_crypto_ed25519_public_keys_version_1
func ext_crypto_ed25519_public_keys_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")
	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()

	keyTypeID := args[0].I32()
	id := memory[keyTypeID : keyTypeID+4]

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		ret, _ := toWasmMemory(runtimeCtx, []byte{0})
		return []wasmer.Value{wasmer.NewI64(ret)}
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"error for id 0x%x: keystore type is %s and not the expected ed25519",
			id, ks.Type())
		ret, _ := toWasmMemory(runtimeCtx, []byte{0})
		return []wasmer.Value{wasmer.NewI64(ret)}
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, _ := toWasmMemory(runtimeCtx, []byte{0})
		return []wasmer.Value{wasmer.NewI64(ret)}
	}

	ret, err := toWasmMemory(runtimeCtx, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, _ = toWasmMemory(runtimeCtx, []byte{0})
		return []wasmer.Value{wasmer.NewI64(ret)}
	}

	return []wasmer.Value{wasmer.NewI64(ret)}
}

//export ext_crypto_ed25519_sign_version_1
func ext_crypto_ed25519_sign_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()

	keyTypeID := args[0].I32()
	key := args[1].I32()
	msg := args[2].I64()
	id := memory[keyTypeID : keyTypeID+4]

	pubKeyData := memory[key : key+32]
	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Errorf("failed to get public keys: %s", err)
		return []wasmer.Value{wasmer.NewI64(0)}
	}

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return mustToWasmMemoryOptionalNil(runtimeCtx)
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		ret, err := toWasmMemoryOptionalNil(runtimeCtx)
		if err != nil {
			logger.Errorf("failed to allocate memory: %s", err)
			return []wasmer.Value{wasmer.NewI64(0)}
		}
		return ret
	}

	sig, err := signingKey.Sign(asMemorySlice(runtimeCtx, msg))
	if err != nil {
		logger.Error("could not sign message")
	}

	ret, err := toWasmMemoryFixedSizeOptional(runtimeCtx, sig)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI64(0)}
	}

	return []wasmer.Value{wasmer.NewI64(ret)}
}

//export ext_crypto_ed25519_verify_version_1
func ext_crypto_ed25519_verify_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()
	sigVerifier := runtimeCtx.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	signature := memory[sig : sig+64]
	message := asMemorySlice(runtimeCtx, msg)
	pubKeyData := memory[key : key+32]

	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("failed to create public key")
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pubKey.Encode(),
			Sign:       signature,
			Msg:        message,
			VerifyFunc: ed25519.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return []wasmer.Value{wasmer.NewI32(1)}
	}

	if ok, err := pubKey.Verify(message, signature); err != nil || !ok {
		logger.Error("failed to verify")
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debug("verified ed25519 signature")
	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_crypto_secp256k1_ecdsa_recover_version_1
func ext_crypto_secp256k1_ecdsa_recover_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()

	sig := args[0].I32()
	msg := args[1].I32()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	pub, err := secp256k1.RecoverPublicKey(message, signature)
	if err != nil {
		logger.Errorf("failed to recover public key: %s", err)
		ret, err := toWasmMemoryResultEmpty(instanceContext)
		if err != nil {
			logger.Errorf("failed to allocate memory: %s", err)
			return []wasmer.Value{wasmer.NewI32(0)}
		}
		return ret
	}

	logger.Debugf(
		"recovered public key of length %d: 0x%x",
		len(pub), pub)

	ret, err := toWasmMemoryResult(instanceContext, pub[1:])
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	return []wasmer.Value{wasmer.NewI32(ret)}
}

//export ext_crypto_secp256k1_ecdsa_recover_version_2
func ext_crypto_secp256k1_ecdsa_recover_version_2(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	return ext_crypto_secp256k1_ecdsa_recover_version_1(env, args)
}

//export ext_crypto_ecdsa_verify_version_2
func ext_crypto_ecdsa_verify_version_2(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")

	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()
	sigVerifier := instanceContext.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]
	pubKey := memory[key : key+33]

	pub := new(secp256k1.PublicKey)
	err := pub.Decode(pubKey)
	if err != nil {
		logger.Errorf("failed to decode public key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debugf("pub=%s, message=0x%x, signature=0x%x",
		pub.Hex(), fmt.Sprintf("0x%x", message), fmt.Sprintf("0x%x", signature))

	hash, err := common.Blake2bHash(message)
	if err != nil {
		logger.Errorf("failed to hash message: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pub.Encode(),
			Sign:       signature,
			Msg:        hash[:],
			VerifyFunc: secp256k1.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return []wasmer.Value{wasmer.NewI32(1)}
	}

	ok, err := pub.Verify(hash[:], signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Errorf(message)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debug("validated signature")
	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_crypto_secp256k1_ecdsa_recover_compressed_version_1
func ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()

	sig := args[0].I32()
	msg := args[1].I32()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	cpub, err := secp256k1.RecoverPublicKeyCompressed(message, signature)
	if err != nil {
		logger.Errorf("failed to recover public key: %s", err)
		return mustToWasmMemoryResultEmpty(instanceContext)
	}

	logger.Debugf(
		"recovered public key of length %d: 0x%x",
		len(cpub), cpub)

	ret, err := toWasmMemoryResult(instanceContext, cpub)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI64(0)}
	}

	return []wasmer.Value{wasmer.NewI64(ret)}
}

//export ext_crypto_secp256k1_ecdsa_recover_compressed_version_2
func ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")
	return ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(env, args)
}

//export ext_crypto_sr25519_generate_version_1
func ext_crypto_sr25519_generate_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")

	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()
	keyTypeID := args[0].I32()
	seedSpan := args[1].I64()

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(instanceContext, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	var kp KeyPair
	if seed != nil {
		kp, err = sr25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = sr25519.GenerateKeypair()
	}

	if err != nil {
		logger.Tracef("cannot generate key: %s", err)
		panic(err)
	}

	ks, err := instanceContext.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id "+common.BytesToHex(id)+": %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	err = ks.Insert(kp)
	if err != nil {
		logger.Warnf("failed to insert key: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	ret, err := toWasmMemorySized(instanceContext, kp.Public().Encode())
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debug("generated sr25519 keypair with public key: " + kp.Public().Hex())
	return []wasmer.Value{wasmer.NewI32(ret)}
}

//export ext_crypto_sr25519_public_keys_version_1
func ext_crypto_sr25519_public_keys_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()
	keyTypeID, ok := args[0].Unwrap().(int32)
	if !ok {
		panic("keyTypeID is not int32")
	}

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := instanceContext.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id "+common.BytesToHex(id)+": %s", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return []wasmer.Value{wasmer.NewI32(ret)}
	}

	if ks.Type() != crypto.Sr25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"keystore type for id 0x%x is %s and not expected sr25519",
			id, ks.Type())
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return []wasmer.Value{wasmer.NewI32(ret)}
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, _ := toWasmMemory(instanceContext, []byte{0})
		return []wasmer.Value{wasmer.NewI32(ret)}
	}

	ret, err := toWasmMemory(instanceContext, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, _ = toWasmMemory(instanceContext, []byte{0})
		return []wasmer.Value{wasmer.NewI32(ret)}
	}

	return []wasmer.Value{wasmer.NewI32(ret)}
}

//export ext_crypto_sr25519_sign_version_1
func ext_crypto_sr25519_sign_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")
	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()
	keyTypeID := args[0].I32()
	key := args[1].I32()
	msg := args[2].I64()

	emptyRet, _ := toWasmMemoryOptional(runtimeCtx, nil)

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := runtimeCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return []wasmer.Value{wasmer.NewI64(emptyRet)}
	}

	var ret int64
	pubKey, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Errorf("failed to get public key: %s", err)
		return []wasmer.Value{wasmer.NewI64(emptyRet)}
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		return []wasmer.Value{wasmer.NewI64(emptyRet)}
	}

	msgData := asMemorySlice(runtimeCtx, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Errorf("could not sign message: %s", err)
		return []wasmer.Value{wasmer.NewI64(emptyRet)}
	}

	ret, err = toWasmMemoryFixedSizeOptional(runtimeCtx, sig)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return []wasmer.Value{wasmer.NewI64(emptyRet)}
	}

	return []wasmer.Value{wasmer.NewI64(ret)}
}

//export ext_crypto_sr25519_verify_version_1
func ext_crypto_sr25519_verify_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()
	sigVerifier := instanceContext.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("invalid sr25519 public key")
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debugf(
		"pub=%s message=0x%x signature=0x%x",
		pub.Hex(), message, signature)

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pub.Encode(),
			Sign:       signature,
			Msg:        message,
			VerifyFunc: sr25519.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return []wasmer.Value{wasmer.NewI32(1)}
	}

	ok, err := pub.VerifyDeprecated(message, signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Debugf(message)
		// this fails at block 3876, which seems to be expected, based on discussions
		return []wasmer.Value{wasmer.NewI32(1)}
	}

	logger.Debug("verified sr25519 signature")
	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_crypto_sr25519_verify_version_2
func ext_crypto_sr25519_verify_version_2(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")

	instanceContext := env.(*Context)
	memory := instanceContext.Memory.Data()
	sigVerifier := instanceContext.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	message := asMemorySlice(instanceContext, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("invalid sr25519 public key")
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debugf(
		"pub=%s; message=0x%x; signature=0x%x",
		pub.Hex(), message, signature)

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pub.Encode(),
			Sign:       signature,
			Msg:        message,
			VerifyFunc: sr25519.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return []wasmer.Value{wasmer.NewI32(1)}
	}

	ok, err := pub.Verify(message, signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Errorf(message)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debug("validated signature")
	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_crypto_start_batch_verify_version_1
func ext_crypto_start_batch_verify_version_1(_ interface{}) {
	logger.Debug("executing...")

	// TODO: fix and re-enable signature verification (#1405)
	// beginBatchVerify(context)
}

//export ext_crypto_finish_batch_verify_version_1
func ext_crypto_finish_batch_verify_version_1(_ interface{}) []wasmer.Value {
	logger.Debug("executing...")

	// TODO: fix and re-enable signature verification (#1405)
	// return finishBatchVerify(context)
	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_trie_blake2_256_root_version_1
func ext_trie_blake2_256_root_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()
	dataSpan := args[0].I64()

	data := asMemorySlice(runtimeCtx, dataSpan)

	t := trie.NewEmptyTrie()

	type kv struct {
		Key, Value []byte
	}

	// this function is expecting an array of (key, value) tuples
	var kvs []kv
	if err := scale.Unmarshal(data, &kvs); err != nil {
		logger.Errorf("failed scale decoding data: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	for _, kv := range kvs {
		err := t.Put(kv.Key, kv.Value)
		if err != nil {
			logger.Errorf("failed putting key 0x%x and value 0x%x into trie: %s",
				kv.Key, kv.Value, err)
			return []wasmer.Value{wasmer.NewI32(0)}
		}
	}

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(32)
	if err != nil {
		logger.Errorf("failed allocating: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Errorf("failed computing trie Merkle root hash: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debugf("root hash is %s", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return []wasmer.Value{wasmer.NewI32(ptr)}
}

//export ext_trie_blake2_256_ordered_root_version_1
func ext_trie_blake2_256_ordered_root_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	runtimeCtx := env.(*Context)
	memory := runtimeCtx.Memory.Data()
	dataSpan := args[0].I64()

	data := asMemorySlice(runtimeCtx, dataSpan)

	t := trie.NewEmptyTrie()
	var values [][]byte
	err := scale.Unmarshal(data, &values)
	if err != nil {
		logger.Errorf("failed scale decoding data: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	for i, value := range values {
		key, err := scale.Marshal(big.NewInt(int64(i)))
		if err != nil {
			logger.Errorf("failed scale encoding value index %d: %s", i, err)
			return []wasmer.Value{wasmer.NewI32(0)}
		}
		logger.Tracef(
			"put key=0x%x and value=0x%x",
			key, value)

		err = t.Put(key, value)
		if err != nil {
			logger.Errorf("failed putting key 0x%x and value 0x%x into trie: %s",
				key, value, err)
			return []wasmer.Value{wasmer.NewI32(0)}
		}
	}

	// allocate memory for value and copy value to memory
	ptr, err := runtimeCtx.Allocator.Allocate(32)
	if err != nil {
		logger.Errorf("failed allocating: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Errorf("failed computing trie Merkle root hash: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	logger.Debugf("root hash is %s", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return []wasmer.Value{wasmer.NewI32(ptr)}
}

//export ext_trie_blake2_256_ordered_root_version_2
func ext_trie_blake2_256_ordered_root_version_2(env interface{}, args []wasmer.Value) []wasmer.Value {
	// TODO: update to use state trie version 1 (#2418)
	return ext_trie_blake2_256_ordered_root_version_1(env, args)
}

//export ext_trie_blake2_256_verify_proof_version_1
func ext_trie_blake2_256_verify_proof_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Debug("executing...")

	instanceContext := env.(*Context)
	rootSpan := args[0].I32()
	proofSpan := args[1].I64()
	keySpan := args[1].I64()
	valueSpan := args[1].I64()

	toDecProofs := asMemorySlice(instanceContext, proofSpan)
	var encodedProofNodes [][]byte
	err := scale.Unmarshal(toDecProofs, &encodedProofNodes)
	if err != nil {
		logger.Errorf("failed scale decoding proof data: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	key := asMemorySlice(instanceContext, keySpan)
	value := asMemorySlice(instanceContext, valueSpan)

	mem := instanceContext.Memory.Data()
	trieRoot := mem[rootSpan : rootSpan+32]

	err = proof.Verify(encodedProofNodes, trieRoot, key, value)
	if err != nil {
		logger.Errorf("failed proof verification: %s", err)
		return []wasmer.Value{wasmer.NewI32(0)}
	}

	return []wasmer.Value{wasmer.NewI32(1)}
}

//export ext_misc_print_hex_version_1
func ext_misc_print_hex_version_1(env interface{}, args []wasmer.Value) {
	logger.Trace("executing...")

	ctx := env.(*Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)
	logger.Debugf("data: 0x%x", data)
}

//export ext_misc_print_num_version_1
func ext_misc_print_num_version_1(_ interface{}, args []wasmer.Value) {
	logger.Trace("executing...")
	data := args[0].I64()
	logger.Debugf("num: %d", data)
}

//export ext_misc_print_utf8_version_1
func ext_misc_print_utf8_version_1(env interface{}, args []wasmer.Value) {
	logger.Trace("executing...")

	ctx := env.(*Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)
	logger.Debug("utf8: " + string(data))
}

//export ext_misc_runtime_version_version_1
func ext_misc_runtime_version_version_1(env interface{}, args []wasmer.Value) []wasmer.Value {
	logger.Trace("executing...")

	instanceContext := env.(*Context)
	dataSpan := args[0].I64()
	//data := asMemorySlice(ctx, dataSpan)
	code := asMemorySlice(instanceContext, dataSpan)

	version, err := GetRuntimeVersion(code)
	if err != nil {
		logger.Errorf("failed to get runtime version: %s", err)
		return mustToWasmMemoryOptionalNil(instanceContext)
	}

	// Note the encoding contains all the latest Core_version fields as defined in
	// https://spec.polkadot.network/#defn-rt-core-version
	// In other words, decoding older version data with missing fields
	// and then encoding it will result in a longer encoding due to the
	// extra version fields. This however remains compatible since the
	// version fields are still encoded in the same order and an older
	// decoder would succeed with the longer encoding.
	encodedData, err := scale.Marshal(version)
	if err != nil {
		logger.Errorf("failed to encode result: %s", err)
		return []wasmer.Value{wasmer.NewI64(0)}
	}

	out, err := toWasmMemoryOptional(instanceContext, encodedData)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return []wasmer.Value{wasmer.NewI64(0)}
	}

	return []wasmer.Value{wasmer.NewI64(out)}
}

//export ext_default_child_storage_read_version_1
func ext_default_child_storage_read_version_1(context unsafe.Pointer,
	childStorageKey, key, valueOut C.int64_t, offset C.int32_t) C.int64_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage
	memory := instanceContext.Memory().Data()

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	keyBytes := asMemorySlice(instanceContext, key)
	value, err := storage.GetChildStorage(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child storage: %s", err)
		return 0
	}

	valueBuf, valueLen := splitPointerSize(int64(valueOut))
	copy(memory[valueBuf:valueBuf+valueLen], value[offset:])

	size := uint32(len(value[offset:]))
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, size)

	sizeSpan, err := toWasmMemoryOptional(instanceContext, sizeBuf)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(sizeSpan)
}

//export ext_default_child_storage_clear_version_1
func ext_default_child_storage_clear_version_1(context unsafe.Pointer, childStorageKey, keySpan C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	key := asMemorySlice(instanceContext, keySpan)

	err := storage.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Errorf("failed to clear child storage: %s", err)
	}
}

//export ext_default_child_storage_clear_prefix_version_1
func ext_default_child_storage_clear_prefix_version_1(context unsafe.Pointer, childStorageKey, prefixSpan C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	prefix := asMemorySlice(instanceContext, prefixSpan)

	err := storage.ClearPrefixInChild(keyToChild, prefix)
	if err != nil {
		logger.Errorf("failed to clear prefix in child: %s", err)
	}
}

//export ext_default_child_storage_exists_version_1
func ext_default_child_storage_exists_version_1(context unsafe.Pointer,
	childStorageKey, key C.int64_t) C.int32_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	keyBytes := asMemorySlice(instanceContext, key)
	child, err := storage.GetChildStorage(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child from child storage: %s", err)
		return 0
	}
	if child != nil {
		return 1
	}
	return 0
}

//export ext_default_child_storage_get_version_1
func ext_default_child_storage_get_version_1(context unsafe.Pointer, childStorageKey, key C.int64_t) C.int64_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	keyBytes := asMemorySlice(instanceContext, key)
	child, err := storage.GetChildStorage(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child from child storage: %s", err)
		return 0
	}

	value, err := toWasmMemoryOptional(instanceContext, child)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(value)
}

//export ext_default_child_storage_next_key_version_1
func ext_default_child_storage_next_key_version_1(context unsafe.Pointer, childStorageKey, key C.int64_t) C.int64_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	keyToChild := asMemorySlice(instanceContext, childStorageKey)
	keyBytes := asMemorySlice(instanceContext, key)
	child, err := storage.GetChildNextKey(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child's next key: %s", err)
		return 0
	}

	value, err := toWasmMemoryOptional(instanceContext, child)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(value)
}

//export ext_default_child_storage_root_version_1
func ext_default_child_storage_root_version_1(context unsafe.Pointer,
	childStorageKey C.int64_t) (ptrSize C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	child, err := storage.GetChild(asMemorySlice(instanceContext, childStorageKey))
	if err != nil {
		logger.Errorf("failed to retrieve child: %s", err)
		return 0
	}

	childRoot, err := child.Hash()
	if err != nil {
		logger.Errorf("failed to encode child root: %s", err)
		return 0
	}

	root, err := toWasmMemoryOptional(instanceContext, childRoot[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(root)
}

//export ext_default_child_storage_set_version_1
func ext_default_child_storage_set_version_1(context unsafe.Pointer,
	childStorageKeySpan, keySpan, valueSpan C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)
	key := asMemorySlice(instanceContext, keySpan)
	value := asMemorySlice(instanceContext, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	err := storage.SetChildStorage(childStorageKey, key, cp)
	if err != nil {
		logger.Errorf("failed to set value in child storage: %s", err)
		return
	}
}

//export ext_default_child_storage_storage_kill_version_1
func ext_default_child_storage_storage_kill_version_1(context unsafe.Pointer, childStorageKeySpan C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)
	err := storage.DeleteChild(childStorageKey)
	panicOnError(err)
}

//export ext_default_child_storage_storage_kill_version_2
func ext_default_child_storage_storage_kill_version_2(context unsafe.Pointer,
	childStorageKeySpan, lim C.int64_t) (allDeleted C.int32_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage
	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit *[]byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warnf("cannot generate limit: %s", err)
		return 0
	}

	_, all, err := storage.DeleteChildLimit(childStorageKey, limit)
	if err != nil {
		logger.Warnf("cannot get child storage: %s", err)
	}

	if all {
		return 1
	}

	return 0
}

type noneRemain uint32

func (noneRemain) Index() uint       { return 0 }
func (nr noneRemain) String() string { return fmt.Sprintf("noneRemain(%d)", nr) }

type someRemain uint32

func (someRemain) Index() uint       { return 1 }
func (sr someRemain) String() string { return fmt.Sprintf("someRemain(%d)", sr) }

//export ext_default_child_storage_storage_kill_version_3
func ext_default_child_storage_storage_kill_version_3(context unsafe.Pointer,
	childStorageKeySpan, lim C.int64_t) (pointerSize C.int64_t) {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage
	childStorageKey := asMemorySlice(instanceContext, childStorageKeySpan)

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit *[]byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warnf("cannot generate limit: %s", err)
	}

	deleted, all, err := storage.DeleteChildLimit(childStorageKey, limit)
	if err != nil {
		logger.Warnf("cannot get child storage: %s", err)
		return C.int64_t(0)
	}

	vdt, err := scale.NewVaryingDataType(noneRemain(0), someRemain(0))
	if err != nil {
		logger.Warnf("cannot create new varying data type: %s", err)
	}

	if all {
		err = vdt.Set(noneRemain(deleted))
	} else {
		err = vdt.Set(someRemain(deleted))
	}
	if err != nil {
		logger.Warnf("cannot set varying data type: %s", err)
		return C.int64_t(0)
	}

	encoded, err := scale.Marshal(vdt)
	if err != nil {
		logger.Warnf("problem marshalling varying data type: %s", err)
		return C.int64_t(0)
	}

	out, err := toWasmMemoryOptional(instanceContext, encoded)
	if err != nil {
		logger.Warnf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(out)
}

//export ext_allocator_free_version_1
func ext_allocator_free_version_1(context unsafe.Pointer, addr C.int32_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	// Deallocate memory
	err := runtimeCtx.Allocator.Deallocate(uint32(addr))
	if err != nil {
		logger.Errorf("failed to free memory: %s", err)
	}
}

//export ext_allocator_malloc_version_1
func ext_allocator_malloc_version_1(context unsafe.Pointer, size C.int32_t) C.int32_t {
	logger.Tracef("executing with size %d...", int64(size))

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)

	// Allocate memory
	res, err := ctx.Allocator.Allocate(uint32(size))
	if err != nil {
		logger.Criticalf("failed to allocate memory: %s", err)
		panic(err)
	}

	return C.int32_t(res)
}

//export ext_hashing_blake2_128_version_1
func ext_hashing_blake2_128_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Blake2b128(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x has hash 0x%x",
		data, hash)

	out, err := toWasmMemorySized(instanceContext, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_blake2_256_version_1
func ext_hashing_blake2_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Blake2bHash(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(instanceContext, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_keccak_256_version_1
func ext_hashing_keccak_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Keccak256(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(instanceContext, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_sha2_256_version_1
func ext_hashing_sha2_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)
	hash := common.Sha256(data)

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(instanceContext, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_256_version_1
func ext_hashing_twox_256_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox256(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := toWasmMemorySized(instanceContext, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_128_version_1
func ext_hashing_twox_128_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox128Hash(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x hash hash 0x%x",
		data, hash)

	out, err := toWasmMemorySized(instanceContext, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_hashing_twox_64_version_1
func ext_hashing_twox_64_version_1(context unsafe.Pointer, dataSpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	data := asMemorySlice(instanceContext, dataSpan)

	hash, err := common.Twox64(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x has hash 0x%x",
		data, hash)

	out, err := toWasmMemorySized(instanceContext, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int32_t(out)
}

//export ext_offchain_index_set_version_1
func ext_offchain_index_set_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	storageKey := asMemorySlice(instanceContext, keySpan)
	newValue := asMemorySlice(instanceContext, valueSpan)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	err := runtimeCtx.NodeStorage.BaseDB.Put(storageKey, cp)
	if err != nil {
		logger.Errorf("failed to set value in raw storage: %s", err)
	}
}

//export ext_offchain_local_storage_clear_version_1
func ext_offchain_local_storage_clear_version_1(context unsafe.Pointer, kind C.int32_t, key C.int64_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
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
		logger.Errorf("failed to clear value from storage: %s", err)
	}
}

//export ext_offchain_is_validator_version_1
func ext_offchain_is_validator_version_1(context unsafe.Pointer) C.int32_t {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Validator {
		return 1
	}
	return 0
}

//export ext_offchain_local_storage_compare_and_set_version_1
func ext_offchain_local_storage_compare_and_set_version_1(context unsafe.Pointer,
	kind C.int32_t, key, oldValue, newValue C.int64_t) (newValueSet C.int32_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
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
		logger.Errorf("failed to get value from storage: %s", err)
		return 0
	}

	oldVal := asMemorySlice(instanceContext, oldValue)
	newVal := asMemorySlice(instanceContext, newValue)
	if reflect.DeepEqual(storedValue, oldVal) {
		cp := make([]byte, len(newVal))
		copy(cp, newVal)
		err = runtimeCtx.NodeStorage.LocalStorage.Put(storageKey, cp)
		if err != nil {
			logger.Errorf("failed to set value in storage: %s", err)
			return 0
		}
	}

	return 1
}

//export ext_offchain_local_storage_get_version_1
func ext_offchain_local_storage_get_version_1(context unsafe.Pointer, kind C.int32_t, key C.int64_t) C.int64_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
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
		logger.Errorf("failed to get value from storage: %s", err)
	}
	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemoryOptional(instanceContext, res)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return 0
	}
	return C.int64_t(ptr)
}

//export ext_offchain_local_storage_set_version_1
func ext_offchain_local_storage_set_version_1(context unsafe.Pointer, kind C.int32_t, key, value C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
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
		logger.Errorf("failed to set value in storage: %s", err)
	}
}

//export ext_offchain_network_state_version_1
func ext_offchain_network_state_version_1(context unsafe.Pointer) C.int64_t {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)
	if runtimeCtx.Network == nil {
		return 0
	}

	nsEnc, err := scale.Marshal(runtimeCtx.Network.NetworkState())
	if err != nil {
		logger.Errorf("failed at encoding network state: %s", err)
		return 0
	}

	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemorySized(instanceContext, nsEnc)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return 0
	}

	return C.int64_t(ptr)
}

//export ext_offchain_random_seed_version_1
func ext_offchain_random_seed_version_1(context unsafe.Pointer) C.int32_t {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		logger.Errorf("failed to generate random seed: %s", err)
	}
	ptr, err := toWasmMemorySized(instanceContext, seed)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
	}
	return C.int32_t(ptr)
}

//export ext_offchain_submit_transaction_version_1
func ext_offchain_submit_transaction_version_1(context unsafe.Pointer, data C.int64_t) C.int64_t {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	extBytes := asMemorySlice(instanceContext, data)

	var extrinsic []byte
	err := scale.Unmarshal(extBytes, &extrinsic)
	if err != nil {
		logger.Errorf("failed to decode extrinsic data: %s", err)
	}

	// validate the transaction
	txv := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	vtx := transaction.NewValidTransaction(extrinsic, txv)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	runtimeCtx.Transaction.AddToPool(vtx)

	ptr, err := toWasmMemoryOptionalNil(instanceContext)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
	}
	return ptr
}

//export ext_offchain_timestamp_version_1
func ext_offchain_timestamp_version_1(_ unsafe.Pointer) C.int64_t {
	logger.Trace("executing...")

	now := time.Now().Unix()
	return C.int64_t(now)
}

//export ext_offchain_sleep_until_version_1
func ext_offchain_sleep_until_version_1(_ unsafe.Pointer, deadline C.int64_t) {
	logger.Trace("executing...")

	dur := time.Until(time.UnixMilli(int64(deadline)))
	if dur > 0 {
		time.Sleep(dur)
	}
}

//export ext_offchain_http_request_start_version_1
func ext_offchain_http_request_start_version_1(context unsafe.Pointer,
	methodSpan, uriSpan, metaSpan C.int64_t) (pointerSize C.int64_t) {
	logger.Debug("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	runtimeCtx := instanceContext.Data().(*runtime.Context)

	httpMethod := asMemorySlice(instanceContext, methodSpan)
	uri := asMemorySlice(instanceContext, uriSpan)

	result := scale.NewResult(int16(0), nil)

	reqID, err := runtimeCtx.OffchainHTTPSet.StartRequest(string(httpMethod), string(uri))
	if err != nil {
		// StartRequest error already was logged
		logger.Errorf("failed to start request: %s", err)
		err = result.Set(scale.Err, nil)
	} else {
		err = result.Set(scale.OK, reqID)
	}

	// note: just check if an error occurs while setting the result data
	if err != nil {
		logger.Errorf("failed to set the result data: %s", err)
		return C.int64_t(0)
	}

	enc, err := scale.Marshal(result)
	if err != nil {
		logger.Errorf("failed to scale marshal the result: %s", err)
		return C.int64_t(0)
	}

	ptr, err := toWasmMemory(instanceContext, enc)
	if err != nil {
		logger.Errorf("failed to allocate result on memory: %s", err)
		return C.int64_t(0)
	}

	return C.int64_t(ptr)
}

//export ext_offchain_http_request_add_header_version_1
func ext_offchain_http_request_add_header_version_1(context unsafe.Pointer,
	reqID C.int32_t, nameSpan, valueSpan C.int64_t) (pointerSize C.int64_t) {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)

	name := asMemorySlice(instanceContext, nameSpan)
	value := asMemorySlice(instanceContext, valueSpan)

	runtimeCtx := instanceContext.Data().(*runtime.Context)
	offchainReq := runtimeCtx.OffchainHTTPSet.Get(int16(reqID))

	result := scale.NewResult(nil, nil)
	resultMode := scale.OK

	err := offchainReq.AddHeader(string(name), string(value))
	if err != nil {
		logger.Errorf("failed to add request header: %s", err)
		resultMode = scale.Err
	}

	err = result.Set(resultMode, nil)
	if err != nil {
		logger.Errorf("failed to set the result data: %s", err)
		return C.int64_t(0)
	}

	enc, err := scale.Marshal(result)
	if err != nil {
		logger.Errorf("failed to scale marshal the result: %s", err)
		return C.int64_t(0)
	}

	ptr, err := toWasmMemory(instanceContext, enc)
	if err != nil {
		logger.Errorf("failed to allocate result on memory: %s", err)
		return C.int64_t(0)
	}

	return C.int64_t(ptr)
}

//export ext_storage_append_version_1
func ext_storage_append_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)
	valueAppend := asMemorySlice(instanceContext, valueSpan)
	logger.Debugf(
		"will append value 0x%x to values at key 0x%x",
		valueAppend, key)

	cp := make([]byte, len(valueAppend))
	copy(cp, valueAppend)

	err := storageAppend(storage, key, cp)
	if err != nil {
		logger.Errorf("failed appending to storage: %s", err)
	}
}

//export ext_storage_changes_root_version_1
func ext_storage_changes_root_version_1(context unsafe.Pointer, parentHashSpan C.int64_t) C.int64_t {
	logger.Trace("executing...")
	logger.Debug("returning None")

	instanceContext := wasmer.IntoInstanceContext(context)

	rootSpan, err := toWasmMemoryOptionalNil(instanceContext)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return rootSpan
}

//export ext_storage_clear_version_1
func ext_storage_clear_version_1(context unsafe.Pointer, keySpan C.int64_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)

	logger.Debugf("key: 0x%x", key)
	err := storage.Delete(key)
	panicOnError(err)
}

//export ext_storage_clear_prefix_version_1
func ext_storage_clear_prefix_version_1(context unsafe.Pointer, prefixSpan C.int64_t) {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	prefix := asMemorySlice(instanceContext, prefixSpan)
	logger.Debugf("prefix: 0x%x", prefix)

	err := storage.ClearPrefix(prefix)
	panicOnError(err)
}

//export ext_storage_clear_prefix_version_2
func ext_storage_clear_prefix_version_2(context unsafe.Pointer, prefixSpan, lim C.int64_t) C.int64_t {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	prefix := asMemorySlice(instanceContext, prefixSpan)
	logger.Debugf("prefix: 0x%x", prefix)

	limitBytes := asMemorySlice(instanceContext, lim)

	var limit []byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warnf("failed scale decoding limit: %s", err)
		return mustToWasmMemoryNil(instanceContext)
	}

	if len(limit) == 0 {
		// limit is None, set limit to max
		limit = []byte{0xff, 0xff, 0xff, 0xff}
	}

	limitUint := binary.LittleEndian.Uint32(limit)
	numRemoved, all, err := storage.ClearPrefixLimit(prefix, limitUint)
	if err != nil {
		logger.Errorf("failed to clear prefix limit: %s", err)
		return mustToWasmMemoryNil(instanceContext)
	}

	encBytes, err := toKillStorageResultEnum(all, numRemoved)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return mustToWasmMemoryNil(instanceContext)
	}

	valueSpan, err := toWasmMemory(instanceContext, encBytes)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return mustToWasmMemoryNil(instanceContext)
	}

	return C.int64_t(valueSpan)
}

//export ext_storage_exists_version_1
func ext_storage_exists_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int32_t {
	logger.Trace("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)
	logger.Debugf("key: 0x%x", key)

	value := storage.Get(key)
	if value != nil {
		return 1
	}

	return 0
}

//export ext_storage_get_version_1
func ext_storage_get_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int64_t {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)
	logger.Debugf("key: 0x%x", key)

	value := storage.Get(key)
	logger.Debugf("value: 0x%x", value)

	valueSpan, err := toWasmMemoryOptional(instanceContext, value)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return mustToWasmMemoryOptionalNil(instanceContext)
	}

	return C.int64_t(valueSpan)
}

//export ext_storage_next_key_version_1
func ext_storage_next_key_version_1(context unsafe.Pointer, keySpan C.int64_t) C.int64_t {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	key := asMemorySlice(instanceContext, keySpan)

	next := storage.NextKey(key)
	logger.Debugf(
		"key: 0x%x; next key 0x%x",
		key, next)

	nextSpan, err := toWasmMemoryOptional(instanceContext, next)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(nextSpan)
}

//export ext_storage_read_version_1
func ext_storage_read_version_1(context unsafe.Pointer, keySpan, valueOut C.int64_t, offset C.int32_t) C.int64_t {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage
	memory := instanceContext.Memory().Data()

	key := asMemorySlice(instanceContext, keySpan)
	value := storage.Get(key)
	logger.Debugf(
		"key 0x%x has value 0x%x",
		key, value)

	if value == nil {
		return mustToWasmMemoryOptionalNil(instanceContext)
	}

	var size uint32
	if uint32(offset) <= uint32(len(value)) {
		size = uint32(len(value[offset:]))
		valueBuf, valueLen := splitPointerSize(int64(valueOut))
		copy(memory[valueBuf:valueBuf+valueLen], value[offset:])
	}

	sizeSpan, err := toWasmMemoryOptionalUint32(instanceContext, &size)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(sizeSpan)
}

//export ext_storage_root_version_1
func ext_storage_root_version_1(context unsafe.Pointer) C.int64_t {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	storage := instanceContext.Data().(*runtime.Context).Storage

	root, err := storage.Root()
	if err != nil {
		logger.Errorf("failed to get storage root: %s", err)
		return 0
	}

	logger.Debugf("root hash is: %s", root)

	rootSpan, err := toWasmMemory(instanceContext, root[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}

	return C.int64_t(rootSpan)
}

//export ext_storage_root_version_2
func ext_storage_root_version_2(context unsafe.Pointer, version C.int32_t) C.int64_t {
	// TODO: update to use state trie version 1 (#2418)
	return ext_storage_root_version_1(context)
}

//export ext_storage_set_version_1
func ext_storage_set_version_1(context unsafe.Pointer, keySpan, valueSpan C.int64_t) {
	logger.Trace("executing...")

	instanceContext := wasmer.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*runtime.Context)
	storage := ctx.Storage

	key := asMemorySlice(instanceContext, keySpan)
	value := asMemorySlice(instanceContext, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	logger.Debugf(
		"key 0x%x has value 0x%x",
		key, value)
	err := storage.Put(key, cp)
	panicOnError(err)
}

//export ext_storage_start_transaction_version_1
func ext_storage_start_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.BeginStorageTransaction()
}

//export ext_storage_rollback_transaction_version_1
func ext_storage_rollback_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.RollbackStorageTransaction()
}

//export ext_storage_commit_transaction_version_1
func ext_storage_commit_transaction_version_1(context unsafe.Pointer) {
	logger.Debug("executing...")
	instanceContext := wasmer.IntoInstanceContext(context)
	instanceContext.Data().(*runtime.Context).Storage.CommitStorageTransaction()
}
