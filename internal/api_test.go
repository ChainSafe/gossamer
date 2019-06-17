package api

import "testing"

const (
	TestPeerCount = 1337
	TestVersion = "1.2.3"
)

// -------------- Mock Apis ------------------
type MockP2pApi struct {}

func (a *MockP2pApi) PeerCount() int {
	return TestPeerCount
}

type MockRuntimeApi struct {}

func (a *MockRuntimeApi) Version() string {
	return TestVersion
}
// -------------------------------------------

func newApiService() *Service {
	return NewApiService(&MockP2pApi{}, &MockRuntimeApi{})
}

func TestCoreModule(t *testing.T) {
	srvc := newApiService()

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