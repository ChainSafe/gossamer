package p2p

import (
	"os"
	"path"
	"testing"
)

var testDir = path.Join(os.TempDir(), "gossamer-test")

func TestBuildOptions(t *testing.T) {
	configA := &Config{
		DataDir: testDir,
	}

	_, err := configA.buildOpts()
	if err != nil {
		t.Fatal(err)
	}

	if configA.privateKey == nil {
		t.Error("Private key was not set.")
	}

	configB := &Config{
		DataDir: testDir,
	}

	_, err = configB.buildOpts()
	if err != nil {
		t.Fatal(err)
	}

	if configA.privateKey == configB.privateKey {
		t.Error("Private keys should not match.")
	}
}

func TestSetupPrivKey(t *testing.T) {
	configA := &Config{
		BootstrapNodes: nil,
		Port:           0,
		RandSeed:       0,
		NoBootstrap:    true,
		NoMdns:         true,
		DataDir:        testDir,
		privateKey:     nil,
	}

	err := configA.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	// Load private key
	configB := &(*configA)
	configB.privateKey = nil

	err = configB.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	if !configA.privateKey.Equals(configB.privateKey) {
		t.Errorf("keys don't match. publicA: %s publicB: %s", configA.privateKey.GetPublic(), configB.privateKey.GetPublic())
	}
}
