package p2p

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../runtime/polkadot_runtime.compact.wasm"
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

func newRuntime(t *testing.T) (*runtime.Runtime, error) {
	_, err := getRuntimeBlob()
	if err != nil {
		t.Fatalf("Fail: could not get polkadot runtime")
	}

	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	tt := &trie.Trie{}

	r, err := runtime.NewRuntime(&runtime.Config{Path: fp}, tt)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r, err
}

func TestP2pRuntime(t *testing.T) {
	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	}

	testServiceConfigA := &Config{
		NoBootstrap: true,
		Port:        7001,
		Runtime:     r,
	}

	sa, err := NewService(testServiceConfigA)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	expected := &runtime.Version{
		Spec_name:         []byte("polkadot"),
		Impl_name:         []byte("parity-polkadot"),
		Authoring_version: 1,
		Spec_version:      1000,
		Impl_version:      0,
	}

	ret, err := sa.Runtime().Exec("Core_version", 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ret)

	res, err := runtime.DecodeToInterface(ret, &runtime.Version{})
	if err != nil {
		t.Fatal(err)
	}

	version := res.(*runtime.Version)
	t.Logf("Spec_name: %s\n", version.Spec_name)
	t.Logf("Impl_name: %s\n", version.Impl_name)
	t.Logf("Authoring_version: %d\n", version.Authoring_version)
	t.Logf("Spec_version: %d\n", version.Spec_version)
	t.Logf("Impl_version: %d\n", version.Impl_version)

	if !reflect.DeepEqual(version, expected) {
		t.Errorf("Fail: got %v expected %v\n", version, expected)
	}
}
