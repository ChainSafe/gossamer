package runtime

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/wasmerio/go-ext-wasm/wasmer"
)

const PAGE_SIZE = 65536

func setOffset(mem wasmer.Memory, offset uint32) {
	mem_vals := make([]byte, offset)
	for i := 0; i < len(mem_vals); i++ {
		mem_vals[i] = 0xff
	}
	copy(mem.Data()[0:len(mem_vals)], mem_vals)
}

func TestAllocatorShouldAllocateProperly(t *testing.T) {
	// give
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	// when
	alloc_res, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestAllocatorShouldAllocateProperly]", "result", alloc_res)
	if alloc_res != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", alloc_res, 8)
	}
}

// todo discuss how we want to handle offset
func TestAllocatorShouldAlignPointersToMultiplesOf8(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	setOffset(mem, 13)
	fbha := newAllocator(&mem)

	// when
	alloc_res, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestAllocatorShouldAlignPointersToMultiplesOf8]", "result", alloc_res)
	if alloc_res != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", alloc_res, 24)
	}
}

func TestAllocatorShouldIncrementPointersProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	// when
	ptr1, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	ptr2, err := fbha.allocate(9)
	if err != nil {
		t.Fatal(err)
	}
	ptr3, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestAllocatorShouldIncrementPointersProperly]", "ptr1", ptr1, "ptr2", ptr2, "ptr3", ptr3)
	// a prefix of 8 bytes is prepended to each pointer
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}
	// the prefix of 8 bytes + the content of ptr1 padded to the lowest possible
	// item size of 8 bytes + the prefix of ptr1
	if ptr2 != 8+16 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}
	// ptr2 + its content of 16 bytes + the prefix of 8 bytes
	if ptr3 != 24+16+8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr3, 24+16+8)
	}
}

func TestAllocatorShouldFreeProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	ptr1, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	// the prefix of 8 bytes is prepended to the pointer
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	// the prefix of 8 bytes + the content of ptr1 is prepended to the pointer
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	// when
	err = fbha.deallocate(ptr2)
	if err != nil {
		t.Fatal(err)
	}

	// then
	// then the heads table should contain a pointer to the prefix of ptr2 in the leftmost entry
	t.Log("[TestAllocatorShouldFreeProperly]", "head0", fbha.heads[0], "ptr2", ptr2-8)
	if fbha.heads[0] != ptr2-8 {
		t.Errorf("Error deallocate, head ptr not equal expected value")
	}
}

// todo we're NOT testing offset in this, discuss
func TestAllocatorShouldDeallocateAndReallocateProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)
	ptr1, err := fbha.allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(9)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	// when
	err = fbha.deallocate(ptr2)
	if err != nil {
		t.Fatal(err)
	}
	ptr3, err := fbha.allocate(9)
	if err != nil {
		t.Fatal(err)
	}

	// then
	// should have re-allocated
	t.Log("ptr3", ptr3)
	if ptr3 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr3, 24)
	}
	// TODO find way to compare head results to expected results
	t.Log("[TestAllocatorShouldDeallocateAndReallocateProperly]", "heads", fbha.heads)
}

func TestAllocatorShouldBuildLinkedListOfFreeAreasProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	ptr1, err := fbha.allocate(8)
	if err != nil {
		t.Fatal(err)
	}

	ptr2, err := fbha.allocate(8)
	if err != nil {
		t.Fatal(err)
	}

	ptr3, err := fbha.allocate(8)
	if err != nil {
		t.Fatal(err)
	}

	// when
	err = fbha.deallocate(ptr1)
	if err != nil {
		t.Fatal(err)
	}

	err = fbha.deallocate(ptr2)
	if err != nil {
		t.Fatal(err)
	}

	err = fbha.deallocate(ptr3)
	if err != nil {
		t.Fatal(err)
	}

	// then
	//expected := make([]uint32, 22)
	//expected[0] = ptr3 - 8
	expected := []uint32{ptr3 - 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	// TODO check slices are equal
	t.Log("[TestAllocatorShouldBuildLinkedListOfFreeAreasProperly], heads", fbha.heads)
	t.Log("[TestAllocatorShouldBuildLinkedListOfFreeAreasProperly], expected", expected)

	ptr4, err := fbha.allocate(8)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr3", ptr3)
	t.Log("ptr4", ptr4)
	if ptr3 != ptr4 {
		t.Errorf("Pointer values not equal")
	}

	expected[0] = ptr2 - 8
	// TODO check slices are equal
	t.Log("[TestAllocatorShouldBuildLinkedListOfFreeAreasProperly], heads   ", fbha.heads)
	t.Log("[TestAllocatorShouldBuildLinkedListOfFreeAreasProperly], expected", expected)
}

