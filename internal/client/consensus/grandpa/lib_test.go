// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

type dummyAuthID uint

func (dummyAuthID) Verify(_ []byte, _ []byte) (bool, error) {
	return true, nil
}

type dummyInvalidAuthID uint

func (dummyInvalidAuthID) Verify(_ []byte, _ []byte) (bool, error) {
	return false, nil
}

func Test_checkMessageSignature(t *testing.T) {
	pubKeyValid := dummyAuthID(1)
	pubKeyInvalid := dummyInvalidAuthID(1)

	message := grandpa.Message[string, uint]{
		Value: 4,
	}

	msg := messageData[string, uint]{
		1,
		2,
		message,
	}

	// Dummy signature
	encMsg, err := scale.Marshal(msg)
	require.NoError(t, err)

	valid, err := checkMessageSignature[string, uint, dummyAuthID](message, pubKeyValid, encMsg, 1, 2)
	require.NoError(t, err)
	require.True(t, valid)

	invalid, err := checkMessageSignature[string, uint, dummyInvalidAuthID](message, pubKeyInvalid, encMsg, 1, 2)
	require.NoError(t, err)
	require.False(t, invalid)
}
