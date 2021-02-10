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

import (
	"bytes"
	"os"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

var testChildKey = []byte("childKey")
var testKey = []byte("key")
var testValue = []byte("value")

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Error("failed to generate runtime wasm file", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func Test_ext_hashing_blake2_128_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_blake2_128_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected, err := common.Blake2b128(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_blake2_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_blake2_256_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected, err := common.Blake2bHash(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_keccak_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_keccak_256_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected, err := common.Keccak256(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_twox_128_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_twox_128_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected, err := common.Twox128Hash(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_twox_64_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_twox_64_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected, err := common.Twox64(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_sha2_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Encode(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_sha2_256_version_1", enc)
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected := common.Sha256(data)
	require.Equal(t, expected[:], hash)
}

func Test_ext_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	err := inst.inst.ctx.Storage.Set(testkey, []byte{1})
	require.NoError(t, err)

	enc, err := scale.Encode(testkey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_version_1", enc)
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Nil(t, val)
}

func Test_ext_storage_clear_prefix_version_1_hostAPI(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("static")
	err := inst.inst.ctx.Storage.Set(testkey, []byte("Inverse"))
	require.NoError(t, err)
	testkey2 := []byte("even-keeled")
	err = inst.inst.ctx.Storage.Set(testkey2, []byte("Future-proofed"))
	require.NoError(t, err)

	enc, err := scale.Encode(testkey[:3])
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_prefix_version_1", enc)
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Nil(t, val)

	val, err = inst.inst.ctx.Storage.Get(testkey2)
	require.NoError(t, err)
	require.NotNil(t, val)
}

func Test_ext_storage_clear_prefix_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	err := inst.inst.ctx.Storage.Set(testkey, []byte{1})
	require.NoError(t, err)
	testkey2 := []byte("spaghet")
	err = inst.inst.ctx.Storage.Set(testkey2, []byte{2})
	require.NoError(t, err)

	enc, err := scale.Encode(testkey[:3])
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_prefix_version_1", enc)
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Nil(t, val)

	val, err = inst.inst.ctx.Storage.Get(testkey2)
	require.NoError(t, err)
	require.NotNil(t, val)
}

func Test_ext_storage_get_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte{1, 2}
	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	enc, err := scale.Encode(testkey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_get_version_1", enc)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	value, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	require.Equal(t, testvalue, value.Value())
}

func Test_ext_storage_exists_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte{1, 2}
	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	enc, err := scale.Encode(testkey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_exists_version_1", enc)
	require.NoError(t, err)
	require.Equal(t, byte(1), ret[0])

	nonexistent := []byte("none")
	enc, err = scale.Encode(nonexistent)
	require.NoError(t, err)

	ret, err = inst.Exec("rtm_ext_storage_exists_version_1", enc)
	require.NoError(t, err)
	require.Equal(t, byte(0), ret[0])
}

func Test_ext_storage_next_key_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	err := inst.inst.ctx.Storage.Set(testkey, []byte{1})
	require.NoError(t, err)

	nextkey := []byte("oot")
	err = inst.inst.ctx.Storage.Set(nextkey, []byte{1})
	require.NoError(t, err)

	enc, err := scale.Encode(testkey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_next_key_version_1", enc)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	next, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	require.Equal(t, nextkey, next.Value())
}

func Test_ext_storage_read_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")
	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	testoffset := uint32(2)
	testBufferSize := uint32(100)

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Encode(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Encode(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	val := read.Value()
	require.Equal(t, testvalue[testoffset:], val[:len(testvalue)-int(testoffset)])
}

func Test_ext_storage_read_version_1_again(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("_was_here_")
	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	testoffset := uint32(8)
	testBufferSize := uint32(5)

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Encode(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Encode(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	val := read.Value()
	require.Equal(t, len(testvalue)-int(testoffset), len(val)) // TODO: fix
	require.Equal(t, testvalue[testoffset:], val[:len(testvalue)-int(testoffset)])
}

func Test_ext_storage_read_version_1_OffsetLargerThanValue(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")
	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	testoffset := uint32(len(testvalue))
	testBufferSize := uint32(8)

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Encode(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Encode(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	val := read.Value()
	require.Equal(t, []byte{}, val)
}

func Test_ext_storage_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	ret, err := inst.Exec("rtm_ext_storage_root_version_1", []byte{})
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	expected := trie.EmptyHash
	require.Equal(t, expected[:], hash)
}

func Test_ext_storage_set_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encValue, err := scale.Encode(testvalue)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_set_version_1", append(encKey, encValue...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Equal(t, testvalue, val)
}

func Test_ext_crypto_ed25519_generate_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	require.Equal(t, 0, inst.inst.ctx.Keystore.Size())

	idData := []byte{2, 2, 2, 2}

	// TODO: we currently don't provide a seed since the spec says the seed is an optional BIP-39 seed
	// clarify whether this is a mnemonic or not
	data := optional.NewBytes(false, nil)
	seedData, err := data.Encode()
	require.NoError(t, err)

	params := append(idData, seedData...)

	// we manually store and call the runtime function here since inst.exec assumes
	// the data returned from the function is a pointer-size, but for ext_crypto_ed25519_generate_version_1,
	// it's just a pointer
	ptr, err := inst.inst.malloc(uint32(len(params)))
	require.NoError(t, err)

	inst.inst.store(params, int32(ptr))
	dataLen := int32(len(params))

	runtimeFunc, ok := inst.inst.vm.Exports["rtm_ext_crypto_ed25519_generate_version_1"]
	require.True(t, ok)

	ret, err := runtimeFunc(int32(ptr), dataLen)
	require.NoError(t, err)

	mem := inst.inst.vm.Memory.Data()
	// TODO: why is this SCALE encoded? it should just be a 32 byte buffer. may be due to way test runtime is written.
	pubKeyBytes := mem[ret.ToI32()+1 : ret.ToI32()+1+32]
	pubKey, err := ed25519.NewPublicKey(pubKeyBytes)
	require.NoError(t, err)

	require.Equal(t, 1, inst.inst.ctx.Keystore.Size())
	kp := inst.inst.ctx.Keystore.GetKeypair(pubKey)
	require.NotNil(t, kp)
}

func Test_ext_crypto_ed25519_public_keys_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	require.Equal(t, 0, inst.inst.ctx.Keystore.Size())

	var pubKeys [][]byte
	for i := 0; i < 5; i++ {
		kp, err := ed25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		inst.inst.ctx.Keystore.Insert(kp)
		pubKeys = append(pubKeys, kp.Public().Encode())
	}

	sort.Slice(pubKeys, func(i int, j int) bool { return bytes.Compare(pubKeys[i], pubKeys[j]) < 0 })

	var expectedPubKeys []byte
	for _, key := range pubKeys {
		expectedPubKeys = append(expectedPubKeys, key...)
	}

	idData := []byte{2, 2, 2, 2}

	res, err := inst.Exec("rtm_ext_crypto_ed25519_public_keys_version_1", idData)
	require.NoError(t, err)

	out, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out.([]byte))

	value, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	require.Equal(t, expectedPubKeys, value.Value())
}

func Test_ext_crypto_ed25519_sign_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	inst.inst.ctx.Keystore.Insert(kp)

	idData := []byte{2, 2, 2, 2}

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Encode(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Encode(msgData)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_crypto_ed25519_sign_version_1", append(append(idData, encPubKey...), encMsg...))
	require.NoError(t, err)

	out, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out.([]byte))

	value, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	ok, err := kp.Public().Verify(msgData, value.Value())
	require.NoError(t, err)

	require.True(t, ok)
}

func Test_ext_crypto_ed25519_verify_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	inst.inst.ctx.Keystore.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Encode(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Encode(msgData)
	require.NoError(t, err)

	sign, err := kp.Private().Sign(msgData)
	require.NoError(t, err)
	encSign, err := scale.Encode(sign)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_ed25519_verify_version_1", append(append(encSign, encMsg...), encPubKey...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	require.True(t, read.Exists())
}

func Test_ext_crypto_sr25519_generate_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	require.Equal(t, 0, inst.inst.ctx.Keystore.Size())

	idData := []byte{2, 2, 2, 2}

	data := optional.NewBytes(false, nil)
	seedData, err := data.Encode()
	require.NoError(t, err)

	params := append(idData, seedData...)

	ret, err := inst.Exec("rtm_ext_crypto_sr25519_generate_version_1", params)
	require.NoError(t, err)

	out, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	pubKey, err := ed25519.NewPublicKey(out.([]byte))
	require.NoError(t, err)
	require.Equal(t, 1, inst.inst.ctx.Keystore.Size())

	kp := inst.inst.ctx.Keystore.GetKeypair(pubKey)
	require.NotNil(t, kp)
}

func Test_ext_crypto_secp256k1_ecdsa_recover_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	msgData := []byte("Hello world!")
	blakeHash, err := common.Blake2bHash(msgData)
	require.NoError(t, err)

	kp, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)

	inst.inst.ctx.Keystore.Insert(kp)
	sigData, err := kp.Private().Sign(blakeHash.ToBytes())
	require.NoError(t, err)

	expectedPubKey := kp.Public().Encode()

	encSign, err := scale.Encode(sigData)
	require.NoError(t, err)
	encMsg, err := scale.Encode(blakeHash.ToBytes())
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_secp256k1_ecdsa_recover_version_1", append(encSign, encMsg...))
	require.NoError(t, err)

	out, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out.([]byte))

	uncomPubKey, err := new(types.Result).Decode(buf)
	require.NoError(t, err)

	publicKey := new(secp256k1.PublicKey)

	// Generates [33]byte compressed key from uncompressed [65]byte public key.
	err = publicKey.UnmarshalPubkey(uncomPubKey.Value())
	require.NoError(t, err)

	require.Equal(t, expectedPubKey, publicKey.Encode())
}

func Test_ext_crypto_sr25519_public_keys_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	require.Equal(t, 0, inst.inst.ctx.Keystore.Size())

	var pubKeys [][]byte
	for i := 0; i < 5; i++ {
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		inst.inst.ctx.Keystore.Insert(kp)
		pubKeys = append(pubKeys, kp.Public().Encode())
	}

	sort.Slice(pubKeys, func(i int, j int) bool { return bytes.Compare(pubKeys[i], pubKeys[j]) < 0 })

	var expectedPubKeys []byte
	for _, key := range pubKeys {
		expectedPubKeys = append(expectedPubKeys, key...)
	}

	res, err := inst.Exec("rtm_ext_crypto_sr25519_public_keys_version_1", []byte{2, 2, 2, 2})
	require.NoError(t, err)

	out, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out.([]byte))

	value, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	require.Equal(t, expectedPubKeys, value.Value())
}

