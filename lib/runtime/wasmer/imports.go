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

import "C"
import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	rtype "github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	wasm "github.com/wasmerio/wasmer-go/wasmer"
)

func ext_logging_log_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_logging_log_version_1] executing...")
	ctx := env.(*runtime.Context)

	level, ok := args[0].Unwrap().(int32)
	if !ok {
		logger.Crit("[ext_logging_log_version_1]", "error", "addr cannot be converted to int32")
	}

	targetData := args[1].I64()
	msgData := args[2].I64()

	target := string(asMemorySlice(ctx, targetData))
	msg := string(asMemorySlice(ctx, msgData))

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

	return nil, nil
}

func ext_offchain_timestamp_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_offchain_timestamp_version_1] executing...")
	logger.Warn("[ext_offchain_timestamp_version_1] unimplemented")
	return nil, nil
}

func ext_offchain_sleep_until_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_offchain_sleep_until_version_1] executing...")
	logger.Warn("[ext_offchain_sleep_until_version_1] unimplemented")
	return nil, nil
}

func ext_default_child_storage_storage_kill_version_2(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_default_child_storage_storage_kill_version_2] executing...")
	logger.Warn("[ext_default_child_storage_storage_kill_version_2] unimplemented")
	return nil, nil
}

func ext_default_child_storage_storage_kill_version_3(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_default_child_storage_storage_kill_version_3] executing...")
	logger.Warn("[ext_default_child_storage_storage_kill_version_3] unimplemented")
	return nil, nil
}

func ext_sandbox_instance_teardown_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_instance_teardown_version_1] executing...")
	logger.Warn("[ext_sandbox_instance_teardown_version_1] unimplemented")
	return nil, nil
}

func ext_sandbox_instantiate_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_instantiate_version_1] executing...")
	logger.Warn("[ext_sandbox_instantiate_version_1] unimplemented")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_sandbox_invoke_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_invoke_version_1] executing...")
	logger.Warn("[ext_sandbox_invoke_version_1] unimplemented")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_sandbox_memory_get_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_memory_get_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_get_version_1] unimplemented")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_sandbox_memory_new_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_memory_new_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_new_version_1] unimplemented")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_sandbox_memory_set_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_memory_set_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_set_version_1] unimplemented")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_sandbox_memory_teardown_version_1(_ interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_sandbox_memory_teardown_version_1] executing...")
	logger.Warn("[ext_sandbox_memory_teardown_version_1] unimplemented")
	return nil, nil
}

func ext_crypto_ed25519_generate_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_crypto_ed25519_generate_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()

	keyTypeID := args[0].I32()
	seedSpan := args[1].I64()

	id := memory[keyTypeID : keyTypeID+4]
	seedBytes := asMemorySlice(ctx, seedSpan)
	buf := &bytes.Buffer{}
	buf.Write(seedBytes)

	seed, err := optional.NewBytes(false, nil).Decode(buf)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] cannot generate key", "error", err)
		return nil, err
	}

	var kp crypto.Keypair

	if seed.Exists() {
		kp, err = ed25519.NewKeypairFromMnenomic(string(seed.Value()), "")
	} else {
		kp, err = ed25519.GenerateKeypair()
	}

	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] cannot generate key", "error", err)
		return nil, err
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1]", "name", id, "error", err)
		return nil, err
	}

	ks.Insert(kp)

	ret, err := toWasmMemorySized(ctx, kp.Public().Encode(), 32)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_generate_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	logger.Debug("[ext_crypto_ed25519_generate_version_1] generated ed25519 keypair", "public", kp.Public().Hex())
	return []wasm.Value{wasm.NewI32(int32(ret))}, nil
}

func ext_crypto_ed25519_public_keys_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_ed25519_public_keys_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()

	keyTypeID := args[0].I32()
	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_public_keys_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemory(ctx, []byte{0})
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	if ks.Type() != crypto.Ed25519Type && ks.Type() != crypto.UnknownType {
		logger.Warn("[ext_crypto_ed25519_public_keys_version_1]", "name", id, "error", "keystore type is not ed25519", "type", ks.Type())
		ret, _ := toWasmMemory(ctx, []byte{0})
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Encode(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory", err)
		return nil, err
	}

	ret, err := toWasmMemory(ctx, append(prefix, encodedKeys...))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_public_keys_version_1] failed to allocate memory", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_ed25519_sign_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_ed25519_sign_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()

	keyTypeID := args[0].I32()
	key := args[1].I32()
	msg := args[2].I64()

	id := memory[keyTypeID : keyTypeID+4]

	pubKeyData := memory[key : key+32]
	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] failed to get public keys", "error", err)
		ret, _ := toWasmMemoryOptional(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_ed25519_sign_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemoryOptional(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not find public key in keystore", "error", pubKey)
		ret, _ := toWasmMemoryOptional(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	sig, err := signingKey.Sign(asMemorySlice(ctx, msg))
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] could not sign message")
		return nil, err
	}

	ret, err := toWasmMemoryFixedSizeOptional(ctx, sig)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_sign_version_1] failed to allocate memory", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_ed25519_verify_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_ed25519_verify_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	sigVerifier := ctx.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	signature := memory[sig : sig+64]
	message := asMemorySlice(ctx, msg)
	pubKeyData := memory[key : key+32]

	pubKey, err := ed25519.NewPublicKey(pubKeyData)
	if err != nil {
		logger.Error("[ext_crypto_ed25519_verify_version_1] failed to create public key")
		return nil, err
	}

	if sigVerifier.IsStarted() {
		signature := runtime.Signature{
			PubKey:    pubKey.Encode(),
			Sign:      signature,
			Msg:       message,
			KeyTypeID: crypto.Ed25519Type,
		}
		sigVerifier.Add(&signature)
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	if ok, err := pubKey.Verify(message, signature); err != nil || !ok {
		logger.Debug("[ext_crypto_ed25519_verify_version_1] failed to verify")
		return []wasm.Value{wasm.NewI32(0)}, nil
	}

	logger.Debug("[ext_crypto_ed25519_verify_version_1] verified ed25519 signature")
	return []wasm.Value{wasm.NewI32(1)}, nil
}

