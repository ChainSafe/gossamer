// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wazero_runtime

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory/proof"
	"github.com/tetratelabs/wazero/api"
)

var (
	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "wazero"),
	)

	emptyByteVectorEncoded []byte = scale.MustMarshal([]byte{})
	noneEncoded            []byte = []byte{0x00}
	allZeroesBytes                = [32]byte{}
)

const (
	validateSignatureFail = "failed to validate signature"
)

// toPointerSize converts an uint32 pointer and uint32 size
// to an int64 pointer size.
func newPointerSize(ptr, size uint32) (pointerSize uint64) {
	return uint64(ptr) | (uint64(size) << 32)
}

// splitPointerSize converts a 64bit pointer size to an
// uint32 pointer and a uint32 size.
func splitPointerSize(pointerSize uint64) (ptr, size uint32) {
	return uint32(pointerSize), uint32(pointerSize >> 32)
}

// read will read from 64 bit pointer size and return a byte slice
func read(m api.Module, pointerSize uint64) (data []byte) {
	ptr, size := splitPointerSize(pointerSize)
	data, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("write overflow")
	}
	return data
}

// copies a Go byte slice to wasm memory and returns the corresponding
// 64 bit pointer size.
func write(m api.Module, allocator runtime.Allocator, data []byte) (pointerSize uint64, err error) {
	size := uint32(len(data))
	pointer, err := allocator.Allocate(m.Memory(), size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	ok := m.Memory().Write(pointer, data)
	if !ok {
		return 0, fmt.Errorf("out of range")
	}
	return newPointerSize(pointer, size), nil
}

func mustWrite(m api.Module, allocator runtime.Allocator, data []byte) (pointerSize uint64) {
	pointerSize, err := write(m, allocator, data)
	if err != nil {
		panic(err)
	}
	return pointerSize
}

func ext_logging_log_version_1(ctx context.Context, m api.Module, level int32, targetData, msgData uint64) {
	target := string(read(m, targetData))
	msg := string(read(m, msgData))

	line := fmt.Sprintf("target=%s message=%s", target, msg)

	switch int(level) {
	case 0:
		logger.Critical(line)
	case 1:
		logger.Warn(line)
	case 2:
		logger.Info(line)
	case 3:
		logger.Debug(line)
	case 4:
		logger.Trace(line)
	default:
		logger.Errorf("level=%d target=%s message=%s", int(level), target, msg)
	}
}

func ext_crypto_ecdsa_generate_version_1(ctx context.Context, m api.Module, _ uint32, _ uint64) uint32 {
	panic("TODO impl: see https://github.com/ChainSafe/gossamer/issues/3769 ")
}

func ext_crypto_ed25519_generate_version_1(
	ctx context.Context, m api.Module, keyTypeID uint32, seedSpan uint64) uint32 {
	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("out of range read")
	}
	seedBytes := read(m, seedSpan)

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

	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
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

	ret, err := write(m, rtCtx.Allocator, kp.Public().Encode())
	if err != nil {
		logger.Warnf("failed to allocate memory: %s", err)
		return 0
	}

	logger.Debug("generated ed25519 keypair with public key: " + kp.Public().Hex())
	return uint32(ret)
}

func ext_crypto_ed25519_public_keys_version_1(ctx context.Context, m api.Module, keyTypeID uint32) uint64 {
	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("out of range read")
	}

	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"error for id 0x%x: keystore type is %s and not the expected ed25519",
			id, ks.Type())
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	keys := ks.PublicKeys()
	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	keysPtr, err := write(m, rtCtx.Allocator, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}
	return keysPtr
}

func ext_crypto_ed25519_sign_version_1(ctx context.Context, m api.Module, keyTypeID, key uint32, msg uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("out of range read")
	}

	pubKeyData, ok := m.Memory().Read(key, 32)
	if !ok {
		panic("out of range read")
	}

	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Errorf("failed to get public keys: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	sig, err := signingKey.Sign(read(m, msg))
	if err != nil {
		logger.Error("could not sign message")
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	var fixedSize [64]byte
	copy(fixedSize[:], sig)
	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&fixedSize))
}

func ext_crypto_ed25519_verify_version_1(ctx context.Context, m api.Module, sig uint32, msg uint64, key uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	memory := m.Memory()
	sigVerifier := rtCtx.SigVerifier

	signature, ok := memory.Read(sig, 64)
	if !ok {
		panic("read overflow")
	}
	message := read(m, msg)
	pubKeyData, ok := memory.Read(key, 32)
	if !ok {
		panic("read overflow")
	}

	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("failed to create public key")
		return 0
	}

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pubKey.Encode(),
			Sign:       signature,
			Msg:        message,
			VerifyFunc: ed25519.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return 1
	}

	if ok, err := pubKey.Verify(message, signature); err != nil || !ok {
		logger.Error("failed to verify")
		return 0
	}

	logger.Debug("verified ed25519 signature")
	return 1
}

func ext_crypto_secp256k1_ecdsa_recover_version_1(ctx context.Context, m api.Module, sig, msg uint32) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message, ok := m.Memory().Read(msg, 32)
	if !ok {
		panic("read overflow")
	}
	signature, ok := m.Memory().Read(sig, 65)
	if !ok {
		panic("read overflow")
	}

	res := scale.NewResult([64]byte{}, nil)

	pub, err := secp256k1.RecoverPublicKey(message, signature)
	if err != nil {
		logger.Errorf("failed to recover public key: %s", err)
		err := res.Set(scale.Err, nil)
		if err != nil {
			panic(err)
		}
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(res))
		if err != nil {
			panic(err)
		}
		return ret
	}

	logger.Debugf(
		"recovered public key of length %d: 0x%x",
		len(pub), pub)

	var fixedSize [64]byte
	copy(fixedSize[:], pub[1:])

	err = res.Set(scale.OK, fixedSize)
	if err != nil {
		panic(err)
	}

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(res))
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_crypto_secp256k1_ecdsa_recover_version_2(ctx context.Context, m api.Module, sig, msg uint32) uint64 {
	return ext_crypto_secp256k1_ecdsa_recover_version_1(ctx, m, sig, msg)
}

