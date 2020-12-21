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
	"fmt"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

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

	var expectedPubKeys [][]byte
	numKps := 5

	for i := 0; i < numKps; i++ {
		kp, err := ed25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		inst.inst.ctx.Keystore.Insert(kp)
		expected := kp.Public().Encode()
		expectedPubKeys = append(expectedPubKeys, expected)
	}

	// put some ed25519 keypairs in the keystore to make sure they don't get returned
	for i := 0; i < numKps; i++ {
		kp, err := ed25519.GenerateKeypair()
		require.Nil(t, err)

		inst.inst.ctx.Keystore.Insert(kp)
	}

	idData := []byte{2, 2, 2, 2}

	ptr, err := inst.inst.malloc(uint32(len(idData)))
	require.NoError(t, err)

	inst.inst.store(idData, int32(ptr))
	dataLen := int32(len(idData))

	require.NoError(t, err)

	runtimeFunc, ok := inst.inst.vm.Exports["rtm_ext_crypto_ed25519_public_keys_version_1"]
	require.True(t, ok)

	out, err := runtimeFunc(int32(ptr), dataLen)
	require.NoError(t, err)

	mem := inst.inst.vm.Memory.Data()
	//resultLenBytes := mem[0:32]
	//resultLen := binary.LittleEndian.Uint32(resultLenBytes)
	pubKeyData := mem[out.ToI32() : out.ToI32()+int32(numKps*32)]

	pubKeys := [][]byte{}
	for i := 0; i < numKps; i++ {
		kpData := pubKeyData[i*32 : i*32+32]
		pubKeys = append(pubKeys, kpData)
	}

	sort.Slice(expectedPubKeys, func(i, j int) bool { return bytes.Compare(expectedPubKeys[i], expectedPubKeys[j]) < 0 })
	sort.Slice(pubKeys, func(i, j int) bool { return bytes.Compare(pubKeys[i], pubKeys[j]) < 0 })

	require.Equal(t, expectedPubKeys, pubKeys)
}

func Test_ext_crypto_ed25519_sign_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	//mem := inst.inst.vm.Memory.Data()

	kp, err := ed25519.GenerateKeypair()
	require.Nil(t, err)

	inst.inst.ctx.Keystore.Insert(kp)

	idData := []byte{2, 2, 2, 2}
	pubKeyData := kp.Public().Encode()
	msgData := []byte("helloworld")
	encMsg, err := scale.Encode(msgData)
	require.NoError(t, err)

	params := append(idData, pubKeyData...)
	params = append(params, encMsg...)

	ptr, err := inst.inst.malloc(uint32(len(params)))
	require.NoError(t, err)

	inst.inst.store(params, int32(ptr))
	dataLen := int32(len(params))

	runtimeFunc, ok := inst.inst.vm.Exports["rtm_ext_crypto_ed25519_sign_version_1"]
	require.True(t, ok)

	ret, err := runtimeFunc(int32(ptr), dataLen)
	require.NoError(t, err)

	fmt.Println(ret.ToI32())
}
