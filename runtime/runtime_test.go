package runtime

import (
	//"bytes"
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	trie "github.com/ChainSafe/gossamer/trie"
	"golang.org/x/crypto/ed25519"
	//"github.com/ChainSafe/gossamer/polkadb"
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

func newEmpty() *trie.Trie {
	db := &trie.Database{
		//db: polkadb.NewMemDatabase(),
	}
	t := trie.NewEmptyTrie(db)
	return t
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

	tt := newEmpty()

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
		Spec_version:      109,
		Impl_version:      0,
	}

	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	res, err := r.Exec("Core_version", 0, 0)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
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

func TestExecAuthorities(t *testing.T) {
	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	pubkey, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey1, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey2, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey3, _, err := ed25519.GenerateKey(rand.Reader)

	authLen, err := scale.Encode(int64(1))
	if err != nil {
		t.Fatal(err)
	}

	r.t.Put([]byte(":auth:len"), authLen)
	r.t.Put(append([]byte(":auth:"), []byte{0, 0, 0, 0}...), []byte(pubkey))
	r.t.Put(append([]byte(":auth:"), []byte{1, 0, 0, 0}...), []byte(pubkey1))
	r.t.Put(append([]byte(":auth:"), []byte{2, 0, 0, 0}...), []byte(pubkey2))
	r.t.Put(append([]byte(":auth:"), []byte{3, 0, 0, 0}...), []byte(pubkey3))

	var offset int64 = 1
	var length int64 = 1
	copy(r.vm.Memory[offset:offset+length], []byte{1})
	res, err := r.Exec("AuthoritiesApi_authorities", offset, length)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Logf("%v\n", res)
}

func TestExecInitializeBlock(t *testing.T) {
	ph, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatalf("Fail when decoding parent hash: %s", err)
	}
	sr, err := common.HexToHash("0x89d0e979afb54e4ba041e942c252fefd83b94b4c8e71821bdf663347fe169eaa")
	if err != nil {
		t.Fatalf("Fail when decoding state root: %s", err)
	}
	er, err := common.HexToHash("0xf6ae75ee1f0895eebee8bc19f5b68fea145ffee1102d00c83950e5a70f907490")
	if err != nil {
		t.Fatalf("Fail when decoding extrinsics root: %s", err)
	}

	header := &common.BlockHeader{
		ParentHash: ph,
		Number: big.NewInt(1),
		StateRoot: sr,
		ExtrinsicsRoot: er,
		Digest: nil,
	}
 
	encHeader, err := scale.Encode(header)
	if err != nil {
		t.Fatalf("Fail: could not encode header: %s", err)
	}

	encHeader = encHeader[:]
	t.Logf("%x", encHeader)

	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	var offset int64 = 16
	var length int64 = int64(len(encHeader))
	copy(r.vm.Memory[offset:offset+length], encHeader)

	res, err := r.Exec("Core_initialize_block", offset, length)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Logf("%v\n", res)
}


func TestExecActiveParachains(t *testing.T) {
	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	pubkey, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey1, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey2, _, err := ed25519.GenerateKey(rand.Reader)
	pubkey3, _, err := ed25519.GenerateKey(rand.Reader)

	authLen, err := scale.Encode(int64(1))
	if err != nil {
		t.Fatal(err)
	}

	r.t.Put([]byte(":auth:len"), authLen)
	r.t.Put(append([]byte(":auth:"), []byte{0, 0, 0, 0}...), []byte(pubkey))
	r.t.Put(append([]byte(":auth:"), []byte{1, 0, 0, 0}...), []byte(pubkey1))
	r.t.Put(append([]byte(":auth:"), []byte{2, 0, 0, 0}...), []byte(pubkey2))
	r.t.Put(append([]byte(":auth:"), []byte{3, 0, 0, 0}...), []byte(pubkey3))


	res, err := r.Exec("ParachainHost_active_parachains", 0, 0)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Logf("%v", res)
	// version := res.(*Version)
	// t.Logf("Spec_name: %s\n", version.Spec_name)
	// t.Logf("Impl_name: %s\n", version.Impl_name)
	// t.Logf("Authoring_version: %d\n", version.Authoring_version)
	// t.Logf("Spec_version: %d\n", version.Spec_version)
	// t.Logf("Impl_version: %d\n", version.Impl_version)

	// if !reflect.DeepEqual(version, expected) {
	// 	t.Errorf("Fail: got %v expected %v\n", version, expected)
	// }
}