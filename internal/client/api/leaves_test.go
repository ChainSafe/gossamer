package api

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/stretchr/testify/assert"
)

func TestLeafSet_ImportWorks(t *testing.T) {
	set := NewLeafSet[uint32, uint32]()
	set.Import(0, 0, 0)

	set.Import(1_1, 1, 0)
	set.Import(2_1, 2, 1_1)
	set.Import(3_1, 3, 2_1)

	assert.Equal(t, uint(1), set.Count())
	assert.True(t, set.Contains(3, 3_1))
	assert.False(t, set.Contains(2, 2_1))
	assert.False(t, set.Contains(1, 1_1))
	assert.False(t, set.Contains(0, 0))

	set.Import(2_2, 2, 1_1)
	set.Import(1_2, 1, 0)
	set.Import(2_3, 2, 1_2)

	assert.Equal(t, uint(3), set.Count())
	assert.True(t, set.Contains(3, 3_1))
	assert.True(t, set.Contains(2, 2_2))
	assert.True(t, set.Contains(2, 2_3))

	// Finally test the undo feature

	outcome := set.Import(2_4, 2, 1_1)
	assert.Equal(t, uint32(2_4), outcome.inserted.hash)
	assert.Nil(t, outcome.removed)
	assert.Equal(t, uint(4), set.Count())
	assert.True(t, set.Contains(2, 2_4))

	set.Undo().UndoImport(outcome)
	assert.Equal(t, uint(3), set.Count())
	assert.True(t, set.Contains(3, 3_1))
	assert.True(t, set.Contains(2, 2_2))
	assert.True(t, set.Contains(2, 2_3))

	outcome = set.Import(3_2, 3, 2_3)
	assert.Equal(t, uint32(3_2), outcome.inserted.hash)
	assert.Equal(t, uint32(2_3), *outcome.removed)
	assert.Equal(t, uint(3), set.Count())
	assert.True(t, set.Contains(3, 3_2))

	set.Undo().UndoImport(outcome)
	assert.Equal(t, uint(3), set.Count())
	assert.True(t, set.Contains(3, 3_1))
	assert.True(t, set.Contains(2, 2_2))
	assert.True(t, set.Contains(2, 2_3))
}

func newUint32(u uint32) *uint32 {
	return &u
}

func TestLeafSet_RemovalWorks(t *testing.T) {
	set := NewLeafSet[uint32, uint32]()
	set.Import(10_1, 10, 0)
	set.Import(11_1, 11, 10_1)
	set.Import(11_2, 11, 10_1)
	set.Import(12_1, 12, 11_1)

	outcome := set.Remove(12_1, 12, newUint32(11_1))
	assert.NotNil(t, outcome)
	assert.Equal(t, uint32(12_1), outcome.removed.hash)
	assert.Equal(t, newUint32(11_1), outcome.inserted)
	assert.Equal(t, uint(2), set.Count())
	assert.True(t, set.Contains(11, 11_1))
	assert.True(t, set.Contains(11, 11_2))

	outcome = set.Remove(11_1, 11, nil)
	assert.NotNil(t, outcome)
	assert.Equal(t, uint32(11_1), outcome.removed.hash)
	assert.Nil(t, outcome.inserted)
	assert.Equal(t, uint(1), set.Count())
	assert.True(t, set.Contains(11, 11_2))

	outcome = set.Remove(11_2, 11, newUint32(10_1))
	assert.NotNil(t, outcome)
	assert.Equal(t, uint32(11_2), outcome.removed.hash)
	assert.Equal(t, newUint32(10_1), outcome.inserted)
	assert.Equal(t, uint(1), set.Count())
	assert.True(t, set.Contains(10, 10_1))

	set.Undo().UndoRemove(*outcome)
	assert.Equal(t, uint(1), set.Count())
	assert.True(t, set.Contains(11, 11_2))
}

func TestLeafSet_FinalizationWorks(t *testing.T) {
	set := NewLeafSet[uint32, uint32]()
	set.Import(9_1, 9, 0)
	set.Import(10_1, 10, 9_1)
	set.Import(10_2, 10, 9_1)
	set.Import(11_1, 11, 10_1)
	set.Import(11_2, 11, 10_1)
	set.Import(12_1, 12, 11_2)

	outcome := set.FinalizeHeight(11)
	assert.Equal(t, uint(2), set.Count())
	assert.True(t, set.Contains(11, 11_1))
	assert.True(t, set.Contains(12, 12_1))
	assert.Equal(t, []uint32{10}, outcome.removed.Keys())
	assert.Equal(t, [][]uint32{{10_2}}, outcome.removed.Values())

	set.Undo().UndoFinalization(outcome)
	assert.Equal(t, uint(3), set.Count())
	assert.True(t, set.Contains(11, 11_1))
	assert.True(t, set.Contains(12, 12_1))
	assert.True(t, set.Contains(10, 10_2))
}

func TestLeafSet_FlushToDisk(t *testing.T) {
	var prefix = []byte("abcdefg")
	db := database.NewMemDB[hash.H256]()

	set := NewLeafSet[uint32, uint32]()
	set.Import(0, 0, 0)

	set.Import(1_1, 1, 0)
	set.Import(2_1, 2, 1_1)
	set.Import(3_1, 3, 2_1)

	var tx database.Transaction[hash.H256]

	set.PrepareTransaction(&tx, 0, prefix)
	assert.NoError(t, db.Commit(tx))

	set2, err := NewLeafSetFromDB[uint32, uint32](&db, 0, prefix)
	assert.NoError(t, err)
	assert.Equal(t, set, set2)
}

func TestLeafSet_TwoLeavesSameHeightCanBeIncluded(t *testing.T) {
	set := NewLeafSet[uint32, uint32]()

	set.Import(1_1, 10, 0)
	set.Import(1_2, 10, 0)

	_, ok := set.storage.Get(10)
	assert.True(t, ok)
	assert.True(t, set.Contains(10, 1_1))
	assert.True(t, set.Contains(10, 1_2))
	assert.False(t, set.Contains(10, 1_3))
}

func TestLeafSet_FinalizationConsistentWithDisk(t *testing.T) {
	var prefix = []byte("prefix")
	db := database.NewMemDB[hash.H256]()

	set := NewLeafSet[uint32, uint32]()
	set.Import(10_1, 10, 0)
	set.Import(11_1, 11, 10_2)
	set.Import(11_2, 11, 10_2)
	set.Import(12_1, 12, 11_123)

	assert.True(t, set.Contains(10, 10_1))

	var tx database.Transaction[hash.H256]
	set.PrepareTransaction(&tx, 0, prefix)
	fmt.Printf("%T%+v\n", tx, tx)
	assert.NoError(t, db.Commit(tx))

	set.FinalizeHeight(11)
	tx = database.Transaction[hash.H256]{}
	set.PrepareTransaction(&tx, 0, prefix)
	fmt.Printf("%T%+v\n", tx, tx)
	assert.NoError(t, db.Commit(tx))

	assert.True(t, set.Contains(11, 11_1))
	assert.True(t, set.Contains(11, 11_2))
	assert.True(t, set.Contains(12, 12_1))
	assert.False(t, set.Contains(10, 10_1))

	set2, err := NewLeafSetFromDB[uint32, uint32](&db, 0, prefix)
	assert.NoError(t, err)
	assert.Equal(t, set.storage.Keys(), set2.storage.Keys())
}
