package runtime

import (
	"bytes"
	"crypto/rand"
	"io"
	//"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	scale "github.com/ChainSafe/gossamer/codec"
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

	fp, err := filepath.Abs("./polkadot_runtime.compact.wasm")
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

func encode(in interface{}) ([]byte, error) {
	buffer := bytes.Buffer{}
	se := scale.Encoder{&buffer}
	_, err := se.Encode(in)
	output := buffer.Bytes()
	return output, err
}

func TestNewVM(t *testing.T) {
	_, err := newRuntime(t)
	if err != nil {
		t.Errorf("Fail: could not create new runtime: %s", err)
	}
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

	authLen, err := encode(int64(1))
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
	res, err := r.Exec("Core_authorities", offset, length)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Logf("%v\n", res)
}

func TestExecInitializeBlock(t *testing.T) {
	// ph, err := common.HexToHash("0x8550326cee1e1b768a254095b412e0db58523c2b5df9b7d2540b4513d475ce7f")
	// if err != nil {
	// 	t.Fatalf("Fail when decoding parent hash: %s", err)
	// }
	// sr, err := common.HexToHash("0x1d9d01423a90032ac600d1e2ff0a54634760d0ae0941cfab855c69bef38689d2")
	// if err != nil {
	// 	t.Fatalf("Fail when decoding state root: %s", err)
	// }	
	// er, err := common.HexToHash("0x118a02e06882254b1d24417d4df4dca6a7b8754e42f5b24419f7170a0de6d027")
	// if err != nil {
	// 	t.Fatalf("Fail when decoding extrinsics root: %s", err)
	// }

	// header := &common.BlockHeader{
	// 	ParentHash: ph,
	// 	Number: big.NewInt(1570578),
	// 	StateRoot: sr,
	// 	ExtrinsicsRoot: er,
	// 	Digest: []byte{},
	// }

	// encHeader, err := scale.Encode(*header)
	// if err != nil {
	// 	t.Fatalf("Fail: could not encode header: %s", err)
	// }

	encHeader, err := common.HexToBytes("0x9aa25e4c67a8a7e1d77572e4c3b97ca8110df952cfc3d345cec5e88cb1e3a96f01dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0489d0e979afb54e4ba041e942c252fefd83b94b4c8e71821bdf663347fe169eaaf6ae75ee1f0895eebee8bc19f5b68fea145ffee1102d00c83950e5a70f907490040330295a0f000000007ea5fce566f9fd4ec6eb49208b420d654a219e4fc3d56caff6ba8c3a7df6fbba950c4b867a8177adac98dc4e37c8c631ceb63fdbe4a8e51fbc1968413277b00c0108200100000320f71c5c10010b0000000000")
	if err != nil {
		t.Fatal(err)
	}

	r, err := newRuntime(t)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}


	var offset int64 = 1049000
	var length int64 = int64(len(encHeader))
	copy(r.vm.Memory[offset:offset+length], encHeader)

	//fr := r.vm.GetCurrentFrame()
	//copy(fr.Locals, []int64{offset, length})
	//r.vm.Ignite(344, offset, length)

	res, err := r.Exec("Core_initialise_block", offset, length)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Logf("%v\n", res)
}