// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

const (
	// NoNetworkRole runs a node without networking
	NoNetworkRole = byte(0)
	// FullNodeRole runs a full node
	FullNodeRole = byte(1)
	// LightClientRole runs a light client
	LightClientRole = byte(2)
	// AuthorityRole runs the node as a block-producing and finalising node
	AuthorityRole = byte(4)
)
