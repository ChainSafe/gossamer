package runtime

import (
	"path/filepath"
	"testing"
)

func TestNewVM(t *testing.T) {
	fp, err := filepath.Abs("./polkadot_runtime.compact.wasm")
	if err != nil {
		t.Fatal("could not create filepath")
	}

	r, err := NewRuntime(fp)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}
}


func TestExec(t *testing.T) {
	fp, err := filepath.Abs("./polkadot_runtime.compact.wasm")
	if err != nil {
		t.Fatal("could not create filepath")
	}

	r, err := NewRuntime(fp)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	res, err := r.Exec()
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Log(res)
}
