package allocator

// TODO: missing test should_read_and_write_u64_correctly

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPagesFromSize(t *testing.T) {
	cases := []struct {
		size          uint64
		expectedPages uint32
	}{
		{0, 0},
		{1, 1},
		{65536, 1},
		{65536 + 1, 2},
		{65536 * 2, 2},
		{65536*2 + 1, 3},
	}

	for _, tt := range cases {
		pages, ok := pagesFromSize(tt.size)
		require.True(t, ok)
		require.Equal(t, tt.expectedPages, pages)
	}
}

func TestShouldAllocatePropertly(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr, err := heap.Allocate(mem, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(HeaderSize), ptr)
}

func TestShouldAlwaysAlignPointerToMultiplesOf8(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(13)

	ptr, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	// the pointer must start at the next multiple of 8 from 13
	// + the prefix of 8 bytes.
	require.Equal(t, uint32(24), ptr)
}

func TestShouldIncrementPointersProperly(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr1, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	ptr2, err := heap.Allocate(mem, 9)
	require.NoError(t, err)

	ptr3, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	// a prefix of 8 bytes is prepended to each pointer
	require.Equal(t, uint32(HeaderSize), ptr1)

	// the prefix of 8 bytes + the content of ptr1 padded to the lowest possible
	// item size of 8 bytes + the prefix of ptr1
	require.Equal(t, uint32(24), ptr2)

	// ptr2 + its content of 16 bytes + the prefix of 8 bytes
	require.Equal(t, uint32(24+16+HeaderSize), ptr3)
}

func TestShouldFreeProperly(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr1, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	// the prefix of 8 bytes is prepend to the pointer
	require.Equal(t, uint32(HeaderSize), ptr1)

	ptr2, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	// the prefix of 8 bytes + the content of ptr 1 is prepended to the ptr
	require.Equal(t, uint32(24), ptr2)

	err = heap.Deallocate(mem, ptr2)
	require.NoError(t, err)

	// the heads table should contain a pointer to the prefix of ptr2 in the leftmost entry
	link := heap.freeLists.heads[0]
	expectedLink := Ptr{headerPtr: ptr2 - HeaderSize}
	require.Equal(t, expectedLink, link)
}

func TestShouldDeallocateAndReallocateProperly(t *testing.T) {
	const paddedOffset = 16
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(13)

	ptr1, err := heap.Allocate(mem, 1)
	require.NoError(t, err)

	// the prefix of 8 bytes is prepended to the pointer
	require.Equal(t, uint32(paddedOffset+HeaderSize), ptr1)

	ptr2, err := heap.Allocate(mem, 9)
	require.NoError(t, err)

	// the padded offset + the prev allocated ptr (8 bytes prefix + 8 bytes content)
	// + the prefix of 8 bytes which is prepend to the current pointer
	require.Equal(t, uint32(paddedOffset+16+HeaderSize), ptr2)

	// deallocate and reallocate
	err = heap.Deallocate(mem, ptr2)
	require.NoError(t, err)

	ptr3, err := heap.Allocate(mem, 9)
	require.NoError(t, err)

	require.Equal(t, uint32(paddedOffset+16+HeaderSize), ptr3)
	var expectedHeads [23]Link
	for i := range expectedHeads {
		expectedHeads[i] = Nil{}
	}
	// should have re-allocated
	require.Equal(t, heap.freeLists.heads, expectedHeads)
}

func TestShouldBuildLinkedListOfFreeAreasProperly(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	// given
	ptr1, err := heap.Allocate(mem, 8)
	require.NoError(t, err)

	ptr2, err := heap.Allocate(mem, 8)
	require.NoError(t, err)

	ptr3, err := heap.Allocate(mem, 8)
	require.NoError(t, err)

	// when
	err = heap.Deallocate(mem, ptr1)
	require.NoError(t, err)

	err = heap.Deallocate(mem, ptr2)
	require.NoError(t, err)

	err = heap.Deallocate(mem, ptr3)
	require.NoError(t, err)

	//then
	require.Equal(t, Ptr{headerPtr: ptr3 - HeaderSize}, heap.freeLists.heads[0])

	// reallocate
	ptr4, err := heap.Allocate(mem, 8)
	require.NoError(t, err)
	require.Equal(t, ptr3, ptr4)

	require.Equal(t, Ptr{headerPtr: ptr2 - HeaderSize}, heap.freeLists.heads[0])
}

