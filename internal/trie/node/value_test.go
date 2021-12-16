// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Branch_GetValue(t *testing.T) {
	t.Parallel()

	branch := &Branch{
		Value: []byte{2},
	}
	value := branch.GetValue()
	assert.Equal(t, []byte{2}, value)
}

func Test_Leaf_GetValue(t *testing.T) {
	t.Parallel()

	leaf := &Leaf{
		Value: []byte{2},
	}
	value := leaf.GetValue()
	assert.Equal(t, []byte{2}, value)
}