func Test_ext_crypto_sr25519_sign_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	inst.inst.ctx.Keystore.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Encode(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Encode(msgData)
	require.NoError(t, err)

	idData := []byte{2, 2, 2, 2}

	res, err := inst.Exec("rtm_ext_crypto_sr25519_sign_version_1", append(append(idData, encPubKey...), encMsg...))
	require.NoError(t, err)

	out, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out.([]byte))

	value, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	ok, err := kp.Public().Verify(msgData, value.Value())
	require.NoError(t, err)

	require.True(t, ok)
}

func Test_ext_crypto_sr25519_verify_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	inst.inst.ctx.Keystore.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Encode(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Encode(msgData)
	require.NoError(t, err)

	sign, err := kp.Private().Sign(msgData)
	require.NoError(t, err)
	encSign, err := scale.Encode(sign)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_sr25519_verify_version_1", append(append(encSign, encMsg...), encPubKey...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	require.True(t, read.Exists())
}

func Test_ext_default_child_storage_read_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	testOffset := uint32(2)
	testBufferSize := uint32(100)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	encBufferSize, err := scale.Encode(testBufferSize)
	require.NoError(t, err)

	encOffset, err := scale.Encode(testOffset)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_read_version_1", append(append(encChildKey, encKey...), append(encOffset, encBufferSize...)...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)

	val := read.Value()
	require.Equal(t, testValue[testOffset:], val[:len(testValue)-int(testOffset)])
}