func ext_crypto_secp256k1_ecdsa_recover_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_version_1] executing...")
	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()

	sig := args[0].I32()
	msg := args[1].I32()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	if signature[64] == 27 {
		signature[64] = 0
	}

	if signature[64] == 28 {
		signature[64] = 1
	}

	pub, err := secp256k1.RecoverPublicKey(message, signature)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to recover public key", "error", err)
		ret, _ := toWasmMemoryResult(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	logger.Debug("[ext_crypto_secp256k1_ecdsa_recover_version_1]", "len", len(pub), "recovered public key", fmt.Sprintf("0x%x", pub))

	ret, err := toWasmMemoryResult(ctx, pub[1:])
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] executing...")
	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()

	sig := args[0].I32()
	msg := args[1].I32()

	// msg must be the 32-byte hash of the message to be signed.
	// sig must be a 65-byte compact ECDSA signature containing the
	// recovery id as the last element
	message := memory[msg : msg+32]
	signature := memory[sig : sig+65]

	if signature[64] == 27 {
		signature[64] = 0
	}

	if signature[64] == 28 {
		signature[64] = 1
	}

	cpub, err := secp256k1.RecoverPublicKeyCompressed(message, signature)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] failed to recover public key", "error", err)
		ret, _ := toWasmMemoryResult(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	logger.Debug("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1]", "len", len(cpub), "recovered public key", fmt.Sprintf("0x%x", cpub))

	ret, err := toWasmMemoryResult(ctx, cpub)
	if err != nil {
		logger.Error("[ext_crypto_secp256k1_ecdsa_recover_compressed_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_sr25519_generate_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_crypto_sr25519_generate_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	keyTypeID := args[0].I32()
	seedSpan := args[1].I64()

	id := memory[keyTypeID : keyTypeID+4]

	seedBytes := asMemorySlice(ctx, seedSpan)
	buf := &bytes.Buffer{}
	buf.Write(seedBytes)

	seed, err := optional.NewBytes(false, nil).Decode(buf)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_generate_version_1] cannot generate key", "error", err)
		return nil, err
	}

	var kp crypto.Keypair
	if seed.Exists() {
		kp, err = sr25519.NewKeypairFromMnenomic(string(seed.Value()), "")
	} else {
		kp, err = sr25519.GenerateKeypair()
	}

	if err != nil {
		logger.Trace("[ext_crypto_sr25519_generate_version_1] cannot generate key", "error", err)
		return nil, err
	}

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_generate_version_1]", "name", id, "error", err)
		return nil, err
	}

	ks.Insert(kp)
	ret, err := toWasmMemorySized(ctx, kp.Public().Encode(), 32)
	if err != nil {
		logger.Error("[ext_crypto_sr25519_generate_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	logger.Debug("[ext_crypto_sr25519_generate_version_1] generated sr25519 keypair", "public", kp.Public().Hex())
	return []wasm.Value{wasm.NewI32(int32(ret))}, nil
}

func ext_crypto_sr25519_public_keys_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_sr25519_public_keys_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	keyTypeID, ok := args[0].Unwrap().(int32)
	if !ok {
		panic("keyTypeID is not int32")
	}

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_public_keys_version_1]", "name", id, "error", err)
		ret, _ := toWasmMemory(ctx, []byte{0})
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	if ks.Type() != crypto.Sr25519Type && ks.Type() != crypto.UnknownType {
		logger.Warn("[ext_crypto_sr25519_public_keys_version_1]", "name", id, "error", "keystore type is not sr25519")
		ret, _ := toWasmMemory(ctx, []byte{0})
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	keys := ks.PublicKeys()

	var encodedKeys []byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, key.Encode()...)
	}

	prefix, err := scale.Encode(big.NewInt(int64(len(keys))))
	if err != nil {
		logger.Error("[ext_crypto_sr25519_public_keys_version_1] failed to encode keys", err)
		ret, _ := toWasmMemory(ctx, []byte{0})
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	ret, err := toWasmMemory(ctx, append(prefix, encodedKeys...))
	if err != nil {
		logger.Error("[ext_crypto_sr25519_public_keys_version_1] failed to allocate memory", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_sr25519_sign_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_sr25519_sign_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	keyTypeID := args[0].I32()
	key := args[1].I32()
	msg := args[2].I64()

	emptyRet, _ := toWasmMemoryOptional(ctx, nil)

	id := memory[keyTypeID : keyTypeID+4]

	ks, err := ctx.Keystore.GetKeystore(id)
	if err != nil {
		logger.Warn("[ext_crypto_sr25519_sign_version_1]", "name", id, "error", err)
		return []wasm.Value{wasm.NewI64(emptyRet)}, nil
	}

	var ret int64
	pubKey, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] failed to get public key", "error", err)
		return []wasm.Value{wasm.NewI64(emptyRet)}, nil
	}

	signingKey := ks.GetKeypair(pubKey)
	if signingKey == nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] could not find public key in keystore", "error", pubKey)
		return []wasm.Value{wasm.NewI64(emptyRet)}, nil
	}

	msgData := asMemorySlice(ctx, msg)
	sig, err := signingKey.Sign(msgData)
	if err != nil {
		logger.Error("[ext_crypto_sr25519_sign_version_1] could not sign message", "error", err)
		return []wasm.Value{wasm.NewI64(emptyRet)}, nil
	}

	ret, err = toWasmMemoryFixedSizeOptional(ctx, sig)
	if err != nil {
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ret)}, nil
}

