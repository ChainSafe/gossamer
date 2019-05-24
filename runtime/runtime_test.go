package runtime

import (
	"path/filepath"
	"testing"
)

func TestExec(t *testing.T) {
	fp, err := filepath.Abs("./polkadot_runtime.compact.wasm")
	if err != nil {
		t.Fatal("could not create filepath")
	}

	res, err := Exec(fp)
	if err != nil {
		t.Fatalf("could not exec wasm runtime: %s", err)
	}

	t.Log(res)
}
