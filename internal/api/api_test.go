package api

import "testing"

// -------------- Mock Apis ------------------
const (
	TestPeerCount = 1337
	TestVersion   = "1.2.3"
	Chain         = "blah"
	Health        = "blah"
	Name          = "ba"
	networkState  = "blah"
	peers         = "blah"
	properties    = "blah"
)

type MockP2pApi struct{}

func (a *MockP2pApi) PeerCount() int {
	return TestPeerCount
}

type MockRuntimeApi struct{}

func (a *MockRuntimeApi) Version() string {
	return TestVersion
}

func (a *MockRuntimeApi) Chain() string {
	return Chain
}

func (a *MockRuntimeApi) Health() string {
	return Health
}

// func (a *MockRuntimeApi) Name() string {
// 	return Name
// }

func (a *MockRuntimeApi) networkState() string {
	return networkState
}

func (a *MockRuntimeApi) peers() string {
	return peers
}

func (a *MockRuntimeApi) properties() string {
	return properties
}

// -------------------------------------------

func TestSystemModule(t *testing.T) {
	srvc := NewApiService(&MockP2pApi{}, &MockRuntimeApi{})

	// System.PeerCount
	c := srvc.Api.System.PeerCount()
	if c != TestPeerCount {
		t.Fatalf("System.PeerCount - expected: %d got: %d\n", TestPeerCount, c)
	}

	// System.Version
	v := srvc.Api.System.version()
	if v != TestVersion {
		t.Fatalf("System.Version - expected: %s got: %s\n", TestVersion, v)
	}
}
