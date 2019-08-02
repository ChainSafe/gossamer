package runtime

import "testing"

func TestAllocator(t *testing.T) {
	t.Log("testing Allocator")
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	alloc_res, err := fbha.allocate(16 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("next:",3 , nextPowerOf2GT8( 15))
	t.Log("fbha:", alloc_res)

}
