package api

import "testing"

// -------------- Mock Apis ------------------
const (
	TestPeerCount = 1337
	TestVersion = "1.2.3"
)

type MockP2pApi struct {}

func (a *MockP2pApi) PeerCount() int {
	return TestPeerCount
}

type MockRuntimeApi struct {}

func (a *MockRuntimeApi) Version() string {
	return TestVersion
}
// -------------------------------------------

func TestCoreModule(t *testing.T) {
	srvc := NewApiService(&MockP2pApi{}, &MockRuntimeApi{})

	// Core.PeerCount
	c := srvc.Api.Core.PeerCount()
	if c != TestPeerCount {
		t.Fatalf("Core.PeerCount - expected: %d got: %d\n", TestPeerCount, c)
	}

	// Core.Version
	v := srvc.Api.Core.Version()
	if v != TestVersion {
		t.Fatalf("Core.Version - expected: %s got: %s\n", TestVersion, v)
	}
}