func ext_crypto_ecdsa_verify_version_2(ctx context.Context, m api.Module, sig uint32, msg uint64, key uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	sigVerifier := rtCtx.SigVerifier

	message := read(m, msg)
	signature, ok := m.Memory().Read(sig, 64)
	if !ok {
		panic("read overflow")
	}
	pubKey, ok := m.Memory().Read(key, 33)
	if !ok {
		panic("read overflow")
	}

	pub := new(secp256k1.PublicKey)
	err := pub.Decode(pubKey)
	if err != nil {
		logger.Errorf("failed to decode public key: %s", err)
		return 0
	}

	logger.Debugf("pub=%s, message=0x%x, signature=0x%x",
		pub.Hex(), fmt.Sprintf("0x%x", message), fmt.Sprintf("0x%x", signature))

	hash, err := common.Blake2bHash(message)
	if err != nil {
		logger.Errorf("failed to hash message: %s", err)
		return 0
	}

	if sigVerifier.IsStarted() {
		signature := crypto.SignatureInfo{
			PubKey:     pub.Encode(),
			Sign:       signature,
			Msg:        hash[:],
			VerifyFunc: secp256k1.VerifySignature,
		}
		sigVerifier.Add(&signature)
		return 1
	}

	ok, err = pub.Verify(hash[:], signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Errorf(message)
		return 0
	}

	logger.Debug("validated signature")
	return 1
}

func ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(
	ctx context.Context, m api.Module, sig, msg uint32) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message, ok := m.Memory().Read(msg, 32)
	if !ok {
		panic("read overflow")
	}
	signature, ok := m.Memory().Read(sig, 65)
	if !ok {
		panic("read overflow")
	}

	res := scale.NewResult([33]byte{}, nil)
	cpub, err := secp256k1.RecoverPublicKeyCompressed(message, signature)
	if err != nil {
		logger.Errorf("failed to recover public key: %s", err)
		err := res.Set(scale.Err, nil)
		if err != nil {
			panic(err)
		}
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(res))
		if err != nil {
			panic(err)
		}
		return ret
	}

	var fixed [33]byte
	copy(fixed[:], cpub)

	err = res.Set(scale.OK, fixed)
	if err != nil {
		panic(err)
	}

	logger.Debugf(
		"recovered public key of length %d: 0x%x",
		len(cpub), cpub)

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(res))
	if err != nil {
		panic(err)
	}

	return ret
}

func ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(
	ctx context.Context, m api.Module, sig, msg uint32) uint64 {
	return ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(ctx, m, sig, msg)
}

func ext_crypto_sr25519_generate_version_1(
	ctx context.Context, m api.Module, keyTypeID uint32, seedSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("read overflow")
	}

	seedBytes := read(m, seedSpan)

	var seed *[]byte
	err := scale.Unmarshal(seedBytes, &seed)
	if err != nil {
		logger.Warnf("cannot generate key: %s", err)
		return 0
	}

	var kp *sr25519.Keypair
	if seed != nil {
		kp, err = sr25519.NewKeypairFromMnenomic(string(*seed), "")
	} else {
		kp, err = sr25519.GenerateKeypair()
	}

	if err != nil {
		logger.Tracef("cannot generate key: %s", err)
		panic(err)
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id "+common.BytesToHex(id)+": %s", err)
		return 0
	}

	err = ks.Insert(kp)
	if err != nil {
		logger.Warnf("failed to insert key: %s", err)
		return 0
	}

	ret, err := write(m, rtCtx.Allocator, kp.Public().Encode())
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return 0
	}

	logger.Debug("generated sr25519 keypair with public key: " + kp.Public().Hex())

	ptr, _ := splitPointerSize(ret)
	return ptr
}

