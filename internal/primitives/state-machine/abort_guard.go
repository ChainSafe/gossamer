// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"log"
	"os"
)

type PanicMode int

const (
	// force immediate exit without unwinding
	Abort PanicMode = iota
	// allow panic to unwind normally
	Unwind
	// catch and suppress panic
	NeverAbort
)

type AbortGuard struct {
	mode PanicMode
}

// NewGuard creates a guard with the specified panic mode.
func NewGuard(mode PanicMode) *AbortGuard {
	return &AbortGuard{mode: mode}
}

// Done is the RAII-style Drop for the guard. It runs in a deferred call.
// If a panic occurred, recover() returns it and we act based on mode:
//   - Abort    => os.Exit(1)        (abort without unwind)
//   - Unwind   => re-panic(r)       (unwind stack normally)
//   - NeverAbort => suppress the panic and continue execution
func (g *AbortGuard) Done() {
	if r := recover(); r != nil {
		switch g.mode {
		case Abort:
			log.Printf("panic caught, aborting: %v", r)
			os.Exit(1)
		case Unwind:
			// re-panic to allow normal unwinding
			panic(r)
		case NeverAbort:
			log.Printf("panic caught and suppressed: %v", r)
			// do not re-panic; execution continues
		}
	}
}