// todo discuss, wasm memory implementation issues regarding setup
func TestShouldNotAllocateIfTooLarge(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory

	fbha := newAllocator(&mem)

	// when
	ptr, err := fbha.allocate(PAGE_SIZE)

	// then
	if err != nil {
		// TODO check that correct error was returned
		t.Fatal(err)
	}
	t.Log("[TestShouldNotAllocateIfTooLarge]", "ptr", ptr)
}

func TestShouldNotAllocateIfFull(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	ptr1, err := fbha.allocate((PAGE_SIZE / 2) - 8)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Expected value of 8")
	}

	// when
	ptr2, err := fbha.allocate((PAGE_SIZE / 2))
	t.Log("ptr2", ptr2)

	// then
	if err != nil {
		t.Error(err)
	}

}

func TestShouldAllocateMaxPossibleAllocationSize(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	// when
	ptr1, err := fbha.allocate(MAX_POSSIBLE_ALLOCATION)
	if err != nil {
		t.Error(err)
	}

	//then
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Expected value of 8")
	}
}

func TestShouldNotAllocateIfRequestSizeTooLarge(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem)

	// when
	_, err = fbha.allocate(MAX_POSSIBLE_ALLOCATION + 1)
	// then
	if err != nil {
		if err.Error() != "Error: size to large" {
			t.Error("Didn't get expected error")
		}
	}

}

func TestShouldIncludePrefixesInTotalHeapSize(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	setOffset(mem, 1)
	fbha := newAllocator(&mem)

	// when
	_, err = fbha.allocate(9)
	if err != nil {
		t.Fatal(err)
	}
	// then
	t.Log("[TestShouldIncludePrefixesInTotalHeapSize]", "tetal_size", fbha.total_size)
	if fbha.total_size != (8 + 16) {
		t.Error("Total heap size not calculating properly")
	}

}

func TestShouldCalculateTotalHeapSizeToZero(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	setOffset(mem, 13)
	fbha := newAllocator(&mem)

	// when
	ptr, err := fbha.allocate(42)
	if err != nil {
		t.Fatal(err)
	}
	err = fbha.deallocate(ptr)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestShouldColculateTotalHeapSizeToZero]", "heap total size", fbha.total_size)
	if fbha.total_size != 0 {
		t.Error("Total heap size does not equal zero, total_size: ", fbha.total_size)
	}

}

func TestShouldColculateTotalSizeOfZero(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	setOffset(mem, 13)
	fbha := newAllocator(&mem)

	// when
	for i := 0; i < 10; i++ {
		ptr, err := fbha.allocate(42)
		if err != nil {
			t.Fatal(err)
		}
		err = fbha.deallocate(ptr)
		if err != nil {
			t.Fatal(err)
		}
	}

	// then
	t.Log("[TestShouldColculateTotalHeapSizeToZero]", "heap total size", fbha.total_size)
	if fbha.total_size != 0 {
		t.Error("Total heap size does not equal zero, total_size: ", fbha.total_size)
	}

}

func TestShouldWriteU32CorrectlyIntoLe(t *testing.T) {
	// NOTE: we used the go's binary.LittleEndianPutUint32 function
	//  so this test isn't necessary, but is included for completeness

	//given
	heap := make([]byte, 5)

	// when
	binary.LittleEndian.PutUint32(heap, 1)

	//then
	// TODO find way to conpare slices
	t.Log("[TestShouldWriteU32CorrectlyIntoLe]", "heap", heap)

}

func TestShouldWriteU32MaxCorrectlyIntoLe(t *testing.T) {
	// NOTE: we used the go's binary.LittleEndianPutUint32 function
	//  so this test isn't necessary, but is included for completeness

	//given
	heap := make([]byte, 5)

	// when
	binary.LittleEndian.PutUint32(heap, math.MaxUint32)

	//then
	// TODO find way to conpare slices
	t.Log("[TestShouldWriteU32MaxCorrectlyIntoLe]", "heap", heap)
}

func TestShouldGetItemFromIndex(t *testing.T) {
	// given
	index := uint(0)

	// when
	item_size := get_item_size_from_index(index)

	//
	t.Log("[TestShouldGetItemFromIndex]", "item_size", item_size)
	if item_size != 8 {
		t.Error("item_size should be 8, got item_size:", item_size)
	}
}

func TestShouldGetMaxFromIndex(t *testing.T) {
	// given
	index := uint(21)

	// when
	item_size := get_item_size_from_index(index)

	//
	t.Log("[TestShouldGetMaxFromIndex]", "item_size", item_size)
	if item_size != MAX_POSSIBLE_ALLOCATION {
		t.Errorf("item_size should be %d, got item_size: %d", MAX_POSSIBLE_ALLOCATION, item_size)
	}
}
