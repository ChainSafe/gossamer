// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package multiaddr

import (
	libp2p "github.com/libp2p/go-libp2p/core"
)

// Multiaddr type used in Gossamer
type Multiaddr struct {
	libp2p.Multiaddr
}
