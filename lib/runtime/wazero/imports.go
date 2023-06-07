package wazero_runtime

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
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

// write copies a Go byte slice to wasm memory and returns the corresponding
// 32 bit pointer. Note the data must have a well known fixed length in the runtime.
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
