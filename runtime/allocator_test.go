package runtime

import (
	"github.com/wasmerio/go-ext-wasm/wasmer"
	"testing"
)

func setOffset(mem wasmer.Memory, offset uint32) {
	// TODO actually implement this so that offset passed in is used to build array
	mem_values := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	copy(mem.Data()[0:len(mem_values) ], mem_values)
}

func TestAllocatorShouldAllocateProperly(t *testing.T) {
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

	t.Log("[allocator_test], should allocate properly", "result", alloc_res)
	if alloc_res != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", alloc_res, 8)
	}
}

func TestAllocatorShouldAlignPointersToMultiplesOf8(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	setOffset(mem, 13)
	fbha := newAllocator(&mem)
	alloc_res, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}

	t.Log("[allocator_test], should allign pointers to multiples of 8", "result", alloc_res)
	if alloc_res != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", alloc_res, 24)
	}
}

func TestAllocatorShouldIncrementPointersProperly(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	ptr1, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}
	ptr2, err := fbha.allocate(9 );
	if err != nil {
		t.Fatal(err)
	}
	ptr3, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[allocator_test], should increment pointers properly", "ptr1", ptr1, "ptr2", ptr2, "ptr3", ptr3)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}
	if ptr3 != 24 + 16 + 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr3, 24 + 16 + 8)
	}
}

func TestAllocatorShouldFreeProperly(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	ptr1, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	err = fbha.deallocate(ptr2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[allocator_test], head[0]", "head0", fbha.heads[0], "ptr2", ptr2 -8)
	if fbha.heads[0] != ptr2 -8 {
		t.Errorf("Error deallocate, head ptr not equal expected value")
	}
}

func TestAllocatorShouldDeallocateAndReallocateProperly(t *testing.T) {
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	ptr1, err := fbha.allocate(1 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(9 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	err = fbha.deallocate(ptr2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("[allocator_test], head[0]", "head0", fbha.heads[0], "ptr2", 0)
	if fbha.heads[0] != 0 {
		t.Errorf("Error deallocate, head ptr not equal expected value")
	}

	ptr3, err := fbha.allocate(9 );
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr3", ptr3)
	if ptr3 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr3, 24)
	}
	t.Log("heap", "heap", fbha.heads)
}
