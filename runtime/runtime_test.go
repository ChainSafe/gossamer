package runtime

import (
	"path/filepath"
	"reflect"
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


func TestExecVersion(t *testing.T) {
	expected := &Version{
		Spec_name: []byte("polkadot"),
		Impl_name: []byte("parity-polkadot"),
		Authoring_version: 1,
		Spec_version: 109,
		Impl_version: 0,
	}

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

	res, err := r.Exec("Core_version")
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