func Test_ext_default_child_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	// Confirm if value is set
	val, err := inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, val)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	val, err = inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Nil(t, val)
}

func Test_ext_default_child_storage_clear_prefix_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	prefix := []byte("key")

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("keyOne"), []byte("value1")},
		{[]byte("keyTwo"), []byte("value2")},
		{[]byte("keyThree"), []byte("value3")},
	}

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	// Confirm if value is set
	keys, err := inst.inst.ctx.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encPrefix, err := scale.Encode(prefix)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_prefix_version_1", append(encChildKey, encPrefix...))
	require.NoError(t, err)

	keys, err = inst.inst.ctx.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 0, len(keys))
}

func Test_ext_default_child_storage_exists_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_exists_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	require.True(t, read.Exists())
}

func Test_ext_default_child_storage_get_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_get_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	require.Equal(t, testValue, read.Value())
}

func Test_ext_default_child_storage_next_key_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("apple"), []byte("value1")},
		{[]byte("key"), []byte("value2")},
	}

	key := testKeyValuePair[0].key

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(key)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_next_key_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(ret)

	read, err := new(optional.Bytes).Decode(buf)
	require.NoError(t, err)
	require.Equal(t, testKeyValuePair[1].key, read.Value())
}

func Test_ext_default_child_storage_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	child, err := inst.inst.ctx.Storage.GetChild(testChildKey)
	require.NoError(t, err)

	rootHash, err := child.Hash()
	require.NoError(t, err)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)
	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_root_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	hash, err := scale.Decode(ret, []byte{})
	require.NoError(t, err)

	// Convert decoded interface to common Hash
	actualValue := common.BytesToHash(hash.([]byte))
	require.Equal(t, rootHash, actualValue)
}

