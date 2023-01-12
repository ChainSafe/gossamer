// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

type ephemeralService interface {
	Run() error
	Stop() error
}