func ext_crypto_sr25519_public_keys_version_1(ctx context.Context, m api.Module, keyTypeID uint32) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("read overflow")
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id "+common.BytesToHex(id)+": %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	if ks.Type() != crypto.Sr25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"keystore type for id 0x%x is %s and not expected sr25519",
			id, ks.Type())
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Marshal(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	keysPtr, err := write(m, rtCtx.Allocator, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	return keysPtr
}

func ext_crypto_sr25519_sign_version_1(ctx context.Context, m api.Module, keyTypeID, key uint32, msg uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	id, ok := m.Memory().Read(keyTypeID, 4)
	if !ok {
		panic("read overflow")
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	kb, ok := m.Memory().Read(key, 32)
	if !ok {
		panic("read overflow")
	}

	pubKey, err := sr25519.NewPublicKey(kb)
	if err != nil {
		logger.Errorf("failed to get public key: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	msgData := read(m, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Errorf("could not sign message: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	var fixedSig [64]byte
	copy(fixedSig[:], sig)
	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&fixedSig))
}

func ext_crypto_sr25519_verify_version_1(ctx context.Context, m api.Module, sig uint32, msg uint64, key uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	sigVerifier := rtCtx.SigVerifier

	message := read(m, msg)
	signature, ok := m.Memory().Read(sig, 64)
	if !ok {
		panic("read overflow")
	}

	pubKeyBytes, ok := m.Memory().Read(key, 32)
	if !ok {
		panic("read overflow")
	}
	pub, err := sr25519.NewPublicKey(pubKeyBytes)
	if err != nil {
		logger.Error("invalid sr25519 public key")
		return 0
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
		return 1
	}

	ok, err = pub.VerifyDeprecated(message, signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Debugf(message)
		// this fails at block 3876, which seems to be expected, based on discussions
		return 1
	}

	logger.Debug("verified sr25519 signature")
	return 1
}

func ext_crypto_sr25519_verify_version_2(ctx context.Context, m api.Module, sig uint32, msg uint64, key uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	pubKeyBytes, ok := m.Memory().Read(key, 32)
	if !ok {
		panic("read overflow")
	}

	// prevents Polkadot zero-address crash using
	// ext_crypto_sr25519_verify_version_1
	// https://pacna.org/dot-zero-addr/
	if bytes.Equal(pubKeyBytes, allZeroesBytes[:]) {
		return ext_crypto_sr25519_verify_version_1(ctx, m, sig, msg, key)
	}

	sigVerifier := rtCtx.SigVerifier

	message := read(m, msg)
	signature, ok := m.Memory().Read(sig, 64)
	if !ok {
		panic("read overflow")
	}

	pub, err := sr25519.NewPublicKey(pubKeyBytes)
	if err != nil {
		logger.Error("invalid sr25519 public key")
		return 0
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
		return 1
	}

	ok, err = pub.Verify(message, signature)
	if err != nil || !ok {
		message := validateSignatureFail
		if err != nil {
			message += ": " + err.Error()
		}
		logger.Errorf(message)
		return 0
	}

	logger.Debug("validated signature")
	return 1
}

func ext_crypto_start_batch_verify_version_1(ctx context.Context, m api.Module) {
	// TODO: fix and re-enable signature verification (#1405)
	// beginBatchVerify(context)
}

func ext_crypto_finish_batch_verify_version_1(ctx context.Context, m api.Module) uint32 {
	// TODO: fix and re-enable signature verification (#1405)
	// return finishBatchVerify(context)
	return 1
}

func ext_trie_blake2_256_root_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	return ext_trie_blake2_256_root_version_2(ctx, m, dataSpan, 0)
}

func ext_trie_blake2_256_root_version_2(ctx context.Context, m api.Module, dataSpan uint64, version uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	stateVersion, err := trie.ParseVersion(uint8(version))
	if err != nil {
		logger.Errorf("failed parsing state version: %s", err)
		return 0
	}

	data := read(m, dataSpan)

	// this function is expecting an array of (key, value) tuples
	var entries trie.Entries
	if err := scale.Unmarshal(data, &entries); err != nil {
		logger.Errorf("failed scale decoding data: %s", err)
		return 0
	}

	hash, err := stateVersion.Root(inmemory.NewEmptyInmemoryTrie(), entries)
	if err != nil {
		logger.Errorf("failed computing trie Merkle root hash: %s", err)
		return 0
	}

	// allocate memory for value and copy value to memory
	ptr, err := rtCtx.Allocator.Allocate(m.Memory(), 32)
	if err != nil {
		logger.Errorf("failed allocating: %s", err)
		return 0
	}

	logger.Debugf("root hash is %s", hash)
	m.Memory().Write(ptr, hash[:])
	return ptr
}

func ext_trie_blake2_256_ordered_root_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	return ext_trie_blake2_256_ordered_root_version_2(ctx, m, dataSpan, 0)
}

func ext_trie_blake2_256_ordered_root_version_2(
	ctx context.Context, m api.Module, dataSpan uint64, version uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	stateVersion, err := trie.ParseVersion(uint8(version))
	if err != nil {
		logger.Errorf("failed parsing state version: %s", err)
		return 0
	}

	var values [][]byte
	err = scale.Unmarshal(data, &values)
	if err != nil {
		logger.Errorf("failed scale decoding data: %s", err)
		return 0
	}

	var entries trie.Entries

	for i, value := range values {
		key, err := scale.Marshal(big.NewInt(int64(i)))
		if err != nil {
			logger.Errorf("failed scale encoding value index %d: %s", i, err)
			return 0
		}

		entries = append(entries, trie.Entry{Key: key, Value: value})
	}

	// allocate memory for value and copy value to memory
	ptr, err := rtCtx.Allocator.Allocate(m.Memory(), 32)
	if err != nil {
		logger.Errorf("failed allocating: %s", err)
		return 0
	}

	hash, err := stateVersion.Root(inmemory.NewEmptyInmemoryTrie(), entries)
	if err != nil {
		logger.Errorf("failed computing trie Merkle root hash: %s", err)
		return 0
	}

	logger.Debugf("root hash is %s", hash)
	m.Memory().Write(ptr, hash[:])
	return ptr
}

func ext_trie_blake2_256_verify_proof_version_1(
	ctx context.Context, m api.Module, rootSpan uint32, proofSpan, keySpan, valueSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	toDecProofs := read(m, proofSpan)
	var encodedProofNodes [][]byte
	err := scale.Unmarshal(toDecProofs, &encodedProofNodes)
	if err != nil {
		logger.Errorf("failed scale decoding proof data: %s", err)
		return 0
	}

	key := read(m, keySpan)
	value := read(m, valueSpan)

	trieRoot, ok := m.Memory().Read(rootSpan, 32)
	if !ok {
		panic("read overflow")
	}

	err = proof.Verify(encodedProofNodes, trieRoot, key, value)
	if err != nil {
		logger.Errorf("failed proof verification: %s", err)
		return 0
	}

	return 1
}

