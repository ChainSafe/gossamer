// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestDirtyKeysSetsPop(t *testing.T) {
	set1 := btree.Set[int]{}
	set1.Insert(1)
	set1.Insert(2)
	set1.Insert(3)

	set2 := btree.Set[int]{}
	set2.Insert(4)
	set2.Insert(5)
	set2.Insert(6)

	dirtyKeys := DirtyKeysSets[int]{
		set1,
		set2,
	}

	reverseSets := []btree.Set[int]{set2, set1}

	for i := 0; i < len(reverseSets); i++ {
		last, has := dirtyKeys.Pop()
		if !has {
			break
		}
		require.Equal(t, last, reverseSets[i])
	}
}
