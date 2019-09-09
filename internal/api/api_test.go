package api

import (
	"testing"

	peer "github.com/libp2p/go-libp2p-peer"
)

// -------------- Mock Apis ------------------
var (
	TestPeerCount   = 1
	TestVersion     = "0.0.1"
	Name            = "Gossamer"
	peerID          = "Qmc85Ephxa3sR7xaTzTq2UpCJ4a4HWAfxxaV6TarXHWVVh"
	isSyncing       = false
	ShouldHavePeers = false
	Peers           = int(len(peers))
	peers           = []peer.ID{"QmeQeqpf3fz3CG2ckQq3CUWwUnyT2cqxJepHpjji7ehVtX"}
)

// var

// Creating a mock peer
type MockP2pApi struct{}

func (a *MockP2pApi) PeerCount() int {
	return TestPeerCount
}

func (a *MockP2pApi) Peers() []peer.ID {
	return peers
}

func (b *MockP2pApi) NetworkState() string {
	return peerID
}

func (b *MockP2pApi) ShouldHavePeers() bool {
	return ShouldHavePeers
}

// Creating a mock runtime API
type MockRuntimeApi struct{}

func (a *MockRuntimeApi) Name() string {
	//TODO: Replace with dynamic name
	return Name
}

func (a *MockRuntimeApi) Version() string {
	return TestVersion
}

// func (a *MockRuntimeApi) Chain() string {
// 	return Chain
// }

// // System properties not implemented yet
// func (b *MockRuntimeApi) properties() string {
// 	return properties
// }

// -------------------------------------------

func TestSystemModule(t *testing.T) {
	srvc := NewApiService(&MockP2pApi{}, &MockRuntimeApi{})

	// // System.Health
	// h := srvc.Api.System.Health()
	// if h != health {
	// 	t.Fatalf("System.Health - expected %+v got: %+v\n", health, h)
	// }

	// System.Name
	n := srvc.Api.System.Name()
	if n != Name {
		t.Fatalf("System.Name - expected %+v got: %+v\n", Name, n)
	}

	// System.networkState
	s := srvc.Api.System.NetworkState()
	if s != peerID {
		t.Fatalf("System.NetworkState - expected %+v got: %+v\n", peerID, s)
	}

	// System.peers
	p := srvc.Api.System.Peers()
	if s != peerID {
		t.Fatalf("System.NetworkState - expected %+v got: %+v\n", peers, p)
	}

	// System.PeerCount
	c := srvc.Api.System.PeerCount()
	if c != TestPeerCount {
		t.Fatalf("System.PeerCount - expected: %d got: %d\n", TestPeerCount, c)
	}

	// System.Version
	v := srvc.Api.System.Version()
	if v != TestVersion {
		t.Fatalf("System.Version - expected: %s got: %s\n", TestVersion, v)
	}
}
