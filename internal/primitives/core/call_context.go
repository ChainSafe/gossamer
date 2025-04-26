// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

// The context in which a call is done.
// Depending on the context the executor may chooses different kind of heap sizes for the runtime
// instance.
type CallContext uint8

const (
	// The call is happening in some offchain context
	CallContextOffchain CallContext = iota
	// The call is happening in some on-chain context like building or importing a block.
	CallContextOnchain
)
