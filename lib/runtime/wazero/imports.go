package wazero_runtime

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tetratelabs/wazero/api"
)

var (
	logger = log.NewFromGlobal(
		log.AddContext("pkg", "runtime"),
		log.AddContext("module", "wazero"),
	)
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
		panic("wtf?")
	}
	return data
}

// copies a Go byte slice to wasm memory and returns the corresponding
// 64 bit pointer size.
func write(m api.Module, allocator *runtime.FreeingBumpHeapAllocator, data []byte) (pointerSize uint64, err error) {
	size := uint32(len(data))
	pointer, err := allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	ok := m.Memory().Write(pointer, data)
	if !ok {
		return 0, fmt.Errorf("out of range")
	}
	return newPointerSize(pointer, size), nil
}

//export ext_logging_log_version_1
func ext_logging_log_version_1(ctx context.Context, m api.Module, level int32, targetData, msgData uint64) {
	target := string(read(m, targetData))
	msg := string(read(m, msgData))

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

//export ext_crypto_ed25519_generate_version_1
func ext_crypto_ed25519_generate_version_1(ctx context.Context, m api.Module, keyTypeID uint32, seedSpan uint64) uint32 {
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
		return uint64(ret)
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warnf(
			"error for id 0x%x: keystore type is %s and not the expected ed25519",
			id, ks.Type())
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return uint64(ret)
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
		return uint64(ret)
	}

	ret, err := write(m, rtCtx.Allocator, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return uint64(ret)
	}
	return ret
}

//export ext_crypto_ed25519_sign_version_1
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
		return 0
	}

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return uint64(ret)
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return uint64(ret)
	}

	sig, err := signingKey.Sign(read(m, msg))
	if err != nil {
		logger.Error("could not sign message")
	}

	var fixedSize [64]byte
	copy(fixedSize[:], sig)

	encoded, err := scale.Marshal(&fixedSize)
	if err != nil {
		logger.Error(fmt.Sprintf("scale encoding: %s", err))
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return uint64(ret)
	}

	ret, err := write(m, rtCtx.Allocator, encoded)
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		return 0
	}

	return ret
}

func ext_crypto_ed25519_verify_version_1(ctx context.Context, m api.Module, sig uint32, msg uint64, key uint32) uint32 {
	rtCtx := ctx.Value(runtimeContextKey).(*runtime.Context)
	if rtCtx == nil {
		panic("nil runtime context")
	}

	memory := m.Memory()
	sigVerifier := rtCtx.SigVerifier

	// signature := memory[sig : sig+64]
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

//export ext_crypto_secp256k1_ecdsa_recover_version_1
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

// //export ext_crypto_secp256k1_ecdsa_recover_version_2
// func ext_crypto_secp256k1_ecdsa_recover_version_2(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
// 	logger.Trace("executing...")
// 	return ext_crypto_secp256k1_ecdsa_recover_version_1(context, sig, msg)
// }

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

func ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(ctx context.Context, m api.Module, sig, msg uint32) uint64 {
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

	res := scale.NewResult([]byte{}, nil)

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

	err = res.Set(scale.OK, cpub)
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

// //export ext_crypto_secp256k1_ecdsa_recover_compressed_version_2
// func ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(context unsafe.Pointer, sig, msg C.int32_t) C.int64_t {
// 	logger.Trace("executing...")
// 	return ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(context, sig, msg)
// }

//export ext_crypto_sr25519_generate_version_1
func ext_crypto_sr25519_generate_version_1(ctx context.Context, m api.Module, keyTypeID uint32, seedSpan uint64) uint32 {
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

	ret, err := write(m, rtCtx.Allocator, append(prefix, encodedKeys...))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, []byte{0})
		if err != nil {
			panic(err)
		}
		return ret
	}

	return ret
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

	var optionSig *[64]byte

	ks, err := rtCtx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warnf("error for id 0x%x: %s", id, err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
		if err != nil {
			panic(err)
		}
		return ret
	}

	kb, ok := m.Memory().Read(key, 32)
	if !ok {
		panic("read overflow")
	}

	pubKey, err := sr25519.NewPublicKey(kb)
	if err != nil {
		logger.Errorf("failed to get public key: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
		if err != nil {
			panic(err)
		}
		return ret
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("could not find public key " + pubKey.Hex() + " in keystore")
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
		if err != nil {
			panic(err)
		}
		return ret
	}

	msgData := read(m, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Errorf("could not sign message: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
		if err != nil {
			panic(err)
		}
		return ret
	}

	var fixedSig [64]byte
	copy(fixedSig[:], sig)
	optionSig = &fixedSig

	ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
	if err != nil {
		logger.Errorf("failed to allocate memory: %s", err)
		ret, err := write(m, rtCtx.Allocator, scale.MustMarshal(optionSig))
		if err != nil {
			panic(err)
		}
		return ret
	}
	return ret
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

// //export ext_crypto_start_batch_verify_version_1
// func ext_crypto_start_batch_verify_version_1(context unsafe.Pointer) {
// 	logger.Debug("executing...")

// 	// TODO: fix and re-enable signature verification (#1405)
// 	// beginBatchVerify(context)
// }

// //export ext_crypto_finish_batch_verify_version_1
// func ext_crypto_finish_batch_verify_version_1(context unsafe.Pointer) C.int32_t {
// 	logger.Debug("executing...")

// 	// TODO: fix and re-enable signature verification (#1405)
// 	// return finishBatchVerify(context)
// 	return 1
// }

func ext_allocator_free_version_1(ctx context.Context, m api.Module, addr uint32) {
	allocator := ctx.Value(runtimeContextKey).(*runtime.Context).Allocator

	// Deallocate memory
	err := allocator.Deallocate(addr)
	if err != nil {
		panic(err)
	}
}

func ext_allocator_malloc_version_1(ctx context.Context, m api.Module, size uint32) uint32 {
	allocator := ctx.Value(runtimeContextKey).(*runtime.Context).Allocator

	// Allocate memory
	res, err := allocator.Allocate(size)
	if err != nil {
		panic(err)
	}

	return res
}
