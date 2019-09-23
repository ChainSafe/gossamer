package manager

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../../polkadot_runtime.wasm"

func newRuntime(t *testing.T) *runtime.Runtime {
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

func TestNewService_Start(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	p2pcfg := &p2p.Config{
		BootstrapNodes: []string{},
		Port:           7001,
	}
	p, err := p2p.NewService(p2pcfg)
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewService(rt, b, p.MsgChan())
	e := mgr.Start()
	err = <-e
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTransaction(t *testing.T) {
	rt := newRuntime(t)
	mgr := NewService(rt, nil, make(chan p2p.Message))
	ext := []byte{0}
	validity, err := mgr.validateTransaction(ext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(validity)
}

func TestProcessTransaction(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	mgr := NewService(rt, b, make(chan p2p.Message))
	ext := []byte{0}
	err := mgr.ProcessTransaction(ext)
	if err != nil {
		t.Fatal(err)
	}
}
