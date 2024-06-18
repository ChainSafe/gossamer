// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNumberIndexKey(t *testing.T) {
	var id uint64 = 72340207214430721
	_, err := newNumberIndexKey(id)
	assert.NotNil(t, err)

	id = uint64(math.MaxUint32)
	key, err := newNumberIndexKey(id)
	assert.Nil(t, err)
	assert.Equal(t, key, numberIndexKey{255, 255, 255, 255})

	id = uint64(9)
	key, err = newNumberIndexKey(id)
	assert.Nil(t, err)
	assert.Equal(t, key, numberIndexKey{0, 0, 0, 9})
}

func TestJoinInput(t *testing.T) {
	buf1 := []byte{1, 2, 3, 4}
	buf2 := []byte{5, 6, 7, 8}

	joined := joinInput(buf1, buf2)
	assert.Equal(t, joined, []byte{1, 2, 3, 4, 5, 6, 7, 8})
}
