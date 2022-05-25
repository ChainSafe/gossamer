// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newGenerator creates a new PRNG seeded with the
// unix nanoseconds value of the current time.
func newGenerator() (prng *rand.Rand) {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	return rand.New(source)
}

func generateRandBytes(t *testing.T, size int,
	generator *rand.Rand) (b []byte) {
	t.Helper()
	b = make([]byte, size)
	_, err := generator.Read(b)
	require.NoError(t, err)
	return b
}
