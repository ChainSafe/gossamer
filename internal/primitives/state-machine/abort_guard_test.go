// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"testing"
)

func TestNeverAbort(t *testing.T) {
	func() {
		guard := NewGuard(NeverAbort)
		defer guard.Done()
		panic("test-panic")
	}()
}

func TestUnwind(t *testing.T) {
	defer func() {
		if r := recover(); r != "test-unwind" {
			t.Fatalf("expected panic 'test-unwind', got %v", r)
		}
	}()
	guard := NewGuard(Unwind)
	defer guard.Done()
	panic("test-unwind")
}
