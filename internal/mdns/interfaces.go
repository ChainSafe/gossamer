// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mdns

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Logger is a logger interface for the mDNS service.
type Logger interface {
	Debugf(format string, args ...any)
	Warnf(format string, args ...any)
}

// IDNetworker can return the peer ID and a network interface.
type IDNetworker interface {
	ID() peer.ID
	Networker
}

// Networker can return a network interface.
type Networker interface {
	Network() network.Network
}

// interfaceListenAddressesGetter returns the listen addresses of the interfaces.
type interfaceListenAddressesGetter interface {
	InterfaceListenAddresses() ([]multiaddr.Multiaddr, error)
}
