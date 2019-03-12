package p2p

import (
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

var testServiceConfig = &ServiceConfig{
	BootstrapNodes: []string{
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	},
	Port: 7001,
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

func TestStart(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := s.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error :%s", err)
	}
}

func TestSend(t *testing.T) {
	sa, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := sa.Start()
	if <-e != nil {
		t.Errorf("Start error: %s", err)
	}

	sb, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e = sb.Start()
	if <-e != nil {
		t.Errorf("Start error: %s", err)
	}

	peer := sb.host.ID()
	t.Log(peer.Pretty())
	msg := []byte("hello there")
	err = sa.Send(peer, msg)
	if err != nil {
		t.Errorf("Send error: %s", err)
	}
}