func ext_crypto_sr25519_verify_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_sr25519_verify_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	sigVerifier := ctx.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	message := asMemorySlice(ctx, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_verify_version_1] invalid sr25519 public key")
		return nil, err
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
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	if ok, err := pub.VerifyDeprecated(message, signature); err != nil || !ok {
		logger.Debug("[ext_crypto_sr25519_verify_version_1] failed to validate signature", "error", err)
		// TODO: fix this, fails at block 3876
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_1] verified sr25519 signature")
	return []wasm.Value{wasm.NewI32(1)}, nil
}

func ext_crypto_sr25519_verify_version_2(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_crypto_sr25519_verify_version_2] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	sigVerifier := ctx.SigVerifier

	sig := args[0].I32()
	msg := args[1].I64()
	key := args[2].I32()

	message := asMemorySlice(ctx, msg)
	signature := memory[sig : sig+64]

	pub, err := sr25519.NewPublicKey(memory[key : key+32])
	if err != nil {
		logger.Error("[ext_crypto_sr25519_verify_version_2] invalid sr25519 public key")
		return nil, err
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
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	if ok, err := pub.Verify(message, signature); err != nil || !ok {
		logger.Debug("[ext_crypto_sr25519_verify_version_2] failed to validate signature")
		return []wasm.Value{wasm.NewI32(0)}, nil
	}

	logger.Debug("[ext_crypto_sr25519_verify_version_2] validated signature")
	return []wasm.Value{wasm.NewI32(1)}, nil
}

func ext_crypto_start_batch_verify_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_start_batch_verify_version_1] executing...")
	// TODO: fix and re-enable signature verification
	// return beginBatchVerify(context)
	return nil, nil
}

func beginBatchVerify(env interface{}) ([]wasm.Value, error) { //nolint
	ctx := env.(*runtime.Context)
	sigVerifier := ctx.SigVerifier

	if sigVerifier.IsStarted() {
		logger.Error("[ext_crypto_start_batch_verify_version_1] previous batch verification is not finished")
		return nil, nil
	}

	sigVerifier.Start()
	return nil, nil
}

func ext_crypto_finish_batch_verify_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_crypto_finish_batch_verify_version_1] executing...")

	// TODO: fix and re-enable signature verification
	// return finishBatchVerify(context)
	return []wasm.Value{wasm.NewI32(1)}, nil
}

func finishBatchVerify(env interface{}) ([]wasm.Value, error) { //nolint
	ctx := env.(*runtime.Context)
	sigVerifier := ctx.SigVerifier

	if !sigVerifier.IsStarted() {
		logger.Error("[ext_crypto_finish_batch_verify_version_1] batch verification is not started", "error")
		return nil, errors.New("batch verification is not started")
	}

	if sigVerifier.Finish() {
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	logger.Error("[ext_crypto_finish_batch_verify_version_1] failed to batch verify; invalid signature")
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_trie_blake2_256_root_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_trie_blake2_256_root_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	dataSpan := args[0].I64()

	data := asMemorySlice(ctx, dataSpan)

	t := trie.NewEmptyTrie()
	// TODO: this is a fix for the length until slices of structs can be decoded
	// length passed in is the # of (key, value) tuples, but we are decoding as a slice of []byte
	data[0] = data[0] << 1

	// this function is expecting an array of (key, value) tuples
	kvs, err := scale.Decode(data, [][]byte{})
	if err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return nil, err
	}

	keyValues := kvs.([][]byte)
	if len(keyValues)%2 != 0 { // TODO: this can be removed when we have decoding of slices of structs
		logger.Warn("[ext_trie_blake2_256_root_version_1] odd number of input key-values, skipping last value")
		keyValues = keyValues[:len(keyValues)-1]
	}

	for i := 0; i < len(keyValues); i = i + 2 {
		t.Put(keyValues[i], keyValues[i+1])
	}

	// allocate memory for value and copy value to memory
	ptr, err := ctx.Allocator.Allocate(32)
	if err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return nil, err
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Error("[ext_trie_blake2_256_root_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_trie_blake2_256_root_version_1]", "root", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return []wasm.Value{wasm.NewI32(int32(ptr))}, nil
}

func ext_trie_blake2_256_ordered_root_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_trie_blake2_256_ordered_root_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	dataSpan := args[0].I64()

	data := asMemorySlice(ctx, dataSpan)

	t := trie.NewEmptyTrie()
	v, err := scale.Decode(data, [][]byte{})
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return nil, err
	}

	values := v.([][]byte)

	for i, val := range values {
		key, err := scale.Encode(big.NewInt(int64(i))) //nolint
		if err != nil {
			logger.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
			return nil, err
		}
		logger.Trace("[ext_trie_blake2_256_ordered_root_version_1]", "key", key, "value", val)

		t.Put(key, val)
	}

	// allocate memory for value and copy value to memory
	ptr, err := ctx.Allocator.Allocate(32)
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return nil, err
	}

	hash, err := t.Hash()
	if err != nil {
		logger.Error("[ext_trie_blake2_256_ordered_root_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_trie_blake2_256_ordered_root_version_1]", "root", hash)
	copy(memory[ptr:ptr+32], hash[:])
	return []wasm.Value{wasm.NewI32(int32(ptr))}, nil
}