func Test_ext_default_child_storage_set_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Check if value is not set
	val, err := inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Nil(t, val)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	encVal, err := scale.Encode(testValue)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_set_version_1", append(append(encChildKey, encKey...), encVal...))
	require.NoError(t, err)

	val, err = inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, val)
}

func Test_ext_default_child_storage_storage_kill_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.inst.ctx.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_storage_kill_version_1", encChildKey)
	require.NoError(t, err)

	child, err = inst.inst.ctx.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.Nil(t, child)
}

func Test_ext_storage_append_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("was")
	testvalueAppend := []byte("here")

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encVal, err := scale.Encode(testvalue)
	require.NoError(t, err)
	doubleEncVal, err := scale.Encode(encVal)
	require.NoError(t, err)

	encArr, err := scale.Encode([][]byte{testvalue})
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncVal...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Equal(t, encArr, val)

	encValueAppend, err := scale.Encode(testvalueAppend)
	require.NoError(t, err)
	doubleEncValueAppend, err := scale.Encode(encValueAppend)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncValueAppend...))
	require.NoError(t, err)

	ret, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	dec, err := scale.Decode(ret, [][]byte{})
	require.NoError(t, err)

	res := dec.([][]byte)
	require.Equal(t, 2, len(res))
	require.Equal(t, testvalue, res[0])
	require.Equal(t, testvalueAppend, res[1])

	expected, err := scale.Encode([][]byte{testvalue, testvalueAppend})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func Test_ext_storage_append_version_1_again(t *testing.T) {
	DefaultTestLogLvl = 5
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("abc")
	testvalueAppend := []byte("def")

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encVal, err := scale.Encode(testvalue)
	require.NoError(t, err)
	doubleEncVal, err := scale.Encode(encVal)
	require.NoError(t, err)

	encArr, err := scale.Encode([][]byte{testvalue})
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncVal...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Equal(t, encArr, val)

	encValueAppend, err := scale.Encode(testvalueAppend)
	require.NoError(t, err)
	doubleEncValueAppend, err := scale.Encode(encValueAppend)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncValueAppend...))
	require.NoError(t, err)

	ret, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	dec, err := scale.Decode(ret, [][]byte{})
	require.NoError(t, err)

	res := dec.([][]byte)
	require.Equal(t, 2, len(res))
	require.Equal(t, testvalue, res[0])
	require.Equal(t, testvalueAppend, res[1])

	expected, err := scale.Encode([][]byte{testvalue, testvalueAppend})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func TestStartTransaction_ext_storage_set_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	// instead of committing the change, it should be stored in the context
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	testkey := []byte("noot")
	testvalue := []byte("washere")

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encValue, err := scale.Encode(testvalue)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_set_version_1", append(encKey, encValue...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Nil(t, val)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.SetOp, changes[0].Operation)
	require.Equal(t, testkey, changes[0].Key)
	require.Equal(t, testvalue, changes[0].Value)
}

