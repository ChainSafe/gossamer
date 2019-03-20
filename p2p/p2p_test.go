package p2p

import (
	"context"
	"fmt"
	"testing"

	ipfs "github.com/ipfs/go-ipfs/core"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func StartIpfsNode() (*ipfs.IpfsNode, error) {
	cfg := &ipfs.BuildCfg{}
	node, err := ipfs.NewNode(context.Background(), cfg)
	return node, err
}

func TestBuildOpts(t *testing.T) {
	testServiceConfig := &ServiceConfig{
		BootstrapNodes: []string{
		},
		Port: 7001,
	}

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
	ipfsNode, err := StartIpfsNode()
	if err != nil {
		t.Fatalf("Could not start IPFS node: %s", err)
	}

	ipfsAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/ipfs/%s", ipfsNode.Identity.String())

	t.Log("ipfsAddr:", ipfsAddr)
	testServiceConfig := &ServiceConfig{
		BootstrapNodes: []string{
			ipfsAddr,
		},
		Port: 7001,
	}

	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := s.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error: %s", err)
	}
}

func TestSend(t *testing.T) {
	ipfsNode, err := StartIpfsNode()
	if err != nil {
		t.Fatalf("Could not start IPFS node: %s", err)
	}

	ipfsAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/ipfs/%s", ipfsNode.Identity.String())

	testServiceConfigA := &ServiceConfig{
		BootstrapNodes: []string{
			ipfsAddr,
		},
		Port: 7001,
	}

	testServiceConfigB := &ServiceConfig{
		BootstrapNodes: []string{
			ipfsAddr,
		},
		Port: 7001,
	}

	sa, err := NewService(testServiceConfigA)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e := sa.Start()
	if <-e != nil {
		t.Errorf("Start error: %s", err)
	}

	sb, err := NewService(testServiceConfigB)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	e = sb.Start()
	if <-e != nil {
		t.Errorf("Start error: %s", err)
	}

	peer, err := sa.dht.FindPeer(sa.ctx,  sb.host.ID())
	if err != nil {
		t.Fatalf("could not find peer: %s", err)
	}

	t.Log(peer)
	msg := []byte("hello there")
	err = sa.Send(peer, msg)
	if err != nil {
		t.Errorf("Send error: %s", err)
	}
}
