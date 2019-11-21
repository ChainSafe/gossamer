package p2p

import (
	"os"
	"path"
	"reflect"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gossamer-test")

	defer os.RemoveAll(testDir)

	keyA, err := generateKey(0, testDir)
	if err != nil {
		t.Fatal(err)
	}

	keyB, err := generateKey(0, testDir)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(keyA, keyB) {
		t.Error("Generated keys should not match")
	}

	keyC, err := generateKey(1, testDir)
	if err != nil {
		t.Fatal(err)
	}

	keyD, err := generateKey(1, testDir)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(keyC, keyD) {
		t.Error("Generated keys should match")
	}
}

func TestSetupPrivateKey(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gossamer-test")

	defer os.RemoveAll(testDir)

	configA := &Config{
		DataDir: testDir,
	}

	err := configA.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	configB := &Config{
		DataDir: testDir,
	}

	err = configB.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(configA.privateKey, configB.privateKey) {
		t.Error("Private keys should match")
	}

	configC := &Config{
		RandSeed: 1,
	}

	err = configC.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	configD := &Config{
		RandSeed: 2,
	}

	err = configD.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(configC.privateKey, configD.privateKey) {
		t.Error("Private keys should not match")
	}
}

func TestBuildOptions(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gossamer-test")

	defer os.RemoveAll(testDir)

	configA := &Config{
		DataDir: testDir,
	}

	_, err := configA.buildOpts()
	if err != nil {
		t.Fatal(err)
	}

	if configA.BootstrapNodes != nil {
		t.Error("BootstrapNodes should be nil")
	}

	if configA.ProtocolId != "" {
		t.Error("ProtocolId should be an empty string")
	}

	if configA.Port != 0 {
		t.Error("Port should be 0")
	}

	if configA.RandSeed != 0 {
		t.Error("RandSeed should be 0")
	}

	if configA.NoBootstrap != false {
		t.Error("NoBootstrap should be false")
	}

	if configA.NoMdns != false {
		t.Error("NoMdns should be false")
	}

	if configA.DataDir != testDir {
		t.Errorf("DataDir should be %s", testDir)
	}

	if configA.privateKey == nil {
		t.Error("pivateKey should defined")
	}
}
