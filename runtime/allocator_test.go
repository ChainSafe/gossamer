package runtime

import (
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

const PAGE_SIZE = 65536

func TestAllocatorShouldAllocateProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem, 0)

	// when
	alloc_res, err := fbha.allocate(&mem, 1)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestAllocatorShouldAllocateProperly]", "result", alloc_res)
	if alloc_res != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", alloc_res, 8)
	}
}

func TestAllocatorShouldAlignPointersToMultiplesOf8(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	// set ptr_offset to simulate 13 bytes used
	fbha := newAllocator(&mem, 13)

	// when
	alloc_res, err := fbha.allocate(&mem, 1)
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
	fbha := newAllocator(&mem, 0)

	// when
	ptr1, err := fbha.allocate(&mem, 1)
	if err != nil {
		t.Fatal(err)
	}
	ptr2, err := fbha.allocate(&mem, 9)
	if err != nil {
		t.Fatal(err)
	}
	ptr3, err := fbha.allocate(&mem, 1)
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
	fbha := newAllocator(&mem, 0)
	ptr1, err := fbha.allocate(&mem, 1)
	if err != nil {
		t.Fatal(err)
	}
	// the prefix of 8 bytes is prepended to the pointer
	t.Log("ptr1", ptr1)
	if ptr1 != 8 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(&mem, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	// the prefix of 8 bytes + the content of ptr1 is prepended to the pointer
	if ptr2 != 24 {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	// when
	err = fbha.deallocate(&mem, ptr2)
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

func TestAllocatorShouldDeallocateAndReallocateProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	// test ptr_offset of 13, which should give is 16 for padding
	fbha := newAllocator(&mem, 13)
	padding_offset := 16
	ptr1, err := fbha.allocate(&mem, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr1", ptr1)
	if ptr1 != uint32(padding_offset+8) {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr1, 8)
	}

	ptr2, err := fbha.allocate(&mem, 9)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr2", ptr2)
	if ptr2 != uint32(padding_offset+16+8) {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr2, 24)
	}

	// when
	err = fbha.deallocate(&mem, ptr2)
	if err != nil {
		t.Fatal(err)
	}
	ptr3, err := fbha.allocate(&mem, 9)
	if err != nil {
		t.Fatal(err)
	}

	// then
	// should have re-allocated
	t.Log("ptr3", ptr3)
	if ptr3 != uint32(padding_offset+16+8) {
		t.Errorf("Returned ptr not correct, got: %d, want: %d.", ptr3, 24)
	}
	expected := make([]uint32, 22)
	if !reflect.DeepEqual(expected, fbha.heads[:]) {
		t.Error("ERROR: Didn't get expected heads")
	}
}

func TestAllocatorShouldBuildLinkedListOfFreeAreasProperly(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem, 0)

	ptr1, err := fbha.allocate(&mem, 8)
	if err != nil {
		t.Fatal(err)
	}

	ptr2, err := fbha.allocate(&mem, 8)
	if err != nil {
		t.Fatal(err)
	}

	ptr3, err := fbha.allocate(&mem,8)
	if err != nil {
		t.Fatal(err)
	}

	// when
	err = fbha.deallocate(&mem, ptr1)
	if err != nil {
		t.Fatal(err)
	}

	err = fbha.deallocate(&mem, ptr2)
	if err != nil {
		t.Fatal(err)
	}

	err = fbha.deallocate(&mem, ptr3)
	if err != nil {
		t.Fatal(err)
	}

	// then
	expected := make([]uint32, 22)
	expected[0] = ptr3 - 8
	if !reflect.DeepEqual(expected, fbha.heads[:]) {
		t.Error("ERROR: Didn't get expected heads")
	}

	ptr4, err := fbha.allocate(&mem, 8)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ptr3", ptr3)
	t.Log("ptr4", ptr4)
	if ptr3 != ptr4 {
		t.Errorf("Pointer values not equal")
	}

	expected[0] = ptr2 - 8
	if !reflect.DeepEqual(expected, fbha.heads[:]) {
		t.Error("ERROR: Didn't get expected heads")
	}
}