func ext_misc_print_hex_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_misc_print_hex_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()

	data := asMemorySlice(ctx, dataSpan)
	logger.Debug("[ext_misc_print_hex_version_1]", "hex", fmt.Sprintf("0x%x", data))
	return nil, nil
}

func ext_misc_print_num_version_1(_ interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_misc_print_num_version_1] executing...")
	data := args[0].I64()
	logger.Debug("[ext_misc_print_num_version_1]", "num", fmt.Sprintf("%d", data))
	return nil, nil
}

//func ext_misc_print_utf8_version_1(context unsafe.Pointer, dataSpan C.int64_t) {
func ext_misc_print_utf8_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_misc_print_utf8_version_1] executing...")
	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()

	data := asMemorySlice(ctx, dataSpan)
	logger.Debug("[ext_misc_print_utf8_version_1]", "utf8", string(data))
	return nil, nil
}

func ext_misc_runtime_version_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_misc_runtime_version_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)

	cfg := &Config{
		Imports: ImportsNodeRuntime,
	}
	cfg.LogLvl = -1 // don't change log level
	cfg.Storage, _ = rtstorage.NewTrieState(nil)

	instance, err := NewInstance(data, cfg)
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to create instance", "error", err)
		return nil, err
	}

	// instance version is set and cached in NewInstance
	version := instance.version
	logger.Debug("[ext_misc_runtime_version_version_1]", "version", version)

	if version == nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to get runtime version")
		return nil, err
	}

	encodedData, err := version.Encode()
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to encode result", "error", err)
		return nil, err
	}

	out, err := toWasmMemoryOptional(ctx, encodedData)
	if err != nil {
		logger.Error("[ext_misc_runtime_version_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(out)}, nil
}

func ext_default_child_storage_read_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_read_version_1] executing...")

	ctx := env.(*runtime.Context)
	memory := ctx.Memory.Data()
	storage := ctx.Storage

	childStorageKey := args[0].I64()
	key := args[1].I64()
	valueOut := args[2].I64()
	offset := args[3].I32()

	value, err := storage.GetChildStorage(asMemorySlice(ctx, childStorageKey), asMemorySlice(ctx, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_read_version_1] failed to get child storage", "error", err)
		return nil, err
	}

	valueBuf, valueLen := int64ToPointerAndSize(valueOut)
	copy(memory[valueBuf:valueBuf+valueLen], value[offset:])

	size := uint32(len(value[offset:]))
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, size)

	sizeSpan, err := toWasmMemoryOptional(ctx, sizeBuf)
	if err != nil {
		logger.Error("[ext_default_child_storage_read_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(sizeSpan)}, nil
}

func ext_default_child_storage_clear_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_clear_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	childStorageKey := args[0].I64()
	keySpan := args[1].I64()

	keyToChild := asMemorySlice(ctx, childStorageKey)
	key := asMemorySlice(ctx, keySpan)

	err := storage.ClearChildStorage(keyToChild, key)
	if err != nil {
		logger.Error("[ext_default_child_storage_clear_version_1] failed to clear child storage", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_default_child_storage_clear_prefix_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_clear_prefix_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := args[0].I64()
	prefixSpan := args[1].I64()

	keyToChild := asMemorySlice(ctx, childStorageKey)
	prefix := asMemorySlice(ctx, prefixSpan)

	err := storage.ClearPrefixInChild(keyToChild, prefix)
	if err != nil {
		logger.Error("[ext_default_child_storage_clear_prefix_version_1] failed to clear prefix in child", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_default_child_storage_exists_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_exists_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := args[0].I64()
	key := args[1].I64()

	child, err := storage.GetChildStorage(asMemorySlice(ctx, childStorageKey), asMemorySlice(ctx, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_exists_version_1] failed to get child from child storage", "error", err)
		return []wasm.Value{wasm.NewI32(0)}, nil
	}

	if child != nil {
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_default_child_storage_get_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_get_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := args[0].I64()
	key := args[1].I64()

	child, err := storage.GetChildStorage(asMemorySlice(ctx, childStorageKey), asMemorySlice(ctx, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_get_version_1] failed to get child from child storage", "error", err)
		return nil, err
	}

	value, err := toWasmMemoryOptional(ctx, child)
	if err != nil {
		logger.Error("[ext_default_child_storage_get_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(value)}, nil
}

func ext_default_child_storage_next_key_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_next_key_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKey, ok := args[0].Unwrap().(int64)
	if !ok {
		panic("childStorageKey is not int64")
	}
	key, ok := args[1].Unwrap().(int64)
	if !ok {
		panic("key is not int64")
	}

	child, err := storage.GetChildNextKey(asMemorySlice(ctx, childStorageKey), asMemorySlice(ctx, key))
	if err != nil {
		logger.Error("[ext_default_child_storage_next_key_version_1] failed to get child's next key", "error", err)
		return nil, err
	}

	value, err := toWasmMemoryOptional(ctx, child)
	if err != nil {
		logger.Error("[ext_default_child_storage_next_key_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(value)}, nil
}

func ext_default_child_storage_root_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_root_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKey := args[0].I64()

	child, err := storage.GetChild(asMemorySlice(ctx, childStorageKey))
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to retrieve child", "error", err)
		return nil, err
	}

	childRoot, err := child.Hash()
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to encode child root", "error", err)
		return nil, err
	}

	root, err := toWasmMemoryOptional(ctx, childRoot[:])
	if err != nil {
		logger.Error("[ext_default_child_storage_root_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(root)}, nil
}

func ext_default_child_storage_set_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_set_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	childStorageKeySpan := args[0].I64()
	keySpan := args[1].I64()
	valueSpan := args[2].I64()

	childStorageKey := asMemorySlice(ctx, childStorageKeySpan)
	key := asMemorySlice(ctx, keySpan)
	value := asMemorySlice(ctx, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	err := storage.SetChildStorage(childStorageKey, key, cp)
	if err != nil {
		logger.Error("[ext_default_child_storage_set_version_1] failed to set value in child storage", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_default_child_storage_storage_kill_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_default_child_storage_storage_kill_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	childStorageKeySpan := args[0].I64()

	childStorageKey := asMemorySlice(ctx, childStorageKeySpan)
	storage.DeleteChild(childStorageKey)
	return nil, nil
}

func ext_allocator_free_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_allocator_free_version_1] executing...")

	ctx := env.(*runtime.Context)
	addr, ok := args[0].Unwrap().(int32)
	if !ok {
		logger.Crit("[ext_allocator_free_version_1]", "error", "addr cannot be converted to int32")
	}

	err := ctx.Allocator.Deallocate(uint32(addr))
	if err != nil {
		logger.Error("[ext_allocator_free_version_1] failed to free memory", "error", err)
	}

	return nil, nil
}

func ext_allocator_malloc_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_allocator_malloc_version_1] executing...")
	size, ok := args[0].Unwrap().(int32)
	if !ok {
		logger.Crit("[ext_allocator_malloc_version_1]", "error", "addr cannot be converted to int32")
	}

	ctx := env.(*runtime.Context)

	res, err := ctx.Allocator.Allocate(uint32(size))
	if err != nil {
		logger.Crit("[ext_allocator_malloc_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	logger.Trace("[ext_allocator_malloc_version_1]", "size", size, "ptr", res)
	return []wasm.Value{wasm.NewI32(int32(res))}, nil
}

func ext_hashing_blake2_128_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_blake2_128_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	ptr, size := int64ToPointerAndSize(dataSpan)
	logger.Info("[ext_hashing_blake2_128_version_1]", "ptr", ptr, "datalen", size)

	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Blake2b128(data)
	if err != nil {
		logger.Error("[ext_hashing_blake2_128_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_blake2_128_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(ctx, hash, 16)
	if err != nil {
		logger.Error("[ext_hashing_blake2_128_version_1] failed to allocate", "error", err)
		return nil, err
	}

	logger.Info("[ext_hashing_blake2_128_version_1]", "ret", out)
	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_blake2_256_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_blake2_256_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()

	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Blake2bHash(data)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_blake2_256_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", hash)

	out, err := toWasmMemorySized(ctx, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_blake2_256_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_keccak_256_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_keccak_256_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Keccak256(data)
	if err != nil {
		logger.Error("[ext_hashing_keccak_256_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_keccak_256_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", hash)

	out, err := toWasmMemorySized(ctx, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_keccak_256_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_sha2_256_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_sha2_256_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)
	hash := common.Sha256(data)

	logger.Debug("[ext_hashing_sha2_256_version_1]", "data", data, "hash", hash)

	out, err := toWasmMemorySized(ctx, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_sha2_256_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_twox_256_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_twox_256_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Twox256(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_256_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_twox_256_version_1]", "data", data, "hash", hash)

	out, err := toWasmMemorySized(ctx, hash[:], 32)
	if err != nil {
		logger.Error("[ext_hashing_twox_256_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_twox_128_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_twox_128_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Twox128Hash(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_twox_128_version_1]", "data", string(data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(ctx, hash, 16)
	if err != nil {
		logger.Error("[ext_hashing_twox_128_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_hashing_twox_64_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_hashing_twox_64_version_1] executing...")

	ctx := env.(*runtime.Context)
	dataSpan := args[0].I64()
	data := asMemorySlice(ctx, dataSpan)

	hash, err := common.Twox64(data)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1]", "error", err)
		return nil, err
	}

	logger.Debug("[ext_hashing_twox_64_version_1]", "data", fmt.Sprintf("0x%x", data), "hash", fmt.Sprintf("0x%x", hash))

	out, err := toWasmMemorySized(ctx, hash, 8)
	if err != nil {
		logger.Error("[ext_hashing_twox_64_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(int32(out))}, nil
}

func ext_offchain_index_set_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_offchain_index_set_version_1] executing...")
	logger.Warn("[ext_offchain_index_set_version_1] unimplemented")
	return nil, nil
}

func ext_offchain_is_validator_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_is_validator_version_1] executing...")
	ctx := env.(*runtime.Context)
	if ctx.Validator {
		return []wasm.Value{wasm.NewI32(1)}, nil
	}
	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_offchain_local_storage_compare_and_set_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_local_storage_compare_and_set_version_1] executing...")

	ctx := env.(*runtime.Context)
	kind := args[0].I32()
	key := args[1].I64()
	oldValue := args[2].I64()
	newValue := args[3].I64()

	storageKey := asMemorySlice(ctx, key)

	var storedValue []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		storedValue, err = ctx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		storedValue, err = ctx.NodeStorage.LocalStorage.Get(storageKey)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_compare_and_set_version_1] failed to get value from storage", "error", err)
		return nil, err
	}

	oldVal := asMemorySlice(ctx, oldValue)
	newVal := asMemorySlice(ctx, newValue)
	if reflect.DeepEqual(storedValue, oldVal) {
		cp := make([]byte, len(newVal))
		copy(cp, newVal)
		err = ctx.NodeStorage.LocalStorage.Put(storageKey, cp)
		if err != nil {
			logger.Error("[ext_offchain_local_storage_compare_and_set_version_1] failed to set value in storage", "error", err)
			return []wasm.Value{wasm.NewI32(0)}, nil
		}
	}

	return []wasm.Value{wasm.NewI32(1)}, nil
}

func ext_offchain_local_storage_get_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_local_storage_get_version_1] executing...")

	ctx := env.(*runtime.Context)
	kind := args[0].I32()
	key := args[1].I64()

	storageKey := asMemorySlice(ctx, key)

	var res []byte
	var err error

	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		res, err = ctx.NodeStorage.PersistentStorage.Get(storageKey)
	case runtime.NodeStorageTypeLocal:
		res, err = ctx.NodeStorage.LocalStorage.Get(storageKey)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_get_version_1] failed to get value from storage", "error", err)
		return nil, err
	}
	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemoryOptional(ctx, res)
	if err != nil {
		logger.Error("[ext_offchain_local_storage_get_version_1] failed to allocate memory", "error", err)
		return []wasm.Value{wasm.NewI64(0)}, nil
	}
	return []wasm.Value{wasm.NewI64(ptr)}, nil
}

func ext_offchain_local_storage_set_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_local_storage_set_version_1] executing...")

	ctx := env.(*runtime.Context)
	kind := args[0].I32()
	key := args[1].I64()
	value := args[2].I64()

	storageKey := asMemorySlice(ctx, key)
	newValue := asMemorySlice(ctx, value)
	cp := make([]byte, len(newValue))
	copy(cp, newValue)

	var err error
	switch runtime.NodeStorageType(kind) {
	case runtime.NodeStorageTypePersistent:
		err = ctx.NodeStorage.PersistentStorage.Put(storageKey, cp)
	case runtime.NodeStorageTypeLocal:
		err = ctx.NodeStorage.LocalStorage.Put(storageKey, cp)
	}

	if err != nil {
		logger.Error("[ext_offchain_local_storage_set_version_1] failed to set value in storage", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_offchain_network_state_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_network_state_version_1] executing...")
	ctx := env.(*runtime.Context)

	if ctx.Network == nil {
		return []wasm.Value{wasm.NewI64(0)}, nil
	}

	nsEnc, err := scale.Encode(ctx.Network.NetworkState())
	if err != nil {
		logger.Error("[ext_offchain_network_state_version_1] failed at encoding network state", "error", err)
		return nil, err
	}

	// copy network state length to memory writtenOut location
	nsEncLen := uint32(len(nsEnc))
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, nsEncLen)

	// allocate memory for value and copy value to memory
	ptr, err := toWasmMemorySized(ctx, nsEnc, nsEncLen)
	if err != nil {
		logger.Error("[ext_offchain_network_state_version_1] failed to allocate memory", "error", err)
		return []wasm.Value{wasm.NewI64(0)}, nil
	}

	return []wasm.Value{wasm.NewI64(ptr)}, nil
}

func ext_offchain_random_seed_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_random_seed_version_1] executing...")
	ctx := env.(*runtime.Context)

	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		logger.Error("[ext_offchain_random_seed_version_1] failed to generate random seed", "error", err)
		return nil, err
	}

	ptr, err := toWasmMemorySized(ctx, seed, 32)
	if err != nil {
		logger.Error("[ext_offchain_random_seed_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI32(ptr)}, nil
}

func ext_offchain_submit_transaction_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_offchain_submit_transaction_version_1] executing...")

	ctx := env.(*runtime.Context)
	data := args[0].I64()
	extBytes := asMemorySlice(ctx, data)

	var decExt interface{}
	decExt, err := scale.Decode(extBytes, decExt)
	if err != nil {
		logger.Error("[ext_offchain_submit_transaction_version_1] failed to decode extrinsic data", "error", err)
		return nil, err
	}

	extrinsic := types.Extrinsic(decExt.([]byte))

	// validate the transaction
	txv := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	vtx := transaction.NewValidTransaction(extrinsic, txv)

	ctx.Transaction.AddToPool(vtx)

	ptr, err := toWasmMemoryOptional(ctx, nil)
	if err != nil {
		logger.Error("[ext_offchain_submit_transaction_version_1] failed to allocate memory", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(ptr)}, nil
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
		// remove length prefix from existing value
		r := &bytes.Buffer{}
		_, _ = r.Write(valueCurr)
		dec := &scale.Decoder{Reader: r}
		currLength, err := dec.DecodeBigInt() //nolint
		if err != nil {
			logger.Trace("[ext_storage_append_version_1] item in storage is not SCALE encoded, overwriting", "key", key)
			storage.Set(key, append([]byte{4}, valueToAppend...))
			return nil
		}

		// append new item
		valueRes = append(r.Bytes(), valueToAppend...)

		// increase length by 1
		nextLength = big.NewInt(0).Add(currLength, big.NewInt(1))
	}

	lengthEnc, err := scale.Encode(nextLength)
	if err != nil {
		logger.Trace("[ext_storage_append_version_1] failed to encode new length", "error", err)
		return err
	}

	// append new length prefix to start of items array
	finalVal := append(lengthEnc, valueRes...)
	logger.Debug("[ext_storage_append_version_1]", "resulting value", fmt.Sprintf("0x%x", finalVal))
	storage.Set(key, finalVal)
	return nil
}