func TestStartTransaction_ext_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	testkey := []byte("noot")
	err := inst.inst.ctx.Storage.Set(testkey, []byte{1})
	require.NoError(t, err)

	enc, err := scale.Encode(testkey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_version_1", enc)
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.NotNil(t, val)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.ClearOp, changes[0].Operation)
	require.Equal(t, testkey, changes[0].Key)
}

func TestStartTransaction_ext_storage_append_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	testkey := []byte("noot")
	testvalue := []byte("was")
	testvalueAppend := []byte("here")

	err := inst.inst.ctx.Storage.Set(testkey, testvalue)
	require.NoError(t, err)

	encKey, err := scale.Encode(testkey)
	require.NoError(t, err)
	encValue, err := scale.Encode(testvalueAppend)
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, encValue...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Equal(t, testvalue, val)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.AppendOp, changes[0].Operation)
	require.Equal(t, testkey, changes[0].Key)
	require.Equal(t, testvalueAppend, changes[0].Value)
}

func TestStartTransaction_ext_default_child_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	// Confirm if value is set
	val, err := inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, val)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	val, err = inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.NotNil(t, val)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.ClearOp, changes[0].Operation)
	require.Equal(t, testChildKey, changes[0].KeyToChild)
	require.Equal(t, testKey, changes[0].Key)
}

func TestStartTransaction_ext_default_child_storage_clear_prefix_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	prefix := []byte("key")

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("keyOne"), []byte("value1")},
		{[]byte("keyTwo"), []byte("value2")},
		{[]byte("keyThree"), []byte("value3")},
	}

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.inst.ctx.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	// Confirm if value is set
	keys, err := inst.inst.ctx.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encPrefix, err := scale.Encode(prefix)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_prefix_version_1", append(encChildKey, encPrefix...))
	require.NoError(t, err)

	keys, err = inst.inst.ctx.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.ClearPrefixOp, changes[0].Operation)
	require.Equal(t, testChildKey, changes[0].KeyToChild)
	require.Equal(t, prefix, changes[0].Prefix)
}

func TestStartTransaction_ext_default_child_storage_set_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Check if value is not set
	val, err := inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Nil(t, val)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Encode(testKey)
	require.NoError(t, err)

	encVal, err := scale.Encode(testValue)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_set_version_1", append(append(encChildKey, encKey...), encVal...))
	require.NoError(t, err)

	val, err = inst.inst.ctx.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Nil(t, val)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.SetOp, changes[0].Operation)
	require.Equal(t, testChildKey, changes[0].KeyToChild)
	require.Equal(t, testKey, changes[0].Key)
	require.Equal(t, testValue, changes[0].Value)
}

func TestStartTransaction_ext_default_child_storage_storage_kill_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	inst.inst.ctx.TransactionStorageChanges = []*runtime.TransactionStorageChange{}

	err := inst.inst.ctx.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.inst.ctx.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_storage_kill_version_1", encChildKey)
	require.NoError(t, err)

	child, err = inst.inst.ctx.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	changes := inst.inst.ctx.TransactionStorageChanges
	require.Equal(t, 1, len(changes))
	require.Equal(t, runtime.DeleteChildOp, changes[0].Operation)
	require.Equal(t, testChildKey, changes[0].KeyToChild)
}

func Test_ext_trie_blake2_256_ordered_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testvalues := []string{"static", "even-keeled", "Future-proofed"}
	encValues, err := scale.Encode(testvalues)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_trie_blake2_256_ordered_root_version_1", encValues)
	require.NoError(t, err)

	hash, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	expected := common.MustHexToHash("0xd847b86d0219a384d11458e829e9f4f4cce7e3cc2e6dcd0e8a6ad6f12c64a737")
	require.Equal(t, expected[:], hash)
}

func Test_ext_trie_blake2_256_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testinput := []string{"noot", "was", "here", "??"}
	encInput, err := scale.Encode(testinput)
	require.NoError(t, err)
	encInput[0] = encInput[0] >> 1

	res, err := inst.Exec("rtm_ext_trie_blake2_256_root_version_1", encInput)
	require.NoError(t, err)

	hash, err := scale.Decode(res, []byte{})
	require.NoError(t, err)

	tt := trie.NewEmptyTrie()
	tt.Put([]byte("noot"), []byte("was"))
	tt.Put([]byte("here"), []byte("??"))

	expected := tt.MustHash()
	require.Equal(t, expected[:], hash)
}
