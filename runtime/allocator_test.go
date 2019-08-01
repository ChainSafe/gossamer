package runtime

import "testing"

func TestAllocator(t *testing.T) {
	t.Log("testing Allocator")
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}

	fbha := FreeingBumpHeapAllocator{bumper: 2}
	alloc_res := fbha.allocate(2);
	t.Log("fbha:", alloc_res)
	mem := runtime.vm.Memory.Data()
	t.Log("mem", mem[:10])
}
