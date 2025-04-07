// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

type CallContext uint8

const (
	CallContextOffchain CallContext = iota
	CallContextOnchain
)
