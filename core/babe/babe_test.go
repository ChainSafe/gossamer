package babe

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../../polkadot_runtime.wasm"

func newRuntime(t *testing.T) (*runtime.Runtime) {
	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	tt := &trie.Trie{}

	r, err := runtime.NewRuntime(fp, tt)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r
}

func TestStartupData(t *testing.T) {
	rt := newRuntime(t)
	babesession := NewBabeSession([32]byte{}, [64]byte{}, rt)
	res, err := babesession.startupData()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}
