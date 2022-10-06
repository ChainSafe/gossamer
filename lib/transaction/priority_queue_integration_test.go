// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_PopWithTimer(t *testing.T) {
	pq := NewPriorityQueue()
	slotTimer := time.NewTimer(time.Second)

	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 2},
		},
	}

	expected := []int{3, 1, 2, 4, 0}

	for _, test := range tests {
		pq.Push(test)
	}

	counter := 0
	for {
		txn := pq.PopWithTimer(slotTimer.C)
		if txn == nil {
			break
		}
		assert.Equal(t, tests[expected[counter]], txn)
		counter++
	}
}
