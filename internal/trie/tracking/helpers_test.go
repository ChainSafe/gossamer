// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPointer(x interface{}) (pointer uintptr, ok bool) {
	func() {
		defer func() {
			ok = recover() == nil
		}()
		valueOfX := reflect.ValueOf(x)
		pointer = valueOfX.Pointer()
	}()
	return pointer, ok
}

func assertPointersNotEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	pointerA, okA := getPointer(a)
	pointerB, okB := getPointer(b)
	require.Equal(t, okA, okB)

	switch {
	case pointerA == 0 && pointerB == 0: // nil and nil
	case okA:
		assert.NotEqual(t, pointerA, pointerB)
	default: // values like `int`
	}
}