func TestShouldNotAllocateIfTooLarge(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	currentSize := mem.Length()

	fbha := newAllocator(&mem, 0)

	// when
	_, err = fbha.allocate(&mem, currentSize + 1)

	// then expect error since trying to over allocate
	if err == nil {
		t.Error("Error, expected out of space error, but didn't get one.")
	}
	if err != nil && err.Error() != "Error: allocator out of space" {
		t.Errorf("Error: got unexpected error: %v", err.Error())
	}
}

func TestShouldNotAllocateIfFull(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	currentSize := mem.Length()
	fbha := newAllocator(&mem, 0)

	ptr1, err := fbha.allocate(&mem,(currentSize / 2) - 8)
	if err != nil {
		t.Fatal(err)
	}
	if ptr1 != 8 {
		t.Errorf("Expected value of 8")
	}

	// when
	_, err = fbha.allocate(&mem,(currentSize / 2))

	// then
	// there is no room after half currentSize including it's 8 byte prefix, so error
	if err == nil {
		t.Error("Error, expected out of space error, but didn't get one.")
	}
	if err != nil && err.Error() != "Error: allocator out of space" {
		t.Errorf("Error: got unexpected error: %v", err.Error())
	}

}

func TestShouldAllocateMaxPossibleAllocationSize(t *testing.T) {
	// given, grow heap memory so that we have at least MAX_POSSIBLE_ALLOCATION available
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	pagesNeeded := (MAX_POSSIBLE_ALLOCATION / PAGE_SIZE) - (mem.Length() / PAGE_SIZE) + 1
	mem.Grow(pagesNeeded)

	fbha := newAllocator(&mem, 0)

	// when
	ptr1, err := fbha.allocate(&mem, MAX_POSSIBLE_ALLOCATION)
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
	fbha := newAllocator(&mem, 0)

	// when
	_, err = fbha.allocate(&mem,MAX_POSSIBLE_ALLOCATION + 1)

	// then
	if err != nil {
		if err.Error() != "Error: size to large" {
			t.Error("Didn't get expected error")
		}
	} else {
		t.Error("Error: Didn't get error but expected one.")
	}

}

func TestShouldIncludePrefixesInTotalHeapSize(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem, 1)

	// when
	_, err = fbha.allocate(&mem,9)
	if err != nil {
		t.Fatal(err)
	}
	// then
	t.Log("[TestShouldIncludePrefixesInTotalHeapSize]", "total_size", fbha.total_size)
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
	fbha := newAllocator(&mem, 13)

	// when
	ptr, err := fbha.allocate(&mem,42)
	if err != nil {
		t.Fatal(err)
	}
	if ptr != (16 + 8) {
		t.Error("Error: Didn't get expected pointer value")
	}
	err = fbha.deallocate(&mem, ptr)
	if err != nil {
		t.Fatal(err)
	}

	// then
	t.Log("[TestShouldColculateTotalHeapSizeToZero]", "heap total size", fbha.total_size)
	if fbha.total_size != 0 {
		t.Error("Total heap size does not equal zero, total_size: ", fbha.total_size)
	}

}

func TestShouldCalculateTotalSizeOfZero(t *testing.T) {
	// given
	runtime, err := newTestRuntime()
	if err != nil {
		t.Fatal(err)
	}
	mem := runtime.vm.Memory
	fbha := newAllocator(&mem, 19)

	// when
	for i := 0; i < 10; i++ {
		ptr, err := fbha.allocate(&mem,42)
		if err != nil {
			t.Fatal(err)
		}
		err = fbha.deallocate(&mem, ptr)
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
	if !reflect.DeepEqual(heap, []byte{1, 0, 0, 0, 0}) {
		t.Error("Error Write U32 to LE")
	}
}

func TestShouldWriteU32MaxCorrectlyIntoLe(t *testing.T) {
	// NOTE: we used the go's binary.LittleEndianPutUint32 function
	//  so this test isn't necessary, but is included for completeness

	//given
	heap := make([]byte, 5)

	// when
	binary.LittleEndian.PutUint32(heap, math.MaxUint32)

	//then
	if !reflect.DeepEqual(heap, []byte{255, 255, 255, 255, 0}) {
		t.Error("Error Write U32 MAX to LE")
	}
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
