package p2p

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/state"
)

// test buildIdentity method
func TestBuildIdentity(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gossamer-test")
	defer os.RemoveAll(testDir)

	configA := &Config{
		DataDir: testDir,
	}

	err := configA.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	configB := &Config{
		DataDir: testDir,
	}

	err = configB.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(configA.privateKey, configB.privateKey) {
		t.Error("Private keys should match")
	}

	configC := &Config{
		RandSeed: 1,
	}

	err = configC.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	configD := &Config{
		RandSeed: 2,
	}

	err = configD.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(configC.privateKey, configD.privateKey) {
		t.Error("Private keys should not match")
	}
}

// test build configuration method
func TestBuild(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gossamer-test")
	defer os.RemoveAll(testDir)

	testRoles := byte(1) // full node

	cfg := &Config{
		BlockState:   &state.BlockState{},
		NetworkState: &state.NetworkState{},
		StorageState: &state.StorageState{},
		DataDir:      testDir,
		Roles:        testRoles,
	}

	err := cfg.build()
	if err != nil {
		t.Fatal(err)
	}

	testKey, err := generateKey(0, testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		BlockState:   &state.BlockState{},
		NetworkState: &state.NetworkState{},
		StorageState: &state.StorageState{},
		DataDir:      testDir,
		Roles:        testRoles,
		Port:         DefaultPort,
		RandSeed:     DefaultRandSeed,
		Bootnodes:    DefaultBootnodes,
		ProtocolID:   DefaultProtocolID,
		NoBootstrap:  false,
		NoMdns:       false,
		privateKey:   testKey,
	}

	if reflect.DeepEqual(cfg, expected) {
		t.Error("Configurations should the same")
	}
}
