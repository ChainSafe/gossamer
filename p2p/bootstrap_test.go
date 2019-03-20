package p2p

import (
	"testing"
)

var testServiceConfig = &ServiceConfig{
	BootstrapNodes: []string{
		GetLocalPeerInfo(),
	},
	Port: 7001,
}

// list of IPFS peers, eventually change this to polkadot bootstrap nodes
var (
	IPFS_PEERS = []string{
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
		"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
)

func TestStringToPeerInfo(t *testing.T) {
	for _, str := range IPFS_PEERS {
		pi, err := stringToPeerInfo(str)
		if err != nil {
			t.Error(err)
		} 

		if pi.ID.Pretty() != str[len(str)-46:] {
			t.Errorf("StringToPeerInfo error: got %s expected %s", pi.ID.Pretty(), str)
		}
	}
}

func TestStringsToPeerInfos(t *testing.T) {
	for _, str := range IPFS_PEERS {
		pi, err := stringToPeerInfo(str)
		if err != nil {
			t.Error(err)
		}

		if pi.ID.Pretty() != str[len(str)-46:] {
			t.Errorf("StringToPeerInfo error: got %s expected %s", pi.ID.Pretty(), str)
		}
	}
}

func TestBootstrapConnect(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	err = s.bootstrapConnect()
	if err != nil {
		t.Errorf("Start error :%s", err)
	}
}

