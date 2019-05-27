package runtime

import (
	"path/filepath"
	"strings"
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
	t.Logf("Apis: %d\n", version.Apis)

	if strings.Compare(string(version.Spec_name), "polkadot") != 0 {
		t.Errorf("Fail when getting Core_version.spec_name: got %s expected %s", version.Spec_name, "polkadot")
	} else if strings.Compare(string(version.Impl_name), "parity-polkadot") != 0 {
		t.Errorf("Fail when getting Core_version.impl_name: got %s expected %s", version.Spec_name, "parity-polkadot")
	} else if version.Authoring_version != 1 {
		t.Errorf("Fail when getting Core_version.authoring_version: got %d expected %d", version.Authoring_version, 1)
	}
}
