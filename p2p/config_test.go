package p2p

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSetupPrivKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "gossamer-test")
	if err != nil {
		t.Fatal(err)
	}
	config := &Config{
		BootstrapNodes: nil,
		Port:           0,
		RandSeed:       0,
		NoBootstrap:    true,
		NoMdns:         true,
		DataDir:        tmpDir,
		privateKey:     nil,
	}

	err = config.setupPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	// Load private key
}
