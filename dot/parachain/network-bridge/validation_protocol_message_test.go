package networkbridge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeValidationHandshake(t *testing.T) {
	t.Parallel()

	testHandshake := &validationHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeValidationHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}
