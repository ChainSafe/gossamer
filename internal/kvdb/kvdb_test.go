package kvdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_EndPrefix(t *testing.T) {
	require.Equal(t, []byte{5, 6, 8}, EndPrefix([]byte{5, 6, 7}))
	require.Equal(t, []byte{5, 7}, EndPrefix([]byte{5, 6, 255}))
	// This is not equal as the result is before start.
	require.NotEqual(t, []byte{5, 255}, EndPrefix([]byte{5, 255, 255}))
	// This is equal ([5, 255] will not be deleted because
	// it is before start).
	require.Equal(t, []byte{6}, EndPrefix([]byte{5, 255, 255}))
	require.Nil(t, EndPrefix([]byte{255, 255, 255}))

	require.Equal(t, []byte{0x01}, EndPrefix([]byte{0x00, 0xff}))
	require.Nil(t, EndPrefix([]byte{0xff}))
	require.Nil(t, EndPrefix([]byte{}))
	require.Equal(t, []byte("1"), EndPrefix([]byte("0")))
}
