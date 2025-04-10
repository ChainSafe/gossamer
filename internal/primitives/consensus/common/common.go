// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

type BlockOrigin uint

const (
	NetworkInitialSync BlockOrigin = iota
	NetworkBroadcast
)