func ext_trie_blake2_256_verify_proof_version_2(
	ctx context.Context, m api.Module, rootSpan uint32, proofSpan, keySpan, valueSpan uint64, version uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	_, err := trie.ParseVersion(uint8(version))
	if err != nil {
		logger.Errorf("failed parsing state version: %s", err)
		return 0
	}

	toDecProofs := read(m, proofSpan)
	var encodedProofNodes [][]byte
	err = scale.Unmarshal(toDecProofs, &encodedProofNodes)
	if err != nil {
		logger.Errorf("failed scale decoding proof data: %s", err)
		return 0
	}

	key := read(m, keySpan)
	value := read(m, valueSpan)

	trieRoot, ok := m.Memory().Read(rootSpan, 32)
	if !ok {
		panic("read overflow")
	}

	err = proof.Verify(encodedProofNodes, trieRoot, key, value)
	if err != nil {
		logger.Errorf("failed proof verification: %s", err)
		return 0
	}

	return 1
}

func ext_misc_print_hex_version_1(ctx context.Context, m api.Module, dataSpan uint64) {
	data := read(m, dataSpan)
	logger.Debugf("data: 0x%x", data)
}

func ext_misc_print_num_version_1(ctx context.Context, m api.Module, data uint64) {
	logger.Debugf("num: %d", int64(data))
}

func ext_misc_print_utf8_version_1(ctx context.Context, m api.Module, dataSpan uint64) {
	data := read(m, dataSpan)
	logger.Debug("utf8: " + string(data))
}

// GetRuntimeVersion finds the runtime version by initiating a temporary
// runtime instance using the WASM code provided, and querying it.
func GetRuntimeVersion(code []byte) (version runtime.Version, err error) {
	config := Config{
		LogLvl: log.DoNotChange,
	}
	instance, err := NewInstance(code, config)
	if err != nil {
		return version, fmt.Errorf("creating runtime instance: %w", err)
	}
	defer instance.Stop()

	version, err = instance.Version()
	if err != nil {
		return version, fmt.Errorf("running runtime: %w", err)
	}

	return version, nil
}

func ext_misc_runtime_version_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	code := read(m, dataSpan)

	version, err := GetRuntimeVersion(code)
	if err != nil {
		logger.Errorf("failed to get runtime version: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
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
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&encodedData))
}

func ext_default_child_storage_read_version_1(
	ctx context.Context, m api.Module, childStorageKey, key, valueOut uint64, offset uint32) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	keyToChild := read(m, childStorageKey)
	keyBytes := read(m, key)
	value, err := rtCtx.Storage.GetChildStorage(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child storage: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	valueBuf, _ := splitPointerSize(valueOut)
	ok := m.Memory().Write(valueBuf, value[offset:])
	if !ok {
		panic("write overflow")
	}

	size := uint32(len(value[offset:]))
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, size)

	// this is expected to be Option(Vec<u8>)
	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&sizeBuf))
}

func ext_default_child_storage_set_version_1(
	ctx context.Context, m api.Module, childStorageKeySpan, keySpan, valueSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	childStorageKey := read(m, childStorageKeySpan)
	key := read(m, keySpan)
	value := read(m, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	err := storage.SetChildStorage(childStorageKey, key, cp)
	if err != nil {
		logger.Errorf("failed to set value in child storage: %s", err)
		panic(err)
	}
}

func ext_default_child_storage_clear_version_1(ctx context.Context, m api.Module, childStorageKey, keySpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	key := read(m, keySpan)

	err := storage.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Errorf("failed to clear child storage: %s", err)
	}
}

func ext_default_child_storage_clear_prefix_version_1(
	ctx context.Context, m api.Module, childStorageKey, prefixSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	prefix := read(m, prefixSpan)

	err := storage.ClearPrefixInChild(keyToChild, prefix)
	if err != nil {
		logger.Errorf("failed to clear prefix in child: %s", err)
	}
}

// NewDigestItem returns a new VaryingDataType to represent a DigestItem
func NewKillStorageResult(deleted uint32, allDeleted bool) scale.VaryingDataType {
	killStorageResult := killStorageResult{}

	var err error
	if allDeleted {
		err = killStorageResult.SetValue(noneRemain(deleted))
	} else {
		err = killStorageResult.SetValue(someRemain(deleted))
	}
	if err != nil {
		panic(err)
	}

	return &killStorageResult
}

//export ext_default_child_storage_clear_prefix_version_2
func ext_default_child_storage_clear_prefix_version_2(ctx context.Context, m api.Module,
	childStorageKey, prefixSpan, limitSpan uint64) uint64 {

	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	prefix := read(m, prefixSpan)
	limitBytes := read(m, limitSpan)

	var limit []byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warnf("failed scale decoding limit: %s", err)
		panic(err)
	}

	if len(limit) == 0 {
		// limit is None, set limit to max
		limit = []byte{0xff, 0xff, 0xff, 0xff}
	}

	limitUint := binary.LittleEndian.Uint32(limit)

	deleted, allDeleted, err := storage.ClearPrefixInChildWithLimit(
		keyToChild, prefix, limitUint)
	if err != nil {
		logger.Errorf("failed to clear prefix in child with limit: %s", err)
	}

	killStorageResult := NewKillStorageResult(deleted, allDeleted)

	encodedKillStorageResult, err := scale.Marshal(killStorageResult)
	if err != nil {
		logger.Errorf("failed to encode result: %s", err)
		return 0
	}

	resultSpan, err := write(m, rtCtx.Allocator, scale.MustMarshal(&encodedKillStorageResult))
	if err != nil {
		panic(err)
	}

	return resultSpan
}

