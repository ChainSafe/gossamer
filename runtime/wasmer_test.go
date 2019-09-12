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

package runtime

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"github.com/ChainSafe/gossamer/codec"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/trie"
	"golang.org/x/crypto/ed25519"
)

const POLKADOT_RUNTIME_FP string = "polkadot_runtime.compact.wasm"
const POLKADOT_RUNTIME_URL string = "https://github.com/w3f/polkadot-re-tests/blob/master/polkadot-runtime/polkadot_runtime.compact.wasm?raw=true"

// getRuntimeBlob checks if the polkadot runtime wasm file exists and if not, it fetches it from github
func getRuntimeBlob() (n int64, err error) {
	if Exists(POLKADOT_RUNTIME_FP) {
		return 0, nil
	}

	out, err := os.Create(POLKADOT_RUNTIME_FP)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	resp, err := http.Get(POLKADOT_RUNTIME_URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func newRuntime(t *testing.T) (*Runtime, error) {
	_, err := getRuntimeBlob()
	if err != nil {
		t.Fatalf("Fail: could not get polkadot runtime")
	}

	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	tt := &trie.Trie{}

	r, err := NewRuntime(fp, tt)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r, err
}

func TestExecVersion(t *testing.T) {
	expected := &Version{
		Spec_name:         []byte("polkadot"),
		Impl_name:         []byte("parity-polkadot"),
		Authoring_version: 1,
		Spec_version:      1000,
		Impl_version:      0,
	}

	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := r.Exec("Core_version", 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ret)

	res, err := decodeToInterface(ret, &Version{})
	if err != nil {
		t.Fatal(err)
	}

	version := res.(*Version)
	t.Logf("Spec_name: %s\n", version.Spec_name)
	t.Logf("Impl_name: %s\n", version.Impl_name)
	t.Logf("Authoring_version: %d\n", version.Authoring_version)
	t.Logf("Spec_version: %d\n", version.Spec_version)
	t.Logf("Impl_version: %d\n", version.Impl_version)

	if !reflect.DeepEqual(version, expected) {
		t.Errorf("Fail: got %v expected %v\n", version, expected)
	}
}

const TESTS_FP string = "./test_wasm.wasm"
const TEST_WASM_URL string = "https://github.com/ChainSafe/gossamer-test-wasm/raw/master/target/wasm32-unknown-unknown/release/test_wasm.wasm"

// getTestBlob checks if the test wasm file exists and if not, it fetches it from github
func getTestBlob() (n int64, err error) {
	if Exists(TESTS_FP) {
		return 0, nil
	}

	out, err := os.Create(TESTS_FP)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	resp, err := http.Get(TEST_WASM_URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

func newTestRuntime() (*Runtime, error) {
	_, err := getTestBlob()
	if err != nil {
		return nil, err
	}

	t := &trie.Trie{}
	fp, err := filepath.Abs(TESTS_FP)
	if err != nil {
		return nil, err
	}
	r, err := NewRuntime(fp, t)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// tests that the function ext_get_storage_into can retrieve a value from the trie
// and store it in the wasm memory
func TestExt_get_storage_into(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	// store kv pair in trie
	key := []byte(":noot")
	value := []byte{1, 3, 3, 7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	// copy key to position `keyData` in memory
	keyData := 170
	// return value will be saved at position `valueData`
	valueData := 200
	// `valueOffset` is the position in the value following which its bytes should be stored
	valueOffset := 0
	copy(mem[keyData:keyData+len(key)], key)

	testFunc, ok := runtime.vm.Exports["test_ext_get_storage_into"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	ret, err := testFunc(keyData, len(key), valueData, len(value), valueOffset)
	if err != nil {
		t.Fatal(err)
	} else if ret.ToI32() != int32(len(value)) {
		t.Error("return value does not match length of value in trie")
	} else if !bytes.Equal(mem[valueData:valueData+len(value)], value[valueOffset:]) {
		t.Error("did not store correct value in memory")
	}

	key = []byte("doesntexist")
	copy(mem[keyData:keyData+len(key)], key)
	expected := 1<<32 - 1
	ret, err = testFunc(keyData, len(key), valueData, len(value), valueOffset)
	if err != nil {
		t.Fatal(err)
	} else if ret.ToI32() != int32(expected) {
		t.Errorf("return value should be 2^32 - 1 since value doesn't exist, got %d", ret.ToI32())
	}
}

// tests that ext_set_storage can storage a value in the trie
func TestExt_set_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	// key,value we wish to store in the trie
	key := []byte(":noot")
	value := []byte{1, 3, 3, 7}

	// copy key and value into wasm memory
	keyData := 170
	valueData := 200
	copy(mem[keyData:keyData+len(key)], key)
	copy(mem[valueData:valueData+len(value)], value)

	testFunc, ok := runtime.vm.Exports["test_ext_set_storage"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(keyData, len(key), valueData, len(value))
	if err != nil {
		t.Fatal(err)
	}

	// make sure we can get the value from the trie
	trieValue, err := runtime.trie.Get(key)
	if err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(value, trieValue) {
		t.Error("did not store correct value in storage trie")
	}

	t.Log(trieValue)
}

// tests that we can retrieve the trie root hash and store it in wasm memory
func TestExt_storage_root(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	// save result at `resultPtr` in memory
	resultPtr := 170
	hash, err := runtime.trie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	testFunc, ok := runtime.vm.Exports["test_ext_storage_root"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(resultPtr)
	if err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(mem[resultPtr:resultPtr+32], hash[:]) {
		t.Error("did not save trie hash to memory")
	}
}

// test that ext_get_allocated_storage can get a value from the trie and store it
// in wasm memory
func TestExt_get_allocated_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	// put kv pair in trie
	key := []byte(":noot")
	value := []byte{1, 3, 3, 7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	// copy key to `keyData` in memory
	keyData := 170
	copy(mem[keyData:keyData+len(key)], key)
	// memory location where length of return value is stored
	var writtenOut int32 = 169

	testFunc, ok := runtime.vm.Exports["test_ext_get_allocated_storage"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	ret, err := testFunc(keyData, len(key), writtenOut)
	if err != nil {
		t.Fatal(err)
	}

	// returns memory location where value is stored
	retInt := uint32(ret.ToI32())
	loc := uint32(mem[writtenOut])
	length := binary.LittleEndian.Uint32(mem[loc : loc+4])
	if length != uint32(len(value)) {
		t.Error("did not save correct value length to memory")
	} else if !bytes.Equal(mem[retInt:retInt+length], value) {
		t.Error("did not save value to memory")
	}

	key = []byte("doesntexist")
	copy(mem[keyData:keyData+len(key)], key)
	ret, err = testFunc(keyData, len(key), writtenOut)
	if err != nil {
		t.Fatal(err)
	} else if ret.ToI32() != int32(0) {
		t.Errorf("return value should be 0 since value doesn't exist, got %d", ret.ToI32())
	}
}

// test that ext_clear_storage can delete a value from the trie
func TestExt_clear_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	// save kv pair in trie
	key := []byte(":noot")
	value := []byte{1, 3, 3, 7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	// copy key to wasm memory
	keyData := 170
	copy(mem[keyData:keyData+len(key)], key)

	testFunc, ok := runtime.vm.Exports["test_ext_clear_storage"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(keyData, len(key))
	if err != nil {
		t.Fatal(err)
	}

	// make sure value is deleted
	ret, err := runtime.trie.Get(key)
	if err != nil {
		t.Fatal(err)
	} else if ret != nil {
		t.Error("did not delete key from storage trie")
	}
}

// test that ext_clear_prefix can delete all trie values with a certain prefix
func TestExt_clear_prefix(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	// store some values in the trie
	tests := []struct {
		key   []byte
		value []byte
	}{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
	}

	for _, test := range tests {
		e := runtime.trie.Put(test.key, test.value)
		if e != nil {
			t.Fatal(e)
		}
	}

	// we are going to delete prefix 0x0135
	expected := []struct {
		key   []byte
		value []byte
	}{
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
	}

	expectedTrie := &trie.Trie{}

	for _, test := range expected {
		e := expectedTrie.Put(test.key, test.value)
		if e != nil {
			t.Fatal(e)
		}
	}

	// copy prefix we want to delete to wasm memory
	prefix := []byte{0x01, 0x35}
	prefixData := 170
	copy(mem[prefixData:prefixData+len(prefix)], prefix)

	testFunc, ok := runtime.vm.Exports["test_ext_clear_prefix"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(prefixData, len(prefix))
	if err != nil {
		t.Fatal(err)
	}

	// make sure entries with that prefix were deleted
	runtimeTrieHash, err := runtime.trie.Hash()
	if err != nil {
		t.Fatal(err)
	}
	expectedHash, err := expectedTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(runtimeTrieHash[:], expectedHash[:]) {
		t.Error("did not get expected trie")
	}
}

// test that ext_blake2_256 performs a blake2b hash of the data
func TestExt_blake2_256(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	// save data in memory
	data := []byte("helloworld")
	pos := 170
	out := 180
	copy(mem[pos:pos+len(data)], data)

	testFunc, ok := runtime.vm.Exports["test_ext_blake2_256"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(pos, len(data), out)
	if err != nil {
		t.Fatal(err)
	}

	// make sure hashes match
	hash, err := common.Blake2bHash(data)
	if err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(hash[:], mem[out:out+32]) {
		t.Error("hash saved in memory does not equal calculated hash")
	}
}

// test that ext_ed25519_verify verifies a valid signature
func TestExt_ed25519_verify(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	// copy message into memory
	msg := []byte("helloworld")
	msgData := 170
	copy(mem[msgData:msgData+len(msg)], msg)

	// create key
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// copy public key into memory
	pubkeyData := 180
	copy(mem[pubkeyData:pubkeyData+len(pub)], pub)

	// sign message, copy signature into memory
	sig := ed25519.Sign(priv, msg)
	sigData := 222
	copy(mem[sigData:sigData+len(sig)], sig)

	testFunc, ok := runtime.vm.Exports["test_ext_ed25519_verify"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	verified, err := testFunc(msgData, len(msg), sigData, pubkeyData)
	if err != nil {
		t.Fatal(err)
	} else if verified.ToI32() != 0 {
		t.Error("did not verify ed25519 signature")
	}

	// verification should fail on wrong signature
	sigData = 1
	verified, err = testFunc(msgData, len(msg), sigData, pubkeyData)
	if err != nil {
		t.Fatal(err)
	} else if verified.ToI32() != 1 {
		t.Error("verified incorrect ed25519 signature")
	}
}

// test that ext_blake2_256_enumerated_trie_root places values in an array into a trie
// with the key being the index of the value and returns the hash
func TestExt_blake2_256_enumerated_trie_root(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	// construct expected trie
	tests := []struct {
		key   []byte
		value []byte
	}{
		{key: []byte{0}, value: []byte("pen")},
		{key: []byte{1}, value: []byte("penguin")},
		{key: []byte{2}, value: []byte("feather")},
		{key: []byte{3}, value: []byte("noot")},
	}

	expectedTrie := &trie.Trie{}
	valuesArray := []byte{}
	lensArray := []byte{}

	for _, test := range tests {
		e := expectedTrie.Put(test.key, test.value)
		if e != nil {
			t.Fatal(e)
		}

		// construct array of values
		valuesArray = append(valuesArray, test.value...)
		lensVal := make([]byte, 4)
		binary.LittleEndian.PutUint32(lensVal, uint32(len(test.value)))
		// construct array of lengths of the values, where each length is int32
		lensArray = append(lensArray, lensVal...)
	}

	// save value array into memory at `valuesData`
	valuesData := 1
	// save lengths array into memory at `lensData`
	lensData := valuesData + len(valuesArray)
	// save length of lengths array in memory at `lensLen`
	lensLen := len(tests)
	// return value will be saved at `result` in memory
	result := lensLen + 1
	copy(mem[valuesData:valuesData+len(valuesArray)], valuesArray)
	copy(mem[lensData:lensData+len(lensArray)], lensArray)

	testFunc, ok := runtime.vm.Exports["test_ext_blake2_256_enumerated_trie_root"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(valuesData, lensData, lensLen, result)
	if err != nil {
		t.Fatal(err)
	}

	expectedHash, err := expectedTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// confirm that returned hash matches expected hash
	if !bytes.Equal(mem[result:result+32], expectedHash[:]) {
		t.Error("did not get expected trie")
	}
}

// test that ext_twox_128 performs a xxHash64 twice on give byte array of the data
func TestExt_twox_128(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	// save data in memory
	// test for empty []byte
	data := []byte(nil)
	pos := 170
	out := pos + len(data)
	copy(mem[pos:pos+len(data)], data)

	// call wasm function
	testFunc, ok := runtime.vm.Exports["test_ext_twox_128"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(pos, len(data), out)
	if err != nil {
		t.Fatal(err)
	}

	//check result against expected value
	t.Logf("Ext_twox_128 data: %s, result: %s", data, hex.EncodeToString(mem[out:out+16]))
	if "99e9d85137db46ef4bbea33613baafd5" != hex.EncodeToString(mem[out:out+16]) {
		t.Error("hash saved in memory does not equal calculated hash")
	}

	// test for data value "Hello world!"
	data = []byte("Hello world!")
	out = pos + len(data)
	copy(mem[pos:pos+len(data)], data)

	// call wasm function
	testFunc, ok = runtime.vm.Exports["test_ext_twox_128"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc(pos, len(data), out)
	if err != nil {
		t.Fatal(err)
	}

	//check result against expected value
	t.Logf("Ext_twox_128 data: %s, result: %s", data, hex.EncodeToString(mem[out:out+16]))
	if "b27dfd7f223f177f2a13647b533599af" != hex.EncodeToString(mem[out:out+16]) {
		t.Error("hash saved in memory does not equal calculated hash")
	}
}

// test ext_malloc returns expected pointer value of 8
func TestExt_malloc(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	testFunc, ok := runtime.vm.Exports["test_ext_malloc"]
	if !ok {
		t.Fatal("could not find exported function")
	}
	// when
	res, err := testFunc(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[TestExt_malloc]", "pointer", res)
	if res.ToI64() != 8 {
		t.Errorf("malloc did not return expected pointer value, expected 8, got %v", res)
	}
}

// test ext_free, confirm ext_free frees memory without error
func TestExt_free(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	initFunc, ok := runtime.vm.Exports["test_ext_malloc"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	ptr, err := initFunc(1)
	if err != nil {
		t.Fatal(err)
	}
	if ptr.ToI64() != 8 {
		t.Errorf("malloc did not return expected pointer value, expected 8, got %v", ptr)
	}

	// when
	testFunc, ok := runtime.vm.Exports["test_ext_free"]
	if !ok {
		t.Fatal("could not find exported function")
	}
	_, err = testFunc(ptr)

	// then
	if err != nil {
		t.Fatal(err)
	}
}

func TestCallCoreExecuteBlock(t *testing.T) {
	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	mem := r.vm.Memory.Data()

	//data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 180, 214, 110, 56, 24, 247, 42, 231, 157, 240, 245, 4, 137, 200, 210, 83, 186, 9, 22, 6, 156, 106, 186, 145, 150, 170, 180, 132, 214, 165, 38, 112, 245, 245, 225, 99, 148, 4, 245, 47, 172, 94, 196, 11, 214, 21, 116, 193, 203, 159, 88, 196, 151, 133, 125, 205, 12, 241, 177, 196, 73, 92, 91, 254, 0, 8, 28, 2, 2, 0, 66, 144, 2, 0, 69, 2, 130, 255, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 192, 138, 76, 246, 11, 75, 33, 19, 237, 153, 203, 25, 35, 55, 53, 57, 254, 216, 179, 42, 173, 134, 215, 83, 20, 254, 80, 30, 22, 135, 2, 92, 232, 126, 155, 24, 168, 178, 125, 228, 126, 154, 165, 10, 243, 37, 144, 177, 105, 23, 191, 251, 1, 76, 23, 156, 111, 195, 56, 142, 15, 228, 30, 15, 7, 0, 0, 0, 5, 0, 255, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 15, 0, 64, 243, 112, 131, 131, 24}
	//data := []byte { 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 180, 214, 110, 56, 24, 247, 42, 231, 157, 240, 245, 4, 137, 200, 210, 83, 186, 9, 22, 6, 156, 106, 186, 145, 150, 170, 180, 132, 214, 165, 38, 112, 90, 29, 44, 123, 160, 207, 130, 187, 41, 234, 74, 138, 1, 175, 124, 34, 172, 18, 16, 162, 60, 202, 248, 225, 41, 40, 49, 196, 25, 32, 96, 177, 0, 8, 28, 2, 2, 0, 66, 144, 2, 0, 69, 2, 130, 255, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 246, 127, 244, 34, 191, 22, 58, 85, 231, 255, 161, 102, 24, 181, 150, 185, 46, 22, 29, 12, 113, 230, 45, 65, 116, 115, 139, 207, 10, 79, 252, 29, 166, 67, 24, 132, 124, 18, 213, 229, 74, 251, 91, 83, 76, 252, 115, 231, 160, 178, 152, 179, 8, 109, 66, 49, 181, 60, 234, 43, 222, 170, 210, 8, 7, 0, 0, 0, 5, 0, 255, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 15, 0, 64, 243, 112, 131, 131, 24 }
	//data := []byte { 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 30, 176, 199, 124, 77, 145, 79, 239, 137, 109, 12, 125, 210, 58, 232, 103, 67, 18, 241, 201, 112, 8, 186, 160, 20, 44, 47, 107, 137, 91, 194, 51, 35, 0, 194, 124, 148, 240, 120, 185, 9, 125, 221, 60, 239, 53, 230, 222, 194, 58, 155, 226, 216, 144, 88, 125, 95, 231, 44, 3, 205, 141, 97, 67, 0, 16, 28, 2, 2, 0, 66, 144, 2, 0, 229, 6, 130, 255, 144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 172, 81, 238, 110, 14, 151, 46, 62, 83, 177, 13, 251, 117, 107, 248, 40, 60, 51, 1, 8, 80, 43, 160, 16, 80, 95, 156, 254, 48, 129, 54, 58, 61, 59, 33, 19, 159, 28, 63, 121, 179, 200, 39, 187, 12, 54, 46, 139, 105, 144, 112, 96, 125, 233, 205, 142, 174, 187, 140, 58, 12, 113, 70, 15, 7, 0, 0, 0, 15, 1, 65, 156, 53, 5, 0, 97, 115, 109, 1, 0, 0, 0, 1, 25, 4, 96, 7, 127, 127, 126, 127, 127, 127, 127, 1, 127, 96, 0, 1, 127, 96, 3, 127, 127, 127, 0, 96, 0, 0, 2, 77, 4, 3, 101, 110, 118, 8, 101, 120, 116, 95, 99, 97, 108, 108, 0, 0, 3, 101, 110, 118, 16, 101, 120, 116, 95, 115, 99, 114, 97, 116, 99, 104, 95, 115, 105, 122, 101, 0, 1, 3, 101, 110, 118, 16, 101, 120, 116, 95, 115, 99, 114, 97, 116, 99, 104, 95, 99, 111, 112, 121, 0, 2, 3, 101, 110, 118, 6, 109, 101, 109, 111, 114, 121, 2, 1, 1, 1, 3, 3, 2, 3, 3, 7, 17, 2, 6, 100, 101, 112, 108, 111, 121, 0, 3, 4, 99, 97, 108, 108, 0, 4, 10, 84, 2, 2, 0, 11, 79, 0, 2, 64, 65, 4, 16, 1, 71, 13, 0, 65, 0, 65, 0, 65, 4, 16, 2, 65, 0, 45, 0, 0, 65, 0, 71, 13, 0, 65, 1, 45, 0, 0, 65, 1, 71, 13, 0, 65, 2, 45, 0, 0, 65, 2, 71, 13, 0, 65, 3, 45, 0, 0, 65, 3, 71, 13, 0, 65, 4, 65, 32, 66, 0, 65, 36, 65, 16, 65, 0, 65, 0, 16, 0, 26, 15, 11, 0, 11, 11, 107, 2, 0, 65, 4, 11, 64, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 65, 36, 11, 32, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 2, 130, 255, 144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 106, 238, 32, 188, 148, 73, 130, 242, 186, 157, 242, 77, 245, 75, 44, 153, 64, 193, 22, 130, 29, 175, 73, 40, 167, 149, 112, 13, 75, 105, 222, 98, 253, 145, 43, 102, 189, 122, 101, 117, 24, 216, 156, 232, 159, 226, 84, 181, 218, 174, 150, 9, 99, 94, 243, 197, 111, 147, 177, 79, 57, 28, 29, 8, 7, 0, 4, 0, 15, 3, 11, 0, 64, 122, 16, 243, 90, 65, 156, 96, 251, 181, 150, 93, 112, 104, 147, 171, 117, 203, 219, 49, 15, 72, 172, 119, 90, 58, 14, 65, 36, 151, 216, 215, 112, 79, 101, 182, 252, 21, 176, 0, 69, 2, 130, 255, 144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 238, 111, 34, 29, 155, 229, 144, 97, 17, 170, 148, 246, 240, 120, 232, 124, 56, 217, 2, 24, 131, 4, 123, 114, 23, 58, 251, 48, 236, 159, 234, 19, 42, 63, 124, 165, 90, 198, 245, 144, 210, 64, 26, 13, 83, 142, 14, 51, 27, 103, 234, 155, 119, 59, 228, 15, 79, 24, 238, 116, 235, 243, 52, 13, 7, 0, 8, 0, 15, 2, 255, 186, 155, 147, 128, 164, 208, 171, 187, 223, 73, 87, 196, 7, 111, 210, 2, 243, 3, 228, 143, 133, 169, 184, 78, 199, 196, 50, 230, 134, 93, 106, 156, 40, 65, 156, 16, 0, 1, 2, 3}
	data := []byte { 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 188, 232, 252, 31, 104, 175, 96, 194, 246, 224, 70, 249, 137, 173, 81, 199, 90, 165, 244, 16, 178, 177, 205, 74, 28, 104, 219, 106, 162, 19, 223, 216, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0 }


	parentHash := []byte{0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45, 0x45}
	stateRoot := []byte{0xb3, 0x26, 0x6d, 0xe1, 0x37, 0xd2, 0x0a, 0x5d, 0x0f, 0xf3, 0xa, 0x64, 0x01e, 0xb5, 0x71, 0x27, 0x52, 0x5f, 0xd9, 0xb2, 0x69, 0x37, 0x01, 0xf0, 0xbf, 0x5a, 0x8a, 0x85, 0x3f, 0xa3, 0xeb, 0xe0}
	extrinsicsRoot := []byte{0x03, 0x17, 0x0a, 0x2e, 0x75, 0x97, 0xb7, 0xb7, 0xe3, 0xd8, 0x4c, 0x05, 0x39, 0x1d, 0x13, 0x9a, 0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0, 0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14}

	type headerStruct  struct {
		ParentHash []byte
		BlockNumber *big.Int
		StateRoot []byte
		ExtrinsicsRoot []byte
		Digest []byte
	}

	testHeader := headerStruct{parentHash,big.NewInt(1), stateRoot, extrinsicsRoot, []byte{}}
	t.Log("encode", "testStruct", testHeader)

	type blockStruct struct {
		Header headerStruct
		Extrinsics []byte
	}

	testBlock := blockStruct{testHeader, []byte{}}

	t.Log("encode", "testBlock", testBlock)
	buffer := bytes.Buffer{}

	encoder := codec.Encoder{ &buffer}
	bytesEncoded, err := encoder.Encode(testBlock)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("eh", "bytesEncoded", bytesEncoded)

	output := buffer.Bytes()
	t.Log("encoded header", "output", output)

	var offset int32 = 16
	//var offset int32 = 1126872
	var length int32 = int32(len(data))
	t.Log("length:", length)
	copy(mem[offset:offset+length], data)
	/*
		ret, err := r.Exec("Core_execute_block", offset, length)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("ret:", ret)
	*/
}



