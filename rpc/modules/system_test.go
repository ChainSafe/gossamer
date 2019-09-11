package modules

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/api"
)

var (
	testRuntimeVersion = "1.2.3"
	testRuntimeName    = "Gossamer"
	testPeerId         = "Qmc85Ephxa3sR7xaTzTq2UpCJ4a4HWAfxxaV6TarXHWVVh"
	peers              = []string{"QmeQeqpf3fz3CG2ckQq3CUWwUnyT2cqxJepHpjji7ehVtX", "AbCDeqpf3fz3CG2ckQq3CUWwUnyT2cqxJepHpjji7ehVtX"}
	isSyncing          = false
)

type mockruntimeApi struct{}
type mockP2PApi struct{}

//Mock runtime API
func (a *mockruntimeApi) Version() string {
	return testRuntimeVersion
}

func (a *mockruntimeApi) Name() string {
	return testRuntimeName
}

//Mock p2p API
func (a *mockP2PApi) PeerCount() int {
	return len(peers)
}

func (a *mockP2PApi) Peers() []string {
	return peers
}

func (a *mockP2PApi) ShouldHavePeers() bool {
	return (len(peers) != 0)
}

func (a *mockP2PApi) ID() string {
	return testPeerId
}

func newMockApi() *api.Api {
	runtimeApi := &mockruntimeApi{}
	p2pApi := &mockP2PApi{}

	return &api.Api{
		P2pSystem: api.NewP2PModule(p2pApi),
		RtSystem:  api.NewRTModule(runtimeApi),
	}
}

func TestSystemModule(t *testing.T) {
	sys := NewSystemModule(newMockApi())

	//Test RPC's System.Peers() response
	peersRes := &SystemPeersResponse{}
	sys.Peers(nil, nil, peersRes)

	//Loop through each peer in input & RPC response
	//Check if arrays are equal
	equalPeers := true
	for i, originalPeer := range peers {
		if originalPeer != peersRes.Peers[i] {
			equalPeers = false
		}
	}

	if len(peers) != len(peersRes.Peers) {
		equalPeers = false
	}

	if equalPeers == false {
		t.Errorf("System.Peers: expected: %+v got: %+v\n", peers, *peersRes)
	}

	//Test RPC's System.NetworkState() response
	netState := &SystemNetworkStateResponse{}
	sys.NetworkState(nil, nil, netState)

	if netState.Id != testPeerId {
		t.Errorf("System.NetworkState: expected: %+v got: %+v\n", testPeerId, netState.Id)
	}

	//Test RPC's System.Health() response
	netHealth := &SystemHealthResponse{}
	sys.Health(nil, nil, netHealth)
	expectedHealth := &SystemHealthResponse{Peers: len(peers), IsSyncing: isSyncing, ShouldHavePeers: (peers != nil)}

	if netHealth.Peers != expectedHealth.Peers {
		t.Errorf("System.Health.Peers: expected: %+v got: %+v\n", netHealth.Peers, expectedHealth.Peers)
	}

	if netHealth.IsSyncing != expectedHealth.IsSyncing {
		t.Errorf("System.Health.IsSyncing: expected: %+v got: %+v\n", netHealth.IsSyncing, expectedHealth.IsSyncing)
	}

	if netHealth.ShouldHavePeers != expectedHealth.ShouldHavePeers {
		t.Errorf("System.Health.ShouldHavePeers: expected: %+v got: %+v\n", netHealth.ShouldHavePeers, expectedHealth.ShouldHavePeers)
	}

}
