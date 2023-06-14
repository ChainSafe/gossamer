package parachaininteraction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeCollationHandshake(t *testing.T) {
	t.Parallel()

	testHandshake := &collatorHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeCollatorHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}