func ext_storage_append_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_append_version_1] executing...")

	ctx := env.(*runtime.Context)
	keySpan := args[0].I64()
	valueSpan := args[1].I64()
	storage := ctx.Storage

	key := asMemorySlice(ctx, keySpan)
	valueAppend := asMemorySlice(ctx, valueSpan)
	logger.Debug("[ext_storage_append_version_1]", "key", fmt.Sprintf("0x%x", key), "value to append", fmt.Sprintf("0x%x", valueAppend))

	cp := make([]byte, len(valueAppend))
	copy(cp, valueAppend)

	err := storageAppend(storage, key, cp)
	if err != nil {
		logger.Error("[ext_storage_append_version_1]", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_storage_changes_root_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_changes_root_version_1] executing...")
	logger.Debug("[ext_storage_changes_root_version_1] returning None")

	ctx := env.(*runtime.Context)

	rootSpan, err := toWasmMemoryOptional(ctx, nil)
	if err != nil {
		logger.Error("[ext_storage_changes_root_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(rootSpan)}, nil
}

func ext_storage_clear_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_clear_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	keySpan := args[0].I64()

	key := asMemorySlice(ctx, keySpan)

	logger.Debug("[ext_storage_clear_version_1]", "key", fmt.Sprintf("0x%x", key))
	storage.Delete(key)
	return nil, nil
}

func ext_storage_clear_prefix_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_clear_prefix_version_1] executing...")

	ctx := env.(*runtime.Context)
	prefixSpan := args[0].I64()

	storage := ctx.Storage

	prefix := asMemorySlice(ctx, prefixSpan)
	logger.Debug("[ext_storage_clear_prefix_version_1]", "prefix", fmt.Sprintf("0x%x", prefix))

	err := storage.ClearPrefix(prefix)
	if err != nil {
		logger.Error("[ext_storage_clear_prefix_version_1]", "error", err)
		return nil, err
	}

	return nil, nil
}

