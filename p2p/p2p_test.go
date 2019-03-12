package p2p

import (
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

var testServiceConfig = &ServiceConfig{
	BootstrapNode: "/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	Port:          7001,
}

func TestStart(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	err = s.Start()
	if err != nil {
		t.Errorf("Start error :%s", err)
	}
}

func TestStartDHT(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	err = s.startDHT()
	if err != nil {
		t.Errorf("TestStartDHT error: %s", err)
	}
}

func TestBuildOpts(t *testing.T) {
	_, err := testServiceConfig.buildOpts()
	if err != nil {
		t.Fatalf("TestBuildOpts error: %s", err)
	}
}


func TestGenerateKey(t *testing.T) {
	privA, err := generateKey(7777)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	privB, err := generateKey(7777)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	t.Log(privA)
	t.Log(privB)
	if !crypto.KeyEqual(privA, privB) {
		t.Error("GenerateKey error: did not create same key for same seed")
	} 

	privC, err := generateKey(0)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	if crypto.KeyEqual(privA, privC) {
		t.Fatal("GenerateKey error: created same key for different seed")
	} 
}