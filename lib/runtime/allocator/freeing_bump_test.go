package allocator

import (
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
