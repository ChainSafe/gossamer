// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/stretchr/testify/assert"
)

func TestChildrenWriteReadRemove(t *testing.T) {
	prefix := []byte("children")
	db := database.NewMemDB[hash.H256]()

	tx := database.Transaction[hash.H256]{}
	children1 := []int32{1_3, 1_5}
	writeChildren(&tx, 0, prefix, 1_1, children1)

	children2 := []int32{1_4, 1_6}
	writeChildren(&tx, 0, prefix, 1_2, children2)

	err := db.Commit(tx)
	assert.NoError(t, err)

	r1, err := readChildren[int32](db, 0, prefix, 1_1)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1_3, 1_5}, r1)
	r2, err := readChildren[int32](db, 0, prefix, 1_2)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1_4, 1_6}, r2)

	removeChildren[int32](&tx, 0, prefix, 1_2)
	err = db.Commit(tx)
	assert.NoError(t, err)

	r1, err = readChildren[int32](db, 0, prefix, 1_1)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1_3, 1_5}, r1)
	r2, err = readChildren[int32](db, 0, prefix, 1_2)
	assert.NoError(t, err)
	assert.Nil(t, r2)
}
