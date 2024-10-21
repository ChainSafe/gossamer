// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LeftNibbles_compare(t *testing.T) {
	n := NewLeftNibbles([]byte("gossamer"))
	m := NewLeftNibbles([]byte("gossamerGossamer"))

	assert.Equal(t, -1, n.compare(m))
	assert.Equal(t, 1, m.compare(n))
	assert.Equal(t, 0, n.compare(m.Truncate(16)))

	truncated := n.Truncate(1)
	assert.Equal(t, -1, truncated.compare(n))
	assert.Equal(t, 1, n.compare(truncated))
	assert.Equal(t, -1, truncated.compare(m))
}

func Test_LeftNibbles_StartsWith(t *testing.T) {
	a := NewLeftNibbles([]byte("polkadot"))
	b := NewLeftNibbles([]byte("go"))
	b.len = 1
	assert.Equal(t, false, a.StartsWith(b))
}