func ext_default_child_storage_exists_version_1(ctx context.Context, m api.Module, childStorageKey, key uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	keyBytes := read(m, key)
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

func ext_default_child_storage_get_version_1(ctx context.Context, m api.Module, childStorageKey, key uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	keyBytes := read(m, key)
	child, err := storage.GetChildStorage(keyToChild, keyBytes)
	var encodedChildOptional []byte

	if err != nil || child == nil {
		logger.Warnf("child storage not found: %s", err)
		encodedChildOptional = noneEncoded
	} else {
		encodedChildOptional = scale.MustMarshal(&child)
	}

	return mustWrite(m, rtCtx.Allocator, encodedChildOptional)
}

func ext_default_child_storage_next_key_version_1(
	ctx context.Context, m api.Module, childStorageKey, key uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	keyToChild := read(m, childStorageKey)
	keyBytes := read(m, key)
	childNextKey, err := storage.GetChildNextKey(keyToChild, keyBytes)
	if err != nil {
		logger.Errorf("failed to get child's next key: %s", err)
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	if childNextKey == nil {
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&childNextKey))
}

func ext_default_child_storage_root_version_1(
	ctx context.Context, m api.Module, childStorageKey uint64) (ptrSize uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage
	child, err := storage.GetChild(read(m, childStorageKey))
	if err != nil {
		logger.Errorf("failed to retrieve child: %s", err)
		return 0
	}

	childRoot, err := trie.V0.Hash(child)
	if err != nil {
		logger.Errorf("failed to encode child root: %s", err)
		return 0
	}
	childRootSlice := childRoot[:]

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(&childRootSlice))
	if err != nil {
		panic(err)
	}
	return ret
}

//export ext_default_child_storage_root_version_2
func ext_default_child_storage_root_version_2(ctx context.Context, m api.Module, childStorageKey uint64,
	version uint32) (ptrSize uint64) { //skipcq: RVV-B0012
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage
	key := read(m, childStorageKey)
	child, err := storage.GetChild(key)
	if err != nil {
		logger.Errorf("failed to retrieve child: %s", err)
		return mustWrite(m, rtCtx.Allocator, emptyByteVectorEncoded)
	}

	stateVersion, err := trie.ParseVersion(uint8(version))
	if err != nil {
		logger.Errorf("failed parsing state version: %s", err)
		return 0
	}

	childRoot, err := stateVersion.Hash(child)
	if err != nil {
		logger.Errorf("failed to encode child root: %s", err)
		return mustWrite(m, rtCtx.Allocator, emptyByteVectorEncoded)
	}
	childRootSlice := childRoot[:]

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(&childRootSlice))
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_default_child_storage_storage_kill_version_1(ctx context.Context, m api.Module, childStorageKeySpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	childStorageKey := read(m, childStorageKeySpan)
	err := storage.DeleteChild(childStorageKey)
	if err != nil {
		panic(err)
	}
}

func ext_default_child_storage_storage_kill_version_2(
	ctx context.Context, m api.Module, childStorageKeySpan, lim uint64) (allDeleted uint32) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage
	childStorageKey := read(m, childStorageKeySpan)

	limitBytes := read(m, lim)

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

type killStorageResult struct {
	inner any
}
type killStorageResultValues interface {
	noneRemain | someRemain
}

func setkillStorageResult[Value killStorageResultValues](mvdt *killStorageResult, value Value) {
	mvdt.inner = value
}

func (mvdt *killStorageResult) SetValue(value any) (err error) {
	switch value := value.(type) {
	case noneRemain:
		setkillStorageResult(mvdt, value)
		return
	case someRemain:
		setkillStorageResult(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt killStorageResult) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case noneRemain:
		return 0, mvdt.inner, nil
	case someRemain:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt killStorageResult) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt killStorageResult) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return noneRemain(0), nil
	case 1:
		return someRemain(0), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

type noneRemain uint32

type someRemain uint32