func TestShouldNotAllocIfTooLarge(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	mem.setMaxWasmPages(1)

	heap := NewFreeingBumpHeapAllocator(13)

	ptr, err := heap.Allocate(mem, PageSize-13)
	require.Zero(t, ptr)
	require.ErrorIs(t, err, ErrCannotGrowLinearMemory)
}

func TestShouldNotAllocateIfFull(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	mem.setMaxWasmPages(1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr1, err := heap.Allocate(mem, (PageSize/2)-HeaderSize)
	require.NoError(t, err)
	require.Equal(t, uint32(HeaderSize), ptr1)

	// there is no room for another half page incl. its 8 byte prefix
	ptr2, err := heap.Allocate(mem, PageSize/2)
	require.Zero(t, ptr2)
	require.ErrorIs(t, err, ErrCannotGrowLinearMemory)
}

func TestShouldAllocateMaxPossibleAllocationSize(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr, err := heap.Allocate(mem, MaxPossibleAllocations)
	require.NoError(t, err)
	require.Equal(t, uint32(HeaderSize), ptr)
}

func TestShouldNotAllocateIfRequestedSizeIsTooLarge(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr, err := heap.Allocate(mem, MaxPossibleAllocations+1)
	require.Zero(t, ptr)
	require.ErrorIs(t, err, ErrRequestedAllocationTooLarge)
}

func TestShouldReturnErrorWhenBumperGreaterThanHeapSize(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	mem.setMaxWasmPages(1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptrs := make([]uint32, 0)
	for idx := 0; idx < (PageSize / 40); idx++ {
		ptr, err := heap.Allocate(mem, 32)
		require.NoError(t, err)

		ptrs = append(ptrs, ptr)
	}

	require.Equal(t, uint32(PageSize-16), heap.stats.bytesAllocated)
	require.Equal(t, uint32(PageSize-16), heap.bumper)

	for _, ptr := range ptrs {
		err := heap.Deallocate(mem, ptr)
		require.NoError(t, err)
	}

	require.Zero(t, heap.stats.bytesAllocated)
	require.Equal(t, uint32(PageSize-16), heap.stats.bytesAllocatedPeak)
	require.Equal(t, uint32(PageSize-16), heap.bumper)

	// Allocate another 8 byte to use the full heap
	_, err := heap.Allocate(mem, 8)
	require.NoError(t, err)

	// the `bumper` value is equal to `size` here and any
	// further allocation which would increment the bumper must fail.
	// we try to allocate 8 bytes here, which will increment the
	// bumper since no 8 byte item has been freed before.
	require.Equal(t, heap.bumper, mem.Size())

	ptr, err := heap.Allocate(mem, 8)
	require.Zero(t, ptr)
	require.ErrorIs(t, err, ErrCannotGrowLinearMemory)
}

func TestShouldIncludePrefixesInTotalHeapSize(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(1)

	ptr, err := heap.Allocate(mem, 9)
	require.NoError(t, err)
	require.NotZero(t, ptr)

	require.Equal(t, uint32(HeaderSize+16), heap.stats.bytesAllocated)
}

func TestShouldCalculateTotalHeapSizeToZero(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(13)

	ptr, err := heap.Allocate(mem, 42)
	require.NoError(t, err)
	require.Equal(t, uint32(16+HeaderSize), ptr)

	err = heap.Deallocate(mem, ptr)
	require.NoError(t, err)

	require.Zero(t, heap.stats.bytesAllocated)
}

func TestShouldCalculateTotalSizeOfZero(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(19)

	for idx := 1; idx < 10; idx++ {
		ptr, err := heap.Allocate(mem, 42)
		require.NoError(t, err)
		require.NotZero(t, ptr)

		err = heap.Deallocate(mem, ptr)
		require.NoError(t, err)
	}

	require.Zero(t, heap.stats.bytesAllocated)
}

func TestShouldGetItemSizeFromOrder(t *testing.T) {
	rawOrder := 0
	order, err := orderFromRaw(uint32(rawOrder))
	require.NoError(t, err)
	require.Equal(t, order.size(), uint32(8))
}

func TestShouldGetMaxItemSizeFromIndex(t *testing.T) {
	rawOrder := 22
	order, err := orderFromRaw(uint32(rawOrder))
	require.NoError(t, err)
	require.Equal(t, order.size(), uint32(MaxPossibleAllocations))
}

func TestDeallocateNeedsToMaintainLinkedList(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	// allocate and free some pointers
	ptrs := make([]uint32, 4)
	for idx := range ptrs {
		ptr, err := heap.Allocate(mem, 8)
		require.NoError(t, err)
		require.NotZero(t, ptr)
		ptrs[idx] = ptr
	}
}

func TestHeaderReadWrite(t *testing.T) {
	roundtrip := func(h Header) {
		mem := NewMemoryInstanceWithPages(t, 1)
		writeHeaderInto(h, mem, 0)

		readHeader, err := readHeaderFromMemory(mem, 0)
		require.NoError(t, err)

		require.Equal(t, h, readHeader)
	}

	roundtrip(Occupied{order: Order(0)})
	roundtrip(Occupied{order: Order(1)})
	roundtrip(Free{link: Nil{}})
	roundtrip(Free{link: Ptr{headerPtr: 0}})
	roundtrip(Free{link: Ptr{headerPtr: 4}})
}

func TestPoisonOOM(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	mem.setMaxWasmPages(1)

	heap := NewFreeingBumpHeapAllocator(0)

	alloc_ptr, err := heap.Allocate(mem, PageSize/2)
	require.NoError(t, err)
	require.NotZero(t, alloc_ptr)

	ptr2, err := heap.Allocate(mem, PageSize)
	require.Zero(t, ptr2)
	require.ErrorIs(t, err, ErrCannotGrowLinearMemory)

	require.True(t, heap.poisoned)

	err = heap.Deallocate(mem, alloc_ptr)
	require.Error(t, err, ErrAllocatorPoisoned)
}

func TestNOrders(t *testing.T) {
	// Test that N_ORDERS is consistent with min and max possible allocation.
	require.Equal(t,
		MinPossibleAllocations*uint32(math.Pow(2, float64(NumOrders-1))),
		MaxPossibleAllocations)
}

func TestAcceptsGrowingMemory(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	ptr1, err := heap.Allocate(mem, PageSize/2)
	require.NoError(t, err)
	require.NotZero(t, ptr1)

	ptr2, err := heap.Allocate(mem, PageSize/2)
	require.NoError(t, err)
	require.NotZero(t, ptr2)

	_, ok := mem.Grow(1)
	require.True(t, ok)

	ptr3, err := heap.Allocate(mem, PageSize/2)
	require.NoError(t, err)
	require.NotZero(t, ptr3)
}

func TestDoesNotAcceptShrinkingMemory(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 2)
	heap := NewFreeingBumpHeapAllocator(0)
	ptr, err := heap.Allocate(mem, PageSize/2)
	require.NoError(t, err)
	require.NotZero(t, ptr)

	truncatedMem := make([]byte, PageSize)
	copy(truncatedMem, mem.data[:PageSize])
	mem.data = truncatedMem

	ptr2, err := heap.Allocate(mem, PageSize/2)
	require.Zero(t, ptr2)
	require.ErrorIs(t, err, ErrMemoryShrinked)
}

func TestShouldGrowMemoryWhenRunningOutOfSpace(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)

	require.Equal(t, uint32(1), mem.pages())
	ptr, err := heap.Allocate(mem, PageSize*2)
	require.NoError(t, err)
	require.NotZero(t, ptr)
	require.Equal(t, uint32(3), mem.pages())
}

func TestModifyingHeaderLeadsToAnError(t *testing.T) {
	mem := NewMemoryInstanceWithPages(t, 1)
	heap := NewFreeingBumpHeapAllocator(0)
	ptr, err := heap.Allocate(mem, 5)
	require.NoError(t, err)
	require.NotZero(t, ptr)

	err = heap.Deallocate(mem, ptr)
	require.NoError(t, err)

	header := Free{link: Ptr{headerPtr: math.MaxUint32 - 1}}
	err = writeHeaderInto(header, mem, ptr-HeaderSize)
	require.NoError(t, err)

	ptr2, err := heap.Allocate(mem, 5)
	require.NoError(t, err)
	require.NotZero(t, ptr2)

	ptr3, err := heap.Allocate(mem, 5)
	require.Zero(t, ptr3)
	require.ErrorIs(t, err, ErrInvalidHeaderPointerDetected)
}