func ext_storage_exists_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_exists_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	keySpan := args[0].I64()

	key := asMemorySlice(ctx, keySpan)
	logger.Debug("[ext_storage_exists_version_1]", "key", fmt.Sprintf("0x%x", key))

	val := storage.Get(key)
	if len(val) > 0 {
		return []wasm.Value{wasm.NewI32(1)}, nil
	}

	return []wasm.Value{wasm.NewI32(0)}, nil
}

func ext_storage_get_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_get_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	keySpan := args[0].I64()

	key := asMemorySlice(ctx, keySpan)
	logger.Debug("[ext_storage_get_version_1]", "key", fmt.Sprintf("0x%x", key))

	value := storage.Get(key)
	logger.Debug("[ext_storage_get_version_1]", "value", fmt.Sprintf("0x%x", value))

	valueSpan, err := toWasmMemoryOptional(ctx, value)
	if err != nil {
		logger.Error("[ext_storage_get_version_1] failed to allocate", "error", err)
		return nil, err
	}

	logger.Debug("[ext_storage_get_version_1] returning...")
	return []wasm.Value{wasm.NewI64(valueSpan)}, nil
}

func ext_storage_next_key_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_next_key_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	keySpan := args[0].I64()

	key := asMemorySlice(ctx, keySpan)

	next := storage.NextKey(key)
	logger.Debug("[ext_storage_next_key_version_1]", "key", fmt.Sprintf("0x%x", key), "next", fmt.Sprintf("0x%x", next))

	nextSpan, err := toWasmMemoryOptional(ctx, next)
	if err != nil {
		logger.Error("[ext_storage_next_key_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(nextSpan)}, nil
}

func ext_storage_read_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_read_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	memory := ctx.Memory.Data()

	keySpan := args[0].I64()
	valueOut := args[1].I64()
	offset, ok := args[2].Unwrap().(int32)
	if !ok {
		logger.Crit("[ext_storage_read_version_1]", "error", "addr cannot be converted to int32")
	}

	key := asMemorySlice(ctx, keySpan)
	value := storage.Get(key)
	logger.Debug("[ext_storage_read_version_1]", "key", fmt.Sprintf("0x%x", key), "value", fmt.Sprintf("0x%x", value))

	if value == nil {
		ret, _ := toWasmMemoryOptional(ctx, nil)
		return []wasm.Value{wasm.NewI64(ret)}, nil
	}

	var size uint32

	if int(offset) > len(value) {
		size = uint32(0)
	} else {
		size = uint32(len(value[offset:]))
		valueBuf, valueLen := int64ToPointerAndSize(valueOut)
		copy(memory[valueBuf:valueBuf+valueLen], value[offset:])
	}

	sizeSpan, err := toWasmMemoryOptionalUint32(ctx, &size)
	if err != nil {
		logger.Error("[ext_storage_read_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(sizeSpan)}, nil
}

func ext_storage_root_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_root_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage

	root, err := storage.Root()
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to get storage root", "error", err)
		return nil, err
	}

	logger.Debug("[ext_storage_root_version_1]", "root", root)

	rootSpan, err := toWasmMemory(ctx, root[:])
	if err != nil {
		logger.Error("[ext_storage_root_version_1] failed to allocate", "error", err)
		return nil, err
	}

	return []wasm.Value{wasm.NewI64(rootSpan)}, nil
}

