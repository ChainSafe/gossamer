package modules

import (
	api "github.com/ChainSafe/gossamer/internal"
	"testing"
)

var (
	testRuntimeVersion = "1.2.3"
)
type mockruntimeApi struct {}


func (a *mockruntimeApi) Version() string {
	return testRuntimeVersion
}

func newMockApi() *api.Api {
	runtimeApi := &mockruntimeApi{}

	return &api.Api{
		Core: api.NewCoreModule(nil, runtimeApi),
	}
}

func TestCoreModule_Version(t *testing.T) {
	core := NewCoreModule(newMockApi())

	vres := &CoreVersionResponse{}
	err := core.Version(nil, nil, vres)
	if err != nil {
		t.Fatal(err)
	}
	if vres.Version != testRuntimeVersion {
		t.Fatalf("Core.Version: expected: %s got: %s\n", vres.Version, testRuntimeVersion)
	}
}
