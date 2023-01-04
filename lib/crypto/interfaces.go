// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto

// Erroer logs formatted messages at the error level.
type Erroer interface {
	Errorf(format string, args ...interface{})
}
