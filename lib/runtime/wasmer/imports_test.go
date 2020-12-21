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
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

var testChildKey = []byte("childKey")
var testKey = []byte("key")
var testValue = []byte("value")

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
	keys, err := inst.inst.ctx.Storage.(*runtime.TestRuntimeStorage).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	encChildKey, err := scale.Encode(testChildKey)
	require.NoError(t, err)

	encPrefix, err := scale.Encode(prefix)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_prefix_version_1", append(encChildKey, encPrefix...))
	require.NoError(t, err)

	keys, err = inst.inst.ctx.Storage.(*runtime.TestRuntimeStorage).GetKeysWithPrefixFromChild(testChildKey, prefix)
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
	encValue, err := scale.Encode(testvalue)
	require.NoError(t, err)
	doubleEncValue, err := scale.Encode(encValue)
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_set_version_1", append(encKey, doubleEncValue...))
	require.NoError(t, err)

	val, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	require.Equal(t, encValue, val)

	encValueAppend, err := scale.Encode(testvalueAppend)
	require.NoError(t, err)
	dpublEncValueAppend, err := scale.Encode(encValueAppend)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, dpublEncValueAppend...))
	require.NoError(t, err)

	res, err := inst.inst.ctx.Storage.Get(testkey)
	require.NoError(t, err)
	dec, err := scale.Decode(res, []byte{})
	require.NoError(t, err)
	require.Equal(t, append(testvalue, testvalueAppend...), dec)
}
