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