func ext_storage_set_version_1(env interface{}, args []wasm.Value) ([]wasm.Value, error) {
	logger.Trace("[ext_storage_set_version_1] executing...")

	ctx := env.(*runtime.Context)
	storage := ctx.Storage
	keySpan := args[0].I64()
	valueSpan := args[1].I64()

	key := asMemorySlice(ctx, keySpan)
	value := asMemorySlice(ctx, valueSpan)

	cp := make([]byte, len(value))
	copy(cp, value)

	logger.Debug("[ext_storage_set_version_1]", "key", fmt.Sprintf("0x%x", key), "val", fmt.Sprintf("0x%x", value))
	storage.Set(key, cp)
	return nil, nil
}

func ext_storage_start_transaction_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_storage_start_transaction_version_1] executing...")
	ctx := env.(*runtime.Context)
	ctx.Storage.BeginStorageTransaction()
	return nil, nil
}

func ext_storage_rollback_transaction_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_storage_rollback_transaction_version_1] executing...")
	ctx := env.(*runtime.Context)
	ctx.Storage.RollbackStorageTransaction()
	return nil, nil
}

func ext_storage_commit_transaction_version_1(env interface{}, _ []wasm.Value) ([]wasm.Value, error) {
	logger.Debug("[ext_storage_commit_transaction_version_1] executing...")
	ctx := env.(*runtime.Context)
	ctx.Storage.CommitStorageTransaction()
	return nil, nil
}

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(ctx *runtime.Context, span int64) []byte {
	memory := ctx.Memory.Data()
	ptr, size := int64ToPointerAndSize(span)
	return memory[ptr : ptr+size]
}

