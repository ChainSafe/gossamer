// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

// Infoer logs strings at the info level.
type Infoer interface {
	Info(s string)
}