func ext_default_child_storage_storage_kill_version_3(
	ctx context.Context, m api.Module, childStorageKeySpan, lim uint64) (pointerSize uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage
	childStorageKey := read(m, childStorageKeySpan)

	var option *[]byte

	limitBytes := read(m, lim)
	var limit *[]byte
	err := scale.Unmarshal(limitBytes, &limit)
	if err != nil {
		logger.Warnf("cannot generate limit: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(option))
		if err != nil {
			panic(err)
		}
		return ret
	}

	deleted, all, err := storage.DeleteChildLimit(childStorageKey, limit)
	if err != nil {
		logger.Warnf("cannot get child storage: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(option))
		if err != nil {
			panic(err)
		}
		return ret
	}

	vdt := killStorageResult{}

	if all {
		err = vdt.SetValue(noneRemain(deleted))
	} else {
		err = vdt.SetValue(someRemain(deleted))
	}
	if err != nil {
		logger.Warnf("cannot set varying data type: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(option))
		if err != nil {
			panic(err)
		}
		return ret
	}

	encoded, err := scale.Marshal(vdt)
	if err != nil {
		logger.Warnf("problem marshalling varying data type: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(option))
		if err != nil {
			panic(err)
		}
		return ret
	}

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(&encoded))
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_hashing_blake2_128_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Blake2b128(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x has hash 0x%x",
		data, hash)

	out, err := write(m, rtCtx.Allocator, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_blake2_256_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Blake2bHash(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := write(m, rtCtx.Allocator, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_keccak_256_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Keccak256(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := write(m, rtCtx.Allocator, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_sha2_256_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)
	hash := common.Sha256(data)

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := write(m, rtCtx.Allocator, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_twox_256_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Twox256(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf("data 0x%x has hash %s", data, hash)

	out, err := write(m, rtCtx.Allocator, hash[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_twox_128_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Twox128Hash(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x hash hash 0x%x",
		data, hash)

	out, err := write(m, rtCtx.Allocator, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_hashing_twox_64_version_1(ctx context.Context, m api.Module, dataSpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	data := read(m, dataSpan)

	hash, err := common.Twox64(data)
	if err != nil {
		logger.Errorf("failed hashing data: %s", err)
		return 0
	}

	logger.Debugf(
		"data 0x%x has hash 0x%x",
		data, hash)

	out, err := write(m, rtCtx.Allocator, hash)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		return 0
	}
	ptr, _ := splitPointerSize(out)
	return ptr
}

func ext_offchain_index_set_version_1(ctx context.Context, m api.Module, keySpan, valueSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, keySpan)
	newValue := read(m, valueSpan)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	err := rtCtx.NodeStorage.BaseDB.Put(storageKey, cp)
	if err != nil {
		logger.Errorf("failed to set value in raw storage: %s", err)
	}
}

//export ext_offchain_index_clear_version_1
func ext_offchain_index_clear_version_1(ctx context.Context, m api.Module, keySpan uint64) {
	// Remove a key and its associated value from the Offchain DB.
	// https://github.com/paritytech/substrate/blob/4d608f9c42e8d70d835a748fa929e59a99497e90/primitives/io/src/lib.rs#L1213

	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, keySpan)
	err := rtCtx.NodeStorage.BaseDB.Del(storageKey)
	if err != nil {
		logger.Errorf("failed to set value in raw storage: %s", err)
	}
}

func ext_offchain_local_storage_clear_version_1(ctx context.Context, m api.Module, kind uint32, key uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, key)

	kindBytes, ok := m.Memory().Read(kind, 4)
	if !ok {
		panic("read overflow")
	}
	kindInt := binary.LittleEndian.Uint32(kindBytes)

	var err error

	switch runtime.NodeStorageType(kindInt) {
	case runtime.NodeStorageTypePersistent:
		err = rtCtx.NodeStorage.PersistentStorage.Del(storageKey)
	case runtime.NodeStorageTypeLocal:
		err = rtCtx.NodeStorage.LocalStorage.Del(storageKey)
	}

	if err != nil {
		logger.Errorf("failed to clear value from storage: %s", err)
	}
}

func ext_offchain_is_validator_version_1(ctx context.Context, _ api.Module) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	if rtCtx.Validator {
		return 1
	}
	return 0
}

func ext_offchain_local_storage_compare_and_set_version_1(
	ctx context.Context, m api.Module, kind uint32, key, oldValue, newValue uint64) (newValueSet uint32) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, key)

	var storedValue []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		storedValue, err = rtCtx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		storedValue, err = rtCtx.NodeStorage.LocalStorage.Get(storageKey)
	}

	if err != nil {
		logger.Errorf("failed to get value from storage: %s", err)
		return 0
	}

	oldVal := read(m, oldValue)
	newVal := read(m, newValue)
	if reflect.DeepEqual(storedValue, oldVal) {
		cp := make([]byte, len(newVal))
		copy(cp, newVal)
		err = rtCtx.NodeStorage.LocalStorage.Put(storageKey, cp)
		if err != nil {
			logger.Errorf("failed to set value in storage: %s", err)
			return 0
		}
	}

	return 1
}

func ext_offchain_local_storage_get_version_1(ctx context.Context, m api.Module, kind uint32, key uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, key)

	var res []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		res, err = rtCtx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		res, err = rtCtx.NodeStorage.LocalStorage.Get(storageKey)
	}

	var encodedOption []byte
	if err != nil || res == nil {
		logger.Errorf("failed to get value from storage: %s", err)
		encodedOption = noneEncoded
	} else {
		encodedOption = res
	}

	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&encodedOption))
}

func ext_offchain_local_storage_set_version_1(ctx context.Context, m api.Module, kind uint32, key, value uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	storageKey := read(m, key)
	newValue := read(m, value)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	var err error
	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		err = rtCtx.NodeStorage.PersistentStorage.Put(storageKey, cp)
	case runtime.NodeStorageTypeLocal:
		err = rtCtx.NodeStorage.LocalStorage.Put(storageKey, cp)
	}

	if err != nil {
		logger.Errorf("failed to set value in storage: %s", err)
	}
}

func ext_offchain_network_state_version_1(ctx context.Context, m api.Module) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	if rtCtx.Network == nil {
		return 0
	}

	// expected to return Result<OpaqueNetworkState, ()

	nsEnc, err := scale.Marshal(rtCtx.Network.NetworkState())
	if err != nil {
		logger.Errorf("failed at encoding network state: %s", err)
		return 0
	}

	ret, err := write(m, rtCtx.Allocator, nsEnc)
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_offchain_random_seed_version_1(ctx context.Context, m api.Module) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		logger.Errorf("failed to generate random seed: %s", err)
	}

	ret, err := write(m, rtCtx.Allocator, seed)
	if err != nil {
		panic(err)
	}
	ptr, _ := splitPointerSize(ret)
	return ptr
}