// Copy a byte slice to wasm memory and return the resulting 64bit span descriptor
func toWasmMemory(ctx *runtime.Context, data []byte) (int64, error) {
	allocator := ctx.Allocator
	size := uint32(len(data))

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := ctx.Memory.Data()

	if uint32(len(memory)) < out+size {
		panic(fmt.Sprintf("length of memory is less than expected, want %d have %d", out+size, len(memory)))
	}

	copy(memory[out:out+size], data)
	return pointerAndSizeToInt64(int32(out), int32(size)), nil
}

// Copy a byte slice of a fixed size to wasm memory and return resulting pointer
func toWasmMemorySized(ctx *runtime.Context, data []byte, size uint32) (uint32, error) {
	if int(size) != len(data) {
		return 0, errors.New("internal byte array size missmatch")
	}

	allocator := ctx.Allocator

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := ctx.Memory.Data()
	copy(memory[out:out+size], data)
	return out, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(ctx *runtime.Context, data []byte) (int64, error) {
	var opt *optional.Bytes
	if data == nil {
		opt = optional.NewBytes(false, nil)
	} else {
		opt = optional.NewBytes(true, data)
	}

	enc, err := opt.Encode()
	if err != nil {
		return 0, err
	}

	return toWasmMemory(ctx, enc)
}

// Wraps slice in Result type and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryResult(ctx *runtime.Context, data []byte) (int64, error) {
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

	return toWasmMemory(ctx, enc)
}

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(ctx *runtime.Context, data *uint32) (int64, error) {
	var opt *optional.Uint32
	if data == nil {
		opt = optional.NewUint32(false, 0)
	} else {
		opt = optional.NewUint32(true, *data)
	}

	enc := opt.Encode()
	return toWasmMemory(ctx, enc)
}

// Wraps slice in optional.FixedSizeBytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryFixedSizeOptional(context *runtime.Context, data []byte) (int64, error) {
	var opt *optional.FixedSizeBytes
	if data == nil {
		opt = optional.NewFixedSizeBytes(false, nil)
	} else {
		opt = optional.NewFixedSizeBytes(true, data)
	}

	enc, err := opt.Encode()
	if err != nil {
		return 0, err
	}

	return toWasmMemory(context, enc)
}
