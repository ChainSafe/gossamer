package modules

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/api"
	peer "github.com/libp2p/go-libp2p-peer"
)

var (
	testRuntimeVersion = "1.2.3"
	testRuntimeName    = "Gossamer"
	testNetworkState   = "Qmc85Ephxa3sR7xaTzTq2UpCJ4a4HWAfxxaV6TarXHWVVh"
	peers              = []peer.ID{"QmeQeqpf3fz3CG2ckQq3CUWwUnyT2cqxJepHpjji7ehVtX", "AbCDeqpf3fz3CG2ckQq3CUWwUnyT2cqxJepHpjji7ehVtX"}
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

func (a *mockP2PApi) Peers() []peer.ID {
	return peers
}

func (a *mockP2PApi) ShouldHavePeers() bool {
	return (peers != nil)
}

func (a *mockP2PApi) NetworkState() string {
	return testNetworkState
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
	equalPeers := true
	for _, peerOriginal := range peers {
		found := false
		for _, peerResponse := range *peersRes {
			//If we found matching peers in both arrays
			if peerOriginal == peerResponse {
				found = true
			}
		}
		//If we dont find matching
		if found == false {
			equalPeers = false
		}
	}

	if len(peers) != len(*peersRes) {
		equalPeers = false
	}

	if equalPeers == false {
		t.Fatalf("System.Peers: expected: %+v got: %+v\n", peers, *peersRes)
	}

	//Test RPC's System.NetworkState() response
	netState := &SystemNetworkStateResponse{}
	sys.NetworkState(nil, nil, netState)

	if netState.PeerId != testNetworkState {
		t.Fatalf("System.NetworkState: expected: %+v got: %+v\n", testNetworkState, netState.PeerId)
	}

	//Test RPC's System.Health() response
	netHealth := &SystemHealthResponse{}
	sys.Health(nil, nil, netHealth)
	expectedHealth := &SystemHealthResponse{Peers: len(peers), IsSyncing: isSyncing, ShouldHavePeers: (peers != nil)}

	if (*netHealth).Peers != expectedHealth.Peers || (*netHealth).IsSyncing != expectedHealth.IsSyncing || (*netHealth).ShouldHavePeers != expectedHealth.ShouldHavePeers {

		t.Fatalf("System.Health: expected: %+v got: %+v\n", (*netHealth), expectedHealth)
	}

}