func ext_offchain_submit_transaction_version_1(ctx context.Context, m api.Module, data uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	extBytes := read(m, data)

	var extrinsic []byte
	err := scale.Unmarshal(extBytes, &extrinsic)
	if err != nil {
		logger.Errorf("failed to decode extrinsic data: %s", err)
		// Error case
		ret, err := write(m, rtCtx.Allocator, []byte{1})
		if err != nil {
			panic(err)
		}
		return ret
	}

	// validate the transaction
	txv := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	vtx := transaction.NewValidTransaction(extrinsic, txv)

	rtCtx.Transaction.AddToPool(vtx)

	// OK case
	ret, err := write(m, rtCtx.Allocator, []byte{0})
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_offchain_timestamp_version_1(_ context.Context, _ api.Module) uint64 {
	now := time.Now().Unix()
	return uint64(now)
}

func ext_offchain_sleep_until_version_1(_ context.Context, _ api.Module, deadline uint64) {
	dur := time.Until(time.UnixMilli(int64(deadline)))
	if dur > 0 {
		time.Sleep(dur)
	}
}

func ext_offchain_http_request_start_version_1(
	ctx context.Context, m api.Module, methodSpan, uriSpan, metaSpan uint64) (pointerSize uint64) { //skipcq: RVV-B0012
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	httpMethod := read(m, methodSpan)
	uri := read(m, uriSpan)

	result := scale.NewResult(int16(0), nil)

	reqID, err := rtCtx.OffchainHTTPSet.StartRequest(string(httpMethod), string(uri))
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
		return uint64(0)
	}

	enc, err := scale.Marshal(result)
	if err != nil {
		logger.Errorf("failed to scale marshal the result: %s", err)
		return uint64(0)
	}

	ptr, err := write(m, rtCtx.Allocator, enc)
	if err != nil {
		logger.Errorf("failed to allocate result on memory: %s", err)
		return uint64(0)
	}

	return ptr
}

func ext_offchain_http_request_add_header_version_1(
	ctx context.Context, m api.Module, reqID uint32, nameSpan, valueSpan uint64) (pointerSize uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	name := read(m, nameSpan)
	value := read(m, valueSpan)

	offchainReq := rtCtx.OffchainHTTPSet.Get(int16(reqID))

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
		return uint64(0)
	}

	enc, err := scale.Marshal(result)
	if err != nil {
		logger.Errorf("failed to scale marshal the result: %s", err)
		return uint64(0)
	}

	ptr, err := write(m, rtCtx.Allocator, enc)
	if err != nil {
		logger.Errorf("failed to allocate result on memory: %s", err)
		return uint64(0)
	}

	return ptr
}

func storageAppend(storage runtime.Storage, key, valueToAppend []byte) (err error) {
	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	currentValue := storage.Get(key)

	var value []byte
	if len(currentValue) == 0 {
		nextLength := big.NewInt(1)
		encodedLength, err := scale.Marshal(nextLength)
		if err != nil {
			return fmt.Errorf("scale encoding: %w", err)
		}
		value = make([]byte, len(encodedLength)+len(valueToAppend))
		// append new length prefix to start of items array
		copy(value, encodedLength)
		copy(value[len(encodedLength):], valueToAppend)
	} else {
		var currentLength *big.Int
		err := scale.Unmarshal(currentValue, &currentLength)
		if err != nil {
			logger.Tracef(
				"item in storage is not SCALE encoded, overwriting at key 0x%x", key)
			value = make([]byte, 1+len(valueToAppend))
			value[0] = 4
			copy(value[1:], valueToAppend)
		} else {
			lengthBytes, err := scale.Marshal(currentLength)
			if err != nil {
				return fmt.Errorf("scale encoding: %w", err)
			}

			// increase length by 1
			nextLength := big.NewInt(0).Add(currentLength, big.NewInt(1))
			nextLengthBytes, err := scale.Marshal(nextLength)
			if err != nil {
				return fmt.Errorf("scale encoding next length bytes: %w", err)
			}

			// append new item, pop off number of bytes required for length encoding,
			// since we're not using old scale.Decoder
			value = make([]byte, len(nextLengthBytes)+len(currentValue)-len(lengthBytes)+len(valueToAppend))
			// append new length prefix to start of items array
			i := 0
			copy(value[i:], nextLengthBytes)
			i += len(nextLengthBytes)
			copy(value[i:], currentValue[len(lengthBytes):])
			i += len(currentValue) - len(lengthBytes)
			copy(value[i:], valueToAppend)
		}
	}

	err = storage.Put(key, value)
	if err != nil {
		return fmt.Errorf("putting key and value in storage: %w", err)
	}

	return nil
}

func ext_storage_append_version_1(ctx context.Context, m api.Module, keySpan, valueSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)
	valueAppend := read(m, valueSpan)
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

// Always returns `None`. This function exists for compatibility reasons.
func ext_storage_changes_root_version_1(
	ctx context.Context, m api.Module, parentHashSpan uint64) uint64 { //skipcq: RVV-B0012
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	var option *[]byte = nil

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(option))
	if err != nil {
		panic(err)
	}
	return ret
}

func ext_storage_clear_version_1(ctx context.Context, m api.Module, keySpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)

	logger.Debugf("key: 0x%x", key)
	err := storage.Delete(key)
	if err != nil {
		panic(err)
	}
}

func ext_storage_clear_prefix_version_1(ctx context.Context, m api.Module, prefixSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	prefix := read(m, prefixSpan)
	logger.Debugf("prefix: 0x%x", prefix)

	err := storage.ClearPrefix(prefix)
	if err != nil {
		panic(err)
	}
}

// toKillStorageResultEnum encodes the `allRemoved` flag and
// the `numRemoved` uint32 to a byte slice and returns it.
// The format used is:
// Byte 0: 1 if allRemoved is false, 0 otherwise
// Byte 1-5: scale encoding of numRemoved (up to 4 bytes)
func toKillStorageResultEnum(allRemoved bool, numRemoved uint32) (
	encodedEnumValue []byte, err error) {
	encodedNumRemoved, err := scale.Marshal(numRemoved)
	if err != nil {
		return nil, fmt.Errorf("scale encoding: %w", err)
	}

	encodedEnumValue = make([]byte, len(encodedNumRemoved)+1)
	if !allRemoved {
		// At least one key resides in the child trie due to the supplied limit.
		encodedEnumValue[0] = 1
	}
	copy(encodedEnumValue[1:], encodedNumRemoved)

	return encodedEnumValue, nil
}

