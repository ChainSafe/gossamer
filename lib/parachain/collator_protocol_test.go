// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

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
