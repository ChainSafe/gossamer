// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import ma "github.com/multiformats/go-multiaddr"

// Health is network information about host needed for the rpc server
type Health struct {
	Peers           int
	IsSyncing       bool
	ShouldHavePeers bool
}

// NetworkState is network information about host needed for the rpc server and the runtime
type NetworkState struct {
	PeerID     string
	Multiaddrs []ma.Multiaddr
}

// PeerInfo is network information about peers needed for the rpc server
type PeerInfo struct {
	PeerID     string
	Roles      byte
	BestHash   Hash
	BestNumber uint64
}
