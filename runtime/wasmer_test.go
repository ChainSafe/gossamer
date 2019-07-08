package runtime

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	trie "github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "polkadot_runtime.compact.wasm"

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

	resp, err := http.Get("https://github.com/w3f/polkadot-re-tests/blob/master/polkadot-runtime/polkadot_runtime.compact.wasm?raw=true")
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

const TESTS_FP string = "./test_wasm/target/wasm32-unknown-unknown/release/test_wasm.wasm"

func newTestRuntime() (*Runtime, error) {
	t := &trie.Trie{}
	fp, err := filepath.Abs(TESTS_FP)
	if err != nil {
		return nil, err
	}
  	return NewRuntime(fp, t)
}

func TestExt_print_utf8(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	testFunc, ok := runtime.vm.Exports["test_ext_print_utf8"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	_, err = testFunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestExt_get_storage_into(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	key := []byte(":noot")
	value := []byte{1,3,3,7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	keyData := 170
	valueData := 200
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
}

func TestExt_set_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()

	key := []byte(":noot")
	value := []byte{1,3,3,7}

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

	trieValue, err := runtime.trie.Get(key)
	if err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(value, trieValue) {
		t.Error("did not store correct value in storage trie")
	}

	t.Log(trieValue)
}

func TestExt_storage_root(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
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

func TestExt_get_allocated_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	key := []byte(":noot")
	value := []byte{1,3,3,7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

	keyData := 170
	copy(mem[keyData:keyData+len(key)], key)
	var writtenOut int32 = 169

	testFunc, ok := runtime.vm.Exports["test_ext_get_allocated_storage"]
	if !ok {
		t.Fatal("could not find exported function")
	}

	ret, err := testFunc(keyData, len(key), writtenOut)
	if err != nil {
		t.Fatal(err)
	} 

	retInt := ret.ToI32()
	length := int32(mem[writtenOut])
	if !bytes.Equal(mem[retInt:retInt+length], value) {
		t.Error("did not save value to memory")
	}
}


func TestExt_clear_storage(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	mem := runtime.vm.Memory.Data()
	key := []byte(":noot")
	value := []byte{1,3,3,7}
	err = runtime.trie.Put(key, value)
	if err != nil {
		t.Fatal(err)
	}

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

	ret, err := runtime.trie.Get(key)
	if err != nil {
		t.Fatal(err)
	} else if ret != nil {
		t.Error("did not delete key from storage trie")
	}
}