func ext_storage_clear_prefix_version_2(ctx context.Context, m api.Module, prefixSpan, lim uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	prefix := read(m, prefixSpan)
	logger.Debugf("prefix: 0x%x", prefix)

	limitBytes := read(m, lim)

	var limitPtr *uint32
	err := scale.Unmarshal(limitBytes, &limitPtr)
	if err != nil {
		logger.Warnf("failed scale decoding limit: %s", err)
		panic(err)
	}

	if limitPtr == nil {
		maxLimit := uint32(math.MaxUint32)
		limitPtr = &maxLimit
	}

	numRemoved, all, err := storage.ClearPrefixLimit(prefix, *limitPtr)
	if err != nil {
		logger.Errorf("failed to clear prefix limit: %s", err)
		panic(err)
	}

	encBytes, err := toKillStorageResultEnum(all, numRemoved)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		panic(err)
	}

	valueSpan, err := write(m, rtCtx.Allocator, encBytes)
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		panic(err)
	}

	return valueSpan
}

func ext_storage_exists_version_1(ctx context.Context, m api.Module, keySpan uint64) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)
	logger.Debugf("key: 0x%x", key)

	value := storage.Get(key)
	if value != nil {
		return 1
	}

	return 0
}

func ext_storage_get_version_1(ctx context.Context, m api.Module, keySpan uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)
	logger.Debugf("key: 0x%x", key)

	value := storage.Get(key)
	logger.Debugf("value: 0x%x", value)

	var encodedOption []byte
	if value != nil {
		encodedOption = scale.MustMarshal(&value)
	} else {
		encodedOption = noneEncoded
	}

	return mustWrite(m, rtCtx.Allocator, encodedOption)
}

func ext_storage_next_key_version_1(ctx context.Context, m api.Module, keySpan uint64) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)

	next := storage.NextKey(key)
	logger.Debugf(
		"key: 0x%x; next key 0x%x",
		key, next)

	var encodedOption []byte
	if len(next) == 0 {
		encodedOption = noneEncoded
	} else {
		encodedOption = scale.MustMarshal(&next)
	}

	return mustWrite(m, rtCtx.Allocator, encodedOption)
}

func ext_storage_read_version_1(ctx context.Context, m api.Module, keySpan, valueOut uint64, offset uint32) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)
	value := storage.Get(key)
	logger.Debugf(
		"key 0x%x has value 0x%x",
		key, value)

	if value == nil {
		return mustWrite(m, rtCtx.Allocator, noneEncoded)
	}

	var data []byte
	switch {
	case offset <= uint32(len(value)):
		data = value[offset:]
	default:
		data = value[len(value):]
	}

	var written uint
	valueOutPtr, valueOutSize := splitPointerSize(valueOut)
	if uint32(len(data)) <= valueOutSize {
		written = uint(len(data))
	} else {
		written = uint(valueOutSize)
	}

	ok := m.Memory().Write(valueOutPtr, data[0:written])
	if !ok {
		panic("write overflow")
	}

	size := uint32(len(data))
	return mustWrite(m, rtCtx.Allocator, scale.MustMarshal(&size))
}

func ext_storage_root_version_1(ctx context.Context, m api.Module) uint64 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	root, err := storage.Root()
	if err != nil {
		logger.Errorf("failed to get storage root: %s", err)
		panic(err)
	}

	logger.Debugf("root hash is: %s", root)

	rootSpan, err := write(m, rtCtx.Allocator, root[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		panic(err)
	}
	return rootSpan
}

func ext_storage_root_version_2(ctx context.Context, m api.Module, _ uint32) uint64 { //skipcq: RVV-B0012
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	root, err := storage.Root()
	if err != nil {
		logger.Errorf("failed to get storage root: %s", err)
		panic(err)
	}

	rootSpan, err := write(m, rtCtx.Allocator, root[:])
	if err != nil {
		logger.Errorf("failed to allocate: %s", err)
		panic(err)
	}
	return rootSpan
}

func ext_storage_set_version_1(ctx context.Context, m api.Module, keySpan, valueSpan uint64) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	storage := rtCtx.Storage

	key := read(m, keySpan)
	value := read(m, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	logger.Debugf(
		"key 0x%x has value 0x%x",
		key, value)
	err := storage.Put(key, cp)
	if err != nil {
		panic(err)
	}
}

func ext_storage_start_transaction_version_1(ctx context.Context, _ api.Module) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	rtCtx.Storage.StartTransaction()
}

func ext_storage_rollback_transaction_version_1(ctx context.Context, _ api.Module) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	rtCtx.Storage.RollbackTransaction()
}

func ext_storage_commit_transaction_version_1(ctx context.Context, _ api.Module) {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}
	rtCtx.Storage.CommitTransaction()
}

func ext_allocator_free_version_1(ctx context.Context, m api.Module, addr uint32) {
	allocator := ctx.Value(runtimeContextKey).(*runtime.Context).Allocator

	// Deallocate memory
	err := allocator.Deallocate(m.Memory(), addr)
	if err != nil {
		panic(err)
	}
}

func ext_allocator_malloc_version_1(ctx context.Context, m api.Module, size uint32) uint32 {
	allocator := ctx.Value(runtimeContextKey).(*runtime.Context).Allocator

	// Allocate memory
	res, err := allocator.Allocate(m.Memory(), size)
	if err != nil {
		panic(err)
	}

	return res
}
