// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

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
